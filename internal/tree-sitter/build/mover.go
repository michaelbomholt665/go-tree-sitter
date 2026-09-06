package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type Mover struct {
	cleaner   *Cleaner
	validator ArtifactValidator
	reporter  _jsii.Reporter
}

// NewMover constructs a Mover.  The stdout and stderr writers are kept for
// backward-compatibility; a TextReporter is derived from stdout.  Prefer
// NewMoverWithReporter for new code.
func NewMover(_ Clock, cleaner *Cleaner, stdout, stderr io.Writer) *Mover {
	if stdout == nil {
		stdout = io.Discard
	}
	return NewMoverWithValidatorAndReporter(
		cleaner,
		NewNativeValidator(),
		_jsii.NewTextReporter(stdout, false),
	)
}

// NewMoverWithValidator is the existing constructor preserved for test
// compatibility.
func NewMoverWithValidator(cleaner *Cleaner, validator ArtifactValidator, stdout, stderr io.Writer) *Mover {
	if stdout == nil {
		stdout = io.Discard
	}
	return NewMoverWithValidatorAndReporter(cleaner, validator, _jsii.NewTextReporter(stdout, false))
}

// NewMoverWithReporter constructs a Mover that reports progress via r.
func NewMoverWithReporter(cleaner *Cleaner, r _jsii.Reporter) *Mover {
	return NewMoverWithValidatorAndReporter(cleaner, NewNativeValidator(), r)
}

// NewMoverWithValidatorAndReporter is the fully-specified constructor.
func NewMoverWithValidatorAndReporter(cleaner *Cleaner, validator ArtifactValidator, r _jsii.Reporter) *Mover {
	if cleaner == nil {
		cleaner = NewCleaner()
	}
	if validator == nil {
		validator = NewNativeValidator()
	}
	if r == nil {
		r = _jsii.SilentReporter{}
	}
	return &Mover{cleaner: cleaner, validator: validator, reporter: r}
}

type stagedLanguage struct {
	language   _jsii.Language
	binaries   []string
	provenance map[string]BinaryProvenance
	nodeTypes  bool
	queries    bool
}

func (m *Mover) Move(ctx context.Context, cfg *_jsii.Config, req _jsii.MoveRequest) error {
	if !cfg.Output.GenerateManifest {
		return errors.New("publication requires a validated manifest; generate_manifest=false is no longer supported")
	}
	languages, err := cfg.ResolveLanguages(req.Language)
	if err != nil {
		return err
	}
	base, err := filepath.Abs(cfg.Output.GrammarBase)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(base), 0o755); err != nil {
		return fmt.Errorf("create output parent: %w", err)
	}
	stage, err := os.MkdirTemp(filepath.Dir(base), ".grammar-staging-*")
	if err != nil {
		return fmt.Errorf("create release staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	if info, statErr := os.Stat(base); statErr == nil && info.IsDir() {
		if err := copyDirContents(base, stage); err != nil {
			return fmt.Errorf("seed staging directory from current release: %w", err)
		}
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect current release: %w", statErr)
	}

	start := time.Now()
	staged := make(map[string]stagedLanguage, len(languages))
	var preparationErrors []error
	for _, lang := range languages {
		m.reporter.Start("move", lang.Name)
		prepared, err := m.stageLanguage(cfg, stage, lang, req)
		if err != nil {
			preparationErrors = append(preparationErrors, fmt.Errorf("stage %s: %w", lang.Name, err))
			m.reporter.Failure("move", lang.Name, err, diagnosticsFrom(err))
			continue
		}
		staged[lang.Name] = prepared
	}
	if err := errors.Join(preparationErrors...); err != nil {
		return err
	}

	items, err := collectReleaseValidationItems(stage, cfg, languages, staged)
	if err != nil {
		return err
	}
	measuredABI, validationErr := m.validator.Validate(ctx, items)
	if validationErr != nil {
		return fmt.Errorf("validate staged native release: %w", validationErr)
	}

	for _, lang := range languages {
		prepared := staged[lang.Name]
		manifest, err := GenerateManifest(lang, prepared.binaries, measuredABI, prepared.provenance, prepared.nodeTypes, prepared.queries)
		if err != nil {
			preparationErrors = append(preparationErrors, fmt.Errorf("manifest %s: %w", lang.Name, err))
			continue
		}
		manifestPath := filepath.Join(stage, lang.Name, "manifest.json")
		if err := manifest.WriteToFile(manifestPath); err != nil {
			preparationErrors = append(preparationErrors, fmt.Errorf("manifest %s: %w", lang.Name, err))
		}
	}
	if err := errors.Join(preparationErrors...); err != nil {
		return err
	}
	if err := validateCatalog(stage, languages, measuredABI); err != nil {
		return fmt.Errorf("validate staged catalog: %w", err)
	}
	if err := publishAtomic(stage, base); err != nil {
		return err
	}

	// Count total binaries across all languages.
	totalBinaries := 0
	for _, sl := range staged {
		totalBinaries += len(sl.binaries)
	}

	var cleanupErrors []error
	if req.Clean {
		for _, lang := range languages {
			if err := m.cleaner.CleanLanguage(cfg.BuildDir, lang.Name); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("clean %s after publication: %w", lang.Name, err))
			}
		}
	}

	detail := fmt.Sprintf("%d grammars, %d binaries", len(languages), totalBinaries)
	m.reporter.Success("move", base, detail, time.Since(start))

	return errors.Join(cleanupErrors...)
}

