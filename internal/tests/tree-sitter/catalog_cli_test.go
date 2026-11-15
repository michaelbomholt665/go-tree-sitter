package treesitter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

func catalogApp(cfg *ts.Config, out *bytes.Buffer) *ts.App {
	return ts.NewApp(out, out, nil, func(string) (*ts.Config, error) { return cfg, nil }, nil, nil, nil)
}

func TestCatalogListSupport(t *testing.T) {
	pin, old := strings.Repeat("a", 40), strings.Repeat("b", 40)
	doc := &ts.LanguageSupportAssessment{AssessedRevision: pin, Claim: "verified", MinVersion: "4", Limitations: []string{"Static syntax only."}, Evidence: []ts.LanguageSupportEvidence{{Kind: "documentation"}}}
	fixture := &ts.LanguageSupportAssessment{AssessedRevision: pin, Claim: "verified", MinVersion: "8", MaxVersion: "9", Evidence: []ts.LanguageSupportEvidence{{Kind: "syntax_fixtures", Result: "passed", SHA256: strings.Repeat("c", 64), TestedVersions: []string{"8", "9"}}}}
	stale := &ts.LanguageSupportAssessment{AssessedRevision: old, Claim: "verified", MinVersion: "2", MaxVersion: "3", Evidence: []ts.LanguageSupportEvidence{{Kind: "documentation"}}}
	cfg := &ts.Config{Languages: []ts.Language{
		{Name: "unknown", Revision: pin},
		{Name: "documented", Revision: pin, LanguageSupport: doc},
		{Name: "tested", Revision: pin, LanguageSupport: fixture},
		{Name: "stale", Revision: pin, LanguageSupport: stale},
		{Name: "estimated", Revision: pin, LanguageSupport: &ts.LanguageSupportAssessment{AssessedRevision: pin, Claim: "unverified", MinVersion: "1"}},
		{Name: "inconclusive", Revision: pin, LanguageSupport: &ts.LanguageSupportAssessment{AssessedRevision: pin, Claim: "unverified"}},
	}}
	var out bytes.Buffer
	if err := catalogApp(cfg, &out).Run(context.Background(), []string{"list", "--json"}); err != nil {
		t.Fatal(err)
	}
	var rows []ts.GrammarSupport
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 || rows[0].Name != "documented" {
		t.Fatalf("unsorted/missing results: %+v", rows)
	}
	wantRule := map[string]int{"unknown": 1, "documented": 2, "tested": 3, "stale": 2, "estimated": 2, "inconclusive": 2}
	for _, row := range rows {
		if row.Rule != wantRule[row.Name] {
			t.Errorf("%s: rule %d", row.Name, row.Rule)
		}
		switch row.Name {
		case "stale":
			if !row.Stale || row.Claim != "unverified" || row.MinVersion != "" || row.MaxVersion != "" || row.Assessment.MinVersion != "2" {
				t.Errorf("stale support leaked: %+v", row)
			}
		case "documented":
			if row.Claim != "verified" || row.MinVersion != "4" || row.MaxVersion != "" {
				t.Errorf("documentation scope lost: %+v", row)
			}
		case "tested":
			if row.Claim != "verified" || len(row.Assessment.Evidence[0].TestedVersions) != 2 {
				t.Errorf("fixture scope lost: %+v", row)
			}
		}
	}
	out.Reset()
	if err := catalogApp(cfg, &out).Run(context.Background(), []string{"list", "-l", "unknown"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "unknown") || !strings.Contains(out.String(), "unverified") || strings.Contains(out.String(), "documented") {
		t.Fatal(out.String())
	}
	out.Reset()
	if err := catalogApp(cfg, &out).Run(context.Background(), []string{"list", "-l", "missing"}); err == nil {
		t.Fatal("unknown grammar accepted")
	}
}

func catalogWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func catalogRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestExportPublishedCatalog(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "repo", "tree-sitter-config.yaml")
	source := filepath.Join(root, "repo", "data", "grammars")
	target := filepath.Join(root, "consumer", "grammars")
	catalogWrite(t, filepath.Join(source, "python", "parser.so"), "binary")
	catalogWrite(t, filepath.Join(source, "python", "node-types.json"), "[]")
	catalogWrite(t, filepath.Join(source, "python", "grammar.js"), "module.exports = grammar({});")
	catalogWrite(t, filepath.Join(source, "python", "queries", "highlights.scm"), "(identifier) @variable")
	cfg := &ts.Config{Output: ts.Output{GrammarBase: "data/grammars"}, Languages: []ts.Language{{Name: "python"}, {Name: "not-yet-built"}}}
	run := func(args ...string) error {
		var out bytes.Buffer
		return catalogApp(cfg, &out).Run(context.Background(), append([]string{"--config", config, "export", "--to", target}, args...))
	}
	if err := run(); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"parser.so", "node-types.json", "grammar.js", "queries/highlights.scm"} {
		if catalogRead(t, filepath.Join(target, "python", file)) != catalogRead(t, filepath.Join(source, "python", file)) {
			t.Fatal(file)
		}
	}
	// There is no manifest: exporting raw Tree-sitter artifacts must still work.
	catalogWrite(t, filepath.Join(target, "python", "obsolete"), "old")
	catalogWrite(t, filepath.Join(target, "unrelated", "keep"), "keep")
	catalogWrite(t, filepath.Join(source, "python", "grammar.js"), "updated")
	if err := run(); err == nil {
		t.Fatal("existing destination overwritten without --force")
	}
	if got := catalogRead(t, filepath.Join(target, "python", "grammar.js")); got == "updated" {
		t.Fatal("refusal changed destination")
	}
	if err := run("--force", "--language", "python"); err != nil {
		t.Fatal(err)
	}
	if catalogRead(t, filepath.Join(target, "python", "grammar.js")) != "updated" {
		t.Fatal("not replaced")
	}
	if _, err := os.Stat(filepath.Join(target, "python", "obsolete")); !os.IsNotExist(err) {
		t.Fatal("obsolete file retained")
	}
	if catalogRead(t, filepath.Join(target, "unrelated", "keep")) != "keep" {
		t.Fatal("unrelated directory changed")
	}
	if err := run("--language", "not-yet-built"); err == nil {
		t.Fatal("explicit unpublished grammar accepted")
	}
	if err := run("--language", "unknown"); err == nil {
		t.Fatal("unknown grammar accepted")
	}
}

