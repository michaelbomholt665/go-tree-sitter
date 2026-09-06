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
	language         _jsii.Language
	binaries         []string
	provenance       map[string]BinaryProvenance
	nodeTypes        bool
	compactNodeTypes bool
	queries          bool
	wasm             bool
	cSource          bool
	js               bool
}

func (m *Mover) Move(ctx context.Context, cfg *_jsii.Config, req _jsii.MoveRequest) error {
	if req.GenerateManifest && !cfg.Output.GenerateManifest {
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

	if req.GenerateManifest {
		for _, lang := range languages {
			prepared := staged[lang.Name]
			manifest, err := GenerateManifest(lang, prepared.binaries, measuredABI, prepared.provenance, ArtifactInfo{
				HasNodeTypes:        prepared.nodeTypes,
				HasCompactNodeTypes: prepared.compactNodeTypes,
				HasQueries:          prepared.queries,
				HasWasm:             prepared.wasm,
				HasCSource:          prepared.cSource,
				HasJS:               prepared.js,
			})
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
	if req.Compact {
		rawJSON, err := os.ReadFile(nodeTypesPath)
		if err != nil {
			return stagedLanguage{}, fmt.Errorf("read node-types.json for %s: %w", lang.Name, err)
		}
		compactYAML, err := CompactNodeTypes(rawJSON)
		if err != nil {
			return stagedLanguage{}, fmt.Errorf("generate compact node types for %s: %w", lang.Name, err)
		}
		discrepancies, _, err := ValidateCapture(rawJSON, string(compactYAML))
		if err != nil {
			return stagedLanguage{}, fmt.Errorf("validate compact node types for %s: %w", lang.Name, err)
		}
		if len(discrepancies) > 0 {
			return stagedLanguage{}, fmt.Errorf("compact validation failed for %s: %s", lang.Name, strings.Join(discrepancies, "; "))
		}
		targetCompact := filepath.Join(outputDir, "compact-node-types.yaml")
		if err := os.WriteFile(targetCompact, compactYAML, 0o644); err != nil {
			return stagedLanguage{}, fmt.Errorf("write compact-node-types.yaml for %s: %w", lang.Name, err)
		}
		result.compactNodeTypes = true
	} else if req.CheckCompact {
		candidateCompactPaths := []string{
			filepath.Join(outputDir, "compact-node-types.yaml"),
			filepath.Join(source, "compact-node-types.yaml"),
			filepath.Join(source, "src", "compact-node-types.yaml"),
			filepath.Join(cfg.Output.GrammarBase, lang.Name, "compact-node-types.yaml"),
		}
		var foundCompact string
		for _, cand := range candidateCompactPaths {
			if info, err := os.Stat(cand); err == nil && info.Mode().IsRegular() {
				foundCompact = cand
				break
			}
		}
		if foundCompact == "" {
			return stagedLanguage{}, fmt.Errorf("compact-node-types.yaml not found for %s; run with --compact to generate", lang.Name)
		}
		rawJSON, err := os.ReadFile(nodeTypesPath)
		if err != nil {
			return stagedLanguage{}, fmt.Errorf("read node-types.json for %s: %w", lang.Name, err)
		}
		yamlBytes, err := os.ReadFile(foundCompact)
		if err != nil {
			return stagedLanguage{}, fmt.Errorf("read compact-node-types.yaml for %s: %w", lang.Name, err)
		}
		discrepancies, _, err := ValidateCapture(rawJSON, string(yamlBytes))
		if err != nil {
			return stagedLanguage{}, fmt.Errorf("validate compact node types for %s: %w", lang.Name, err)
		}
		if len(discrepancies) > 0 {
			return stagedLanguage{}, fmt.Errorf("compact validation failed for %s: %s", lang.Name, strings.Join(discrepancies, "; "))
		}
		targetCompact := filepath.Join(outputDir, "compact-node-types.yaml")
		if foundCompact != targetCompact {
			if err := copyFile(foundCompact, targetCompact, true); err != nil {
				return stagedLanguage{}, fmt.Errorf("copy compact-node-types.yaml: %w", err)
			}
		}
		result.compactNodeTypes = true
	}
	if req.CopySCM {
		targetQueriesDir := filepath.Join(outputDir, "queries")
		copied, err := copyLanguageQueries(source, buildRootDir(cfg, lang), targetQueriesDir, lang)
		if err != nil {
			return stagedLanguage{}, err
		}
		result.queries = copied
	}
	if req.CopySource {
		srcDir := filepath.Join(source, "src")
		if info, err := os.Stat(srcDir); err != nil || !info.IsDir() {
			return stagedLanguage{}, fmt.Errorf("source directory %q not found; run `tree-sitter build` first", srcDir)
		}
		targetSrc := filepath.Join(outputDir, "src")
		if err := copySourceDir(srcDir, targetSrc); err != nil {
			return stagedLanguage{}, fmt.Errorf("copy source files from %q: %w", srcDir, err)
		}
		parserPath := filepath.Join(targetSrc, "parser.c")
		if _, err := os.Stat(parserPath); err != nil {
			return stagedLanguage{}, fmt.Errorf("expected parser source %q not found", parserPath)
		}
		result.cSource = true
	}
	if req.CopyJS {
		grammarJS := filepath.Join(source, "grammar.js")
		if _, err := os.Stat(grammarJS); err != nil {
			grammarJS = filepath.Join(buildRootDir(cfg, lang), "grammar.js")
		}
		if info, err := os.Stat(grammarJS); err != nil || !info.Mode().IsRegular() {
			return stagedLanguage{}, fmt.Errorf("grammar.js not found under %q; run `tree-sitter build` first", source)
		}
		targetJS := filepath.Join(outputDir, "grammar.js")
		if err := copyFile(grammarJS, targetJS, true); err != nil {
			return stagedLanguage{}, fmt.Errorf("copy grammar.js: %w", err)
		}
		result.js = true
	}
	if req.IncludeWasm {
		wasmFilename := fmt.Sprintf("tree-sitter-%s.wasm", lang.Name)
		candidatePaths := []string{
			filepath.Join(buildRootDir(cfg, lang), wasmFilename),
			filepath.Join(source, wasmFilename),
			filepath.Join(binaryDir(cfg, lang), wasmFilename),
			filepath.Join(buildRootDir(cfg, lang), "bin", wasmFilename),
			filepath.Join(buildRootDir(cfg, lang), fmt.Sprintf("tree-sitter-%s.wasm", lang.Grammar)),
			filepath.Join(source, fmt.Sprintf("tree-sitter-%s.wasm", lang.Grammar)),
		}
		var sourceWasm string
		for _, cand := range candidatePaths {
			if info, err := os.Stat(cand); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
				sourceWasm = cand
				break
			}
		}
		if sourceWasm == "" {
			return stagedLanguage{}, fmt.Errorf("wasm artifact %q not found; run `ts-build compile --wasm` first", wasmFilename)
		}
		targetWasm := filepath.Join(outputDir, wasmFilename)
		if err := copyFile(sourceWasm, targetWasm, true); err != nil {
			return stagedLanguage{}, fmt.Errorf("copy wasm artifact: %w", err)
		}
		if _, err := checksumFile(targetWasm); err != nil {
			return stagedLanguage{}, fmt.Errorf("checksum wasm artifact %q: %w", targetWasm, err)
		}
		result.wasm = true
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

type tsConfigGrammarQueries struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Highlights any    `json:"highlights"`
	Injections any    `json:"injections"`
	Locals     any    `json:"locals"`
	Tags       any    `json:"tags"`
	Folds      any    `json:"folds"`
	Indents    any    `json:"indents"`
}

type tsConfigQueryFile struct {
	Grammars []tsConfigGrammarQueries `json:"grammars"`
}

func extractQueryPaths(val any) []string {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			return []string{v}
		}
	case []any:
		var paths []string
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				paths = append(paths, s)
			}
		}
		return paths
	}
	return nil
}

