package treesitter_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

func TestConfigPathCommand(t *testing.T) {
	for _, mode := range []string{"production", "development", "explicit"} {
		t.Run(mode, func(t *testing.T) {
			work := isolatedConfigHome(t)
			args := []string{"config", "path"}
			var want string
			switch mode {
			case "production":
				dir, err := os.UserConfigDir()
				if err != nil {
					t.Fatal(err)
				}
				want = filepath.Join(dir, "ts-build", ts.DefaultConfigPath)
				catalogWrite(t, filepath.Join(work, ts.DefaultConfigPath), "ignored local YAML")
			case "development":
				catalogWrite(t, filepath.Join(work, ".env"), "TS_BUILD_ENV=DEVELOPMENT\n")
				want = filepath.Join(work, ts.DefaultConfigPath)
				// Reporting the location must not depend on parsing valid YAML.
				catalogWrite(t, want, "invalid: [")
			case "explicit":
				catalogWrite(t, filepath.Join(work, ".env"), "TS_BUILD_ENV=INVALID\n")
				want = filepath.Join(work, "custom.yaml")
				args = append(args, "--config", "custom.yaml")
			}
			var out bytes.Buffer
			app := ts.NewApp(&out, &out, nil, nil, nil, nil, nil)
			if err := app.Run(context.Background(), args); err != nil {
				t.Fatal(err)
			}
			if out.String() != want+"\n" {
				t.Fatalf("got %q, want %q", out.String(), want+"\n")
			}
			if mode == "production" {
				if _, err := ts.LoadConfig(want); err != nil {
					t.Fatalf("first-run configuration not initialized: %v", err)
				}
			}
		})
	}
}

func TestConfigPathCommandErrors(t *testing.T) {
	work := isolatedConfigHome(t)
	var out bytes.Buffer
	app := ts.NewApp(&out, &out, nil, nil, nil, nil, nil)
	if err := app.Run(context.Background(), []string{"config", "path", "extra"}); err == nil {
		t.Fatal("unexpected positional argument accepted")
	}
	catalogWrite(t, filepath.Join(work, ".env"), "TS_BUILD_ENV=DEVELOPMENT\n")
	out.Reset()
	if err := app.Run(context.Background(), []string{"config", "path"}); err == nil || !strings.Contains(err.Error(), "development configuration") {
		t.Fatalf("expected missing local configuration error, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("printed a path despite selection error: %q", out.String())
	}
}
