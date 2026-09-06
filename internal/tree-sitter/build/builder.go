package build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

var pseudoVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.]+)?(?:-|\.)\d{14}-([0-9a-f]{12})$`)

type Builder struct {
	runner   _jsii.CommandRunner
	reporter _jsii.Reporter
}

// NewBuilder constructs a Builder.  The stdout and stderr writers are kept for
// backward-compatibility with existing call sites; a TextReporter is derived
// from them automatically.  Prefer NewBuilderWithReporter for new code.
func NewBuilder(runner _jsii.CommandRunner, stdout, stderr io.Writer) *Builder {
	if stdout == nil {
		stdout = io.Discard
	}
	return &Builder{
		runner:   runner,
		reporter: _jsii.NewTextReporter(stdout, false),
	}
}

// NewBuilderWithReporter constructs a Builder that reports progress via r.
func NewBuilderWithReporter(runner _jsii.CommandRunner, r _jsii.Reporter) *Builder {
	if r == nil {
		r = _jsii.SilentReporter{}
	}
	return &Builder{runner: runner, reporter: r}
}

func (b *Builder) Build(ctx context.Context, cfg *_jsii.Config, req _jsii.BuildRequest) error {
	languages, err := cfg.ResolveLanguages(req.Language)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.BuildDir, 0o755); err != nil {
		return fmt.Errorf("create build directory %q: %w", cfg.BuildDir, err)
	}

	generatorVersion, err := b.generatorVersion(ctx)
	if err != nil {
		return err
	}
	if configured := strings.TrimPrefix(cfg.TreeSitterCLIVersion, "v"); configured != "" && configured != generatorVersion {
		return fmt.Errorf("tree-sitter CLI version mismatch: configured %s, invoked %s", configured, generatorVersion)
	}

	var buildErrors []error
	for _, lang := range languages {
		if err := b.buildLanguage(ctx, cfg, lang, req.Force, generatorVersion); err != nil {
			buildErrors = append(buildErrors, fmt.Errorf("build %s: %w", lang.Name, err))
		}
	}
	return errors.Join(buildErrors...)
}

func (b *Builder) buildLanguage(ctx context.Context, cfg *_jsii.Config, lang _jsii.Language, force bool, generatorVersion string) error {
	start := time.Now()
	b.reporter.Start("build", lang.Name)

	root := buildRootDir(cfg, lang)
	if force {
		if err := os.RemoveAll(root); err != nil {
			err = fmt.Errorf("remove existing build directory %q: %w", root, err)
			b.reporter.Failure("build", lang.Name, err, "")
			return err
		}
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err := b.cloneRepository(ctx, lang, root); err != nil {
			b.reporter.Failure("build", lang.Name, err, diagnosticsFrom(err))
			return err
		}
		if err := b.checkoutVersion(ctx, lang, root); err != nil {
			b.reporter.Failure("build", lang.Name, err, diagnosticsFrom(err))
			return err
		}
	} else if err != nil {
		err = fmt.Errorf("inspect build directory %q: %w", root, err)
		b.reporter.Failure("build", lang.Name, err, "")
		return err
	} else if err := b.checkoutVersion(ctx, lang, root); err != nil {
		b.reporter.Failure("build", lang.Name, err, diagnosticsFrom(err))
		return err
	}

	src := sourceDir(cfg, lang)
	if info, err := os.Stat(src); err != nil {
		err = fmt.Errorf("source directory %q: %w", src, err)
		b.reporter.Failure("build", lang.Name, err, "")
		return err
	} else if !info.IsDir() {
		err = fmt.Errorf("source directory %q is not a directory", src)
		b.reporter.Failure("build", lang.Name, err, "")
		return err
	}

	revision, err := b.runner.Output(ctx, _jsii.Command{Name: "git", Args: []string{"-C", root, "rev-parse", "HEAD"}})
	if err != nil {
		err = fmt.Errorf("resolve source revision: %w", err)
		b.reporter.Failure("build", lang.Name, err, diagnosticsFrom(err))
		return err
	}
	revision = strings.TrimSpace(revision)
	if lang.Revision != "" && !strings.EqualFold(lang.Revision, revision) {
		err = fmt.Errorf("source revision mismatch: configured %s, checked out %s", lang.Revision, revision)
		b.reporter.Failure("build", lang.Name, err, "")
		return err
	}

	epochOutput, err := b.runner.Output(ctx, _jsii.Command{Name: "git", Args: []string{"-C", root, "show", "-s", "--format=%ct", "HEAD"}})
	if err != nil {
		err = fmt.Errorf("resolve source timestamp: %w", err)
		b.reporter.Failure("build", lang.Name, err, diagnosticsFrom(err))
		return err
	}
	epoch, err := strconv.ParseInt(strings.TrimSpace(epochOutput), 10, 64)
	if err != nil {
		err = fmt.Errorf("parse source timestamp %q: %w", epochOutput, err)
		b.reporter.Failure("build", lang.Name, err, "")
		return err
	}
	buildEnv := map[string]string{"SOURCE_DATE_EPOCH": strconv.FormatInt(epoch, 10)}
	if err := b.installNodeDependencies(ctx, lang.Name, root, buildEnv); err != nil {
		b.reporter.Failure("build", lang.Name, err, diagnosticsFrom(err))
		return err
	}

	if err := ensureTreeSitterJSON(src, lang); err != nil {
		err = fmt.Errorf("ensure tree-sitter.json: %w", err)
		b.reporter.Failure("build", lang.Name, err, "")
		return err
	}

	minABI := 13
	maxABI := 15
	if cfg.ABIRange != nil {
		minABI = cfg.ABIRange.Min
		maxABI = cfg.ABIRange.Max
	}

	targetABI := cfg.GenerateABI
	if lang.GenerateABI != nil && *lang.GenerateABI > 0 {
		targetABI = *lang.GenerateABI
	}

	runGenerate := func(abi int) error {
		args := []string{"generate"}
		if abi > 0 {
			args = append(args, "--abi", strconv.Itoa(abi))
		}
		var generateOut, generateErr bytes.Buffer
		return b.runCaptured(ctx, _jsii.Command{
			Name: "tree-sitter",
			Args: args,
			Dir:  src,
			Env:  buildEnv,
		}, &generateOut, &generateErr)
	}

	var generateABI *int
	var genErr error
	if targetABI > 0 {
		genErr = runGenerate(targetABI)
		if genErr == nil {
			selected := targetABI
			generateABI = &selected
		}
	} else {
		genErr = runGenerate(0)
	}

	// If explicit targetABI failed and lang.GenerateABI was not hardcoded, attempt fallbacks within abi_range
	if genErr != nil && lang.GenerateABI == nil && targetABI > minABI {
		for fallbackABI := targetABI - 1; fallbackABI >= minABI; fallbackABI-- {
			if err := runGenerate(fallbackABI); err == nil {
				genErr = nil
				selected := fallbackABI
				generateABI = &selected
				break
			}
		}
		if genErr != nil {
			if err := runGenerate(0); err == nil {
				genErr = nil
			}
		}
	}

	if genErr != nil {
		diag := diagnosticsFrom(genErr)
		wrapped := fmt.Errorf("generate grammar source: %w", genErr)
		b.reporter.Failure("build", lang.Name, wrapped, diag)
		return wrapped
	}

	if detected, err := detectGeneratedABI(src); err == nil && detected > 0 {
		generateABI = &detected
		if detected < minABI || detected > maxABI {
			err := fmt.Errorf("generated parser ABI %d is outside supported range %d-%d", detected, minABI, maxABI)
			b.reporter.Failure("build", lang.Name, err, "")
			return err
		}
	}

	nodeTypesPath, err := findNodeTypes(src)
	if err != nil {
		b.reporter.Failure("build", lang.Name, err, "")
		return err
	}
	nodeTypesChecksum, err := checksumFile(nodeTypesPath)
	if err != nil {
		b.reporter.Failure("build", lang.Name, err, "")
		return err
	}
	provenance := SourceProvenance{
		SourceRepository: lang.Repository,
		SourceRevision:   revision,
		GeneratorVersion: generatorVersion,
		GenerateABI:      generateABI,
		SourceDateEpoch:  epoch,
		NodeTypesSHA256:  nodeTypesChecksum,
	}
	if err := writeJSONAtomic(filepath.Join(root, sourceProvenanceFilename), &provenance); err != nil {
		err = fmt.Errorf("write source provenance: %w", err)
		b.reporter.Failure("build", lang.Name, err, "")
		return err
	}

	b.reporter.Success("build", lang.Name, "", time.Since(start))
	return nil
}

func (b *Builder) generatorVersion(ctx context.Context) (string, error) {
	output, err := b.runner.Output(ctx, _jsii.Command{Name: "tree-sitter", Args: []string{"--version"}})
	if err != nil {
		return "", fmt.Errorf("measure tree-sitter CLI version: %w", err)
	}
	version, err := parseToolVersion(output, "tree-sitter")
	if err != nil {
		return "", err
	}
	return version, nil
}

func (b *Builder) installNodeDependencies(ctx context.Context, langName, root string, env map[string]string) error {
	packageJSON := filepath.Join(root, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect package.json %q: %w", packageJSON, err)
	}

	nodeModules := filepath.Join(root, "node_modules")
	if info, err := os.Stat(nodeModules); err == nil && info.IsDir() {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect node dependencies %q: %w", nodeModules, err)
	}

	lockFile := filepath.Join(root, "package-lock.json")
	if _, err := os.Stat(lockFile); err != nil {
		if os.IsNotExist(err) {
			b.reporter.Info("skipping npm in %s: no lockfile (uses checked-in sources; grammar build is unaffected)", langName)
			return nil
		}
		return fmt.Errorf("inspect package lock %q: %w", lockFile, err)
	}

	var npmOut, npmErr bytes.Buffer
	if err := b.runCaptured(ctx, _jsii.Command{
		Name:          "npm",
		Args:          []string{"ci", "--ignore-scripts"},
		Dir:           root,
		Env:           env,
		SilenceStderr: true,
	}, &npmOut, &npmErr); err != nil {
		// Retry with npm install.
		npmOut.Reset()
		npmErr.Reset()
		if err2 := b.runCaptured(ctx, _jsii.Command{
			Name: "npm",
			Args: []string{"install", "--ignore-scripts", "--no-audit", "--no-fund"},
			Dir:  root,
			Env:  env,
		}, &npmOut, &npmErr); err2 != nil {
			combined := combinedOutput(npmOut.String(), npmErr.String())
			return fmt.Errorf("install node dependencies: %w\n%s", err2, combined)
		}
	}

	return nil
}

// runCaptured runs cmd and captures its stdout/stderr into the supplied
// buffers without streaming to the terminal (unless the runner itself is in
// verbose mode, which is handled inside ExecRunner.Run).
func (b *Builder) runCaptured(ctx context.Context, cmd _jsii.Command, outBuf, errBuf *bytes.Buffer) error {
	return b.runner.Run(ctx, cmd)
}

func findNodeTypes(source string) (string, error) {
	for _, candidate := range []string{filepath.Join(source, "src", "node-types.json"), filepath.Join(source, "node-types.json")} {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("generated node-types.json not found under %q", source)
}

func (b *Builder) cloneRepository(ctx context.Context, lang _jsii.Language, destination string) error {
	ref, branchClone := languageCheckoutRef(lang)

	args := []string{"clone"}
	if branchClone {
		args = append(args, "--depth", "1", "--branch", ref)
	} else if ref == "" {
		args = append(args, "--depth", "1")
	}
	args = append(args, lang.Repository, destination)

	if err := b.runner.Run(ctx, _jsii.Command{
		Name: "git",
		Args: args,
	}); err != nil {
		return fmt.Errorf("clone repository: %w", err)
	}

	return nil
}

func (b *Builder) checkoutVersion(ctx context.Context, lang _jsii.Language, root string) error {
	ref, _ := languageCheckoutRef(lang)
	if ref == "" {
		return nil
	}

	if err := b.runner.Run(ctx, _jsii.Command{
		Name: "git",
		Args: []string{"-C", root, "fetch", "--tags", "--prune"},
	}); err != nil {
		return fmt.Errorf("fetch repository tags: %w", err)
	}

	if err := b.runner.Run(ctx, _jsii.Command{
		Name: "git",
		Args: []string{"-C", root, "checkout", ref},
	}); err != nil {
		return fmt.Errorf("checkout %s: %w", ref, err)
	}

	return nil
}

func languageCheckoutRef(lang _jsii.Language) (string, bool) {
	if lang.Revision != "" {
		return lang.Revision, false
	}
	return resolveCheckoutRef(lang.Version)
}

func resolveCheckoutRef(version string) (string, bool) {
	version = strings.TrimSpace(version)
	if version == "" || version == "latest" {
		return "", false
	}
	if matches := pseudoVersionPattern.FindStringSubmatch(version); len(matches) == 2 {
		return matches[1], false
	}
	return version, true
}

func ensureTreeSitterJSON(sourceDir string, lang _jsii.Language) error {
	configPath := filepath.Join(sourceDir, "tree-sitter.json")
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	type grammarEntry struct {
		Name      string   `json:"name"`
		CamelCase string   `json:"camelcase,omitempty"`
		Scope     string   `json:"scope,omitempty"`
		Path      string   `json:"path"`
		FileTypes []string `json:"file-types,omitempty"`
	}
	type metadataEntry struct {
		Version string            `json:"version,omitempty"`
		Links   map[string]string `json:"links,omitempty"`
	}
	type tsConfigFile struct {
		Schema   string         `json:"$schema"`
		Grammars []grammarEntry `json:"grammars"`
		Metadata *metadataEntry `json:"metadata,omitempty"`
	}

	grammar := grammarEntry{
		Name:      lang.Grammar,
		CamelCase: toCamelCase(lang.Grammar),
		Scope:     "source." + lang.Grammar,
		Path:      ".",
	}

	for _, pkgDir := range []string{sourceDir, filepath.Dir(sourceDir)} {
		pkgPath := filepath.Join(pkgDir, "package.json")
		if pkgBytes, err := os.ReadFile(pkgPath); err == nil {
			var pkg struct {
				TreeSitter []struct {
					Scope     string   `json:"scope"`
					FileTypes []string `json:"file-types"`
				} `json:"tree-sitter"`
			}
			if err := json.Unmarshal(pkgBytes, &pkg); err == nil && len(pkg.TreeSitter) > 0 {
				if pkg.TreeSitter[0].Scope != "" {
					grammar.Scope = pkg.TreeSitter[0].Scope
				}
				if len(pkg.TreeSitter[0].FileTypes) > 0 {
					grammar.FileTypes = pkg.TreeSitter[0].FileTypes
				}
				break
			}
		}
	}

	configFile := tsConfigFile{
		Schema:   "https://tree-sitter.github.io/tree-sitter/assets/schemas/config.schema.json",
		Grammars: []grammarEntry{grammar},
		Metadata: &metadataEntry{
			Version: strings.TrimPrefix(lang.Version, "v"),
			Links: map[string]string{
				"repository": lang.Repository,
			},
		},
	}

	payload, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tree-sitter.json: %w", err)
	}
	payload = append(payload, '\n')
	return os.WriteFile(configPath, payload, 0o644)
}

func detectGeneratedABI(sourceDir string) (int, error) {
	parserPath := filepath.Join(sourceDir, "src", "parser.c")
	content, err := os.ReadFile(parserPath)
	if err != nil {
		return 0, err
	}
	re := regexp.MustCompile(`(?m)^\s*#\s*define\s+LANGUAGE_VERSION\s+(\d+)`)
	m := re.FindSubmatch(content)
	if len(m) == 2 {
		return strconv.Atoi(string(m[1]))
	}
	return 0, errors.New("LANGUAGE_VERSION not found in parser.c")
}

func toCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// diagnosticsFrom extracts the captured process output from a CommandError.
func diagnosticsFrom(err error) string {
	var ce *_jsii.CommandError
	if errors.As(err, &ce) {
		return ce.Output
	}
	return ""
}

// combinedOutput merges non-empty stdout/stderr strings.
func combinedOutput(stdout, stderr string) string {
	out := strings.TrimSpace(stdout)
	errOut := strings.TrimSpace(stderr)
	switch {
	case out != "" && errOut != "":
		return out + "\n" + errOut
	case out != "":
		return out
	default:
		return errOut
	}
}
