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

type stubCleaner struct {
	req ts.CleanRequest
}

func (s *stubCleaner) Clean(_ context.Context, _ *ts.Config, req ts.CleanRequest) error {
	s.req = req
	return nil
}

type stubCompactor struct {
	req ts.CompactRequest
}

func (s *stubCompactor) Compact(_ context.Context, _ *ts.Config, req ts.CompactRequest) error {
	s.req = req
	return nil
}

// ─── Existing tests (preserved) ───────────────────────────────────────────────

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

// ─── Cobra short-flag tests ───────────────────────────────────────────────────

func newTestApp(cfg *ts.Config, builder ts.Builder, compiler ts.Compiler, mover ts.Mover, lookup testutil.StaticLookup) *ts.App {
	return ts.NewApp(
		&bytes.Buffer{},
		&bytes.Buffer{},
		lookup,
		func(string) (*ts.Config, error) { return cfg, nil },
		builder,
		compiler,
		mover,
	)
}

func TestBuildShortFlags(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "python"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	builder := &stubBuilder{}
	app := newTestApp(cfg, builder, &stubCompiler{}, &stubMover{}, testutil.StaticLookup{Paths: map[string]string{
		"git":         "/usr/bin/git",
		"tree-sitter": "/usr/bin/tree-sitter",
	}})

	// -l and -f are the short forms of --language and --force
	if err := app.Run(context.Background(), []string{"build", "-l", "python", "-f"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if builder.req.Language != "python" || !builder.req.Force {
		t.Fatalf("unexpected build request: %+v", builder.req)
	}
}

func TestCompileShortFlags(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "rust"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	compiler := &stubCompiler{}
	app := newTestApp(cfg, &stubBuilder{}, compiler, &stubMover{}, testutil.StaticLookup{Paths: map[string]string{
		"tree-sitter": "/usr/bin/tree-sitter",
	}})

	if err := app.Run(context.Background(), []string{"compile", "-l", "rust", "-o", "linux", "-a", "amd64"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if compiler.req.Language != "rust" {
		t.Fatalf("unexpected compile language: %q", compiler.req.Language)
	}
	if compiler.req.OS != "linux" || compiler.req.Arch != "amd64" {
		t.Fatalf("unexpected compile target: os=%q arch=%q", compiler.req.OS, compiler.req.Arch)
	}
}

func TestPersistentConfigFlag(t *testing.T) {
	t.Parallel()

	var capturedPath string
	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "go"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	app := ts.NewApp(
		&bytes.Buffer{},
		&bytes.Buffer{},
		testutil.StaticLookup{Paths: map[string]string{
			"git":         "/usr/bin/git",
			"tree-sitter": "/usr/bin/tree-sitter",
		}},
		func(path string) (*ts.Config, error) {
			capturedPath = path
			return cfg, nil
		},
		&stubBuilder{},
		&stubCompiler{},
		&stubMover{},
	)

	const customConfig = "my-config.yaml"
	if err := app.Run(context.Background(), []string{"-c", customConfig, "build"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if capturedPath != customConfig {
		t.Fatalf("expected config path %q, got %q", customConfig, capturedPath)
	}
}

func TestHelpDoesNotError(t *testing.T) {
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

	// Cobra's --help flag exits with nil error.
	if err := app.Run(context.Background(), []string{"--help"}); err != nil {
		t.Fatalf("--help returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{"build", "--help"}); err != nil {
		t.Fatalf("build --help returned error: %v", err)
	}
}

func TestVersionCommandAndFlag(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "go"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	var stdout bytes.Buffer
	app := ts.NewApp(
		&stdout,
		&bytes.Buffer{},
		testutil.StaticLookup{},
		func(string) (*ts.Config, error) { return cfg, nil },
		&stubBuilder{},
		&stubCompiler{},
		&stubMover{},
	)

	if err := app.Run(context.Background(), []string{"version"}); err != nil {
		t.Fatalf("version command returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "1.0.0") {
		t.Fatalf("expected version 1.0.0, got %q", stdout.String())
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"--version"}); err != nil {
		t.Fatalf("--version flag returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "1.0.0") {
		t.Fatalf("expected version 1.0.0, got %q", stdout.String())
	}
}

func TestAppMoveGranularManifestAndSCMFlags(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "python"}},
		Output: ts.Output{
			GrammarBase:        "data/tree-sitter/grammar",
			GenerateManifest:   true,
			DefaultMoveMode:    "--both",
			SupportedMoveModes: []string{"--json", "--scm", "--both"},
		},
	}

	testCases := []struct {
		name                 string
		args                 []string
		wantGenerateManifest bool
		wantCopySCM          bool
		wantMode             ts.MoveMode
	}{
		{
			name:                 "default behavior",
			args:                 []string{"move", "-l", "python"},
			wantGenerateManifest: true,
			wantCopySCM:          true,
			wantMode:             ts.MoveModeBoth,
		},
		{
			name:                 "manifest=false",
			args:                 []string{"move", "-l", "python", "--manifest=false"},
			wantGenerateManifest: false,
			wantCopySCM:          true,
			wantMode:             ts.MoveModeBoth,
		},
		{
			name:                 "no-manifest alias",
			args:                 []string{"move", "-l", "python", "--no-manifest"},
			wantGenerateManifest: false,
			wantCopySCM:          true,
			wantMode:             ts.MoveModeBoth,
		},
		{
			name:                 "scm=false",
			args:                 []string{"move", "-l", "python", "--scm=false"},
			wantGenerateManifest: true,
			wantCopySCM:          false,
			wantMode:             ts.MoveModeBoth,
		},
		{
			name:                 "no-scm alias",
			args:                 []string{"move", "-l", "python", "--no-scm"},
			wantGenerateManifest: true,
			wantCopySCM:          false,
			wantMode:             ts.MoveModeBoth,
		},
		{
			name:                 "both manifest=false and scm=false",
			args:                 []string{"move", "-l", "python", "--manifest=false", "--scm=false"},
			wantGenerateManifest: false,
			wantCopySCM:          false,
			wantMode:             ts.MoveModeBoth,
		},
		{
			name:                 "legacy both with manifest=false",
			args:                 []string{"move", "-l", "python", "--both", "--manifest=false"},
			wantGenerateManifest: false,
			wantCopySCM:          true,
			wantMode:             ts.MoveModeBoth,
		},
		{
			name:                 "legacy both with scm=false",
			args:                 []string{"move", "-l", "python", "--both", "--scm=false"},
			wantGenerateManifest: true,
			wantCopySCM:          false,
			wantMode:             ts.MoveModeBoth,
		},
		{
			name:                 "legacy json mode default",
			args:                 []string{"move", "-l", "python", "--json"},
			wantGenerateManifest: true,
			wantCopySCM:          false,
			wantMode:             ts.MoveModeJSON,
		},
		{
			name:                 "legacy json with manifest=false",
			args:                 []string{"move", "-l", "python", "--json", "--manifest=false"},
			wantGenerateManifest: false,
			wantCopySCM:          false,
			wantMode:             ts.MoveModeJSON,
		},
		{
			name:                 "legacy json with scm=false",
			args:                 []string{"move", "-l", "python", "--json", "--scm=false"},
			wantGenerateManifest: true,
			wantCopySCM:          false,
			wantMode:             ts.MoveModeJSON,
		},
		{
			name:                 "legacy json with no-scm",
			args:                 []string{"move", "-l", "python", "--json", "--no-scm"},
			wantGenerateManifest: true,
			wantCopySCM:          false,
			wantMode:             ts.MoveModeJSON,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			if err := app.Run(context.Background(), tc.args); err != nil {
				t.Fatalf("Run returned error: %v", err)
			}

			if mover.req.GenerateManifest != tc.wantGenerateManifest {
				t.Errorf("GenerateManifest: got %v, want %v", mover.req.GenerateManifest, tc.wantGenerateManifest)
			}
			if mover.req.CopySCM != tc.wantCopySCM {
				t.Errorf("CopySCM: got %v, want %v", mover.req.CopySCM, tc.wantCopySCM)
			}
			if mover.req.Mode != tc.wantMode {
				t.Errorf("Mode: got %v, want %v", mover.req.Mode, tc.wantMode)
			}
		})
	}
}

func TestAppBuildPruneFlag(t *testing.T) {
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

	if err := app.Run(context.Background(), []string{"build", "-l", "go", "--prune"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !builder.req.Prune {
		t.Fatalf("expected Prune to be true, got %v", builder.req.Prune)
	}
}

func TestAppCleanCommand(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "python"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	cleaner := &stubCleaner{}
	app := ts.NewApp(
		&bytes.Buffer{},
		&bytes.Buffer{},
		testutil.StaticLookup{},
		func(string) (*ts.Config, error) { return cfg, nil },
		&stubBuilder{},
		&stubCompiler{},
		&stubMover{},
	).WithCleaner(cleaner)

	// Test regular clean
	if err := app.Run(context.Background(), []string{"clean", "-l", "python"}); err != nil {
		t.Fatalf("Run clean returned error: %v", err)
	}
	if cleaner.req.Language != "python" || cleaner.req.Prune != false {
		t.Fatalf("unexpected clean request: %+v", cleaner.req)
	}

	// Test prune clean
	if err := app.Run(context.Background(), []string{"clean", "-l", "python", "--prune"}); err != nil {
		t.Fatalf("Run clean --prune returned error: %v", err)
	}
	if cleaner.req.Language != "python" || cleaner.req.Prune != true {
		t.Fatalf("unexpected clean prune request: %+v", cleaner.req)
	}
}

func TestAppCompileWasmFlag(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "python"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	compiler := &stubCompiler{}
	app := newTestApp(cfg, &stubBuilder{}, compiler, &stubMover{}, testutil.StaticLookup{Paths: map[string]string{
		"tree-sitter": "/usr/bin/tree-sitter",
	}})

	if err := app.Run(context.Background(), []string{"compile", "-l", "python", "--wasm"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !compiler.req.BuildWasm {
		t.Fatalf("expected BuildWasm to be true")
	}
}

func TestAppMoveSourceFlags(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "python"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	// Test --source
	mover1 := &stubMover{}
	app1 := newTestApp(cfg, &stubBuilder{}, &stubCompiler{}, mover1, testutil.StaticLookup{})
	if err := app1.Run(context.Background(), []string{"move", "-l", "python", "--source"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !mover1.req.CopySource {
		t.Fatalf("expected CopySource to be true with --source")
	}

	// Test --c-source alias
	mover2 := &stubMover{}
	app2 := newTestApp(cfg, &stubBuilder{}, &stubCompiler{}, mover2, testutil.StaticLookup{})
	if err := app2.Run(context.Background(), []string{"move", "-l", "python", "--c-source"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !mover2.req.CopySource {
		t.Fatalf("expected CopySource to be true with --c-source")
	}
}

func TestAppMoveJSAndWasmFlags(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "python"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	mover := &stubMover{}
	app := newTestApp(cfg, &stubBuilder{}, &stubCompiler{}, mover, testutil.StaticLookup{})
	if err := app.Run(context.Background(), []string{"move", "-l", "python", "--js", "--wasm"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !mover.req.CopyJS {
		t.Fatalf("expected CopyJS to be true")
	}
	if !mover.req.IncludeWasm {
		t.Fatalf("expected IncludeWasm to be true")
	}
}

func TestAppMoveCompactFlags(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "python"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar", GenerateManifest: true},
	}

	mover := &stubMover{}
	app := newTestApp(cfg, &stubBuilder{}, &stubCompiler{}, mover, testutil.StaticLookup{})
	if err := app.Run(context.Background(), []string{"move", "-l", "python", "--compact", "--check-compact"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !mover.req.Compact {
		t.Fatalf("expected Compact to be true")
	}
	if !mover.req.CheckCompact {
		t.Fatalf("expected CheckCompact to be true")
	}
}

func TestAppDispatchesCompactFlags(t *testing.T) {
	t.Parallel()

	cfg := &ts.Config{
		Version:   "1.0",
		BuildDir:  "build",
		Languages: []ts.Language{{Name: "python"}},
		Output:    ts.Output{GrammarBase: "data/tree-sitter/grammar"},
	}

	compactor := &stubCompactor{}
	app := newTestApp(cfg, &stubBuilder{}, &stubCompiler{}, &stubMover{}, testutil.StaticLookup{}).WithCompactor(compactor)

	// test --language and --check
	if err := app.Run(context.Background(), []string{"compact", "--language", "python", "--check"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if compactor.req.Language != "python" || !compactor.req.Check {
		t.Fatalf("unexpected compact request: %+v", compactor.req)
	}

	// test shorthand -l
	if err := app.Run(context.Background(), []string{"compact", "-l", "python"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if compactor.req.Language != "python" || compactor.req.Check {
		t.Fatalf("unexpected compact request: %+v", compactor.req)
	}

	// test global check (all languages)
	if err := app.Run(context.Background(), []string{"compact", "--check"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if compactor.req.Language != "" || !compactor.req.Check {
		t.Fatalf("unexpected compact request: %+v", compactor.req)
	}

	// test positional arg
	if err := app.Run(context.Background(), []string{"compact", "python"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if compactor.req.Language != "python" || compactor.req.Check {
		t.Fatalf("unexpected compact request: %+v", compactor.req)
	}
}
