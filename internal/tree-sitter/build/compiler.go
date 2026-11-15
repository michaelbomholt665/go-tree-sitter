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
	"time"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type Compiler struct {
	runner   _jsii.CommandRunner
	lookup   _jsii.PathLookup
	reporter _jsii.Reporter
	policy   _jsii.RuntimeConfig
}

// NewCompiler constructs a Compiler.  The stdout and stderr writers are kept
// for backward-compatibility; a TextReporter is derived from stdout.  Prefer
// NewCompilerWithReporter for new code.
func NewCompiler(runner _jsii.CommandRunner, lookup _jsii.PathLookup, stdout, _ io.Writer) *Compiler {
	if lookup == nil {
		lookup = _jsii.OSPathLookup{}
	}
	if stdout == nil {
		stdout = io.Discard
	}
	return &Compiler{
		runner:   runner,
		lookup:   lookup,
		reporter: _jsii.NewTextReporter(stdout, false),
	}
}

// NewCompilerWithReporter constructs a Compiler that reports progress via r.
func NewCompilerWithReporter(runner _jsii.CommandRunner, lookup _jsii.PathLookup, r _jsii.Reporter) *Compiler {
	if lookup == nil {
		lookup = _jsii.OSPathLookup{}
	}
	if r == nil {
		r = _jsii.SilentReporter{}
	}
	return &Compiler{runner: runner, lookup: lookup, reporter: r}
}

func (c *Compiler) Compile(ctx context.Context, cfg *_jsii.Config, req _jsii.CompileRequest) error {
	if cfg.Output.GrammarBase != "" {
		if err := cfg.ValidateBuildPaths(); err != nil {
			return err
		}
	}
	local := *c
	local.policy = cfg.EffectiveRuntime()
	local.runner = _jsii.WithToolPaths(c.runner, local.policy.Tools)
	c = &local
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
		if err := cfg.ValidateWorkspacePath(buildRootDir(cfg, lang)); err != nil {
			return err
		}
		if err := cfg.ValidateWorkspacePath(binaryDirForTarget(cfg, lang, target)); err != nil {
			return err
		}
		if err := c.compileLanguage(ctx, cfg, lang, target); err != nil {
			compileErrors = append(compileErrors, fmt.Errorf("compile %s for %s/%s: %w", lang.Name, target.Platform, target.Arch, err))
		}
		if req.BuildWasm {
			if err := c.CompileWasm(ctx, lang, buildRootDir(cfg, lang)); err != nil {
				compileErrors = append(compileErrors, fmt.Errorf("compile wasm %s: %w", lang.Name, err))
			}
		}
	}
	return errors.Join(compileErrors...)
}

