package treesitter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/michaelbomholt665/go-tree-sitter/internal/filelock"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	"gopkg.in/yaml.v3"
)

func runConfig(t *testing.T, path string, args ...string) (string, error) {
	t.Helper()
	var out, stderr bytes.Buffer
	app := ts.NewApp(&out, &stderr, nil, nil, nil, nil, nil)
	err := app.Run(context.Background(), append([]string{"--config", path, "config"}, args...))
	return out.String(), err
}

func TestConfigShowGetValidate(t *testing.T) {
	path, _, _ := supportFixture(t)
	before := supportConfigBytes(t, path)
	out, err := runConfig(t, path, "show", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]any
	if err := json.Unmarshal([]byte(out), &values); err != nil {
		t.Fatal(err)
	}
	if _, ok := values["languages"]; ok {
		t.Fatal("config editor exposes language catalog")
	}
	if values["build_dir"] == nil || values["runtime"] == nil || values["output"] == nil || values["targets"] == nil {
		t.Fatalf("missing non-language settings: %s", out)
	}
	for _, key := range []string{"runtime.cleanup.after_move", "runtime.publication.scm", "build_flags", "targets", "abi_range.min", "prune"} {
		if out, err := runConfig(t, path, "get", key, "--json"); err != nil || !json.Valid([]byte(out)) {
			t.Fatalf("get %s: %s, %v", key, out, err)
		}
	}
	out, err = runConfig(t, path, "get", "prune", "--json")
	if err != nil || strings.TrimSpace(out) != "false" {
		t.Fatalf("omitted bool default lost: %s, %v", out, err)
	}
	if _, err := runConfig(t, path, "validate"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, supportConfigBytes(t, path)) {
		t.Fatal("inspection modified config")
	}
}

