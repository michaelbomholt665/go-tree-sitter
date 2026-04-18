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

func TestBuilderClonesAndGeneratesGrammarSources(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: filepath.Join(dir, "build"),
		ABIVersions: map[string]ts.ABIRange{
			"0.25.0": {Min: 13, Max: 14},
		},
		Languages: []ts.Language{{
			Name:              "tsx",
			Version:           "v0.23.2",
			TreeSitterVersion: "0.25.0",
			Repository:        "https://github.com/tree-sitter/tree-sitter-typescript.git",
			SourceSubdir:      "tsx",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	runner := &testutil.RecordingRunner{
		OnRun: func(_ context.Context, cmd ts.Command) error {
			switch cmd.Name {
			case "git":
				if len(cmd.Args) > 0 && cmd.Args[0] == "clone" {
					dest := cmd.Args[len(cmd.Args)-1]
					return os.MkdirAll(filepath.Join(dest, "tsx"), 0o755)
				}
			case "tree-sitter":
				if cmd.Dir != filepath.Join(cfg.BuildDir, "tsx", "tsx") {
					t.Fatalf("expected generate dir %q, got %q", filepath.Join(cfg.BuildDir, "tsx", "tsx"), cmd.Dir)
				}
			}
			return nil
		},
	}

	builder := tsbuild.NewBuilder(runner, ioDiscard{}, ioDiscard{})
	if err := builder.Build(context.Background(), cfg, ts.BuildRequest{Language: "tsx"}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if len(runner.Runs) != 4 {
		t.Fatalf("expected clone, fetch, checkout, and generate commands, got %d", len(runner.Runs))
	}
}

func TestBuilderChecksOutExistingRepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	root := filepath.Join(dir, "build", "python")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create build root: %v", err)
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

	runner := &testutil.RecordingRunner{}
	builder := tsbuild.NewBuilder(runner, ioDiscard{}, ioDiscard{})
	if err := builder.Build(context.Background(), cfg, ts.BuildRequest{Language: "python"}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if got := len(runner.Runs); got != 3 {
		t.Fatalf("expected fetch, checkout, and generate commands, got %d", got)
	}
	if runner.Runs[0].Name != "git" || runner.Runs[1].Name != "git" || runner.Runs[2].Name != "tree-sitter" {
		t.Fatalf("unexpected command order: %+v", runner.Runs)
	}
}

func TestBuilderInstallsNodeDependenciesWhenPackageJSONExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: filepath.Join(dir, "build"),
		ABIVersions: map[string]ts.ABIRange{
			"0.23.3": {Min: 14, Max: 14},
		},
		Languages: []ts.Language{{
			Name:              "tsx",
			Version:           "v0.23.3-0.20250130221139-75b3874edb2d",
			TreeSitterVersion: "0.23.3",
			Repository:        "https://github.com/tree-sitter/tree-sitter-typescript.git",
			SourceSubdir:      "tsx",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	runner := &testutil.RecordingRunner{
		OnRun: func(_ context.Context, cmd ts.Command) error {
			switch cmd.Name {
			case "git":
				if len(cmd.Args) > 0 && cmd.Args[0] == "clone" {
					dest := cmd.Args[len(cmd.Args)-1]
					if err := os.MkdirAll(filepath.Join(dest, "tsx"), 0o755); err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(dest, "package.json"), []byte(`{"dependencies":{"tree-sitter-javascript":"^0.23.1"}}`), 0o644)
				}
			case "npm":
				return os.MkdirAll(filepath.Join(cmd.Dir, "node_modules"), 0o755)
			}
			return nil
		},
	}

	builder := tsbuild.NewBuilder(runner, ioDiscard{}, ioDiscard{})
	if err := builder.Build(context.Background(), cfg, ts.BuildRequest{Language: "tsx"}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	foundNPM := false
	for _, run := range runner.Runs {
		if run.Name == "npm" {
			foundNPM = true
			if got, want := run.Args, []string{"install", "--ignore-scripts"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("unexpected npm args: %v", got)
			}
		}
	}
	if !foundNPM {
		t.Fatalf("expected npm install to run before generation, got %+v", runner.Runs)
	}
}

func TestBuilderUsesPseudoVersionCommitForCheckout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: filepath.Join(dir, "build"),
		ABIVersions: map[string]ts.ABIRange{
			"0.25.0": {Min: 15, Max: 15},
		},
		Languages: []ts.Language{{
			Name:              "proto",
			Version:           "v0.0.0-20260315065021-d65a18ce7c22",
			TreeSitterVersion: "0.25.0",
			Repository:        "https://github.com/coder3101/tree-sitter-proto.git",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	runner := &testutil.RecordingRunner{
		OnRun: func(_ context.Context, cmd ts.Command) error {
			if cmd.Name == "git" && len(cmd.Args) > 0 && cmd.Args[0] == "clone" {
				dest := cmd.Args[len(cmd.Args)-1]
				return os.MkdirAll(dest, 0o755)
			}
			return nil
		},
	}

	builder := tsbuild.NewBuilder(runner, ioDiscard{}, ioDiscard{})
	if err := builder.Build(context.Background(), cfg, ts.BuildRequest{Language: "proto"}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if got := len(runner.Runs); got != 4 {
		t.Fatalf("expected clone, fetch, checkout, and generate commands, got %d", got)
	}

	got := runner.Runs[0].Args
	if len(got) < 3 || got[0] != "clone" {
		t.Fatalf("unexpected clone args for pseudo-version checkout: %v", got)
	}
	for _, arg := range got {
		if arg == "--branch" {
			t.Fatalf("pseudo-version clone should not use --branch: %v", got)
		}
	}
	if runner.Runs[2].Args[len(runner.Runs[2].Args)-1] != "d65a18ce7c22" {
		t.Fatalf("expected pseudo-version checkout to use commit hash, got %v", runner.Runs[2].Args)
	}
}

func TestBuilderUsesPrereleasePseudoVersionCommitForCheckout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: filepath.Join(dir, "build"),
		ABIVersions: map[string]ts.ABIRange{
			"0.23.3": {Min: 14, Max: 14},
		},
		Languages: []ts.Language{{
			Name:              "tsx",
			Version:           "v0.23.3-0.20250130221139-75b3874edb2d",
			TreeSitterVersion: "0.23.3",
			Repository:        "https://github.com/tree-sitter/tree-sitter-typescript.git",
			SourceSubdir:      "tsx",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	runner := &testutil.RecordingRunner{
		OnRun: func(_ context.Context, cmd ts.Command) error {
			if cmd.Name == "git" && len(cmd.Args) > 0 && cmd.Args[0] == "clone" {
				dest := cmd.Args[len(cmd.Args)-1]
				return os.MkdirAll(filepath.Join(dest, "tsx"), 0o755)
			}
			return nil
		},
	}

	builder := tsbuild.NewBuilder(runner, ioDiscard{}, ioDiscard{})
	if err := builder.Build(context.Background(), cfg, ts.BuildRequest{Language: "tsx"}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	for _, arg := range runner.Runs[0].Args {
		if arg == "--branch" {
			t.Fatalf("prerelease pseudo-version clone should not use --branch: %v", runner.Runs[0].Args)
		}
	}
	if runner.Runs[2].Args[len(runner.Runs[2].Args)-1] != "75b3874edb2d" {
		t.Fatalf("expected prerelease pseudo-version checkout to use commit hash, got %v", runner.Runs[2].Args)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
