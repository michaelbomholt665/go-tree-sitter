package treesitter_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/michaelbomholt665/go-tree-sitter/internal/filelock"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	bundledconfig "github.com/michaelbomholt665/go-tree-sitter"
	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	"gopkg.in/yaml.v3"
)

func supportFixture(t *testing.T) (string, string, ts.LanguageSupportAssessment) {
	t.Helper()
	dir := t.TempDir()
	config := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(config, []byte("# preserve this user comment\n"+bundledconfig.YAML), 0640); err != nil {
		t.Fatal(err)
	}
	cfg, err := ts.LoadConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	var pin string
	for _, lang := range cfg.Languages {
		if lang.Name == "python" {
			pin = lang.Revision
		}
	}
	report := []byte(`{"coverage":"User-owned Python syntax suite; no complete conformance claim"}`)
	if err := os.WriteFile(filepath.Join(dir, "report.json"), report, 0600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(report)
	assessment := ts.LanguageSupportAssessment{
		AssessedRevision: pin, Claim: "verified", MinVersion: "3.12", MaxVersion: "3.12",
		Limitations: []string{"User-owned syntax coverage only; not full Python conformance."},
		Evidence:    []ts.LanguageSupportEvidence{{Kind: "syntax_fixtures", Source: "report.json", Description: "Version-specific syntax assertions and exclusions recorded in the external report.", SHA256: hex.EncodeToString(hash[:]), Result: "passed", TestedVersions: []string{"3.12"}}},
	}
	return config, filepath.Join(dir, "assessment.json"), assessment
}

func writeSupportJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func runSupport(t *testing.T, config, input string, args ...string) (string, error) {
	t.Helper()
	var out, stderr bytes.Buffer
	app := ts.NewApp(&out, &stderr, nil, nil, nil, nil, nil)
	cmd := app.NewRootCmd()
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(append([]string{"--config", config, "support"}, args...))
	err := cmd.ExecuteContext(context.Background())
	return out.String(), err
}

func supportConfigBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSupportPreviewApplyExternalFixturesAndShow(t *testing.T) {
	config, assessmentPath, assessment := supportFixture(t)
	writeSupportJSON(t, assessmentPath, assessment)
	before := supportConfigBytes(t, config)
	out, err := runSupport(t, config, "", "update", "-l", "python", "--assessment", assessmentPath, "--check-report", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Applied        bool              `json:"applied"`
		Changed        bool              `json:"changed"`
		ReportsChecked int               `json:"reports_checked"`
		Before         ts.GrammarSupport `json:"before"`
		After          ts.GrammarSupport `json:"after"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if result.Applied || !result.Changed || result.ReportsChecked != 1 || result.After.Rule != 3 || result.After.Claim != "verified" {
		t.Fatalf("bad preview: %s", out)
	}
	if !bytes.Equal(before, supportConfigBytes(t, config)) {
		t.Fatal("preview modified config")
	}
	out, err = runSupport(t, config, "", "update", "-l", "python", "--assessment", assessmentPath, "--expect-revision", assessment.AssessedRevision, "--check-report", "--apply", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil || !result.Applied {
		t.Fatalf("apply result: %s, %v", out, err)
	}
	after := supportConfigBytes(t, config)
	if !bytes.Contains(after, []byte("# preserve this user comment")) {
		t.Fatal("unrelated comment lost")
	}
	var oldDoc, newDoc map[string]any
	if err := yaml.Unmarshal(before, &oldDoc); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(after, &newDoc); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []map[string]any{oldDoc, newDoc} {
		for _, entry := range doc["languages"].([]any) {
			lang := entry.(map[string]any)
			if lang["name"] == "python" {
				delete(lang, "language_support")
			}
		}
	}
	if !reflect.DeepEqual(oldDoc, newDoc) {
		t.Fatal("update changed settings outside the selected assessment")
	}
	info, err := os.Stat(config)
	if err != nil {
		t.Fatal(err)
	}
	if os.PathSeparator != '\\' && info.Mode().Perm() != 0640 {
		t.Fatal("config permissions changed")
	}
	out, err = runSupport(t, config, "", "show", "--language", "python", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []ts.GrammarSupport
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Rule != 3 || rows[0].Assessment.Evidence[0].Source != filepath.Join(filepath.Dir(assessmentPath), "report.json") {
		t.Fatalf("external assessment not saved: %s", out)
	}
	out, err = runSupport(t, config, "", "update", "-l", "python", "--assessment", assessmentPath, "--apply", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil || result.Changed {
		t.Fatalf("repeat update not a no-op: %s, %v", out, err)
	}
	if !bytes.Equal(after, supportConfigBytes(t, config)) {
		t.Fatal("no-op update rewrote file")
	}
}

func TestSupportRejectsUnsafeOrInvalidUpdates(t *testing.T) {
	for _, name := range []string{"stale", "expected pin", "digest", "missing report", "remote report", "missing evidence", "unknown field", "extra object", "flags", "unknown grammar", "lock", "empty claim"} {
		t.Run(name, func(t *testing.T) {
			config, path, assessment := supportFixture(t)
			args := []string{"update", "--language", "python", "--assessment", path, "--apply", "--json"}
			switch name {
			case "stale":
				assessment.AssessedRevision = strings.Repeat("f", 40)
			case "expected pin":
				args = append(args, "--expect-revision", strings.Repeat("f", 40))
			case "digest":
				assessment.Evidence[0].SHA256 = strings.Repeat("0", 64)
				args = append(args, "--check-report")
			case "missing report":
				assessment.Evidence[0].Source = "absent.json"
				args = append(args, "--check-report")
			case "remote report":
				assessment.Evidence[0].Source = "https://example.invalid/report.json"
				args = append(args, "--check-report")
			case "missing evidence":
				assessment.Evidence = nil
			case "flags":
				args = append(args, "--dry-run")
			case "unknown grammar":
				args[2] = "unknown-grammar"
			case "lock":
				held, err := filelock.Lock(config + ".support.lock")
				if err != nil {
					t.Fatal(err)
				}
				defer held()
			case "empty claim":
				assessment.Claim = ""
			}
			writeSupportJSON(t, path, assessment)
			if name == "unknown field" || name == "extra object" {
				data := supportConfigBytes(t, path)
				if name == "unknown field" {
					data = append([]byte(`{"build_dir":"not-allowed",`), data[1:]...)
				} else {
					data = append(data, []byte(" {}")...)
				}
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := supportConfigBytes(t, config)
			out, err := runSupport(t, config, "", args...)
			if err == nil {
				t.Fatalf("invalid update accepted: %s", out)
			}
			var failure map[string]string
			if err := json.Unmarshal([]byte(out), &failure); err != nil || failure["error"] == "" {
				t.Fatalf("missing structured error: %s, %v", out, err)
			}
			if !bytes.Equal(before, supportConfigBytes(t, config)) {
				t.Fatal("rejected update changed config")
			}
		})
	}
}

func TestSupportDocumentationAndStdin(t *testing.T) {
	config, _, assessment := supportFixture(t)
	assessment.Evidence = []ts.LanguageSupportEvidence{{Kind: "documentation", Source: "https://example.invalid/grammar/" + assessment.AssessedRevision + "/README.md", Description: "Explicit version support statement for this revision."}}
	data, err := json.Marshal(assessment)
	if err != nil {
		t.Fatal(err)
	}
	out, err := runSupport(t, config, string(data), "update", "-l", "python", "--assessment", "-", "--apply", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		After ts.GrammarSupport `json:"after"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil || result.After.Rule != 2 || result.After.Claim != "verified" {
		t.Fatalf("documentation misclassified: %s, %v", out, err)
	}
}

func TestSupportRejectsSymlinkConfig(t *testing.T) {
	config, path, assessment := supportFixture(t)
	writeSupportJSON(t, path, assessment)
	link := filepath.Join(filepath.Dir(config), "link.yaml")
	if err := os.Symlink(config, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	before := supportConfigBytes(t, config)
	if _, err := runSupport(t, link, "", "update", "-l", "python", "--assessment", path, "--apply"); err == nil {
		t.Fatal("symlink config replaced")
	}
	if !bytes.Equal(before, supportConfigBytes(t, config)) {
		t.Fatal("symlink target changed")
	}
}
