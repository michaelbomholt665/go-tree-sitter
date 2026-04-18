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
