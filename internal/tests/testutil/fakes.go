package testutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type RecordingRunner struct {
	Runs     []_jsii.Command
	Outputs  []_jsii.Command
	OnRun    func(context.Context, _jsii.Command) error
	OnOutput func(context.Context, _jsii.Command) (string, error)
}

func (r *RecordingRunner) Run(ctx context.Context, cmd _jsii.Command) error {
	r.Runs = append(r.Runs, cmd)
	if r.OnRun != nil {
		if err := r.OnRun(ctx, cmd); err != nil {
			return err
		}
	}
	if cmd.Name == "tree-sitter" && len(cmd.Args) > 0 && cmd.Args[0] == "generate" {
		path := filepath.Join(cmd.Dir, "src", "node-types.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte("[]\n"), 0o644)
	}
	return nil
}

func (r *RecordingRunner) Output(ctx context.Context, cmd _jsii.Command) (string, error) {
	r.Outputs = append(r.Outputs, cmd)
	if r.OnOutput != nil {
		return r.OnOutput(ctx, cmd)
	}
	switch cmd.Name {
	case "tree-sitter":
		return "tree-sitter 0.26.8", nil
	case "git":
		if len(cmd.Args) >= 2 && cmd.Args[len(cmd.Args)-2] == "--format=%ct" {
			return "1700000000", nil
		}
		return "0123456789abcdef0123456789abcdef01234567", nil
	default:
		return "test-tool 1.0.0", nil
	}
}

type StaticLookup struct {
	Paths map[string]string
}

func (l StaticLookup) LookPath(name string) (string, error) {
	if path, ok := l.Paths[name]; ok {
		return path, nil
	}
	return "", exec.ErrNotFound
}
