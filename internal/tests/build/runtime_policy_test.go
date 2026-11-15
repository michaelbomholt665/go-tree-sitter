package build_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/michaelbomholt665/go-tree-sitter/internal/tests/testutil"
	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

func policyWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

func policyWorkspace(t *testing.T) (*ts.Config, string, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &ts.Config{
		BuildDir: filepath.Join(dir, "workspace"), OSTarget: "linux",
		Targets:   map[string]ts.BuildTarget{"linux": {OS: "linux", Arch: "amd64"}},
		Languages: []ts.Language{{Name: "python", Grammar: "python", Constructor: "tree_sitter_python", Version: "v0.25.0"}},
		Output:    ts.Output{GrammarBase: filepath.Join(dir, "published"), GrammarBuild: "{build_dir}/sources/{lang}/{arch}", GrammarCompile: "{build_dir}/compiled/{lang}/{os}/{arch}", GenerateManifest: true},
	}
	source := filepath.Join(cfg.BuildDir, "sources", "python", "amd64")
	binary := filepath.Join(cfg.BuildDir, "compiled", "python", "linux", "amd64", "python-v0.25.0-linux-amd64.so")
	policyWrite(t, filepath.Join(source, "src", "node-types.json"), "[]")
	policyWrite(t, filepath.Join(source, "src", "parser.c"), "parser")
	policyWrite(t, filepath.Join(source, "drop.tmp"), "temporary")
	policyWrite(t, filepath.Join(source, "discard", "cache"), "temporary")
	policyWrite(t, filepath.Join(source, "editor", "queries", "custom.scm"), "(old)")
	policyWrite(t, binary, "synthetic binary")
	writeBinaryProvenance(t, binary, filepath.Join(source, "src", "node-types.json"))
	cfg.Runtime.Queries.Paths = map[string][]string{"python": {"editor/queries/custom.scm"}}
	cfg.Runtime.Queries.Rewrites = []ts.QueryRewrite{{Language: "python", File: "custom.scm", Find: "old", Replace: "new"}}
	cfg.Runtime.Cleanup.Prune.Directories = []string{"discard"}
	cfg.Runtime.Cleanup.Prune.Files = []string{"*.tmp"}
	prepareArtifactConfig(cfg)
	return cfg, source, binary
}

type policyRejectingValidator struct{}

func (policyRejectingValidator) Validate(context.Context, []tsbuild.ValidationItem) (map[string]uint32, error) {
	return nil, errors.New("deliberate validation failure")
}

