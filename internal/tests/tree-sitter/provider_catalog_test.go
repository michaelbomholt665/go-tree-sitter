package treesitter_test

import (
	"strings"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

// Keep this inventory independent of the optional static Go registry: native-only
// upstreams are valid build targets too. See docs/reference/provider-grammar-coverage.md.
var providerAdditions = []string{
	"powershell", "fsharp", "fsharp-signature", "kotlin", "groovy", "lean",
	"swift", "dart", "gleam", "typst", "clojure", "cue", "hcl", "nix",
	"zsh", "fish", "prisma", "vue", "svelte", "objc", "json5", "less", "mdx",
}

func TestProviderCatalogGrammarConfiguration(t *testing.T) {
	cfg, err := ts.LoadConfig("../../../tree-sitter-config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range providerAdditions {
		t.Run(name, func(t *testing.T) {
			langs, err := cfg.ResolveLanguages(name)
			if err != nil || len(langs) != 1 {
				t.Fatalf("resolve %s: %v (%d results)", name, err, len(langs))
			}
			lang := langs[0]
			if len(lang.Revision) != 40 || lang.Constructor != "tree_sitter_"+lang.Grammar || strings.TrimSpace(lang.Sample) == "" {
				t.Fatalf("missing immutable source, constructor, or smoke sample: %+v", lang)
			}
			assessment := lang.LanguageSupport
			if assessment == nil || assessment.AssessedRevision != lang.Revision {
				t.Fatalf("assessment must describe the selected revision: %+v", assessment)
			}
			if len(assessment.Limitations) == 0 || (assessment.Claim == "verified" && len(assessment.Evidence) == 0) {
				t.Fatal("record assessment limitations and evidence for verified claims")
			}
			for _, evidence := range assessment.Evidence {
				if evidence.Kind != "documentation" || !strings.Contains(evidence.Source, "/"+lang.Revision+"/") {
					t.Fatalf("expected exact-revision documentation, got %+v", evidence)
				}
			}
		})
	}
	// Preserve the original catalog, including grammars absent from the providers.
	for _, name := range []string{
		"bash", "blade", "c", "csharp", "cmake", "cpp", "css", "cypher", "dockerfile", "elixir",
		"go", "go.mod", "go.sum", "haskell", "html", "java", "javascript", "json", "julia", "lua",
		"make", "markdown", "markdown-inline", "ocaml", "ocaml-interface", "php", "php_only", "proto",
		"python", "regex", "ruby", "rust", "sass", "scala", "scss", "solidity", "sql", "toml", "tsx",
		"typescript", "yaml", "zig",
	} {
		if _, err := cfg.ResolveLanguages(name); err != nil {
			t.Errorf("original grammar %s was removed: %v", name, err)
		}
	}
	for _, name := range []string{"deno", "terraform-vars", "sh", "jsonc"} {
		if _, err := cfg.ResolveLanguages(name); err == nil {
			t.Errorf("routing alias %s must not duplicate a grammar", name)
		}
	}
}