func (c *Compiler) compileLanguage(ctx context.Context, cfg *_jsii.Config, lang _jsii.Language, target target) error {
	targetLabel := target.Platform + "/" + target.Arch
	start := time.Now()
	c.reporter.Start("compile", lang.Name+" ["+targetLabel+"]")

	src := sourceDir(cfg, lang)
	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		err = fmt.Errorf("source directory %q not found; run `tree-sitter build` first", src)
		c.reporter.Failure("compile", lang.Name, err, "")
		return err
	}

	outDir := binaryDirForTarget(cfg, lang, target)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		err = fmt.Errorf("create binary output directory %q: %w", outDir, err)
		c.reporter.Failure("compile", lang.Name+" ["+targetLabel+"]", err, "")
		return err
	}

	outputPath := filepath.Join(outDir, binaryFilename(lang, target))
	absoluteOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		err = fmt.Errorf("resolve output path %q: %w", outputPath, err)
		c.reporter.Failure("compile", lang.Name+" ["+targetLabel+"]", err, "")
		return err
	}

	var sourceProvenance SourceProvenance
	if err := readJSON(filepath.Join(buildRootDir(cfg, lang), sourceProvenanceFilename), &sourceProvenance); err != nil {
		err = fmt.Errorf("load source provenance; run `tree-sitter build` first: %w", err)
		c.reporter.Failure("compile", lang.Name+" ["+targetLabel+"]", err, "")
		return err
	}

	if err := verifySourceInputs(src, sourceProvenance); err != nil {
		return err
	}
	if !strings.EqualFold(sourceProvenance.SourceRevision, lang.Revision) || sourceProvenance.SourceRepository != lang.Repository ||
		strings.TrimPrefix(sourceProvenance.GeneratorVersion, "v") != strings.TrimPrefix(cfg.TreeSitterCLIVersion, "v") {
		return fmt.Errorf("source provenance does not match selected configuration; rebuild %s", lang.Name)
	}
	tempDir, err := os.MkdirTemp(outDir, ".compile-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	finalOutputPath := absoluteOutputPath
	absoluteOutputPath, err = filepath.Abs(filepath.Join(tempDir, filepath.Base(outputPath)))
	if err != nil {
		return err
	}
	toolchainEnv, compiler, cleanup, err := c.toolchainEnv(ctx, outDir, target)
	if err != nil {
		c.reporter.Failure("compile", lang.Name+" ["+targetLabel+"]", err, "")
		return err
	}
	defer cleanup()

	toolchainEnv["SOURCE_DATE_EPOCH"] = fmt.Sprintf("%d", sourceProvenance.SourceDateEpoch)
	if len(cfg.BuildFlags) > 0 {
		toolchainEnv["CFLAGS"] = strings.Join(cfg.BuildFlags, " ")
		toolchainEnv["CXXFLAGS"] = strings.Join(cfg.BuildFlags, " ")
	}

	isCross := !isHostTarget(target.Platform, target.Arch)
	buildErr := c.runner.Run(ctx, _jsii.Command{
		Name:          "tree-sitter",
		Args:          []string{"build", "--output", absoluteOutputPath},
		Dir:           src,
		Env:           toolchainEnv,
		SilenceStderr: isCross,
	})
	if buildErr != nil {
		if isCross && isDlopenError(buildErr) {
			if stat, statErr := os.Stat(absoluteOutputPath); statErr == nil && stat.Mode().IsRegular() && stat.Size() > 0 {
				buildErr = nil
			}
		}
		if buildErr != nil {
			diag := diagnosticsFrom(buildErr)
			wrapped := fmt.Errorf("run tree-sitter build: %w", buildErr)
			c.reporter.Failure("compile", lang.Name+" ["+targetLabel+"]", wrapped, diag)
			return wrapped
		}
	}

	if info, err := os.Stat(absoluteOutputPath); err != nil {
		err = fmt.Errorf("expected compiled library %q was not created", absoluteOutputPath)
		c.reporter.Failure("compile", lang.Name+" ["+targetLabel+"]", err, "")
		return err
	} else if !info.Mode().IsRegular() {
		err = fmt.Errorf("compiled library %q is not a regular file", absoluteOutputPath)
		c.reporter.Failure("compile", lang.Name+" ["+targetLabel+"]", err, "")
		return err
	}

	provenance := BinaryProvenance{
		Filename:         filepath.Base(finalOutputPath),
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
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := verifySourceInputs(src, sourceProvenance); err != nil {
		return err
	}
	if err := validateBinaryFormat(absoluteOutputPath, target.Platform, target.Arch, lang.Constructor); err != nil {
		return err
	}
	provenance.SHA256, err = checksumFile(absoluteOutputPath)
	if err != nil {
		return err
	}
	if err := os.Rename(absoluteOutputPath, finalOutputPath); err != nil {
		return err
	}
	if err := writeJSONAtomic(binaryProvenancePath(finalOutputPath), &provenance); err != nil {
		err = fmt.Errorf("write binary provenance: %w", err)
		c.reporter.Failure("compile", lang.Name+" ["+targetLabel+"]", err, "")
		return err
	}

	c.reporter.Success("compile", lang.Name, targetLabel, time.Since(start))
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
		if cc, err := c.lookup.LookPath(target.Compiler); err == nil {
			cxxFound := true
			var cxxPath string
			if target.CXX != "" {
				if cxx, err := c.lookup.LookPath(target.CXX); err == nil {
					cxxPath = cxx
				} else {
					cxxFound = false
				}
			}
			if cxxFound {
				if identity, err := c.measureCompiler(ctx, cc, target.CompilerVersion); err == nil {
					env["CC"] = cc
					if cxxPath != "" {
						env["CXX"] = cxxPath
					}
					return env, identity, func() {}, nil
				}
			}
		}
	}

	if !_jsii.BoolValue(c.policy.Compiler.AllowFallback, true) {
		return nil, compilerIdentity{}, nil, fmt.Errorf("configured compiler %q is unavailable or does not match its version; compiler fallback is disabled", target.Compiler)
	}
	if target.GOOS == runtime.GOOS && target.Arch == runtime.GOARCH {
		cc := c.policy.Compiler.HostCC
		if found, err := c.lookup.LookPath(cc); err == nil {
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
		identity, err := c.measureCompiler(ctx, cc, "")
		if err == nil {
			return env, identity, func() {}, nil
		}
	}

	zigPath, err := c.lookup.LookPath(c.policy.Tools.Zig)
	if err != nil {
		if target.Compiler != "" {
			return nil, compilerIdentity{}, nil, fmt.Errorf("no compiler toolchain available for %s/%s; install %s, zig, or a target-specific cross-compiler", target.Platform, target.Arch, target.Compiler)
		}
		return nil, compilerIdentity{}, nil, fmt.Errorf("no compiler toolchain available for %s/%s; install zig or a target-specific cross-compiler", target.Platform, target.Arch)
	}

	if runtime.GOOS == "windows" {
		return nil, compilerIdentity{}, nil, fmt.Errorf("Zig shell wrappers are unsupported on Windows; configure a native target-specific compiler")
	}
	tempDir, err := os.MkdirTemp(baseDir, "zig-toolchain-")
	if err != nil {
		return nil, compilerIdentity{}, nil, fmt.Errorf("create zig toolchain wrappers: %w", err)
	}
	if abs, err := filepath.Abs(tempDir); err == nil {
		tempDir = abs
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

	identity, err := c.measureCompiler(ctx, zigPath, "")
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, compilerIdentity{}, nil, err
	}
	return env, identity, func() {
		_ = os.RemoveAll(tempDir)
	}, nil
}

func (c *Compiler) measureCompiler(ctx context.Context, executable, expectedVersion string) (compilerIdentity, error) {
	versionArg := "--version"
	if base := filepath.Base(executable); base == "zig" || strings.HasPrefix(base, "zig") {
		versionArg = "version"
	}
	output, err := c.runner.Output(ctx, _jsii.Command{Name: executable, Args: []string{versionArg}})
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
	for _, candidate := range c.policy.Compiler.Candidates[target.Platform+"/"+target.Arch] {
		ccPath, err := c.lookup.LookPath(candidate.CC)
		if err != nil {
			continue
		}

		cxxPath := ""
		if candidate.CXX != "" {
			cxxPath, err = c.lookup.LookPath(candidate.CXX)
			if err != nil {
				continue
			}
		}

		return ccPath, cxxPath, true
	}

	return "", "", false
}

func zigTargetTriple(target target) string {
	switch target.GOOS {
	case "windows":
		if target.Arch == "arm64" {
			return "aarch64-windows-gnu"
		}
		return "x86_64-windows-gnu"
	case "darwin":
		if target.Arch == "arm64" {
			return "aarch64-macos"
		}
		return "x86_64-macos"
	default:
		if target.Arch == "arm64" {
			return "aarch64-linux-gnu"
		}
		return "x86_64-linux-gnu"
	}
}

func writeWrapper(path, zigPath, compiler, target string) error {
	content := fmt.Sprintf(`#!/usr/bin/env bash
args=()
skip=0
for arg in "$@"; do
    if [ "$skip" -eq 1 ]; then
        skip=0
        continue
    fi
    if [ "$arg" = "-target" ]; then
        skip=1
        continue
    fi
    case "$arg" in
        --target=*) ;;
        *) args+=("$arg") ;;
    esac
done
exec %q %s -target %s "${args[@]}"
`, zigPath, compiler, target)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return fmt.Errorf("write compiler wrapper %q: %w", path, err)
	}
	return nil
}

