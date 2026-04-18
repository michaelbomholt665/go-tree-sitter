package testutil

import (
	"context"
	"os/exec"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type RecordingRunner struct {
	Runs  []_jsii.Command
	OnRun func(context.Context, _jsii.Command) error
}

func (r *RecordingRunner) Run(ctx context.Context, cmd _jsii.Command) error {
	r.Runs = append(r.Runs, cmd)
	if r.OnRun != nil {
		return r.OnRun(ctx, cmd)
	}
	return nil
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
