package treesitter_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/michaelbomholt665/go-tree-sitter/internal/tests/testutil"
	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type stubBuilder struct {
	cfg *ts.Config
	req ts.BuildRequest
}

func (s *stubBuilder) Build(_ context.Context, cfg *ts.Config, req ts.BuildRequest) error {
	s.cfg = cfg
	s.req = req
	return nil
}

type stubCompiler struct {
	req ts.CompileRequest
}

func (s *stubCompiler) Compile(_ context.Context, _ *ts.Config, req ts.CompileRequest) error {
	s.req = req
	return nil
}

type stubMover struct {
	req ts.MoveRequest
}

func (s *stubMover) Move(_ context.Context, _ *ts.Config, req ts.MoveRequest) error {
	s.req = req
	return nil
}

func TestAppDispatchesBuildFlags(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "go"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	builder := &stubBuilder{}
	app := ts.NewApp(
		&bytes.Buffer{},
		&bytes.Buffer{},
		testutil.StaticLookup{Paths: map[string]string{
			"git":         "/usr/bin/git",
			"tree-sitter": "/usr/bin/tree-sitter",
		}},
		func(string) (*ts.Config, error) { return cfg, nil },
		builder,
		&stubCompiler{},
		&stubMover{},
	)

	if err := app.Run(context.Background(), []string{"build", "--language", "go", "--force"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if builder.cfg != cfg {
		t.Fatalf("expected build config to be forwarded")
	}
	if builder.req.Language != "go" || !builder.req.Force {
		t.Fatalf("unexpected build request: %+v", builder.req)
	}
}

func TestAppMoveRejectsConflictingModes(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "go"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	app := ts.NewApp(
		&bytes.Buffer{},
		&bytes.Buffer{},
		testutil.StaticLookup{},
		func(string) (*ts.Config, error) { return cfg, nil },
		&stubBuilder{},
		&stubCompiler{},
		&stubMover{},
	)

	err := app.Run(context.Background(), []string{"move", "--json", "--scm"})
	if err == nil {
		t.Fatalf("expected conflicting move flags to fail")
	}
	if !strings.Contains(err.Error(), "move mode flags are mutually exclusive") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMoveModeDefaultsToBoth(t *testing.T) {
	t.Parallel()

	mode, err := ts.ResolveMoveMode(false, false, false, "--both")
	if err != nil {
		t.Fatalf("ResolveMoveMode returned error: %v", err)
	}
	if mode != ts.MoveModeBoth {
		t.Fatalf("expected default move mode %q, got %q", ts.MoveModeBoth, mode)
	}
}

func TestAppUsesConfiguredMoveDefault(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "go"}},
		Output: ts.Output{
			GrammarBase:        "data/tree-sitter/grammar",
			GenerateManifest:   true,
			DefaultMoveMode:    "--json",
			SupportedMoveModes: []string{"--json", "--scm", "--both"},
		},
	}

	mover := &stubMover{}
	app := ts.NewApp(
		&bytes.Buffer{},
		&bytes.Buffer{},
		testutil.StaticLookup{},
		func(string) (*ts.Config, error) { return cfg, nil },
		&stubBuilder{},
		&stubCompiler{},
		mover,
	)

	if err := app.Run(context.Background(), []string{"move"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if mover.req.Mode != ts.MoveModeJSON {
		t.Fatalf("expected configured default move mode %q, got %q", ts.MoveModeJSON, mover.req.Mode)
	}
}