func isDlopenError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "dlopen failed") ||
		strings.Contains(msg, "Error opening dynamic library") ||
		strings.Contains(msg, "cannot open shared object file")
}

func (c *Compiler) CompileWasm(ctx context.Context, lang _jsii.Language, buildDir string) error {
	local := *c
	local.policy = (&_jsii.Config{Runtime: c.policy}).EffectiveRuntime()
	local.runner = _jsii.WithToolPaths(c.runner, local.policy.Tools)
	c = &local
	start := time.Now()
	c.reporter.Start("compile wasm", lang.Name)

	grammarDir := buildDir
	if lang.SourceSubdir != "" {
		candidate := filepath.Join(buildDir, filepath.FromSlash(lang.SourceSubdir))
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			grammarDir = candidate
		}
	}

	info, err := os.Stat(grammarDir)
	if err != nil || !info.IsDir() {
		err = fmt.Errorf("grammar directory %q not found; run `tree-sitter build` first", grammarDir)
		c.reporter.Failure("compile wasm", lang.Name, err, "")
		return err
	}

	wasmFilename := fmt.Sprintf("tree-sitter-%s.wasm", lang.Name)
	outputPath := filepath.Join(buildDir, wasmFilename)
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		absOutputPath = outputPath
	}

	backend := c.policy.WASM.Backend
	tools := c.policy.Tools
	var selected string
	for _, candidate := range []struct{ name, executable string }{{"emcc", tools.EMCC}, {"docker", tools.Docker}, {"podman", tools.Podman}} {
		if backend != "auto" && backend != candidate.name {
			continue
		}
		if path, err := c.lookup.LookPath(candidate.executable); err == nil {
			backend, selected = candidate.name, path
			break
		}
	}
	if selected == "" {
		return fmt.Errorf("WASM backend %q is unavailable; configure runtime.tools and runtime.wasm.backend", backend)
	}
	cmdEnv := make(map[string]string)
	if backend != "emcc" || tools.EMCC != "emcc" {
		wrapperDir, cleanup, err := c.createEmscriptenWrapper(buildDir, grammarDir, backend, selected)
		if err != nil {
			return err
		}
		defer cleanup()
		cmdEnv["PATH"] = wrapperDir + string(os.PathListSeparator) + os.Getenv("PATH")
	}

	buildErr := c.runner.Run(ctx, _jsii.Command{
		Name: "tree-sitter",
		Args: []string{"build", "--wasm", "--output", absOutputPath},
		Dir:  grammarDir,
		Env:  cmdEnv,
	})
	if buildErr != nil {
		diag := diagnosticsFrom(buildErr)
		wrapped := fmt.Errorf("run tree-sitter build --wasm: %w", buildErr)
		c.reporter.Failure("compile wasm", lang.Name, wrapped, diag)
		return wrapped
	}

	if _, err := os.Stat(absOutputPath); err != nil {
		candidates := []string{
			filepath.Join(grammarDir, wasmFilename),
			filepath.Join(grammarDir, fmt.Sprintf("tree-sitter-%s.wasm", lang.Grammar)),
		}
		found := false
		for _, cand := range candidates {
			if stat, sErr := os.Stat(cand); sErr == nil && stat.Mode().IsRegular() && stat.Size() > 0 {
				if err := copyFile(cand, absOutputPath, true); err == nil {
					found = true
					break
				}
			}
		}
		if !found {
			err := fmt.Errorf("expected compiled wasm %q was not created", absOutputPath)
			c.reporter.Failure("compile wasm", lang.Name, err, "")
			return err
		}
	}

	c.reporter.Success("compile wasm", lang.Name, "wasm", time.Since(start))
	return nil
}

