package build_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/michaelbomholt665/go-tree-sitter/internal/tests/testutil"
	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

func TestCompilerBuildsNamedBinary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "build", "python")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	writeSourceProvenance(t, sourceDir)

	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: filepath.Join(dir, "build"),
		ABIVersions: map[string]ts.ABIRange{
			"0.25.0": {Min: 13, Max: 14},
		},
		Languages: []ts.Language{{
			Name:              "python",
			Version:           "v0.25.0",
			TreeSitterVersion: "0.25.0",
			Repository:        "https://github.com/tree-sitter/tree-sitter-python.git",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	runner := &testutil.RecordingRunner{
		OnRun: func(_ context.Context, cmd ts.Command) error {
			outputPath := cmd.Args[len(cmd.Args)-1]
			return os.WriteFile(outputPath, []byte("binary"), 0o644)
		},
	}

	compiler := tsbuild.NewCompiler(runner, testutil.StaticLookup{}, ioDiscard{}, ioDiscard{})
	if err := compiler.Compile(context.Background(), cfg, ts.CompileRequest{Language: "python", OS: "linux", Arch: "amd64"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if len(runner.Runs) != 1 {
		t.Fatalf("expected a single build command, got %d", len(runner.Runs))
	}

	output := filepath.Join(dir, "build", "python", "bin", "python-v0.25.0-linux-amd64.so")
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("expected binary %q to exist: %v", output, err)
	}
	if runner.Runs[0].Env["GOOS"] != "linux" || runner.Runs[0].Env["GOARCH"] != "amd64" {
		t.Fatalf("expected target env to be set, got %+v", runner.Runs[0].Env)
	}
}

func TestCompilerUsesConfiguredOSTarget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "build", "python", "amd64")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	writeSourceProvenance(t, sourceDir)

	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: filepath.Join(dir, "build"),
		OSTarget: "windows",
		Targets: map[string]ts.BuildTarget{
			"windows": {OS: "windows", Arch: "amd64"},
		},
		ABIVersions: map[string]ts.ABIRange{
			"0.25.0": {Min: 13, Max: 14},
		},
		Languages: []ts.Language{{
			Name:              "python",
			Version:           "v0.25.0",
			TreeSitterVersion: "0.25.0",
			Repository:        "https://github.com/tree-sitter/tree-sitter-python.git",
		}},
		Output: ts.Output{
			GrammarBase:      filepath.Join(dir, "out"),
			GrammarBuild:     filepath.Join(dir, "build", "{lang}", "{arch}"),
			GrammarCompile:   filepath.Join(dir, "build", "{lang}", "{arch}"),
			GenerateManifest: true,
		},
	}

	runner := &testutil.RecordingRunner{
		OnRun: func(_ context.Context, cmd ts.Command) error {
			outputPath := cmd.Args[len(cmd.Args)-1]
			return os.WriteFile(outputPath, []byte("binary"), 0o644)
		},
	}

	compiler := tsbuild.NewCompiler(runner, testutil.StaticLookup{Paths: map[string]string{
		"x86_64-w64-mingw32-gcc": "/usr/bin/x86_64-w64-mingw32-gcc",
		"x86_64-w64-mingw32-g++": "/usr/bin/x86_64-w64-mingw32-g++",
	}}, ioDiscard{}, ioDiscard{})
	if err := compiler.Compile(context.Background(), cfg, ts.CompileRequest{Language: "python"}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if len(runner.Runs) != 1 {
		t.Fatalf("expected one build command, got %d", len(runner.Runs))
	}
	expected := filepath.Join(dir, "build", "python", "amd64", "python-v0.25.0-windows-amd64.dll")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected binary %q to exist: %v", expected, err)
	}
}

func writeSourceProvenance(t *testing.T, dir string) {
	t.Helper()
	payload := []byte(`{"source_repository":"https://example.invalid/grammar","source_revision":"0123456789012345678901234567890123456789","generator_version":"0.26.8","source_date_epoch":1700000000,"node_types_sha256":"test"}`)
	if err := os.WriteFile(filepath.Join(dir, ".source-provenance.json"), payload, 0o644); err != nil {
		t.Fatalf("write source provenance: %v", err)
	}
}

func TestCompilerRequiresBuiltSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: filepath.Join(dir, "build"),
		ABIVersions: map[string]ts.ABIRange{
			"0.25.0": {Min: 13, Max: 14},
		},
		Languages: []ts.Language{{
			Name:              "python",
			Version:           "v0.25.0",
			TreeSitterVersion: "0.25.0",
			Repository:        "https://github.com/tree-sitter/tree-sitter-python.git",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	compiler := tsbuild.NewCompiler(&testutil.RecordingRunner{}, testutil.StaticLookup{}, ioDiscard{}, ioDiscard{})
	if err := compiler.Compile(context.Background(), cfg, ts.CompileRequest{Language: "python", OS: "linux", Arch: "amd64"}); err == nil {
		t.Fatalf("expected missing source directory to fail")
	}
}

func TestCompilerCompileWasmWithEmcc(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "build", "python")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}

	lang := ts.Language{
		Name:    "python",
		Grammar: "python",
		Version: "v0.25.0",
	}

	runner := &testutil.RecordingRunner{
		OnRun: func(_ context.Context, cmd ts.Command) error {
			outputPath := cmd.Args[len(cmd.Args)-1]
			return os.WriteFile(outputPath, []byte("wasm bytes"), 0o644)
		},
	}

	compiler := tsbuild.NewCompiler(runner, testutil.StaticLookup{Paths: map[string]string{
		"emcc": "/usr/bin/emcc",
	}}, ioDiscard{}, ioDiscard{})

	if err := compiler.CompileWasm(context.Background(), lang, sourceDir); err != nil {
		t.Fatalf("CompileWasm returned error: %v", err)
	}

	if len(runner.Runs) != 1 {
		t.Fatalf("expected 1 run command, got %d", len(runner.Runs))
	}
	runCmd := runner.Runs[0]
	if runCmd.Name != "tree-sitter" || runCmd.Args[0] != "build" || runCmd.Args[1] != "--wasm" {
		t.Fatalf("unexpected command run: %+v", runCmd)
	}
	expectedWasm := filepath.Join(sourceDir, "tree-sitter-python.wasm")
	if _, err := os.Stat(expectedWasm); err != nil {
		t.Fatalf("expected wasm artifact %q to exist: %v", expectedWasm, err)
	}
}

func TestCompilerCompileWasmWithDockerFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "build", "python")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}

	lang := ts.Language{
		Name:    "python",
		Grammar: "python",
		Version: "v0.25.0",
	}

	runner := &testutil.RecordingRunner{
		OnRun: func(_ context.Context, cmd ts.Command) error {
			outputPath := cmd.Args[len(cmd.Args)-1]
			return os.WriteFile(outputPath, []byte("wasm bytes"), 0o644)
		},
	}

	// docker is available, emcc is NOT
	compiler := tsbuild.NewCompiler(runner, testutil.StaticLookup{Paths: map[string]string{
		"docker": "/usr/bin/docker",
	}}, ioDiscard{}, ioDiscard{})

	if err := compiler.CompileWasm(context.Background(), lang, sourceDir); err != nil {
		t.Fatalf("CompileWasm returned error: %v", err)
	}

	if len(runner.Runs) != 1 {
		t.Fatalf("expected 1 run command, got %d", len(runner.Runs))
	}
	runCmd := runner.Runs[0]
	if runCmd.Name != "tree-sitter" || runCmd.Args[0] != "build" || runCmd.Args[1] != "--wasm" {
		t.Fatalf("unexpected command run: %+v", runCmd)
	}
	if runCmd.Env["PATH"] == "" {
		t.Fatalf("expected PATH in Env to contain wrapper directory")
	}
	expectedWasm := filepath.Join(sourceDir, "tree-sitter-python.wasm")
	if _, err := os.Stat(expectedWasm); err != nil {
		t.Fatalf("expected wasm artifact %q to exist: %v", expectedWasm, err)
	}
}

func TestCompilerCompileWasmNoToolchainFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "build", "python")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}

	lang := ts.Language{
		Name:    "python",
		Grammar: "python",
		Version: "v0.25.0",
	}

	compiler := tsbuild.NewCompiler(&testutil.RecordingRunner{}, testutil.StaticLookup{}, ioDiscard{}, ioDiscard{})
	err := compiler.CompileWasm(context.Background(), lang, sourceDir)
	if err == nil {
		t.Fatalf("expected error when no wasm toolchain available")
	}
}

func TestCompilerCompileWithBuildWasm(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "build", "python")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	writeSourceProvenance(t, sourceDir)

	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: filepath.Join(dir, "build"),
		ABIVersions: map[string]ts.ABIRange{
			"0.25.0": {Min: 13, Max: 14},
		},
		Languages: []ts.Language{{
			Name:              "python",
			Version:           "v0.25.0",
			TreeSitterVersion: "0.25.0",
			Repository:        "https://github.com/tree-sitter/tree-sitter-python.git",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	runner := &testutil.RecordingRunner{
		OnRun: func(_ context.Context, cmd ts.Command) error {
			outputPath := cmd.Args[len(cmd.Args)-1]
			return os.WriteFile(outputPath, []byte("artifact"), 0o644)
		},
	}

	compiler := tsbuild.NewCompiler(runner, testutil.StaticLookup{Paths: map[string]string{
		"emcc": "/usr/bin/emcc",
	}}, ioDiscard{}, ioDiscard{})

	if err := compiler.Compile(context.Background(), cfg, ts.CompileRequest{
		Language:  "python",
		OS:        "linux",
		Arch:      "amd64",
		BuildWasm: true,
	}); err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if len(runner.Runs) != 2 {
		t.Fatalf("expected 2 runs (native + wasm), got %d", len(runner.Runs))
	}
	expectedNative := filepath.Join(dir, "build", "python", "bin", "python-v0.25.0-linux-amd64.so")
	expectedWasm := filepath.Join(dir, "build", "python", "tree-sitter-python.wasm")
	if _, err := os.Stat(expectedNative); err != nil {
		t.Fatalf("expected native binary to exist: %v", err)
	}
	if _, err := os.Stat(expectedWasm); err != nil {
		t.Fatalf("expected wasm artifact to exist: %v", err)
	}
}