func TestExportRejectsOverlapAndSymlinks(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "catalog")
	catalogWrite(t, filepath.Join(source, "python", "parser.so"), "source")
	cfg := &ts.Config{Output: ts.Output{GrammarBase: source}, Languages: []ts.Language{{Name: "python"}}}
	run := func(ctx context.Context, target string) error {
		var out bytes.Buffer
		return catalogApp(cfg, &out).Run(ctx, []string{"export", "--to", target, "--force"})
	}
	for _, target := range []string{source, root, filepath.Join(source, "nested", "new")} {
		if err := run(context.Background(), target); err == nil {
			t.Errorf("overlap accepted: %s", target)
		}
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(source, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := run(context.Background(), filepath.Join(alias, "new")); err == nil {
		t.Fatal("symlink overlap accepted")
	}
	target := filepath.Join(root, "export")
	catalogWrite(t, filepath.Join(target, "python", "parser.so"), "existing")
	if err := os.Symlink(filepath.Join(source, "python", "parser.so"), filepath.Join(source, "python", "link")); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), target); err == nil {
		t.Fatal("source symlink accepted")
	}
	if catalogRead(t, filepath.Join(target, "python", "parser.so")) != "existing" {
		t.Fatal("failed staging changed destination")
	}
	if err := os.Remove(filepath.Join(source, "python", "link")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx, filepath.Join(root, "cancelled")); err == nil {
		t.Fatal("cancelled export succeeded")
	}
	destinationLink := filepath.Join(root, "destination-link")
	if err := os.MkdirAll(destinationLink, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(source, "python"), filepath.Join(destinationLink, "python")); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), destinationLink); err == nil {
		t.Fatal("destination grammar symlink accepted")
	}
}

func TestSyncConfigSubcommand(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "tree-sitter-config.yaml")
	gomod := filepath.Join(root, "go.mod")
	pin := strings.Repeat("a", 40)
	original := "languages:\n  - name: python\n    version: v0.0.1\n    revision: " + pin + "\n    language_support:\n      assessed_revision: " + pin + "\n      claim: unverified\n"
	catalogWrite(t, config, original)
	catalogWrite(t, gomod, "module fixture\nrequire github.com/tree-sitter/tree-sitter-python v1.2.3\n")
	var out bytes.Buffer
	app := catalogApp(nil, &out) // sync-config must accept raw YAML without requiring a build configuration.
	if err := app.Run(context.Background(), []string{"--config", config, "sync-config", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if catalogRead(t, config) != original || !strings.Contains(out.String(), "v1.2.3") {
		t.Fatal("dry run failed")
	}
	if err := app.Run(context.Background(), []string{"--config", config, "sync-config", "--gomod", gomod}); err != nil {
		t.Fatal(err)
	}
	data := catalogRead(t, config)
	if !strings.Contains(data, "v1.2.3") || strings.Count(data, pin) != 2 {
		t.Fatal("version/pins not preserved: " + data)
	}
}

func TestRegenManifestsSubcommandRefusesUnsafeOperation(t *testing.T) {
	var out bytes.Buffer
	err := catalogApp(nil, &out).Run(context.Background(), []string{"regen-manifests"})
	if err == nil || !strings.Contains(err.Error(), "unsafe") || !strings.Contains(err.Error(), "move --manifest=true") {
		t.Fatalf("unexpected result: %v", err)
	}
}

func TestMoveRawGrammarJS(t *testing.T) {
	var out bytes.Buffer
	mover := &stubMover{}
	cfg := &ts.Config{Output: ts.Output{GenerateManifest: true}}
	app := ts.NewApp(&out, &out, nil, func(string) (*ts.Config, error) { return cfg, nil }, nil, nil, mover)
	if err := app.Run(context.Background(), []string{"move", "--js", "--manifest=false"}); err != nil {
		t.Fatal(err)
	}
	if !mover.req.CopyJS || mover.req.GenerateManifest || !mover.req.Clean {
		t.Fatalf("unexpected move request: %+v", mover.req)
	}
}