func (m *Mover) stageLanguage(cfg *_jsii.Config, stage string, lang _jsii.Language, req _jsii.MoveRequest) (stagedLanguage, error) {
	sourceBinaries, err := collectSourceBinaries(cfg, lang)
	if err != nil || len(sourceBinaries) == 0 {
		return stagedLanguage{}, fmt.Errorf("compiled binaries not found in %q; run `tree-sitter compile` first", binaryDir(cfg, lang))
	}
	outputDir := filepath.Join(stage, lang.Name)
	if err := os.RemoveAll(outputDir); err != nil {
		return stagedLanguage{}, fmt.Errorf("replace staged grammar directory: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return stagedLanguage{}, fmt.Errorf("create staged grammar directory: %w", err)
	}
	result := stagedLanguage{language: lang, provenance: make(map[string]BinaryProvenance, len(sourceBinaries))}
	for _, sourcePath := range sourceBinaries {
		targetPath := filepath.Join(outputDir, filepath.Base(sourcePath))
		if err := copyFile(sourcePath, targetPath, true); err != nil {
			return stagedLanguage{}, err
		}
		var provenance BinaryProvenance
		if err := readJSON(binaryProvenancePath(sourcePath), &provenance); err != nil {
			return stagedLanguage{}, fmt.Errorf("load provenance for %q: %w", sourcePath, err)
		}
		result.binaries = append(result.binaries, targetPath)
		result.provenance[targetPath] = provenance
	}
	source := sourceDir(cfg, lang)
	nodeTypesPath, err := findNodeTypes(source)
	if err != nil {
		return stagedLanguage{}, err
	}
	if err := copyFile(nodeTypesPath, filepath.Join(outputDir, "node-types.json"), true); err != nil {
		return stagedLanguage{}, err
	}
	result.nodeTypes = true
	for binaryPath, provenance := range result.provenance {
		checksum, err := checksumFile(filepath.Join(outputDir, "node-types.json"))
		if err != nil {
			return stagedLanguage{}, err
		}
		if checksum != provenance.NodeTypesSHA256 {
			return stagedLanguage{}, fmt.Errorf("node-types.json revision mismatch for %q: generated %s, staged %s", binaryPath, provenance.NodeTypesSHA256, checksum)
		}
	}
	if req.Mode.IncludesQueries() {
		queriesSource, found := findQueries(source, buildRootDir(cfg, lang))
		if found {
			if err := copyDirContents(queriesSource, filepath.Join(outputDir, "queries")); err != nil {
				return stagedLanguage{}, err
			}
			result.queries = true
		}
	}
	sort.Strings(result.binaries)
	return result, nil
}

func candidateBinaryDirs(cfg *_jsii.Config, lang _jsii.Language) []string {
	var dirs []string
	seen := make(map[string]struct{})
	add := func(dir string) {
		if dir == "" {
			return
		}
		cleaned := filepath.Clean(dir)
		if _, ok := seen[cleaned]; !ok {
			seen[cleaned] = struct{}{}
			dirs = append(dirs, cleaned)
		}
	}

	add(binaryDir(cfg, lang))
	add(filepath.Join(buildRootDir(cfg, lang), "bin"))
	if cfg != nil {
		for _, t := range cfg.Targets {
			resolved, err := resolveTarget(t.OS, t.Arch)
			if err == nil {
				add(binaryDirForTarget(cfg, lang, resolved))
			}
		}
	}
	return dirs
}

func collectSourceBinaries(cfg *_jsii.Config, lang _jsii.Language) ([]string, error) {
	candidateDirs := candidateBinaryDirs(cfg, lang)
	seenFiles := make(map[string]string)
	for _, dir := range candidateDirs {
		binaries, err := collectBinaries(dir, lang)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, b := range binaries {
			filename := filepath.Base(b)
			if _, exists := seenFiles[filename]; !exists {
				seenFiles[filename] = b
			}
		}
	}
	if len(seenFiles) == 0 {
		return nil, fmt.Errorf("compiled binaries not found in %q; run `tree-sitter compile` first", binaryDir(cfg, lang))
	}
	var allBinaries []string
	for _, b := range seenFiles {
		allBinaries = append(allBinaries, b)
	}
	sort.Strings(allBinaries)
	return allBinaries, nil
}

func collectReleaseValidationItems(stage string, cfg *_jsii.Config, languages []_jsii.Language, staged ...map[string]stagedLanguage) ([]ValidationItem, error) {
	var items []ValidationItem
	var collectionErrors []error
	for _, lang := range languages {
		grammarDir := filepath.Join(stage, lang.Name)
		binaries, err := collectBinaries(grammarDir, lang)
		if err != nil || len(binaries) == 0 {
			collectionErrors = append(collectionErrors, fmt.Errorf("%s: no staged binaries", lang.Name))
			continue
		}
		queryPaths, err := collectQueryPaths(filepath.Join(grammarDir, "queries"))
		if err != nil {
			collectionErrors = append(collectionErrors, fmt.Errorf("%s: collect queries: %w", lang.Name, err))
			continue
		}
		for _, binaryPath := range binaries {
			platform, arch, err := parseBinaryMetadata(filepath.Base(binaryPath), lang)
			if err != nil {
				collectionErrors = append(collectionErrors, fmt.Errorf("%s: %w", lang.Name, err))
				continue
			}

			var prov *BinaryProvenance
			var genABI uint32
			if len(staged) > 0 && staged[0] != nil {
				if st, ok := staged[0][lang.Name]; ok {
					if p, ok := st.provenance[binaryPath]; ok {
						prov = &p
						if p.GenerateABI != nil && *p.GenerateABI > 0 {
							genABI = uint32(*p.GenerateABI)
						}
					}
				}
			}
			if genABI == 0 && lang.GenerateABI != nil && *lang.GenerateABI > 0 {
				genABI = uint32(*lang.GenerateABI)
			}
			if genABI == 0 && cfg != nil && cfg.GenerateABI > 0 {
				genABI = uint32(cfg.GenerateABI)
			}

			items = append(items, ValidationItem{
				Language:      lang,
				BinaryPath:    binaryPath,
				Platform:      platform,
				Arch:          arch,
				NodeTypesPath: filepath.Join(grammarDir, "node-types.json"),
				QueryPaths:    queryPaths,
				Sample:        lang.Sample,
				GenerateABI:   genABI,
				Provenance:    prov,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].BinaryPath < items[j].BinaryPath })
	return items, errors.Join(collectionErrors...)
}

func validateCatalog(stage string, languages []_jsii.Language, measuredABI map[string]uint32) error {
	var validationErrors []error
	for _, lang := range languages {
		manifestPath := filepath.Join(stage, lang.Name, "manifest.json")
		payload, err := os.ReadFile(manifestPath)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("%s: read manifest: %w", lang.Name, err))
			continue
		}
		var manifest ManifestData
		if err := json.Unmarshal(payload, &manifest); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("%s: parse manifest: %w", lang.Name, err))
			continue
		}
		if manifest.Grammar != lang.Grammar {
			validationErrors = append(validationErrors, fmt.Errorf("%s: manifest grammar %q does not match constructor identifier %q", lang.Name, manifest.Grammar, lang.Grammar))
		}
		if err := ValidateManifest(&manifest, filepath.Dir(manifestPath), measuredABI); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("%s: %w", lang.Name, err))
		}
	}
	return errors.Join(validationErrors...)
}

