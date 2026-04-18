package build_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 4, 18, 18, 30, 0, 0, time.UTC)
}

func TestMoverCopiesArtifactsGeneratesManifestAndCleans(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	binDir := filepath.Join(buildDir, "python", "bin")
	sourceDir := filepath.Join(buildDir, "python")

	if err := os.MkdirAll(filepath.Join(sourceDir, "src"), 0o755); err != nil {
		t.Fatalf("create source src dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sourceDir, "queries"), 0o755); err != nil {
		t.Fatalf("create queries dir: %v", err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(sourceDir, "src", "node-types.json"), []byte(`{"node":"type"}`), 0o644); err != nil {
		t.Fatalf("write node-types: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "queries", "highlights.scm"), []byte("(identifier)"), 0o644); err != nil {
		t.Fatalf("write query: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "python-v0.25.0-linux-amd64.so"), []byte("binary"), 0o644); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: buildDir,
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

	mover := tsbuild.NewMover(fixedClock{}, tsbuild.NewCleaner(), ioDiscard{}, ioDiscard{})
	if err := mover.Move(context.Background(), cfg, ts.MoveRequest{Language: "python", Mode: ts.MoveModeBoth, Clean: true}); err != nil {
		t.Fatalf("Move returned error: %v", err)
	}

	outputDir := filepath.Join(dir, "out", "python")
	for _, expected := range []string{
		"python-v0.25.0-linux-amd64.so",
		"node-types.json",
		filepath.Join("queries", "highlights.scm"),
		"manifest.json",
	} {
		if _, err := os.Stat(filepath.Join(outputDir, expected)); err != nil {
			t.Fatalf("expected %q to exist: %v", expected, err)
		}
	}

	if _, err := os.Stat(filepath.Join(buildDir, "python")); !os.IsNotExist(err) {
		t.Fatalf("expected build directory to be cleaned, got err=%v", err)
	}

	manifestContent, err := os.ReadFile(filepath.Join(outputDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest tsbuild.ManifestData
	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if !manifest.Artifacts.HasNodeTypes || !manifest.Artifacts.HasQueries {
		t.Fatalf("unexpected artifact flags: %+v", manifest.Artifacts)
	}
}

func TestMoverSkipsLanguageWhenBuildArtifactsWereAlreadyMoved(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	outputDir := filepath.Join(dir, "out", "python")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "python-v0.25.0-linux-amd64.so"), []byte("binary"), 0o644); err != nil {
		t.Fatalf("write moved binary: %v", err)
	}

	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: buildDir,
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

	mover := tsbuild.NewMover(fixedClock{}, tsbuild.NewCleaner(), ioDiscard{}, ioDiscard{})
	if err := mover.Move(context.Background(), cfg, ts.MoveRequest{Language: "python", Mode: ts.MoveModeBoth, Clean: true, Force: true}); err != nil {
		t.Fatalf("Move returned error: %v", err)
	}
}

func TestMoverCopiesQueriesFromRepositoryRootForSubdirGrammar(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	repoRoot := filepath.Join(buildDir, "tsx")
	sourceDir := filepath.Join(repoRoot, "tsx")
	binDir := filepath.Join(repoRoot, "bin")

	if err := os.MkdirAll(filepath.Join(sourceDir, "src"), 0o755); err != nil {
		t.Fatalf("create source src dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "queries"), 0o755); err != nil {
		t.Fatalf("create root queries dir: %v", err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "src", "node-types.json"), []byte(`{"node":"type"}`), 0o644); err != nil {
		t.Fatalf("write node-types: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "queries", "highlights.scm"), []byte("(identifier)"), 0o644); err != nil {
		t.Fatalf("write root query: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "tsx-v0.23.3-linux-amd64.so"), []byte("binary"), 0o644); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: buildDir,
		ABIVersions: map[string]ts.ABIRange{
			"0.23.3": {Min: 14, Max: 14},
		},
		Languages: []ts.Language{{
			Name:              "tsx",
			Version:           "v0.23.3",
			TreeSitterVersion: "0.23.3",
			Repository:        "https://github.com/tree-sitter/tree-sitter-typescript.git",
			SourceSubdir:      "tsx",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	mover := tsbuild.NewMover(fixedClock{}, tsbuild.NewCleaner(), ioDiscard{}, ioDiscard{})
	if err := mover.Move(context.Background(), cfg, ts.MoveRequest{Language: "tsx", Mode: ts.MoveModeBoth, Clean: true, Force: true}); err != nil {
		t.Fatalf("Move returned error: %v", err)
	}

	copiedQuery := filepath.Join(dir, "out", "tsx", "queries", "highlights.scm")
	if _, err := os.Stat(copiedQuery); err != nil {
		t.Fatalf("expected root query to be copied: %v", err)
	}
}
