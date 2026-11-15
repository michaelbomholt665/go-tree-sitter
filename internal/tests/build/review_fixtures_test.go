package build_test

import (
	"debug/pe"
	"os"
	"os/exec"
	"strings"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

const reviewFixtureRevision = "0123456789abcdef0123456789abcdef01234567"

func prepareBuildConfig(cfg *ts.Config) {
	if cfg.TreeSitterCLIVersion == "" {
		cfg.TreeSitterCLIVersion = "0.26.8"
	}
	for i := range cfg.Languages {
		if cfg.Languages[i].Revision == "" {
			cfg.Languages[i].Revision = reviewFixtureRevision
		}
	}
}
func prepareArtifactConfig(cfg *ts.Config) {
	prepareBuildConfig(cfg)
	for i := range cfg.Languages {
		cfg.Languages[i].Repository = "https://example.invalid/grammar"
		cfg.Languages[i].Revision = reviewFixtureRevision
		if cfg.Languages[i].Constructor == "" {
			cfg.Languages[i].Constructor = "tree_sitter_" + cfg.Languages[i].Name
		}
	}
}
func writeCompilerFixture(t *testing.T, path string) error {
	t.Helper()
	if strings.HasSuffix(path, ".dll") {
		return os.WriteFile(path, createMockPEDLL(t, pe.IMAGE_FILE_MACHINE_AMD64, "tree_sitter_python"), 0644)
	}
	if strings.HasSuffix(path, ".wasm") {
		return os.WriteFile(path, []byte("wasm fixture"), 0644)
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("compiler fixture requires cc")
	}
	cmd := exec.Command(cc, "-shared", "-fPIC", "-x", "c", "-", "-o", path)
	cmd.Stdin = strings.NewReader("void *tree_sitter_python(void) { return 0; }\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile fixture: %v: %s", err, out)
	}
	return nil
}
