package treesitter_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type dryRunRuntime struct{ calls int }

func (r *dryRunRuntime) Build(context.Context, *ts.Config, ts.BuildRequest) error {
	r.calls++
	return errors.New("runtime reached")
}
func (r *dryRunRuntime) Compile(context.Context, *ts.Config, ts.CompileRequest) error {
	r.calls++
	return errors.New("runtime reached")
}
func (r *dryRunRuntime) Move(context.Context, *ts.Config, ts.MoveRequest) error {
	r.calls++
	return errors.New("runtime reached")
}
func (r *dryRunRuntime) Clean(context.Context, *ts.Config, ts.CleanRequest) error {
	r.calls++
	return errors.New("runtime reached")
}
func (r *dryRunRuntime) Compact(context.Context, *ts.Config, ts.CompactRequest) error {
	r.calls++
	return errors.New("runtime reached")
}
func (r *dryRunRuntime) LookPath(name string) (string, error) { r.calls++; return name, nil }

func dryRunSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		out[rel] = info.Mode().String()
		if entry.Type()&os.ModeSymlink != 0 {
			out[rel], err = os.Readlink(path)
			return err
		}
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			hash := sha256.Sum256(data)
			out[rel] += string(hash[:])
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGlobalDryRunRuntimeCommands(t *testing.T) {
	for _, args := range [][]string{
		{"build", "-l", "python", "--force", "--prune"},
		{"compile", "-l", "python", "--os", "linux", "--arch", "amd64"},
		{"compile", "-l", "python", "--wasm"},
		{"move", "-l", "python", "--force", "--clean", "--manifest=false", "--js"},
		{"clean", "-l", "python"},
		{"clean", "-l", "python", "--prune"},
		{"compact", "-l", "python"},
		{"compact", "-l", "python", "--check"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			path, _, _ := supportFixture(t)
			root := filepath.Dir(path)
			for _, dir := range []string{"build/python", "data/tree-sitter/grammar/python"} {
				if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, dir, "sentinel"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := dryRunSnapshot(t, root)
			var out bytes.Buffer
			runtime := &dryRunRuntime{}
			app := ts.NewApp(&out, &out, runtime, nil, runtime, runtime, runtime).WithCleaner(runtime).WithCompactor(runtime)
			argv := append([]string{"--config", path, "--dry-run"}, args...)
			if err := app.Run(context.Background(), argv); err != nil {
				t.Fatal(err, out.String())
			}
			if runtime.calls != 0 {
				t.Fatal("runtime or tool lookup executed")
			}
			var plan struct {
				DryRun  bool           `json:"dry_run"`
				Command string         `json:"command"`
				Request map[string]any `json:"request"`
			}
			if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
				t.Fatal(err, out.String())
			}
			if !plan.DryRun || plan.Command != args[0] || plan.Request["Language"] != "python" {
				t.Fatalf("incorrect plan: %+v", plan)
			}
			if args[0] == "move" && (plan.Request["GenerateManifest"] != false || plan.Request["CopyJS"] != true || plan.Request["CleanupAction"] != "remove") {
				t.Fatalf("move flags not preserved: %+v", plan.Request)
			}
			if !reflect.DeepEqual(before, dryRunSnapshot(t, root)) {
				t.Fatal("dry run changed files")
			}
		})
	}
}

