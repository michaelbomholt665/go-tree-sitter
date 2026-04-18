package build_test

import (
	"os"
	"path/filepath"
	"testing"

	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

func TestCleanerRemovesLanguageDirectoryAndPreservesBuildRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	languageDir := filepath.Join(buildDir, "python")
	if err := os.MkdirAll(languageDir, 0o755); err != nil {
		t.Fatalf("create language dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(languageDir, "artifact"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	cleaner := tsbuild.NewCleaner()
	if err := cleaner.CleanLanguage(buildDir, "python"); err != nil {
		t.Fatalf("CleanLanguage returned error: %v", err)
	}

	if _, err := os.Stat(languageDir); !os.IsNotExist(err) {
		t.Fatalf("expected language directory to be removed, got err=%v", err)
	}
	if info, err := os.Stat(buildDir); err != nil || !info.IsDir() {
		t.Fatalf("expected build root to remain, err=%v", err)
	}
}
