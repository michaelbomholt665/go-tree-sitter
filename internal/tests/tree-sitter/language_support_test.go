package treesitter_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

func TestLanguageSupportConfigPinSurvivesDependencyUpdates(t *testing.T) {
	pin := strings.Repeat("a", 40)
	newPin := strings.Repeat("b", 40)
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := fmt.Sprintf(`version: "2.0"
tree_sitter_cli_version: "0.26.8"
languages:
  - name: example
    version: v1.0.0
    repository: https://github.com/example/grammar.git
    revision: %s
    sample: "example"
    language_support:
      assessed_revision: %s
      claim: verified
      min_version: "2020"
      max_version: "2024"
      limitations: ["Selected syntax only"]
      evidence:
        - kind: documentation
          source: https://github.com/example/grammar/blob/%s/README.md
          description: "Explicit upstream claim (synthetic test data)."
`, newPin, pin, pin)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ts.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	langs, err := cfg.ResolveLanguages("example")
	if err != nil {
		t.Fatal(err)
	}
	lang := langs[0]
	if lang.Revision != newPin || lang.LanguageSupport.AssessedRevision != pin || lang.LanguageSupport.Claim != "verified" {
		t.Fatalf("dependency update moved historical assessment: %+v", lang)
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(content, "assessed_revision: "+pin, "assessed_revision: short")), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ts.LoadConfig(path); err == nil || !strings.Contains(err.Error(), "languages[0].language_support") {
		t.Fatalf("expected indexed support-pin validation error, got %v", err)
	}
}

func TestLanguageSupportRejectsUnknownFields(t *testing.T) {
	for _, field := range []string{"limitations", "min_version", "description"} {
		t.Run(field, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			content := `version: "2.0"
tree_sitter_cli_version: "0.26.8"
languages:
  - name: example
    version: v1.0.0
    repository: https://github.com/example/grammar.git
    revision: "0123456789012345678901234567890123456789"
    sample: "example"
    language_support:
      assessed_revision: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
      claim: verified
      min_version: "2020"
      limitations: ["Feature X is excluded"]
      evidence:
        - kind: documentation
          source: https://example.com/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/README.md
          description: "Explicit upstream syntax claim"
`
			typo := field + "_typo"
			content = strings.Replace(content, field+":", typo+":", 1)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := ts.LoadConfig(path); err == nil || !strings.Contains(err.Error(), typo) {
				t.Fatalf("expected rejection of unknown field %q, got %v", typo, err)
			}
		})
	}
}

func TestLanguageSupportAbsentDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`version: "2.0"
tree_sitter_cli_version: "0.26.8"
languages:
  - name: example
    version: v1.0.0
    repository: https://github.com/example/grammar.git
    revision: "0123456789012345678901234567890123456789"
    sample: "example"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ts.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Languages[0].LanguageSupport != nil {
		t.Fatal("loading an unassessed language must not invent a support claim")
	}
}
