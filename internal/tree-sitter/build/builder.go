package build

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
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

	for _, lang := range languages {
		if err := b.buildLanguage(ctx, cfg, lang, req.Force); err != nil {
			return fmt.Errorf("build %s: %w", lang.Name, err)
		}
	}

	return nil
}

func (b *Builder) buildLanguage(ctx context.Context, cfg *_jsii.Config, lang _jsii.Language, force bool) error {
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

	if err := b.installNodeDependencies(ctx, root); err != nil {
		return err
	}

	src := sourceDir(cfg, lang)
	if info, err := os.Stat(src); err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("%q is not a directory", src)
		}
		return fmt.Errorf("source directory %q not found", src)
	}

	fmt.Fprintf(b.stdout, "generating parser sources for %s\n", lang.Name)
	if err := b.runner.Run(ctx, _jsii.Command{
		Name: "tree-sitter",
		Args: []string{"generate"},
		Dir:  src,
	}); err != nil {
		return fmt.Errorf("generate grammar source: %w", err)
	}

	return nil
}

func (b *Builder) installNodeDependencies(ctx context.Context, root string) error {
	packageJSON := root + string(os.PathSeparator) + "package.json"
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect package.json %q: %w", packageJSON, err)
	}

	nodeModules := root + string(os.PathSeparator) + "node_modules"
	if info, err := os.Stat(nodeModules); err == nil && info.IsDir() {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect node dependencies %q: %w", nodeModules, err)
	}

	fmt.Fprintf(b.stdout, "installing node dependencies in %s\n", root)
	if err := b.runner.Run(ctx, _jsii.Command{
		Name: "npm",
		Args: []string{"install", "--ignore-scripts"},
		Dir:  root,
	}); err != nil {
		return fmt.Errorf("install node dependencies: %w", err)
	}

	return nil
}

func (b *Builder) cloneRepository(ctx context.Context, lang _jsii.Language, destination string) error {
	ref, branchClone := resolveCheckoutRef(lang.Version)

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
	ref, _ := resolveCheckoutRef(lang.Version)
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
