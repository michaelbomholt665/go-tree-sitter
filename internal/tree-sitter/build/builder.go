package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

var pseudoVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.]+)?(?:-|\.)\d{14}-([0-9a-f]{12})$`)

type Builder struct {
	runner _jsii.CommandRunner
	stdout io.Writer
}

func NewBuilder(runner _jsii.CommandRunner, stdout, _ io.Writer) *Builder {
	if stdout == nil {
		stdout = io.Discard
	}
	return &Builder{
		runner: runner,
		stdout: stdout,
	}
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
	root := buildRootDir(cfg, lang)
	if force {
		if err := os.RemoveAll(root); err != nil {
			return fmt.Errorf("remove existing build directory %q: %w", root, err)
		}
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		fmt.Fprintf(b.stdout, "cloning %s into %s\n", lang.Name, root)
		if err := b.cloneRepository(ctx, lang, root); err != nil {
			return err
		}
		if err := b.checkoutVersion(ctx, lang, root); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("inspect build directory %q: %w", root, err)
	} else if err := b.checkoutVersion(ctx, lang, root); err != nil {
		return err
	}

	src := sourceDir(cfg, lang)
	if info, err := os.Stat(src); err != nil {
		return fmt.Errorf("source directory %q: %w", src, err)
	} else if !info.IsDir() {
		return fmt.Errorf("source directory %q is not a directory", src)
	}
	revision, err := b.runner.Output(ctx, _jsii.Command{Name: "git", Args: []string{"-C", root, "rev-parse", "HEAD"}})
	if err != nil {
		return fmt.Errorf("resolve source revision: %w", err)
	}
	revision = strings.TrimSpace(revision)
	if lang.Revision != "" && !strings.EqualFold(lang.Revision, revision) {
		return fmt.Errorf("source revision mismatch: configured %s, checked out %s", lang.Revision, revision)
	}
	epochOutput, err := b.runner.Output(ctx, _jsii.Command{Name: "git", Args: []string{"-C", root, "show", "-s", "--format=%ct", "HEAD"}})
	if err != nil {
		return fmt.Errorf("resolve source timestamp: %w", err)
	}
	epoch, err := strconv.ParseInt(strings.TrimSpace(epochOutput), 10, 64)
	if err != nil {
		return fmt.Errorf("parse source timestamp %q: %w", epochOutput, err)
	}
	buildEnv := map[string]string{"SOURCE_DATE_EPOCH": strconv.FormatInt(epoch, 10)}
	if err := b.installNodeDependencies(ctx, root, buildEnv); err != nil {
		return err
	}

	fmt.Fprintf(b.stdout, "generating parser sources for %s\n", lang.Name)
	generateArgs := []string{"generate"}
	var generateABI *int
	if cfg.GenerateABI > 0 {
		generateArgs = append(generateArgs, "--abi", strconv.Itoa(cfg.GenerateABI))
		selected := cfg.GenerateABI
		generateABI = &selected
	}
	if err := b.runner.Run(ctx, _jsii.Command{
		Name: "tree-sitter",
		Args: generateArgs,
		Dir:  src,
		Env:  buildEnv,
	}); err != nil {
		return fmt.Errorf("generate grammar source: %w", err)
	}

	nodeTypesPath, err := findNodeTypes(src)
	if err != nil {
		return err
	}
	nodeTypesChecksum, err := checksumFile(nodeTypesPath)
	if err != nil {
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
		return fmt.Errorf("write source provenance: %w", err)
	}
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

func (b *Builder) installNodeDependencies(ctx context.Context, root string, env map[string]string) error {
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
			fmt.Fprintf(b.stdout, "skipping npm in %s because no lockfile is present; generation must use only the pinned external CLI and checked-in sources\n", root)
			return nil
		}
		return fmt.Errorf("inspect package lock %q: %w", lockFile, err)
	}

	fmt.Fprintf(b.stdout, "installing pinned node dependencies in %s\n", root)
	if err := b.runner.Run(ctx, _jsii.Command{
		Name: "npm",
		Args: []string{"ci", "--ignore-scripts"},
		Dir:  root,
		Env:  env,
	}); err != nil {
		return fmt.Errorf("install node dependencies: %w", err)
	}

	return nil
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