func TestConfigSetPreviewApplyAndPreserveCatalog(t *testing.T) {
	for _, test := range []struct{ key, value string }{
		{"runtime.cleanup.after_move", "keep"},
		{"runtime.publication.js", "true"},
		{"runtime.publication.scm", "null"},
		{"runtime.compiler.allow_fallback", "false"},
		{"generate_abi", "14"},
		{"runtime.tools.git", "./tools/my git"},
		{"build_flags", `["-O1","-fPIC"]`},
		{"runtime.sync_modules", `{"example.com/custom/module":["python"]}`},
		{"runtime.sync_modules", `{}`},
		{"runtime.compiler.candidates", `{"linux/amd64":[{"cc":"gcc","cxx":"g++"}]}`},
		{"runtime.queries.rewrites", `[{"language":"python","file":"highlights.scm","find":"old","replace":"new"}]`},
	} {
		t.Run(test.key+"="+test.value, func(t *testing.T) {
			path, _, _ := supportFixture(t)
			before := supportConfigBytes(t, path)
			oldCfg, err := ts.LoadConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			out, err := runConfig(t, path, "set", test.key, test.value, "--json")
			if err != nil {
				t.Fatalf("preview: %s, %v", out, err)
			}
			var result struct {
				Applied, Changed bool
				Before, After    any
			}
			if err := json.Unmarshal([]byte(out), &result); err != nil || result.Applied {
				t.Fatalf("invalid preview %s, %v", out, err)
			}
			if !bytes.Equal(before, supportConfigBytes(t, path)) {
				t.Fatal("preview saved data")
			}
			out, err = runConfig(t, path, "set", test.key, test.value, "--apply", "--json")
			if err != nil {
				t.Fatalf("apply: %s, %v", out, err)
			}
			if err := json.Unmarshal([]byte(out), &result); err != nil || !result.Applied {
				t.Fatalf("invalid result %s, %v", out, err)
			}
			after := supportConfigBytes(t, path)
			if !bytes.Contains(after, []byte("# preserve this user comment")) {
				t.Fatal("unrelated comment lost")
			}
			cfg, err := ts.LoadConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(oldCfg.Languages, cfg.Languages) {
				t.Fatal("language catalog changed")
			}
			var node yaml.Node
			if err := yaml.Unmarshal(after, &node); err != nil {
				t.Fatal(err)
			}
			content := node.Content[0].Content
			if content[len(content)-2].Value != "languages" {
				t.Fatal("languages is no longer the final block")
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if os.PathSeparator != '\\' && info.Mode().Perm() != 0640 {
				t.Fatal("permissions changed")
			}
		})
	}
}

func TestConfigSetRejectsRestrictedInvalidAndConflictingChanges(t *testing.T) {
	for _, test := range []struct {
		key, value string
		flags      []string
	}{
		{"languages", "[]", nil},
		{"languages.python.revision", strings.Repeat("f", 40), nil},
		{"runtime.tools.typo", "git", nil},
		{"runtime", "{}", nil},
		{"generate_abi", "14.5", nil},
		{"runtime.publication.js", `"false"`, nil},
		{"runtime.tools.git", `["git"]`, nil},
		{"runtime.tools.git", ` ["git"] `, nil},
		{"runtime.tools.git", ` {"command":"git"} `, nil},
		{"runtime.cleanup.after_move", "delete-everything", nil},
		{"runtime.node.install_args", `["install","--ignore-scripts=false"]`, nil},
		{"build_flags", "[-O2]", nil},
		{"runtime.compiler.candidates", `{"linux/amd64":[{"cc":"gcc","evil":"x"}]}`, nil},
		{"runtime.cleanup.after_move", "keep", []string{"--dry-run"}},
	} {
		t.Run(test.key+test.value, func(t *testing.T) {
			path, _, _ := supportFixture(t)
			before := supportConfigBytes(t, path)
			args := append([]string{"set", test.key, test.value, "--apply", "--json"}, test.flags...)
			out, err := runConfig(t, path, args...)
			if err == nil {
				t.Fatalf("invalid setting accepted: %s", out)
			}
			var failure map[string]string
			if err := json.Unmarshal([]byte(out), &failure); err != nil || failure["error"] == "" {
				t.Fatalf("invalid structured error: %s, %v", out, err)
			}
			if !bytes.Equal(before, supportConfigBytes(t, path)) {
				t.Fatal("invalid change saved")
			}
		})
	}
}

func TestConfigSetSharesSupportLockAndRejectsSymlinks(t *testing.T) {
	path, _, _ := supportFixture(t)
	before := supportConfigBytes(t, path)
	lock := path + ".support.lock"
	held, err := filelock.Lock(lock)
	if err != nil {
		t.Fatal(err)
	}
	defer held()
	if _, err := runConfig(t, path, "set", "runtime.cleanup.after_move", "keep", "--apply"); err == nil {
		t.Fatal("ignored shared update lock")
	}
	held()
	link := filepath.Join(filepath.Dir(path), "link.yaml")
	if err := os.Symlink(path, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := runConfig(t, link, "set", "runtime.cleanup.after_move", "keep", "--apply"); err == nil {
		t.Fatal("symlink configuration replaced")
	}
	if !bytes.Equal(before, supportConfigBytes(t, path)) {
		t.Fatal("unsafe update changed original config")
	}
}

func TestConfigSetCreatesOmittedSettingsBeforeLanguages(t *testing.T) {
	path, _, _ := supportFixture(t)
	var doc yaml.Node
	if err := yaml.Unmarshal(supportConfigBytes(t, path), &doc); err != nil {
		t.Fatal(err)
	}
	root := doc.Content[0]
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "runtime" {
			root.Content = append(root.Content[:i], root.Content[i+2:]...)
			break
		}
	}
	content, err := yaml.Marshal(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0640); err != nil {
		t.Fatal(err)
	}
	if out, err := runConfig(t, path, "set", "runtime.cleanup.after_move", "keep", "--apply"); err != nil {
		t.Fatalf("omitted settings: %s, %v", out, err)
	}
	if err := yaml.Unmarshal(supportConfigBytes(t, path), &doc); err != nil {
		t.Fatal(err)
	}
	root = doc.Content[0]
	if root.Content[len(root.Content)-2].Value != "languages" {
		t.Fatal("new setting appended after languages")
	}
}
