package treesitter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

func TestLoadConfigAppliesDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "tree-sitter-config.yaml")
	content := `
version: "1.0"
abi_versions:
  ">=0.25": { min: 13, max: 15 }
languages:
  - name: "python"
    version: "v0.25.0"
    tree_sitter_version: "0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-python.git"
output:
  generate_manifest: false
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := ts.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.BuildDir != "build" {
		t.Fatalf("expected default build dir %q, got %q", "build", cfg.BuildDir)
	}
	if cfg.Output.GrammarBase != filepath.Join("data", "tree-sitter", "grammar") {
		t.Fatalf("expected default grammar base, got %q", cfg.Output.GrammarBase)
	}
	if cfg.Output.GrammarBuild != "build/{lang}/{arch}" {
		t.Fatalf("expected default grammar build template, got %q", cfg.Output.GrammarBuild)
	}
	if cfg.Output.GrammarCompile != "build/{lang}/{arch}" {
		t.Fatalf("expected default grammar compile template, got %q", cfg.Output.GrammarCompile)
	}
	if cfg.Output.GenerateManifest {
		t.Fatalf("expected explicit generate_manifest=false to be preserved")
	}
	if cfg.Output.DefaultMoveMode != "--both" {
		t.Fatalf("expected default move mode %q, got %q", "--both", cfg.Output.DefaultMoveMode)
	}
	if len(cfg.Output.SupportedMoveModes) != 3 {
		t.Fatalf("expected default supported move modes, got %v", cfg.Output.SupportedMoveModes)
	}
}

func TestLoadConfigRejectsDuplicateLanguages(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "tree-sitter-config.yaml")
	content := `
version: "1.0"
build_dir: "build"
abi_versions:
  ">=0.25": { min: 13, max: 15 }
languages:
  - name: "python"
    version: "v0.25.0"
    tree_sitter_version: "0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-python.git"
  - name: "python"
    version: "v0.25.0"
    tree_sitter_version: "0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-python.git"
output:
  grammar_base: "data/tree-sitter/grammar"
  generate_manifest: true
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := ts.LoadConfig(configPath)
	if err == nil {
		t.Fatalf("expected duplicate language validation error")
	}
	if !strings.Contains(err.Error(), "duplicate language name") {
		t.Fatalf("expected duplicate language error, got %v", err)
	}
}

func TestLoadConfigValidatesOSTarget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "tree-sitter-config.yaml")
	content := `
version: "1.0"
build_dir: "build"
OS_TARGET: "linux"
targets:
  windows:
    os: "windows"
    arch: "amd64"
abi_versions:
  ">=0.25": { min: 13, max: 15 }
languages:
  - name: "python"
    version: "v0.25.0"
    tree_sitter_version: "0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-python.git"
output:
  grammar_base: "data/tree-sitter/grammar"
  generate_manifest: true
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := ts.LoadConfig(configPath)
	if err == nil {
		t.Fatalf("expected missing OS_TARGET validation error")
	}
	if !strings.Contains(err.Error(), `OS_TARGET "linux" is not defined in targets`) {
		t.Fatalf("expected missing OS_TARGET error, got %v", err)
	}
}

func TestLoadConfigV2ValidatesABIRange(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "tree-sitter-config.yaml")
	content := `
version: "2.0"
tree_sitter_cli_version: "0.26.8"
abi_range:
  min: 13
  max: 15
generate_abi: 15
OS_TARGET: "linux"
targets:
  linux:
    os: "linux"
    arch: "amd64"
    triple: "x86_64-linux-gnu"
    compiler: "gcc"
    compiler_version: "14.2.0"
languages:
  - name: "python"
    version: "v0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-python.git"
    revision: "293fdc02038ee2bf0e2e206711b69c90ac0d413f"
    sample: "x = 1\n"
    generate_abi: 14
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := ts.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.ABIRange == nil || cfg.ABIRange.Min != 13 || cfg.ABIRange.Max != 15 {
		t.Fatalf("expected abi_range [13, 15], got %+v", cfg.ABIRange)
	}
	if cfg.GenerateABI != 15 {
		t.Fatalf("expected generate_abi 15, got %d", cfg.GenerateABI)
	}
	if len(cfg.Languages) != 1 || cfg.Languages[0].GenerateABI == nil || *cfg.Languages[0].GenerateABI != 14 {
		t.Fatalf("expected language generate_abi 14, got %+v", cfg.Languages[0].GenerateABI)
	}
}

func TestLoadConfigV2RejectsInvalidGenerateABI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "tree-sitter-config.yaml")
	content := `
version: "2.0"
tree_sitter_cli_version: "0.26.8"
abi_range:
  min: 13
  max: 15
generate_abi: 16
OS_TARGET: "linux"
targets:
  linux:
    os: "linux"
    arch: "amd64"
    triple: "x86_64-linux-gnu"
    compiler: "gcc"
    compiler_version: "14.2.0"
languages:
  - name: "python"
    version: "v0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-python.git"
    revision: "293fdc02038ee2bf0e2e206711b69c90ac0d413f"
    sample: "x = 1\n"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := ts.LoadConfig(configPath)
	if err == nil {
		t.Fatalf("expected error for generate_abi outside abi_range")
	}
	if !strings.Contains(err.Error(), "must be within supported abi_range 13-15") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
