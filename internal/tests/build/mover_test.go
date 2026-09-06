package build_test

import (
	"context"
	"encoding/json"
	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
	"os"
	"path/filepath"
	"testing"
)

type acceptingValidator struct{}

func (acceptingValidator) Validate(_ context.Context, items []tsbuild.ValidationItem) (map[string]uint32, error) {
	result := make(map[string]uint32, len(items))
	for _, item := range items {
		result[item.BinaryPath] = 15
	}
	return result, nil
}

func writeBinaryProvenance(t *testing.T, binaryPath, nodeTypesPath string) {
	t.Helper()
	checksums, err := tsbuild.CalculateChecksums([]string{nodeTypesPath})
	if err != nil {
		t.Fatalf("checksum node types: %v", err)
	}
	payload, err := json.Marshal(tsbuild.BinaryProvenance{
		Filename: filepath.Base(binaryPath), SourceRepository: "https://example.invalid/grammar",
		SourceRevision: "0123456789012345678901234567890123456789", GeneratorVersion: "0.26.8",
		SourceDateEpoch: 1_700_000_000, NodeTypesSHA256: checksums[nodeTypesPath], TargetTriple: "x86_64-linux-gnu",
		Compiler: "gcc", CompilerVersion: "gcc 14.2.0",
	})
	if err != nil {
		t.Fatalf("marshal provenance: %v", err)
	}
	if err := os.WriteFile(binaryPath+".provenance.json", payload, 0o644); err != nil {
		t.Fatalf("write provenance: %v", err)
	}
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
	writeBinaryProvenance(t, filepath.Join(binDir, "python-v0.25.0-linux-amd64.so"), filepath.Join(sourceDir, "src", "node-types.json"))

	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: buildDir,
		ABIVersions: map[string]ts.ABIRange{
			">=0.25": {Min: 13, Max: 15},
		},
		Languages: []ts.Language{{
			Name:              "python",
			Grammar:           "python",
			Constructor:       "tree_sitter_python",
			Version:           "v0.25.0",
			TreeSitterVersion: "0.25.0",
			Repository:        "https://github.com/tree-sitter/tree-sitter-python.git",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	mover := tsbuild.NewMoverWithValidator(tsbuild.NewCleaner(), acceptingValidator{}, ioDiscard{}, ioDiscard{})
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
			">=0.25": {Min: 13, Max: 15},
		},
		Languages: []ts.Language{{
			Name:              "python",
			Version:           "v0.25.0",
			TreeSitterVersion: "0.25.0",
			Repository:        "https://github.com/tree-sitter/tree-sitter-python.git",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	mover := tsbuild.NewMoverWithValidator(tsbuild.NewCleaner(), acceptingValidator{}, ioDiscard{}, ioDiscard{})
	if err := mover.Move(context.Background(), cfg, ts.MoveRequest{Language: "python", Mode: ts.MoveModeBoth, Clean: true, Force: true}); err == nil {
		t.Fatalf("expected publication without staged build provenance to fail")
	}
	preserved, err := os.ReadFile(filepath.Join(outputDir, "python-v0.25.0-linux-amd64.so"))
	if err != nil || string(preserved) != "binary" {
		t.Fatalf("expected previous usable release to remain unchanged, bytes=%q err=%v", preserved, err)
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
	writeBinaryProvenance(t, filepath.Join(binDir, "tsx-v0.23.3-linux-amd64.so"), filepath.Join(sourceDir, "src", "node-types.json"))

	cfg := &ts.Config{
		Version:  "1.0",
		BuildDir: buildDir,
		ABIVersions: map[string]ts.ABIRange{
			">=0.20.3, <=0.24": {Min: 13, Max: 14},
		},
		Languages: []ts.Language{{
			Name:              "tsx",
			Grammar:           "tsx",
			Constructor:       "tree_sitter_tsx",
			Version:           "v0.23.3",
			TreeSitterVersion: "0.23.3",
			Repository:        "https://github.com/tree-sitter/tree-sitter-typescript.git",
			SourceSubdir:      "tsx",
		}},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "out"), GenerateManifest: true},
	}

	mover := tsbuild.NewMoverWithValidator(tsbuild.NewCleaner(), acceptingValidator{}, ioDiscard{}, ioDiscard{})
	if err := mover.Move(context.Background(), cfg, ts.MoveRequest{Language: "tsx", Mode: ts.MoveModeBoth, Clean: true, Force: true}); err != nil {
		t.Fatalf("Move returned error: %v", err)
	}

	copiedQuery := filepath.Join(dir, "out", "tsx", "queries", "highlights.scm")
	if _, err := os.Stat(copiedQuery); err != nil {
		t.Fatalf("expected root query to be copied: %v", err)
	}
}

func TestMoverCrossPlatformPublishing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	sourceDir := filepath.Join(buildDir, "python", "amd64")
	arm64Dir := filepath.Join(buildDir, "python", "arm64")

	if err := os.MkdirAll(filepath.Join(sourceDir, "src"), 0o755); err != nil {
		t.Fatalf("create source src dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sourceDir, "queries"), 0o755); err != nil {
		t.Fatalf("create queries dir: %v", err)
	}
	if err := os.MkdirAll(arm64Dir, 0o755); err != nil {
		t.Fatalf("create arm64 dir: %v", err)
	}

	nodeTypesPath := filepath.Join(sourceDir, "src", "node-types.json")
	if err := os.WriteFile(nodeTypesPath, []byte(`{"node":"type"}`), 0o644); err != nil {
		t.Fatalf("write node-types: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "queries", "highlights.scm"), []byte("(identifier)"), 0o644); err != nil {
		t.Fatalf("write query: %v", err)
	}

	binaries := map[string]string{
		filepath.Join(sourceDir, "python-v0.25.0-linux-amd64.so"):    "x86_64-linux-gnu",
		filepath.Join(sourceDir, "python-v0.25.0-windows-amd64.dll"): "x86_64-windows-gnu",
		filepath.Join(sourceDir, "python-v0.25.0-macos-amd64.dylib"): "x86_64-apple-darwin",
		filepath.Join(arm64Dir, "python-v0.25.0-macos-arm64.dylib"):  "aarch64-apple-darwin",
	}

	for binPath, triple := range binaries {
		if err := os.WriteFile(binPath, []byte("mock binary bytes for "+filepath.Base(binPath)), 0o644); err != nil {
			t.Fatalf("write binary %s: %v", binPath, err)
		}
		checksums, err := tsbuild.CalculateChecksums([]string{nodeTypesPath})
		if err != nil {
			t.Fatalf("checksum node types: %v", err)
		}
		abi := 15
		payload, err := json.Marshal(tsbuild.BinaryProvenance{
			Filename:         filepath.Base(binPath),
			SourceRepository: "https://github.com/tree-sitter/tree-sitter-python.git",
			SourceRevision:   "0123456789012345678901234567890123456789",
			GeneratorVersion: "0.26.8",
			GenerateABI:      &abi,
			SourceDateEpoch:  1_700_000_000,
			NodeTypesSHA256:  checksums[nodeTypesPath],
			TargetTriple:     triple,
			Compiler:         "gcc",
			CompilerVersion:  "14.2.0",
		})
		if err != nil {
			t.Fatalf("marshal provenance: %v", err)
		}
		if err := os.WriteFile(binPath+".provenance.json", payload, 0o644); err != nil {
			t.Fatalf("write provenance: %v", err)
		}
	}

	cfg := &ts.Config{
		Version:     "2.0",
		BuildDir:    buildDir,
		OSTarget:    "linux",
		GenerateABI: 15,
		Targets: map[string]ts.BuildTarget{
			"linux":       {OS: "linux", Arch: "amd64", Triple: "x86_64-linux-gnu", Compiler: "gcc", CompilerVersion: "14.2.0"},
			"windows":     {OS: "windows", Arch: "amd64", Triple: "x86_64-windows-gnu", Compiler: "x86_64-w64-mingw32-gcc", CompilerVersion: "14.2.0"},
			"macos-amd64": {OS: "macos", Arch: "amd64", Triple: "x86_64-apple-darwin", Compiler: "o64-clang", CompilerVersion: "18.1.8"},
			"macos-arm64": {OS: "macos", Arch: "arm64", Triple: "aarch64-apple-darwin", Compiler: "oa64-clang", CompilerVersion: "18.1.8"},
		},
		Languages: []ts.Language{{
			Name:        "python",
			Grammar:     "python",
			Constructor: "tree_sitter_python",
			Version:     "v0.25.0",
			Repository:  "https://github.com/tree-sitter/tree-sitter-python.git",
			Revision:    "0123456789012345678901234567890123456789",
			Sample:      "x = 1\n",
		}},
		Output: ts.Output{
			GrammarBase:      filepath.Join(dir, "out"),
			GrammarBuild:     filepath.Join(dir, "build", "{lang}", "{arch}"),
			GrammarCompile:   filepath.Join(dir, "build", "{lang}", "{arch}"),
			GenerateManifest: true,
			DefaultMoveMode:  "--both",
		},
	}

	mover := tsbuild.NewMoverWithValidator(tsbuild.NewCleaner(), acceptingValidator{}, ioDiscard{}, ioDiscard{})
	if err := mover.Move(context.Background(), cfg, ts.MoveRequest{
		Language:             "python",
		Mode:                 ts.MoveModeBoth,
		Clean:                true,
		AllowCrossValidation: true,
	}); err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	outputDir := filepath.Join(dir, "out", "python")
	for binName := range map[string]struct{}{
		"python-v0.25.0-linux-amd64.so":    {},
		"python-v0.25.0-windows-amd64.dll": {},
		"python-v0.25.0-macos-amd64.dylib": {},
		"python-v0.25.0-macos-arm64.dylib": {},
		"node-types.json":                  {},
		"manifest.json":                    {},
	} {
		if _, err := os.Stat(filepath.Join(outputDir, binName)); err != nil {
			t.Fatalf("expected output file %q to exist: %v", binName, err)
		}
	}

	manifestBytes, err := os.ReadFile(filepath.Join(outputDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest tsbuild.ManifestData
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if len(manifest.Binaries) != 4 {
		t.Fatalf("expected 4 binaries in manifest, got %d", len(manifest.Binaries))
	}
	if len(manifest.BuildProvenance) != 4 {
		t.Fatalf("expected 4 provenance entries in manifest, got %d", len(manifest.BuildProvenance))
	}

	platforms := make(map[string]bool)
	for _, b := range manifest.Binaries {
		platforms[b.Platform+"-"+b.Arch] = true
		if len(b.ChecksumSHA256) != 64 {
			t.Errorf("expected 64-char sha256 checksum, got %q", b.ChecksumSHA256)
		}
	}

	for _, expectedKey := range []string{"linux-amd64", "windows-amd64", "macos-amd64", "macos-arm64"} {
		if !platforms[expectedKey] {
			t.Errorf("manifest missing binary for target %q", expectedKey)
		}
	}
}
