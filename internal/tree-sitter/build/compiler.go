package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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

	var compileErrors []error
	for _, lang := range languages {
		if err := c.compileLanguage(ctx, cfg, lang, target); err != nil {
			compileErrors = append(compileErrors, fmt.Errorf("compile %s for %s/%s: %w", lang.Name, target.Platform, target.Arch, err))
		}
	}
	return errors.Join(compileErrors...)
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
	var sourceProvenance SourceProvenance
	if err := readJSON(filepath.Join(buildRootDir(cfg, lang), sourceProvenanceFilename), &sourceProvenance); err != nil {
		return fmt.Errorf("load source provenance; run `tree-sitter build` first: %w", err)
	}
	toolchainEnv, compiler, cleanup, err := c.toolchainEnv(ctx, outDir, target)
	if err != nil {
		return err
	}
	defer cleanup()
	toolchainEnv["SOURCE_DATE_EPOCH"] = fmt.Sprintf("%d", sourceProvenance.SourceDateEpoch)
	if len(cfg.BuildFlags) > 0 {
		toolchainEnv["CFLAGS"] = strings.Join(cfg.BuildFlags, " ")
		toolchainEnv["CXXFLAGS"] = strings.Join(cfg.BuildFlags, " ")
	}

	fmt.Fprintf(c.stdout, "compiling %s for %s/%s\n", lang.Name, target.Platform, target.Arch)
	if err := c.runner.Run(ctx, _jsii.Command{
		Name: "tree-sitter",
		Args: []string{"build", "--output", absoluteOutputPath},
		Dir:  src,
		Env:  toolchainEnv,
	}); err != nil {
		return fmt.Errorf("run tree-sitter build: %w", err)
	}

	if info, err := os.Stat(absoluteOutputPath); err != nil {
		return fmt.Errorf("expected compiled library %q was not created", absoluteOutputPath)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("compiled library %q is not a regular file", absoluteOutputPath)
	}

	provenance := BinaryProvenance{
		Filename:         filepath.Base(absoluteOutputPath),
		SourceRepository: sourceProvenance.SourceRepository,
		SourceRevision:   sourceProvenance.SourceRevision,
		GeneratorVersion: sourceProvenance.GeneratorVersion,
		GenerateABI:      sourceProvenance.GenerateABI,
		SourceDateEpoch:  sourceProvenance.SourceDateEpoch,
		NodeTypesSHA256:  sourceProvenance.NodeTypesSHA256,
		TargetTriple:     targetTriple(target),
		Compiler:         compiler.Name,
		CompilerVersion:  compiler.Version,
		BuildFlags:       append([]string(nil), cfg.BuildFlags...),
	}
	if err := writeJSONAtomic(binaryProvenancePath(absoluteOutputPath), &provenance); err != nil {
		return fmt.Errorf("write binary provenance: %w", err)
	}

	return nil
}

type compilerIdentity struct {
	Name    string
	Version string
}

func (c *Compiler) toolchainEnv(ctx context.Context, baseDir string, target target) (map[string]string, compilerIdentity, func(), error) {
	env := map[string]string{
		"GOOS":        target.GOOS,
		"GOARCH":      target.Arch,
		"CGO_ENABLED": "1",
	}

	if target.Compiler != "" {
		cc, err := c.lookup.LookPath(target.Compiler)
		if err != nil {
			return nil, compilerIdentity{}, nil, fmt.Errorf("configured compiler %q not found: %w", target.Compiler, err)
		}
		env["CC"] = cc
		if target.CXX != "" {
			cxx, err := c.lookup.LookPath(target.CXX)
			if err != nil {
				return nil, compilerIdentity{}, nil, fmt.Errorf("configured C++ compiler %q not found: %w", target.CXX, err)
			}
			env["CXX"] = cxx
		}
		identity, err := c.measureCompiler(ctx, cc, target.CompilerVersion)
		return env, identity, func() {}, err
	}

	if target.GOOS == runtime.GOOS && target.Arch == runtime.GOARCH {
		cc := "cc"
		if found, err := c.lookup.LookPath("cc"); err == nil {
			cc = found
		}
		env["CC"] = cc
		identity, err := c.measureCompiler(ctx, cc, target.CompilerVersion)
		return env, identity, func() {}, err
	}

	if cc, cxx, ok := c.findCrossCompiler(target); ok {
		env["CC"] = cc
		if cxx != "" {
			env["CXX"] = cxx
		}
		identity, err := c.measureCompiler(ctx, cc, target.CompilerVersion)
		return env, identity, func() {}, err
	}

	zigPath, err := c.lookup.LookPath("zig")
	if err != nil {
		return nil, compilerIdentity{}, nil, fmt.Errorf("no compiler toolchain available for %s/%s; install zig or a target-specific cross-compiler", target.Platform, target.Arch)
	}

	tempDir, err := os.MkdirTemp(baseDir, "zig-toolchain-")
	if err != nil {
		return nil, compilerIdentity{}, nil, fmt.Errorf("create zig toolchain wrappers: %w", err)
	}

	ccWrapper := filepath.Join(tempDir, "cc-wrapper")
	cxxWrapper := filepath.Join(tempDir, "cxx-wrapper")
	zigTarget := zigTargetTriple(target)

	if err := writeWrapper(ccWrapper, zigPath, "cc", zigTarget); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, compilerIdentity{}, nil, err
	}
	if err := writeWrapper(cxxWrapper, zigPath, "c++", zigTarget); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, compilerIdentity{}, nil, err
	}

	env["CC"] = ccWrapper
	env["CXX"] = cxxWrapper

	identity, err := c.measureCompiler(ctx, zigPath, target.CompilerVersion)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, compilerIdentity{}, nil, err
	}
	return env, identity, func() {
		_ = os.RemoveAll(tempDir)
	}, nil
}

func (c *Compiler) measureCompiler(ctx context.Context, executable, expectedVersion string) (compilerIdentity, error) {
	output, err := c.runner.Output(ctx, _jsii.Command{Name: executable, Args: []string{"--version"}})
	if err != nil {
		return compilerIdentity{}, fmt.Errorf("measure compiler version: %w", err)
	}
	firstLine, _, _ := strings.Cut(strings.TrimSpace(output), "\n")
	if expectedVersion != "" && !strings.Contains(firstLine, expectedVersion) {
		return compilerIdentity{}, fmt.Errorf("compiler version mismatch: configured %s, invoked %q", expectedVersion, firstLine)
	}
	return compilerIdentity{Name: filepath.Base(executable), Version: firstLine}, nil
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
