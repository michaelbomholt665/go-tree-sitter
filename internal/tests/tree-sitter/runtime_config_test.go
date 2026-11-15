package treesitter_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	bundledconfig "github.com/michaelbomholt665/go-tree-sitter"
	"github.com/michaelbomholt665/go-tree-sitter/internal/configsync"
	"github.com/michaelbomholt665/go-tree-sitter/internal/tests/testutil"
	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	"gopkg.in/yaml.v3"
)

func runtimeBool(v bool) *bool { return &v }

func TestRuntimePolicyDefaultsAndExplicitSettings(t *testing.T) {
	defaults := (*ts.Config)(nil).EffectiveRuntime()
	if defaults.WASM.Backend != "auto" || defaults.Cleanup.AfterMove != "remove" || !*defaults.Generation.ABIFallback {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
	cfg := &ts.Config{Runtime: ts.RuntimeConfig{
		Generation:       ts.GenerationPolicy{ABIFallback: runtimeBool(false)},
		Compiler:         ts.CompilerPolicy{AllowFallback: runtimeBool(false), Candidates: map[string][]ts.CompilerPair{}},
		Node:             ts.NodePolicy{Install: "never", RetryInstall: runtimeBool(false), CIArgs: []string{}, InstallArgs: []string{}},
		Cleanup:          ts.CleanupPolicy{AfterMove: "keep", Prune: ts.PruneRules{Directories: []string{}, Files: []string{}}},
		Queries:          ts.QueryPolicy{Rewrites: []ts.QueryRewrite{}},
		SyncModules:      map[string][]string{},
		SupportedTargets: map[string][]string{"linux": {"arm64"}},
	}}
	r := cfg.EffectiveRuntime()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if *r.Generation.ABIFallback || *r.Compiler.AllowFallback || *r.Node.RetryInstall || len(r.Queries.Rewrites) != 0 || len(r.SyncModules) != 0 || len(r.Cleanup.Prune.Files) != 0 {
		t.Fatalf("explicit false or empty settings were replaced: %+v", r)
	}
	// Disabling a target does not require deleting unused compiler candidates.
	cfg.Runtime.Compiler.Candidates = nil
	if err := cfg.EffectiveRuntime().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimePolicyRejectsUnsafeOrInvalidSettings(t *testing.T) {
	cases := map[string]func(*ts.RuntimeConfig){
		"backend":         func(r *ts.RuntimeConfig) { r.WASM.Backend = "shell" },
		"cleanup":         func(r *ts.RuntimeConfig) { r.Cleanup.AfterMove = "delete-everything" },
		"target":          func(r *ts.RuntimeConfig) { r.SupportedTargets = map[string][]string{"linux": {"mips"}} },
		"wizard":          func(r *ts.RuntimeConfig) { r.Wizard.Platforms = []string{"invalid/amd64"} },
		"artifact":        func(r *ts.RuntimeConfig) { r.Wizard.Artifacts = []string{"invalid"} },
		"prune traversal": func(r *ts.RuntimeConfig) { r.Cleanup.Prune.Directories = []string{"../src"} },
		"prune glob":      func(r *ts.RuntimeConfig) { r.Cleanup.Prune.Files = []string{"["} },
		"query traversal": func(r *ts.RuntimeConfig) { r.Queries.Paths = map[string][]string{"go": {"../query.scm"}} },
		"rewrite traversal": func(r *ts.RuntimeConfig) {
			r.Queries.Rewrites = []ts.QueryRewrite{{Language: "go", File: "../x.scm", Find: "x"}}
		},
		"lifecycle scripts": func(r *ts.RuntimeConfig) { r.Node.CIArgs = []string{"ci"} },
		"override safety": func(r *ts.RuntimeConfig) {
			r.Node.CIArgs = []string{"ci", "--ignore-scripts", "--ignore-scripts=false"}
		},
		"tool": func(r *ts.RuntimeConfig) { r.Tools.Git = "git\nother" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r := (*ts.Config)(nil).EffectiveRuntime()
			change(&r)
			if err := r.Validate(); err == nil {
				t.Fatal("invalid policy accepted")
			}
		})
	}
}

func TestBundledCatalogHasAllSettingsBeforeLanguages(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(bundledconfig.YAML), &doc); err != nil {
		t.Fatal(err)
	}
	root := doc.Content[0]
	if root.Content[len(root.Content)-2].Value != "languages" {
		t.Fatal("languages must be the final top-level setting")
	}
	var cfg ts.Config
	if err := yaml.Unmarshal([]byte(bundledconfig.YAML), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Languages) != 65 {
		t.Fatalf("catalog has %d languages", len(cfg.Languages))
	}
	actual, err := yaml.Marshal(cfg.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := yaml.Marshal((*ts.Config)(nil).EffectiveRuntime())
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(expected) {
		t.Fatal("documented runtime defaults differ from implementation")
	}
}

func TestRuntimePublicationDefaultsAndFlagPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		clean     bool
		action    string
		artifacts bool
	}{
		{"configured", nil, true, "prune", true},
		{"explicit false", []string{"--clean=false", "--manifest=false", "--compact=false", "--source=false", "--js=false", "--wasm=false", "--force=false", "--scm=false"}, false, "keep", false},
		{"explicit remove", []string{"--clean"}, true, "remove", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &ts.Config{Runtime: ts.RuntimeConfig{
				Cleanup:     ts.CleanupPolicy{AfterMove: "prune"},
				Publication: ts.PublicationDefaults{Manifest: runtimeBool(true), SCM: runtimeBool(true), Compact: true, Source: true, JS: true, WASM: true, Force: true},
			}, Output: ts.Output{DefaultMoveMode: "--both"}}
			mover := &stubMover{}
			app := newTestApp(cfg, &stubBuilder{}, &stubCompiler{}, mover, testutil.StaticLookup{})
			if err := app.Run(context.Background(), append([]string{"move"}, tc.args...)); err != nil {
				t.Fatal(err)
			}
			r := mover.req
			if r.Clean != tc.clean || r.CleanupAction != tc.action || r.GenerateManifest != tc.artifacts || r.Compact != tc.artifacts || r.CopySource != tc.artifacts || r.CopyJS != tc.artifacts || r.IncludeWasm != tc.artifacts || r.Force != tc.artifacts || r.CopySCM != tc.artifacts {
				t.Fatalf("unexpected request: %+v", r)
			}
		})
	}
}