func publishAtomic(stage, destination string) error {
	backup := destination + ".previous"
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("remove stale release backup %q: %w", backup, err)
	}
	hadRelease := false
	if _, err := os.Stat(destination); err == nil {
		hadRelease = true
		if err := os.Rename(destination, backup); err != nil {
			return fmt.Errorf("preserve current release: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect current release: %w", err)
	}
	if err := os.Rename(stage, destination); err != nil {
		if hadRelease {
			_ = os.Rename(backup, destination)
		}
		return fmt.Errorf("publish staged release: %w", err)
	}
	if hadRelease {
		if err := os.RemoveAll(backup); err != nil {
			return fmt.Errorf("remove previous release after successful publication: %w", err)
		}
	}
	return nil
}

func findQueries(sourceDir, repoRoot string) (string, bool) {
	for _, candidate := range []string{filepath.Join(sourceDir, "queries"), filepath.Join(repoRoot, "queries")} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func collectBinaries(root string, lang _jsii.Language) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, lang.Name+"-"+lang.Version+"-") {
			continue
		}
		switch filepath.Ext(name) {
		case ".so", ".dylib", ".dll":
			files = append(files, filepath.Join(root, name))
		}
	}
	sort.Strings(files)
	return files, nil
}

func copyFile(sourcePath, targetPath string, force bool) error {
	if !force {
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("target file %q already exists; rerun with --force to overwrite", targetPath)
		}
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", targetPath, err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("read %q: %w", sourcePath, err)
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("write %q: %w", targetPath, err)
	}
	if _, err := io.Copy(target, source); err != nil {
		target.Close()
		return fmt.Errorf("copy %q to %q: %w", sourcePath, targetPath, err)
	}
	if err := target.Sync(); err != nil {
		target.Close()
		return fmt.Errorf("sync %q: %w", targetPath, err)
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("close %q: %w", targetPath, err)
	}
	return nil
}

func copyDirContents(sourceDir, targetDir string) error {
	return filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(targetDir, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse to publish symbolic link %q", path)
		}
		return copyFile(path, destination, true)
	})
}