func TestMoveUsesWorkspaceCleanupAndQueryPolicies(t *testing.T) {
	for _, action := range []string{"keep", "prune", "remove"} {
		t.Run(action, func(t *testing.T) {
			cfg, source, binary := policyWorkspace(t)
			cfg.Runtime.Cleanup.AfterMove = action
			mover := tsbuild.NewMoverWithValidator(tsbuild.NewCleaner(), acceptingValidator{}, ioDiscard{}, ioDiscard{})
			prepareBuildConfig(cfg)
			if err := mover.Move(context.Background(), cfg, ts.MoveRequest{Clean: true, Mode: ts.MoveModeBoth, CopySCM: true}); err != nil {
				t.Fatal(err)
			}
			query, err := os.ReadFile(filepath.Join(cfg.Output.GrammarBase, "python", "queries", "custom.scm"))
			if err != nil || string(query) != "(new)" {
				t.Fatalf("query policy not applied: %q, %v", query, err)
			}
			for _, path := range []string{source, filepath.Dir(binary)} {
				_, err := os.Stat(path)
				if action == "remove" {
					if !os.IsNotExist(err) {
						t.Fatalf("workspace root retained: %s, %v", path, err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			}
			_, err = os.Stat(filepath.Join(source, "drop.tmp"))
			if action == "keep" && err != nil {
				t.Fatal(err)
			}
			if action != "keep" && !os.IsNotExist(err) {
				t.Fatalf("temporary file retained: %v", err)
			}
			if action == "prune" {
				if _, err := os.Stat(filepath.Join(source, "src", "parser.c")); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(binary); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(filepath.Join(source, "discard")); !os.IsNotExist(err) {
					t.Fatalf("discard directory retained: %v", err)
				}
			}
		})
	}
}

func TestFailedMovePreservesWorkspaceAndPublishedCatalog(t *testing.T) {
	cfg, source, binary := policyWorkspace(t)
	marker := filepath.Join(cfg.Output.GrammarBase, "existing.txt")
	policyWrite(t, marker, "preserve")
	mover := tsbuild.NewMoverWithValidator(tsbuild.NewCleaner(), policyRejectingValidator{}, ioDiscard{}, ioDiscard{})
	prepareBuildConfig(cfg)
	if err := mover.Move(context.Background(), cfg, ts.MoveRequest{Clean: true, Mode: ts.MoveModeBoth}); err == nil {
		t.Fatal("expected validation failure")
	}
	for _, path := range []string{source, binary, marker} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("failed publication removed %s: %v", path, err)
		}
	}
}

func TestUnsafeCleanupRejectedBeforePublication(t *testing.T) {
	for _, mode := range []string{"invalid action", "external source", "shared source", "symlink escape"} {
		t.Run(mode, func(t *testing.T) {
			cfg, _, _ := policyWorkspace(t)
			marker := filepath.Join(cfg.Output.GrammarBase, "existing.txt")
			policyWrite(t, marker, "preserve")
			action := "remove"
			switch mode {
			case "invalid action":
				action = "invalid"
			case "external source":
				cfg.Output.GrammarBuild = filepath.Join(t.TempDir(), "{lang}")
			case "shared source":
				cfg.Output.GrammarBuild = "{build_dir}/shared"
			case "symlink escape":
				link := filepath.Join(cfg.BuildDir, "escape")
				if err := os.Symlink(t.TempDir(), link); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				cfg.Output.GrammarBuild = "{build_dir}/escape/{lang}"
			}
			mover := tsbuild.NewMoverWithValidator(tsbuild.NewCleaner(), acceptingValidator{}, ioDiscard{}, ioDiscard{})
			prepareBuildConfig(cfg)
			if err := mover.Move(context.Background(), cfg, ts.MoveRequest{Clean: true, CleanupAction: action, Mode: ts.MoveModeBoth}); err == nil {
				t.Fatal("unsafe cleanup accepted")
			}
			if data, err := os.ReadFile(marker); err != nil || string(data) != "preserve" {
				t.Fatalf("published catalog changed: %q %v", data, err)
			}
			if _, err := os.Stat(filepath.Join(cfg.Output.GrammarBase, "python")); !os.IsNotExist(err) {
				t.Fatalf("publication occurred before rejecting cleanup: %v", err)
			}
		})
	}
}

func TestCleanUsesConfiguredWorkspaceTemplates(t *testing.T) {
	cfg, source, binary := policyWorkspace(t)
	cleaner := tsbuild.NewCleaner()
	if err := cleaner.Clean(context.Background(), cfg, ts.CleanRequest{Language: "python", Prune: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(source, "drop.tmp")); !os.IsNotExist(err) {
		t.Fatalf("prune ignored actual source root: %v", err)
	}
	if err := cleaner.Clean(context.Background(), cfg, ts.CleanRequest{Language: "python"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(binary); !os.IsNotExist(err) {
		t.Fatalf("clean ignored actual binary root: %v", err)
	}
}

func TestConfiguredWasmBackendAndImage(t *testing.T) {
	cfg, source, _ := policyWorkspace(t)
	writeSourceProvenance(t, source)
	cfg.Targets["linux"] = ts.BuildTarget{OS: "linux", Arch: "amd64", Compiler: "fixture-cc"}
	cfg.Runtime.WASM = ts.WASMConfig{Backend: "podman", Image: "example.invalid/emsdk:pinned"}
	cfg.Runtime.Tools = ts.ToolPaths{TreeSitter: "custom-ts", Podman: "custom-podman"}
	var wrapper string
	runner := &testutil.RecordingRunner{OnRun: func(_ context.Context, cmd ts.Command) error {
		if cmd.Name != "custom-ts" {
			return errors.New("configured tree-sitter executable not used")
		}
		if len(cmd.Args) > 1 && cmd.Args[1] == "--wasm" {
			data, err := os.ReadFile(filepath.Join(strings.Split(cmd.Env["PATH"], string(os.PathListSeparator))[0], "emcc"))
			if err != nil {
				return err
			}
			wrapper = string(data)
		}
		return writeCompilerFixture(t, cmd.Args[len(cmd.Args)-1])
	}}
	lookup := testutil.StaticLookup{Paths: map[string]string{"fixture-cc": "/tools/cc", "emcc": "/unused/emcc", "docker": "/unused/docker", "custom-podman": "/tools/podman"}}
	compiler := tsbuild.NewCompiler(runner, lookup, ioDiscard{}, ioDiscard{})
	prepareBuildConfig(cfg)
	if err := compiler.Compile(context.Background(), cfg, ts.CompileRequest{OS: "linux", Arch: "amd64", BuildWasm: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(wrapper, "'/tools/podman' run") || !strings.Contains(wrapper, "'example.invalid/emsdk:pinned'") {
		t.Fatalf("wrong backend/image: %s", wrapper)
	}
}

func TestCompilerFallbackCanBeDisabled(t *testing.T) {
	cfg, source, _ := policyWorkspace(t)
	writeSourceProvenance(t, source)
	no := false
	cfg.Runtime.Compiler.AllowFallback = &no
	cfg.Targets["linux"] = ts.BuildTarget{OS: "linux", Arch: "amd64", Compiler: "missing-compiler"}
	runner := &testutil.RecordingRunner{}
	compiler := tsbuild.NewCompiler(runner, testutil.StaticLookup{}, ioDiscard{}, ioDiscard{})
	prepareBuildConfig(cfg)
	err := compiler.Compile(context.Background(), cfg, ts.CompileRequest{})
	if err == nil || !strings.Contains(err.Error(), "compiler fallback is disabled") {
		t.Fatalf("expected strict compiler selection: %v", err)
	}
	if len(runner.Runs) != 0 {
		t.Fatal("compiler ran despite disabled fallback")
	}
}

func TestBuilderNodeInstallPolicies(t *testing.T) {
	for _, tc := range []struct {
		name, mode                    string
		lock, existing, failCI, retry bool
		calls                         int
		fails                         bool
	}{
		{"never", "never", true, false, false, true, 0, false},
		{"auto without lock", "auto", false, false, false, true, 0, false},
		{"auto existing", "auto", true, true, false, true, 0, false},
		{"always without lock", "always", false, true, false, true, 1, false},
		{"CI retry", "auto", true, false, true, true, 2, false},
		{"retry disabled", "auto", true, false, true, false, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, source, _ := policyWorkspace(t)
			cfg.TreeSitterCLIVersion = "0.26.8"
			cfg.Runtime.Tools.NPM = "custom-npm"
			cfg.Runtime.Node = ts.NodePolicy{Install: tc.mode, RetryInstall: &tc.retry, CIArgs: []string{"ci", "--ignore-scripts", "--offline"}, InstallArgs: []string{"install", "--ignore-scripts", "--offline"}}
			policyWrite(t, filepath.Join(source, "package.json"), "{}")
			if tc.lock {
				policyWrite(t, filepath.Join(source, "package-lock.json"), "{}")
			}
			if tc.existing {
				if err := os.MkdirAll(filepath.Join(source, "node_modules"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			runner := &testutil.RecordingRunner{
				OnRun: func(_ context.Context, cmd ts.Command) error {
					if cmd.Name == "custom-npm" {
						calls++
						if cmd.Args[len(cmd.Args)-1] != "--offline" {
							t.Fatal("configured npm args ignored")
						}
						if tc.failCI && cmd.Args[0] == "ci" {
							return errors.New("CI install failed")
						}
					}
					return nil
				},
				OnOutput: func(_ context.Context, cmd ts.Command) (string, error) {
					if cmd.Name == "tree-sitter" {
						return "tree-sitter 0.26.8", nil
					}
					if strings.Contains(strings.Join(cmd.Args, " "), "--format=%ct") {
						return "1700000000", nil
					}
					return reviewFixtureRevision, nil
				},
			}
			prepareBuildConfig(cfg)
			err := tsbuild.NewBuilder(runner, ioDiscard{}, ioDiscard{}).Build(context.Background(), cfg, ts.BuildRequest{})
			if (err != nil) != tc.fails || calls != tc.calls {
				t.Fatalf("npm calls=%d want %d, error=%v", calls, tc.calls, err)
			}
		})
	}
}

func TestBuilderCanDisableABIFallback(t *testing.T) {
	cfg, _, _ := policyWorkspace(t)
	cfg.GenerateABI = 15
	cfg.ABIRange = &ts.ABIRange{Min: 13, Max: 15}
	no := false
	cfg.Runtime.Generation.ABIFallback = &no
	attempts := 0
	runner := &testutil.RecordingRunner{
		OnRun: func(_ context.Context, cmd ts.Command) error {
			if cmd.Name == "tree-sitter" {
				attempts++
				return errors.New("ABI unsupported")
			}
			return nil
		},
		OnOutput: func(_ context.Context, cmd ts.Command) (string, error) {
			if cmd.Name == "tree-sitter" {
				return "tree-sitter 0.26.8", nil
			}
			if strings.Contains(strings.Join(cmd.Args, " "), "--format=%ct") {
				return "1700000000", nil
			}
			return reviewFixtureRevision, nil
		},
	}
	prepareBuildConfig(cfg)
	err := tsbuild.NewBuilder(runner, ioDiscard{}, ioDiscard{}).Build(context.Background(), cfg, ts.BuildRequest{})
	if err == nil || !strings.Contains(err.Error(), "ABI unsupported") || attempts != 1 {
		t.Fatalf("attempts=%d, error=%v", attempts, err)
	}
}
