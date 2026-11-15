package build_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	"github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

// Opt-in: validate temporary native builds without moving or publishing artifacts.
// TS_PROVIDER_SMOKE_ROOT contains tree-sitter-<name>/ sources and compiled/<name>.so.
func TestProviderNativeSmoke(t *testing.T) {
	root := os.Getenv("TS_PROVIDER_SMOKE_ROOT")
	if root == "" {
		t.Skip("set TS_PROVIDER_SMOKE_ROOT to validate temporary provider grammar builds")
	}
	if runtime.GOOS != "linux" {
		t.Skip("temporary smoke libraries use Linux .so files")
	}
	cfg, err := ts.LoadConfig("../../../tree-sitter-config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"powershell", "fsharp", "fsharp-signature", "kotlin", "groovy", "lean", "swift", "dart",
		"gleam", "typst", "clojure", "cue", "hcl", "nix", "zsh", "fish", "prisma", "vue",
		"svelte", "objc", "json5", "less", "mdx",
	} {
		t.Run(name, func(t *testing.T) {
			langs, err := cfg.ResolveLanguages(name)
			if err != nil {
				t.Fatal(err)
			}
			lang := langs[0]
			source := filepath.Join(root, "tree-sitter-"+name, lang.SourceSubdir)
			_, err = build.NewNativeValidator().Validate(context.Background(), []build.ValidationItem{{
				Language: lang, BinaryPath: filepath.Join(root, "compiled", name+".so"),
				Platform: "linux", Arch: runtime.GOARCH,
				NodeTypesPath: filepath.Join(source, "src", "node-types.json"), Sample: lang.Sample,
			}})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
