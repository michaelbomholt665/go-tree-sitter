package treesitter_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	bundledconfig "github.com/michaelbomholt665/go-tree-sitter"
	"github.com/michaelbomholt665/go-tree-sitter/internal/tests/testutil"
	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	"gopkg.in/yaml.v3"
)

func isolatedConfigHome(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for key, value := range map[string]string{
		"HOME": root, "USERPROFILE": root,
		"XDG_CONFIG_HOME": filepath.Join(root, "config"),
		"XDG_CACHE_HOME":  filepath.Join(root, "cache"),
		"XDG_DATA_HOME":   filepath.Join(root, "data"),
		"APPDATA":         filepath.Join(root, "roaming"),
		"LOCALAPPDATA":    filepath.Join(root, "local"),
	} {
		t.Setenv(key, value)
	}
	work := filepath.Join(root, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	return work
}

func TestUserConfigurationInitialization(t *testing.T) {
	work := isolatedConfigHome(t)
	// A local YAML alone must not select development mode.
	catalogWrite(t, filepath.Join(work, ts.DefaultConfigPath), "invalid local YAML: [")
	path, err := ts.ResolveConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	user, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(user, "ts-build", ts.DefaultConfigPath) {
		t.Fatalf("wrong OS path: %s", path)
	}
	cfg, err := ts.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	var embedded ts.Config
	if err := yaml.Unmarshal([]byte(bundledconfig.YAML), &embedded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.Languages, embedded.Languages) {
		t.Fatal("embedded grammar pins or assessments changed")
	}
	for _, setting := range []string{cfg.BuildDir, cfg.Output.GrammarBase, cfg.Output.GrammarBuild, cfg.Output.GrammarCompile} {
		if !filepath.IsAbs(setting) || strings.HasPrefix(setting, work+string(filepath.Separator)) {
			t.Fatalf("nonportable initialized path: %s", setting)
		}
	}
	if cfg.Output.GrammarBase == cfg.BuildDir {
		t.Fatal("cache and published data share a directory")
	}
	// An upgrade or another run must never overwrite user edits.
	catalogWrite(t, path, "user customization\n")
	again, err := ts.ResolveConfigPath()
	if err != nil || again != path || catalogRead(t, path) != "user customization\n" {
		t.Fatalf("existing file changed: %s %v", again, err)
	}
}

func TestDevelopmentConfigMarker(t *testing.T) {
	for _, marker := range []string{
		"TS_BUILD_ENV=DEVELOPMENT\n",
		"# repository mode\nexport TS_BUILD_ENV = \"DEVELOPMENT\" # comment\n",
		"TS_BUILD_ENV='DEVELOPMENT'\nUNRELATED=value\n",
	} {
		t.Run(marker, func(t *testing.T) {
			work := isolatedConfigHome(t)
			catalogWrite(t, filepath.Join(work, ".env"), marker)
			if _, err := ts.ResolveConfigPath(); err == nil || !strings.Contains(err.Error(), "development configuration") {
				t.Fatalf("missing local config did not fail: %v", err)
			}
			local := filepath.Join(work, ts.DefaultConfigPath)
			catalogWrite(t, local, bundledconfig.YAML)
			path, err := ts.ResolveConfigPath()
			if err != nil || path != local {
				t.Fatalf("wrong development path: %s %v", path, err)
			}
			user, _ := os.UserConfigDir()
			if _, err := os.Stat(filepath.Join(user, "ts-build", ts.DefaultConfigPath)); !os.IsNotExist(err) {
				t.Fatal("development mode created user configuration")
			}
		})
	}
}

func TestNonDevelopmentMarkers(t *testing.T) {
	for _, marker := range []string{"", "# nothing configured\n", "OTHER_SETTING=DEVELOPMENT\n", "TS_BUILD_ENV=PRODUCTION\n"} {
		t.Run(marker, func(t *testing.T) {
			work := isolatedConfigHome(t)
			catalogWrite(t, filepath.Join(work, ".env"), marker)
			path, err := ts.ResolveConfigPath()
			if err != nil || filepath.Dir(path) == work {
				t.Fatalf("wrong production selection: %s %v", path, err)
			}
		})
	}
}

func TestInvalidDevelopmentMarkerFails(t *testing.T) {
	for _, marker := range []string{"TS_BUILD_ENV=TYPO", "TS_BUILD_ENV=\"DEVELOPMENT", "TS_BUILD_ENV=\"DEVELOPMENT\"garbage"} {
		t.Run(marker, func(t *testing.T) {
			work := isolatedConfigHome(t)
			catalogWrite(t, filepath.Join(work, ".env"), marker)
			if _, err := ts.ResolveConfigPath(); err == nil {
				t.Fatal("invalid marker accepted")
			}
		})
	}
}

func TestDefaultCLIDiscoversConfigAndResolvesPaths(t *testing.T) {
	work := isolatedConfigHome(t)
	catalogWrite(t, filepath.Join(work, ".env"), "TS_BUILD_ENV=DEVELOPMENT\n")
	catalogWrite(t, filepath.Join(work, ts.DefaultConfigPath), bundledconfig.YAML)
	builder := &stubBuilder{}
	lookup := testutil.StaticLookup{Paths: map[string]string{"git": "git", "tree-sitter": "tree-sitter"}}
	app := ts.NewApp(&bytes.Buffer{}, &bytes.Buffer{}, lookup, nil, builder, nil, nil)
	if err := app.Run(context.Background(), []string{"build", "-l", "python"}); err != nil {
		t.Fatal(err)
	}
	if builder.cfg.BuildDir != filepath.Join(work, "build") || builder.cfg.Output.GrammarBase != filepath.Join(work, "data", "tree-sitter", "grammar") {
		t.Fatalf("relative development paths resolved incorrectly: %+v", builder.cfg.Output)
	}
	// The explicit compatibility flag still overrides automatic selection.
	if err := os.WriteFile(filepath.Join(work, ".env"), []byte("TS_BUILD_ENV=TYPO\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(context.Background(), []string{"--config", filepath.Join(work, ts.DefaultConfigPath), "build", "-l", "python"}); err != nil {
		t.Fatal(err)
	}
}

func TestCLIProductionIgnoresLocalYAMLAndVersionNeedsNoConfig(t *testing.T) {
	work := isolatedConfigHome(t)
	catalogWrite(t, filepath.Join(work, ts.DefaultConfigPath), "invalid: [")
	var out bytes.Buffer
	app := ts.NewApp(&out, &out, nil, nil, nil, nil, nil)
	if err := app.Run(context.Background(), []string{"version"}); err != nil {
		t.Fatal(err)
	}
	user, _ := os.UserConfigDir()
	path := filepath.Join(user, "ts-build", ts.DefaultConfigPath)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("version initialized a configuration")
	}
	if err := app.Run(context.Background(), []string{"list", "-l", "python"}); err != nil {
		t.Fatal(err)
	}
	// Invalid existing user files must fail, never silently fall back to defaults.
	catalogWrite(t, path, "invalid: [")
	if err := app.Run(context.Background(), []string{"list"}); err == nil {
		t.Fatal("invalid user configuration ignored")
	}
}