func (c *Compiler) createEmscriptenWrapper(baseDir, grammarDir, backend, executable string) (string, func(), error) {
	if runtime.GOOS == "windows" {
		return "", nil, fmt.Errorf("container WASM shell wrappers are unsupported on Windows; configure native emcc")
	}
	tempDir, err := os.MkdirTemp(baseDir, "wasm-toolchain-")
	if err != nil {
		return "", nil, fmt.Errorf("create wasm toolchain wrapper dir: %w", err)
	}
	absGrammarDir, err := filepath.Abs(grammarDir)
	if err != nil {
		absGrammarDir = grammarDir
	}
	wrapperPath := filepath.Join(tempDir, "emcc")
	content := "#!/bin/sh\nexec " + shellQuote(executable) + " \"$@\"\n"
	if backend != "emcc" {
		absBuildDir, err := filepath.Abs(baseDir)
		if err != nil {
			return "", nil, err
		}
		content = "#!/bin/sh\nexec " + shellQuote(executable) + " run --rm -v " + shellQuote(absGrammarDir+":"+absGrammarDir) + " -v " + shellQuote(absBuildDir+":"+absBuildDir) + " -w " + shellQuote(absGrammarDir) + " " + shellQuote(c.policy.WASM.Image) + " emcc \"$@\"\n"
	}
	if err := os.WriteFile(wrapperPath, []byte(content), 0o755); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("write emcc wrapper: %w", err)
	}
	return tempDir, func() {
		_ = os.RemoveAll(tempDir)
	}, nil
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
