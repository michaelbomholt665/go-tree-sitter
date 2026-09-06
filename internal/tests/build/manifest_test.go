package build_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

func TestGenerateManifestIncludesChecksumsAndABI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	linuxBinary := filepath.Join(dir, "python-v0.25.0-linux-amd64.so")
	macosBinary := filepath.Join(dir, "python-v0.25.0-macos-arm64.dylib")
	if err := os.WriteFile(linuxBinary, []byte("linux"), 0o644); err != nil {
		t.Fatalf("write linux binary: %v", err)
	}
	if err := os.WriteFile(macosBinary, []byte("macos"), 0o644); err != nil {
		t.Fatalf("write macos binary: %v", err)
	}

	lang := ts.Language{
		Name:    "python",
		Grammar: "python",
		Version: "v0.25.0",
	}
	abi := map[string]uint32{linuxBinary: 15, macosBinary: 15}
	provenance := map[string]tsbuild.BinaryProvenance{}
	for _, path := range []string{linuxBinary, macosBinary} {
		provenance[path] = tsbuild.BinaryProvenance{
			Filename: filepath.Base(path), GeneratorVersion: "0.26.8",
			SourceRepository: "https://example.invalid/python", SourceRevision: "0123456789012345678901234567890123456789",
			SourceDateEpoch: 1_700_000_000, TargetTriple: "test", Compiler: "cc", CompilerVersion: "cc 1",
		}
	}

	manifest, err := tsbuild.GenerateManifest(lang, []string{macosBinary, linuxBinary}, abi, provenance, tsbuild.ArtifactInfo{
		HasNodeTypes: true,
		HasQueries:   true,
	})
	if err != nil {
		t.Fatalf("GenerateManifest returned error: %v", err)
	}

	if manifest.ParserABI != 15 || manifest.ABI.MinVersion != 15 || manifest.ABI.MaxVersion != 15 {
		t.Fatalf("unexpected ABI info: %+v", manifest.ABI)
	}
	if len(manifest.Binaries) != 2 {
		t.Fatalf("expected two binaries, got %d", len(manifest.Binaries))
	}
	if manifest.Binaries[0].Filename != "python-v0.25.0-linux-amd64.so" {
		t.Fatalf("expected binaries to be sorted by filename, got %+v", manifest.Binaries)
	}

	outputPath := filepath.Join(dir, "manifest.json")
	if err := manifest.WriteToFile(outputPath); err != nil {
		t.Fatalf("WriteToFile returned error: %v", err)
	}

	payload, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var persisted tsbuild.ManifestData
	if err := json.Unmarshal(payload, &persisted); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if persisted.Grammar != "python" {
		t.Fatalf("unexpected persisted manifest: %+v", persisted)
	}
}

func TestManifestRejectsABI15WithLegacyBounds13To14(t *testing.T) {
	t.Parallel()
	manifest, dir, measured := validTestManifest(t)
	manifest.ABI.MinVersion, manifest.ABI.MaxVersion = 13, 14
	if err := tsbuild.ValidateManifest(manifest, dir, measured); err == nil || !strings.Contains(err.Error(), "must both equal parser_abi 15") {
		t.Fatalf("expected legacy ABI mismatch, got %v", err)
	}
}

func TestManifestRejectsParserABIDifferentFromLoadedBinary(t *testing.T) {
	t.Parallel()
	manifest, dir, measured := validTestManifest(t)
	for path := range measured {
		measured[path] = 14
	}
	if err := tsbuild.ValidateManifest(manifest, dir, measured); err == nil || !strings.Contains(err.Error(), "manifest 15, measured 14") {
		t.Fatalf("expected measured ABI mismatch, got %v", err)
	}
}

func TestManifestRejectsChecksumComputedBeforeFinalPackaging(t *testing.T) {
	t.Parallel()
	manifest, dir, measured := validTestManifest(t)
	path := filepath.Join(dir, manifest.Binaries[0].Filename)
	if err := os.WriteFile(path, []byte("bytes after strip or signing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tsbuild.ValidateManifest(manifest, dir, measured); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected final-byte checksum mismatch, got %v", err)
	}
}

func TestManifestRejectsGeneratorVersionCopiedFromSourceVersion(t *testing.T) {
	t.Parallel()
	manifest, dir, measured := validTestManifest(t)
	manifest.TreeSitterVer = manifest.Version
	manifest.ABI.ParserVersion = manifest.Version
	if err := tsbuild.ValidateManifest(manifest, dir, measured); err == nil || !strings.Contains(err.Error(), "generator version mismatch") {
		t.Fatalf("expected measured generator mismatch, got %v", err)
	}
}