func copyLanguageQueries(sourceDir, repoRoot, targetQueriesDir string, lang _jsii.Language) (bool, error) {
	// 1. Try to read tree-sitter.json to identify exact queries declared for this grammar.
	for _, configDir := range []string{sourceDir, repoRoot} {
		configPath := filepath.Join(configDir, "tree-sitter.json")
		if data, err := os.ReadFile(configPath); err == nil {
			var cfg tsConfigQueryFile
			if err := json.Unmarshal(data, &cfg); err == nil && len(cfg.Grammars) > 0 {
				var matchingGrammar *tsConfigGrammarQueries
				for i := range cfg.Grammars {
					g := &cfg.Grammars[i]
					if g.Name == lang.Grammar || g.Name == lang.Name {
						matchingGrammar = g
						break
					}
					if lang.SourceSubdir != "" && (g.Path == lang.SourceSubdir || strings.Trim(filepath.ToSlash(g.Path), "./") == strings.Trim(filepath.ToSlash(lang.SourceSubdir), "./")) {
						matchingGrammar = g
						break
					}
				}
				if matchingGrammar != nil {
					var qPaths []string
					qPaths = append(qPaths, extractQueryPaths(matchingGrammar.Highlights)...)
					qPaths = append(qPaths, extractQueryPaths(matchingGrammar.Injections)...)
					qPaths = append(qPaths, extractQueryPaths(matchingGrammar.Locals)...)
					qPaths = append(qPaths, extractQueryPaths(matchingGrammar.Tags)...)
					qPaths = append(qPaths, extractQueryPaths(matchingGrammar.Folds)...)
					qPaths = append(qPaths, extractQueryPaths(matchingGrammar.Indents)...)

					if len(qPaths) > 0 {
						if err := os.MkdirAll(targetQueriesDir, 0o755); err != nil {
							return false, fmt.Errorf("create queries directory: %w", err)
						}
						copiedAny := false
						for _, qp := range qPaths {
							candidate := filepath.Join(repoRoot, filepath.FromSlash(qp))
							if _, err := os.Stat(candidate); os.IsNotExist(err) {
								candidate = filepath.Join(sourceDir, filepath.FromSlash(qp))
							}
							if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
								dest := filepath.Join(targetQueriesDir, filepath.Base(qp))
								if (lang.Grammar == "ocaml_interface" || lang.Name == "ocaml-interface") && filepath.Base(qp) == "highlights.scm" {
									raw, err := os.ReadFile(candidate)
									if err != nil {
										return false, fmt.Errorf("read query file %q: %w", candidate, err)
									}
									cleaned := strings.ReplaceAll(string(raw), " (shebang)", "")
									if err := os.WriteFile(dest, []byte(cleaned), 0o644); err != nil {
										return false, fmt.Errorf("write cleaned query file %q: %w", dest, err)
									}
								} else {
									if err := copyFile(candidate, dest, true); err != nil {
										return false, fmt.Errorf("copy query file %q: %w", candidate, err)
									}
								}
								copiedAny = true
							}
						}
						if copiedAny {
							return true, nil
						}
					}
				}
			}
		}
	}

	// 2. Fall back to conventional queries directory if tree-sitter.json didn't specify queries.
	queriesSource, found := findQueries(sourceDir, repoRoot)
	if found {
		if err := copyDirContents(queriesSource, targetQueriesDir); err != nil {
			return false, err
		}
		if lang.Grammar == "ocaml_interface" || lang.Name == "ocaml-interface" {
			dest := filepath.Join(targetQueriesDir, "highlights.scm")
			if raw, err := os.ReadFile(dest); err == nil {
				cleaned := strings.ReplaceAll(string(raw), " (shebang)", "")
				_ = os.WriteFile(dest, []byte(cleaned), 0o644)
			}
		}
		return true, nil
	}
	return false, nil
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

func copySourceDir(sourceDir, targetDir string) error {
	return filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(targetDir, 0o755)
		}
		dest := filepath.Join(targetDir, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse to copy symbolic link %q", path)
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".c", ".h", ".cc", ".cpp", ".hpp", ".cxx":
			return copyFile(path, dest, true)
		default:
			return nil
		}
	})
}
