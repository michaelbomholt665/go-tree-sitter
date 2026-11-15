package build

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReviewSourceInputHashes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	parser := filepath.Join(root, "src", "parser.c")
	if err := os.WriteFile(parser, []byte("parser"), 0644); err != nil {
		t.Fatal(err)
	}
	hashes, err := hashSourceInputs(root)
	if err != nil {
		t.Fatal(err)
	}
	provenance := SourceProvenance{SourceHashes: hashes}
	if err := verifySourceInputs(root, provenance); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parser, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifySourceInputs(root, provenance); err == nil {
		t.Fatal("accepted changed source")
	}
	if err := verifySourceInputs(root, SourceProvenance{}); err == nil {
		t.Fatal("accepted missing hashes")
	}
}
func TestReviewRecoverPreviousRelease(t *testing.T) {
	base := filepath.Join(t.TempDir(), "grammar")
	if err := os.Mkdir(base+".previous", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base+".previous", "release"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := recoverRelease(base); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(base, "release"))
	if err != nil || string(raw) != "old" {
		t.Fatalf("lost previous release: %q %v", raw, err)
	}
}
func TestReviewAtomicPublication(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("directory exchange is currently supported on Linux and macOS")
	}
	root := t.TempDir()
	base, stage := filepath.Join(root, "grammar"), filepath.Join(root, "stage")
	for _, dir := range []string{base, stage} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(base, "old"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "new"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := publishAtomic(stage, base); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, "new")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(stage, "old")); err != nil {
		t.Fatal("exchange did not preserve old catalog", err)
	}
}
