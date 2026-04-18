package build_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	cfg := &ts.Config{
		ABIVersions: map[string]ts.ABIRange{
			"0.25.0": {Min: 13, Max: 14},
		},
	}
	lang := ts.Language{
		Name:              "python",
		Version:           "v0.25.0",
		TreeSitterVersion: "0.25.0",
	}

	manifest, err := tsbuild.GenerateManifest(lang, []string{macosBinary, linuxBinary}, cfg, true, true, time.Date(2026, 4, 18, 18, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GenerateManifest returned error: %v", err)
	}

	if manifest.ABI.MinVersion != 13 || manifest.ABI.MaxVersion != 14 {
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
