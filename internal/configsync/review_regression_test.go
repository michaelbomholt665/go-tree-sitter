package configsync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRequireWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go.mod")
	raw := "module example.com/test\n\ngo 1.26\n\nrequire (\n    github.com/tree-sitter/tree-sitter-go v0.25.0 // indirect\n\tgithub.com/tree-sitter/tree-sitter-python v0.25.0\n)\nrequire github.com/tree-sitter/tree-sitter-rust v0.24.0\n"
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	versions, err := parseGomod(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 3 || versions["github.com/tree-sitter/tree-sitter-go"] != "v0.25.0" {
		t.Fatalf("parsed versions: %v", versions)
	}
}
