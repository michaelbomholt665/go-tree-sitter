//go:build grammars

package grammars

import (
	"testing"
)

func TestGetLanguage(t *testing.T) {
	languages := []string{
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
			lang := GetLanguage(name)
			if lang == nil {
				t.Fatalf("expected non-nil Language for %q", name)
			}
		})
	}

	// Unknown language returns nil
	if lang := GetLanguage("unknown-language"); lang != nil {
		t.Fatalf("expected nil Language for unknown language, got %v", lang)
	}
}