func TestDryRunDoesNotLeakIntoLaterExecution(t *testing.T) {
	path, _, _ := supportFixture(t)
	runtime := &dryRunRuntime{}
	app := ts.NewApp(nil, nil, runtime, nil, runtime, runtime, runtime)
	if err := app.Run(context.Background(), []string{"build", "-c", path, "-l", "python", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if runtime.calls != 0 {
		t.Fatal("dry run executed runtime")
	}
	err := app.Run(context.Background(), []string{"build", "-c", path, "-l", "python"})
	if err == nil || !strings.Contains(err.Error(), "runtime reached") || runtime.calls == 0 {
		t.Fatal("preview leaked into next execution", err)
	}
}

func TestDryRunStillRejectsInvalidInputs(t *testing.T) {
	path, _, _ := supportFixture(t)
	for _, args := range [][]string{
		{"build", "-l", "does-not-exist"},
		{"compile", "--os", "linux", "--arch", "imaginary"},
		{"move", "--json", "--both"},
		{"export"},
		{"config", "set", "languages", "[]"},
		{"config", "set", "runtime.cleanup.after_move", "keep", "--apply"},
		{"regen-manifests"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			before := dryRunSnapshot(t, filepath.Dir(path))
			app := ts.NewApp(nil, nil, nil, nil, nil, nil, nil)
			if err := app.Run(context.Background(), append([]string{"-c", path, "--dry-run"}, args...)); err == nil {
				t.Fatal("invalid dry run accepted")
			}
			if !reflect.DeepEqual(before, dryRunSnapshot(t, filepath.Dir(path))) {
				t.Fatal("invalid dry run changed files")
			}
		})
	}
}

func TestDryRunConfigReadAndChangeCommands(t *testing.T) {
	path, _, assessment := supportFixture(t)
	input := filepath.Join(filepath.Dir(path), "assessment.json")
	data, err := json.Marshal(assessment)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, data, 0600); err != nil {
		t.Fatal(err)
	}
	gomod := filepath.Join(filepath.Dir(path), "go.mod")
	if err := os.WriteFile(gomod, []byte("module test\n\nrequire (\n\tgithub.com/tree-sitter/tree-sitter-python v0.99.0\n)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"config", "path"}, {"config", "show", "--json"}, {"config", "get", "runtime.cleanup.after_move"}, {"config", "validate"},
		{"config", "set", "runtime.cleanup.after_move", "keep", "--json"},
		{"support", "show", "-l", "python", "--json"},
		{"support", "update", "-l", "python", "--assessment", input, "--check-report"},
		{"sync-config", "--gomod", gomod}, {"list", "--json"}, {"version"}, {"completion", "bash"},
	} {
		t.Run(strings.Join(args[:1], " "), func(t *testing.T) {
			before := dryRunSnapshot(t, filepath.Dir(path))
			var out bytes.Buffer
			app := ts.NewApp(&out, &out, nil, nil, nil, nil, nil)
			if err := app.Run(context.Background(), append([]string{"-c", path, "--dry-run"}, args...)); err != nil {
				t.Fatal(err, out.String())
			}
			if !reflect.DeepEqual(before, dryRunSnapshot(t, filepath.Dir(path))) {
				t.Fatal("dry run changed files")
			}
		})
	}
}

func TestDryRunExportPreflightWithoutCopy(t *testing.T) {
	path, _, _ := supportFixture(t)
	root := filepath.Dir(path)
	source := filepath.Join(root, "data/tree-sitter/grammar/python")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "node-types.json"), []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "external")
	before := dryRunSnapshot(t, root)
	var out bytes.Buffer
	app := ts.NewApp(&out, &out, nil, nil, nil, nil, nil)
	if err := app.Run(context.Background(), []string{"export", "-c", path, "-l", "python", "--to", destination, "--dry-run"}); err != nil {
		t.Fatal(err, out.String())
	}
	if !strings.Contains(out.String(), `"dry_run":true`) {
		t.Fatal("missing export plan", out.String())
	}
	if !reflect.DeepEqual(before, dryRunSnapshot(t, root)) {
		t.Fatal("export created destination or stage")
	}
	if err := app.Run(context.Background(), []string{"export", "-c", path, "-l", "python", "--to", filepath.Dir(source), "--dry-run"}); err == nil {
		t.Fatal("overlapping export accepted")
	}
	if err := os.Symlink(filepath.Join(source, "node-types.json"), filepath.Join(source, "bad-link")); err != nil {
		t.Skip(err)
	}
	if err := app.Run(context.Background(), []string{"export", "-c", path, "-l", "python", "--to", destination, "--dry-run"}); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestDryRunMissingUserConfigStaysInMemory(t *testing.T) {
	root := filepath.Dir(isolatedConfigHome(t))
	before := dryRunSnapshot(t, root)
	for _, args := range [][]string{
		{"build", "-l", "python"}, {"config", "show", "--json"}, {"config", "validate"},
		{"config", "set", "runtime.cleanup.after_move", "keep"}, {"list", "-l", "python"}, {"config", "path"},
	} {
		var out bytes.Buffer
		app := ts.NewApp(&out, &out, nil, nil, nil, nil, nil)
		if err := app.Run(context.Background(), append(args, "--dry-run")); err != nil {
			t.Fatal(args, err, out.String())
		}
		if !strings.Contains(out.String(), "no file created") {
			t.Fatal("missing initialization notice")
		}
		if !reflect.DeepEqual(before, dryRunSnapshot(t, root)) {
			t.Fatal("dry run initialized user directories")
		}
	}
}
