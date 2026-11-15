package syncconfig_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func runSyncConfig(t *testing.T, config, modules string) []byte {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	configPath, modulePath := filepath.Join(dir, "config.yaml"), filepath.Join(dir, "go.mod")
	for path, content := range map[string]string{configPath: config, modulePath: modules} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "run", "./cmd/sync-config", "--config", configPath, "--gomod", modulePath)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sync-config: %v\n%s", err, output)
	}
	result, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestLegacyGoBindingsDoNotOverrideNativeGrammarVersions(t *testing.T) {
	modules := map[string][]string{
		"github.com/tree-sitter-grammars/tree-sitter-make":   {"make"},
		"github.com/tree-sitter-grammars/tree-sitter-zig":    {"zig"},
		"github.com/tree-sitter/tree-sitter-cpp":             {"cpp"},
		"github.com/tree-sitter/tree-sitter-ruby":            {"ruby"},
		"github.com/tree-sitter/tree-sitter-scala":           {"scala"},
		"github.com/airbus-cert/tree-sitter-powershell":      {"powershell"},
		"github.com/ionide/tree-sitter-fsharp":               {"fsharp", "fsharp-signature"},
		"github.com/fwcd/tree-sitter-kotlin":                 {"kotlin"},
		"github.com/tree-sitter/tree-sitter-swift":           {"swift"},
		"github.com/UserNobody14/tree-sitter-dart":           {"dart"},
		"github.com/gleam-lang/tree-sitter-gleam":            {"gleam"},
		"github.com/tree-sitter-grammars/tree-sitter-hcl":    {"hcl"},
		"github.com/tree-sitter/tree-sitter-zsh":             {"zsh"},
		"github.com/victorhqc/tree-sitter-prisma":            {"prisma"},
		"github.com/tree-sitter-grammars/tree-sitter-svelte": {"svelte"},
		"github.com/tree-sitter-grammars/tree-sitter-objc":   {"objc"},
		"github.com/joakker/tree-sitter-json5.git":           {"json5"},
		"www.github.com/srazzak/tree-sitter-mdx":             {"mdx"},
	}
	config, gomod := "languages:\n", "module fixture\n\ngo 1.26\n\nrequire (\n"
	for module, names := range modules {
		gomod += fmt.Sprintf("\t%s v1.9.0\n", module)
		for _, name := range names {
			config += fmt.Sprintf("  - name: %s\n    version: v1.0.0\n", name)
		}
	}
	gomod += ")\n"
	var result struct {
		Languages []struct{ Name, Version string }
	}
	if err := yaml.Unmarshal(runSyncConfig(t, config, gomod), &result); err != nil {
		t.Fatal(err)
	}
	for _, language := range result.Languages {
		want := "v1.9.0"
		if language.Name == "make" || language.Name == "zig" {
			want = "v1.0.0"
		}
		if language.Version != want {
			t.Errorf("%s version = %s, want %s", language.Name, language.Version, want)
		}
	}
}

func TestSyncConfigPreservesLanguageSupportAssessment(t *testing.T) {
	config := `languages:
  - name: python
    version: v1.0.0
    revision: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    language_support:
      assessed_revision: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
      claim: unverified
      min_version: "3.10"
      limitations: ["Not yet verified"]
`
	modules := "module fixture\n\ngo 1.26\n\nrequire github.com/tree-sitter/tree-sitter-python v1.9.0\n"
	var before, after struct{ Languages []map[string]any }
	if err := yaml.Unmarshal([]byte(config), &before); err != nil {
		t.Fatal(err)
	}
	result := runSyncConfig(t, config, modules)
	if err := yaml.Unmarshal(result, &after); err != nil {
		t.Fatal(err)
	}
	before.Languages[0]["version"] = "v1.9.0"
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("synchronization changed more than version:\n%s", result)
	}
	if !strings.Contains(string(result), "assessed_revision: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") {
		t.Fatal("historical assessment pin was not preserved")
	}
}
