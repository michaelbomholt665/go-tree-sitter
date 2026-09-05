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

	manifest, err := tsbuild.GenerateManifest(lang, []string{macosBinary, linuxBinary}, abi, provenance, true, true)
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
	manifest, err := tsbuild.GenerateManifest(ts.Language{Name: "python", Grammar: "python", Version: "v1.0.0"}, []string{path}, measured, provenance, true, true)
	if err != nil {
		t.Fatal(err)
	}
	return manifest, dir, measured
}