func TestConfiguredToolsUsedWithoutShellInterpolation(t *testing.T) {
	runner := &testutil.RecordingRunner{OnOutput: func(context.Context, ts.Command) (string, error) { return "version", nil }}
	wrapped := ts.WithToolPaths(runner, ts.ToolPaths{Git: "/tools/custom git", NPM: "npm-custom", TreeSitter: "ts-custom"})
	if err := wrapped.Run(context.Background(), ts.Command{Name: "git", Args: []string{"clone", "a;b"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := wrapped.Output(context.Background(), ts.Command{Name: "tree-sitter"}); err != nil {
		t.Fatal(err)
	}
	if err := wrapped.Run(context.Background(), ts.Command{Name: "npm"}); err != nil {
		t.Fatal(err)
	}
	if runner.Runs[0].Name != "/tools/custom git" || runner.Runs[0].Args[1] != "a;b" || runner.Runs[1].Name != "npm-custom" {
		t.Fatalf("unexpected commands: %+v", runner.Runs)
	}
}

func TestSyncConfigUsesConfiguredModuleMapping(t *testing.T) {
	for _, mapping := range []string{"{example.com/custom: [custom]}", "{}"} {
		t.Run(mapping, func(t *testing.T) {
			dir := t.TempDir()
			config := filepath.Join(dir, "config.yaml")
			gomod := filepath.Join(dir, "go.mod")
			text := "runtime:\n  sync_modules: " + mapping + "\nlanguages:\n  - name: custom\n    version: v1.0.0\n    revision: unchanged\n"
			if err := os.WriteFile(config, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(gomod, []byte("module example.com/app\nrequire (\n\texample.com/custom v2.0.0\n)\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := configsync.Run(config, gomod, false, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}
			want := "v2.0.0"
			if mapping == "{}" {
				want = "v1.0.0"
			}
			if !bytes.Contains(data, []byte(want)) || !bytes.Contains(data, []byte("revision: unchanged")) {
				t.Fatalf("unexpected config: %s", data)
			}
		})
	}
}
