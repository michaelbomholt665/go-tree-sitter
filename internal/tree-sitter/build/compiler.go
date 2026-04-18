package build

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type Compiler struct {
	runner _jsii.CommandRunner
	lookup _jsii.PathLookup
	stdout io.Writer
}

func NewCompiler(runner _jsii.CommandRunner, lookup _jsii.PathLookup, stdout, _ io.Writer) *Compiler {
	if lookup == nil {
		lookup = _jsii.OSPathLookup{}
	}
	if stdout == nil {
		stdout = io.Discard
	}
	return &Compiler{
		runner: runner,
		lookup: lookup,
		stdout: stdout,
	}
}

func (c *Compiler) Compile(ctx context.Context, cfg *_jsii.Config, req _jsii.CompileRequest) error {
	target, err := resolveConfiguredTarget(cfg, req.OS, req.Arch)
	if err != nil {
		return err
	}

	languages, err := cfg.ResolveLanguages(req.Language)
	if err != nil {
		return err
	}

	for _, lang := range languages {
		if err := c.compileLanguage(ctx, cfg, lang, target); err != nil {
			return fmt.Errorf("compile %s for %s/%s: %w", lang.Name, target.Platform, target.Arch, err)
		}
	}

	return nil
}

func (c *Compiler) compileLanguage(ctx context.Context, cfg *_jsii.Config, lang _jsii.Language, target target) error {
	src := sourceDir(cfg, lang)
	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("source directory %q not found; run `tree-sitter build` first", src)
	}

	outDir := binaryDir(cfg, lang)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create binary output directory %q: %w", outDir, err)
	}

	outputPath := filepath.Join(outDir, binaryFilename(lang, target))
	absoluteOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolve output path %q: %w", outputPath, err)
	}
	toolchainEnv, cleanup, err := c.toolchainEnv(outDir, target)
	if err != nil {
		return err
	}
	defer cleanup()

	fmt.Fprintf(c.stdout, "compiling %s for %s/%s\n", lang.Name, target.Platform, target.Arch)
	if err := c.runner.Run(ctx, _jsii.Command{
		Name: "tree-sitter",
		Args: []string{"build", "--output", absoluteOutputPath},
		Dir:  src,
		Env:  toolchainEnv,
	}); err != nil {
		return fmt.Errorf("run tree-sitter build: %w", err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("expected compiled library %q was not created", outputPath)
	}

	return nil
}

func (c *Compiler) toolchainEnv(baseDir string, target target) (map[string]string, func(), error) {
	env := map[string]string{
		"GOOS":        target.GOOS,
		"GOARCH":      target.Arch,
		"CGO_ENABLED": "1",
	}

	if target.GOOS == runtime.GOOS && target.Arch == runtime.GOARCH {
		return env, func() {}, nil
	}

	if cc, cxx, ok := c.findCrossCompiler(target); ok {
		env["CC"] = cc
		if cxx != "" {
			env["CXX"] = cxx
		}
		return env, func() {}, nil
	}

	zigPath, err := c.lookup.LookPath("zig")
	if err != nil {
		return nil, nil, fmt.Errorf("no compiler toolchain available for %s/%s; install zig or a target-specific cross-compiler", target.Platform, target.Arch)
	}

	tempDir, err := os.MkdirTemp(baseDir, "zig-toolchain-")
	if err != nil {
		return nil, nil, fmt.Errorf("create zig toolchain wrappers: %w", err)
	}

	ccWrapper := filepath.Join(tempDir, "cc-wrapper")
	cxxWrapper := filepath.Join(tempDir, "cxx-wrapper")
	zigTarget := zigTargetTriple(target)

	if err := writeWrapper(ccWrapper, zigPath, "cc", zigTarget); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, nil, err
	}
	if err := writeWrapper(cxxWrapper, zigPath, "c++", zigTarget); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, nil, err
	}

	env["CC"] = ccWrapper
	env["CXX"] = cxxWrapper

	return env, func() {
		_ = os.RemoveAll(tempDir)
	}, nil
}

func (c *Compiler) findCrossCompiler(target target) (string, string, bool) {
	for _, candidate := range compilerCandidates(target) {
		ccPath, err := c.lookup.LookPath(candidate.cc)
		if err != nil {
			continue
		}

		cxxPath := ""
		if candidate.cxx != "" {
			cxxPath, err = c.lookup.LookPath(candidate.cxx)
			if err != nil {
				continue
			}
		}

		return ccPath, cxxPath, true
	}

	return "", "", false
}

type compilerPair struct {
	cc  string
	cxx string
}

func compilerCandidates(target target) []compilerPair {
	switch target.GOOS {
	case "windows":
		return []compilerPair{
			{cc: "x86_64-w64-mingw32-gcc", cxx: "x86_64-w64-mingw32-g++"},
		}
	case "darwin":
		if target.Arch == "arm64" {
			return []compilerPair{
				{cc: "oa64-clang", cxx: "oa64-clang++"},
				{cc: "aarch64-apple-darwin-clang", cxx: "aarch64-apple-darwin-clang++"},
			}
		}
		return []compilerPair{
			{cc: "o64-clang", cxx: "o64-clang++"},
			{cc: "x86_64-apple-darwin-clang", cxx: "x86_64-apple-darwin-clang++"},
		}
	default:
		return []compilerPair{
			{cc: "gcc", cxx: "g++"},
			{cc: "clang", cxx: "clang++"},
		}
	}
}

func zigTargetTriple(target target) string {
	switch target.GOOS {
	case "windows":
		return "x86_64-windows-gnu"
	case "darwin":
		if target.Arch == "arm64" {
			return "aarch64-macos-none"
		}
		return "x86_64-macos-none"
	default:
		return "x86_64-linux-gnu"
	}
}

func writeWrapper(path, zigPath, compiler, target string) error {
	content := fmt.Sprintf("#!/bin/sh\nexec %q %s -target %s \"$@\"\n", zigPath, compiler, target)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return fmt.Errorf("write compiler wrapper %q: %w", path, err)
	}
	return nil
}