func validTestManifest(t *testing.T) (*tsbuild.ManifestData, string, map[string]uint32) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "python-v1.0.0-linux-amd64.so")
	if err := os.WriteFile(path, []byte("final bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	provenance := map[string]tsbuild.BinaryProvenance{path: {
		Filename: filepath.Base(path), GeneratorVersion: "0.26.8", SourceDateEpoch: 1_700_000_000,
		SourceRepository: "https://example.invalid/python", SourceRevision: "0123456789012345678901234567890123456789",
		TargetTriple: "x86_64-linux-gnu", Compiler: "gcc", CompilerVersion: "gcc 14.2.0",
	}}
	measured := map[string]uint32{path: 15}
	manifest, err := tsbuild.GenerateManifest(ts.Language{Name: "python", Grammar: "python", Version: "v1.0.0"}, []string{path}, measured, provenance, tsbuild.ArtifactInfo{
		HasNodeTypes: true,
		HasQueries:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return manifest, dir, measured
}

func TestManifestArtifactFlagsSerializationAndValidation(t *testing.T) {
	t.Parallel()

	manifest, dir, measured := validTestManifest(t)
	manifest.Artifacts.HasWasm = true
	manifest.Artifacts.HasCSource = true
	manifest.Artifacts.HasJS = true

	// Validation should fail initially because the artifacts do not exist in dir
	err := tsbuild.ValidateManifest(manifest, dir, measured)
	if err == nil {
		t.Fatalf("expected validation failure when declared artifacts are missing")
	}

	// Create the expected artifacts
	if err := os.WriteFile(filepath.Join(dir, "tree-sitter-python.wasm"), []byte("wasm"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "parser.c"), []byte("int parser;"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "grammar.js"), []byte("module.exports = {};"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Validation should now succeed
	if err := tsbuild.ValidateManifest(manifest, dir, measured); err != nil {
		t.Fatalf("expected validation success after creating artifacts, got: %v", err)
	}

	// Test JSON serialization
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	encodedStr := string(encoded)
	for _, expectedKey := range []string{`"has_wasm":true`, `"has_c_source":true`, `"has_js":true`} {
		if !strings.Contains(encodedStr, expectedKey) {
			t.Errorf("expected serialized manifest to contain %s, got %s", expectedKey, encodedStr)
		}
	}

	// Test backward-compatible unmarshaling of legacy JSON without has_c_source and has_js
	legacyJSON := []byte(`{
		"schema_version": 2,
		"grammar": "python",
		"version": "v1.0.0",
		"compiled_at": "2024-11-20T00:54:15Z",
		"tree_sitter_version": "0.26.8",
		"parser_abi": 15,
		"binaries": [],
		"abi": {"min_version": 15, "max_version": 15, "parser_version": "0.26.8"},
		"artifacts": {"has_node_types": true, "has_queries": true, "has_wasm": false},
		"build_provenance": []
	}`)
	var legacyManifest tsbuild.ManifestData
	if err := json.Unmarshal(legacyJSON, &legacyManifest); err != nil {
		t.Fatalf("unmarshal legacy manifest: %v", err)
	}
	if legacyManifest.Artifacts.HasCSource || legacyManifest.Artifacts.HasJS || legacyManifest.Artifacts.HasCompactNodeTypes {
		t.Errorf("expected missing fields to deserialize to false, got: %+v", legacyManifest.Artifacts)
	}
}

func TestManifestValidatesHasCompactNodeTypes(t *testing.T) {
	t.Parallel()
	manifest, dir, measured := validTestManifest(t)

	manifest.Artifacts.HasCompactNodeTypes = true

	// Missing compact-node-types.yaml should fail validation
	if err := tsbuild.ValidateManifest(manifest, dir, measured); err == nil || !strings.Contains(err.Error(), "has_compact_node_types=true but") {
		t.Fatalf("expected error for missing compact-node-types.yaml, got: %v", err)
	}

	// Create compact-node-types.yaml -> should pass
	compactPath := filepath.Join(dir, "compact-node-types.yaml")
	if err := os.WriteFile(compactPath, []byte("supertypes:\n\nnodes:\n"), 0o644); err != nil {
		t.Fatalf("write compact-node-types.yaml: %v", err)
	}

	if err := tsbuild.ValidateManifest(manifest, dir, measured); err != nil {
		t.Fatalf("expected validation success after creating compact-node-types.yaml, got: %v", err)
	}

	// Check JSON serialization
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if !strings.Contains(string(encoded), `"has_compact_node_types":true`) {
		t.Errorf("expected serialized manifest to contain \"has_compact_node_types\":true, got %s", string(encoded))
	}
}
