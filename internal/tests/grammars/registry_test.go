//go:build grammars

package grammars_test

import (
	grammars "github.com/michaelbomholt665/go-tree-sitter/internal/grammars"
	"path/filepath"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"testing"
)

func TestGetLanguage(t *testing.T) {
	languages := []string{
		"powershell", "fsharp", "fsharp-signature", "fsharp_signature", "kotlin", "swift",
		"dart", "gleam", "hcl", "terraform", "terraform-vars", "zsh", "prisma", "svelte",
		"objc", "json5", "mdx",
		"bash",
		"c",
		"c-sharp", "c_sharp", "csharp",
		"cpp", "c++",
		"css",
		"cypher",
		"go",
		"haskell",
		"html",
		"java",
		"javascript",
		"json",
		"julia",
		"lua",
		"make", "makefile",
		"ocaml",
		"ocaml-interface", "ocaml_interface",
		"php",
		"php_only", "php-only",
		"proto",
		"python",
		"regex",
		"ruby",
		"rust",
		"scala",
		"toml",
		"tsx",
		"typescript",
		"yaml",
		"zig",
	}

	for _, name := range languages {
		t.Run(name, func(t *testing.T) {
			lang := grammars.GetLanguage(name)
			if lang == nil {
				t.Fatalf("expected non-nil Language for %q", name)
			}
		})
	}

	// Unknown language returns nil
	if lang := grammars.GetLanguage("unknown-language"); lang != nil {
		t.Fatalf("expected nil Language for unknown language, got %v", lang)
	}
}

// Smoke samples check integration, not source-language version conformance.
func TestProviderGrammarSamples(t *testing.T) {
	cfg, err := ts.LoadConfig(filepath.Join("..", "..", "..", "tree-sitter-config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"powershell", "fsharp", "fsharp-signature", "kotlin", "swift", "dart", "gleam",
		"hcl", "zsh", "prisma", "svelte", "objc", "json5", "mdx",
	} {
		t.Run(name, func(t *testing.T) {
			lang := grammars.GetLanguage(name)
			if lang == nil {
				t.Fatal("missing Go binding")
			}
			langs, err := cfg.ResolveLanguages(name)
			if err != nil {
				t.Fatal(err)
			}
			parser := sitter.NewParser()
			defer parser.Close()
			if err := parser.SetLanguage(lang); err != nil {
				t.Fatal(err)
			}
			tree := parser.Parse([]byte(langs[0].Sample), nil)
			if tree == nil {
				t.Fatal("sample returned no parse tree")
			}
			defer tree.Close()
			if tree.RootNode().HasError() {
				t.Fatalf("sample parse error: %s", tree.RootNode().ToSexp())
			}
		})
	}
}
