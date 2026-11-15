// internal/tests/build/manifest_naming_test.go
package build_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

func TestManifestPublishedNames(t *testing.T) {
	t.Parallel()
	for _, language := range []ts.Language{
		{Name: "csharp", Grammar: "c_sharp", Constructor: "tree_sitter_c_sharp"},
		{Name: "go.mod", Grammar: "gomod", Constructor: "tree_sitter_gomod"},
		{Name: "go.sum", Grammar: "gosum", Constructor: "tree_sitter_gosum"},
		{Name: "markdown-inline", Grammar: "markdown_inline", Constructor: "tree_sitter_markdown_inline"},
		{Name: "ocaml-interface", Grammar: "ocaml_interface", Constructor: "tree_sitter_ocaml_interface"},
		{Name: "php_only", Grammar: "php_only", Constructor: "tree_sitter_php_only"},
	} {
		t.Run(language.Name, func(t *testing.T) {
			t.Parallel()
			manifestCheckPublishedName(t, language)
		})
	}
}

// manifestCheckPublishedName checks the complete publication path with distinct parser identifiers.
func manifestCheckPublishedName(t *testing.T, language ts.Language) {
	t.Helper()
	language.Version = "v1.0.0"
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	sourceDir := filepath.Join(buildDir, language.Name)
	if err := os.MkdirAll(filepath.Join(sourceDir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	binaryDir := filepath.Join(sourceDir, "bin")
	if err := os.MkdirAll(binaryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nodeTypes := filepath.Join(sourceDir, "src", "node-types.json")
	if err := os.WriteFile(nodeTypes, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	filename := language.Name + "-v1.0.0-linux-amd64.so"
	binary := filepath.Join(binaryDir, filename)
	if err := os.WriteFile(binary, []byte("synthetic binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeBinaryProvenance(t, binary, nodeTypes)
	cfg := &ts.Config{
		BuildDir: buildDir, Languages: []ts.Language{language},
		Output: ts.Output{GrammarBase: filepath.Join(dir, "catalog"), GenerateManifest: true},
	}
	mover := tsbuild.NewMoverWithValidator(tsbuild.NewCleaner(), acceptingValidator{}, ioDiscard{}, ioDiscard{})
	request := ts.MoveRequest{Language: language.Name, Mode: ts.MoveModeBoth, GenerateManifest: true}
	prepareArtifactConfig(cfg)
	if err := mover.Move(context.Background(), cfg, request); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(cfg.Output.GrammarBase, language.Name)
	payload, err := os.ReadFile(filepath.Join(output, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest tsbuild.ManifestData
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Grammar != language.Name {
		t.Fatalf("manifest grammar = %q, want directory name %q", manifest.Grammar, language.Name)
	}
	if len(manifest.Binaries) != 1 || manifest.Binaries[0].Filename != filename {
		t.Fatalf("unexpected published binaries: %+v", manifest.Binaries)
	}
	if _, err := os.Stat(filepath.Join(output, filename)); err != nil {
		t.Fatal(err)
	}
}
