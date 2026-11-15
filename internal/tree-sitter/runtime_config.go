package treesitter

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/michaelbomholt665/go-tree-sitter/internal/configsync"
)

// RuntimeConfig controls execution policy; grammar source pins remain in Languages.
type RuntimeConfig struct {
	Tools            ToolPaths           `yaml:"tools"`
	WASM             WASMConfig          `yaml:"wasm"`
	Compiler         CompilerPolicy      `yaml:"compiler"`
	Node             NodePolicy          `yaml:"node"`
	Generation       GenerationPolicy    `yaml:"generation"`
	Cleanup          CleanupPolicy       `yaml:"cleanup"`
	Publication      PublicationDefaults `yaml:"publication"`
	SupportedTargets map[string][]string `yaml:"supported_targets"`
	Wizard           WizardDefaults      `yaml:"wizard"`
	Queries          QueryPolicy         `yaml:"queries"`
	SyncModules      map[string][]string `yaml:"sync_modules,omitempty"`
}

type ToolPaths struct {
	TreeSitter string `yaml:"tree_sitter"`
	Git        string `yaml:"git"`
	NPM        string `yaml:"npm"`
	EMCC       string `yaml:"emcc"`
	Docker     string `yaml:"docker"`
	Podman     string `yaml:"podman"`
	Zig        string `yaml:"zig"`
}
type WASMConfig struct {
	Backend string `yaml:"backend"`
	Image   string `yaml:"image"`
}
type CompilerPair struct {
	CC  string `yaml:"cc"`
	CXX string `yaml:"cxx"`
}
type CompilerPolicy struct {
	AllowFallback *bool                     `yaml:"allow_fallback"`
	HostCC        string                    `yaml:"host_cc"`
	Candidates    map[string][]CompilerPair `yaml:"candidates"`
}
type NodePolicy struct {
	Install      string   `yaml:"install"`
	CIArgs       []string `yaml:"ci_args"`
	InstallArgs  []string `yaml:"install_args"`
	RetryInstall *bool    `yaml:"retry_install"`
}
type GenerationPolicy struct {
	ABIFallback *bool `yaml:"abi_fallback"`
}
type CleanupPolicy struct {
	AfterMove string     `yaml:"after_move"`
	Prune     PruneRules `yaml:"prune"`
}
type PruneRules struct {
	Directories         []string `yaml:"directories"`
	Files               []string `yaml:"files"`
	PreserveFiles       []string `yaml:"preserve_files"`
	PreserveDirectories []string `yaml:"preserve_directories"`
	ProtectedFiles      []string `yaml:"protected_files"` // Files removable inside preserved directories.
}
type PublicationDefaults struct {
	Manifest             *bool `yaml:"manifest"`
	SCM                  *bool `yaml:"scm,omitempty"` // nil follows output.default_move_mode.
	Compact              bool  `yaml:"compact"`
	CheckCompact         bool  `yaml:"check_compact"`
	Source               bool  `yaml:"source"`
	JS                   bool  `yaml:"js"`
	WASM                 bool  `yaml:"wasm"`
	Force                bool  `yaml:"force"`
	AllowCrossValidation bool  `yaml:"allow_cross_validation"`
}
type WizardDefaults struct {
	Platforms []string `yaml:"platforms"`
	Artifacts []string `yaml:"artifacts"`
}
type QueryPolicy struct {
	Paths    map[string][]string `yaml:"paths"` // Relative to the checked-out repository.
	Rewrites []QueryRewrite      `yaml:"rewrites"`
}
type QueryRewrite struct {
	Language string `yaml:"language"`
	File     string `yaml:"file"`
	Find     string `yaml:"find"`
	Replace  string `yaml:"replace"`
}

func BoolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
func boolPointer(v bool) *bool { return &v }

// EffectiveRuntime preserves legacy defaults for programmatically constructed configs too.
func (c *Config) EffectiveRuntime() RuntimeConfig {
	var r RuntimeConfig
	if c != nil {
		r = c.Runtime
	}
	defaults := []struct {
		value    *string
		fallback string
	}{
		{&r.Tools.TreeSitter, "tree-sitter"}, {&r.Tools.Git, "git"}, {&r.Tools.NPM, "npm"},
		{&r.Tools.EMCC, "emcc"}, {&r.Tools.Docker, "docker"}, {&r.Tools.Podman, "podman"}, {&r.Tools.Zig, "zig"},
		{&r.WASM.Backend, "auto"}, {&r.WASM.Image, "emscripten/emsdk"}, {&r.Compiler.HostCC, "cc"},
		{&r.Node.Install, "auto"}, {&r.Cleanup.AfterMove, "remove"},
	}
	for _, d := range defaults {
		if *d.value == "" {
			*d.value = d.fallback
		}
	}
	if r.Compiler.AllowFallback == nil {
		r.Compiler.AllowFallback = boolPointer(true)
	}
	if r.Generation.ABIFallback == nil {
		r.Generation.ABIFallback = boolPointer(true)
	}
	if r.Node.RetryInstall == nil {
		r.Node.RetryInstall = boolPointer(true)
	}
	if r.Publication.Manifest == nil {
		r.Publication.Manifest = boolPointer(true)
	}
	if r.Node.CIArgs == nil {
		r.Node.CIArgs = []string{"ci", "--ignore-scripts"}
	}
	if r.Node.InstallArgs == nil {
		r.Node.InstallArgs = []string{"install", "--ignore-scripts", "--no-audit", "--no-fund"}
	}
	if r.Compiler.Candidates == nil {
		r.Compiler.Candidates = map[string][]CompilerPair{
			"linux/amd64":   {{CC: "gcc", CXX: "g++"}, {CC: "clang", CXX: "clang++"}},
			"windows/amd64": {{CC: "x86_64-w64-mingw32-gcc", CXX: "x86_64-w64-mingw32-g++"}},
			"macos/amd64":   {{CC: "o64-clang", CXX: "o64-clang++"}, {CC: "x86_64-apple-darwin-clang", CXX: "x86_64-apple-darwin-clang++"}},
			"macos/arm64":   {{CC: "oa64-clang", CXX: "oa64-clang++"}, {CC: "aarch64-apple-darwin-clang", CXX: "aarch64-apple-darwin-clang++"}},
		}
	}
	if r.SupportedTargets == nil {
		r.SupportedTargets = map[string][]string{"linux": {"amd64"}, "windows": {"amd64"}, "macos": {"amd64", "arm64"}}
	}
	if r.Wizard.Platforms == nil {
		r.Wizard.Platforms = []string{"host"}
	}
	if r.Wizard.Artifacts == nil {
		r.Wizard.Artifacts = []string{"manifest", "scm"}
	}
	p := &r.Cleanup.Prune
	if p.Directories == nil {
		p.Directories = []string{".git", ".github", ".vscode", ".idea", "node_modules", "bindings", "test", "tests", "corpus", "examples", "target", "docs", "doc", "script", "scripts", "tools", "benchmark", "benchmarks", "_layouts", "build/Release"}
	}
	if p.PreserveDirectories == nil {
		p.PreserveDirectories = []string{"src", "queries"}
	}
	if p.PreserveFiles == nil {
		p.PreserveFiles = []string{"tree-sitter.json", "package.json", "go.mod", "LICENSE*", "LICENCE*", "NOTICE*", "grammar.js", "node-types.json", ".source-provenance.json"}
	}
	if p.ProtectedFiles == nil {
		p.ProtectedFiles = []string{"*.o", "*.obj", "*.a", "*.lib"}
	}
	if p.Files == nil {
		p.Files = []string{"*.o", "*.obj", "*.a", "*.lib", "*.lock", "*.resolved", "go.sum", "package-lock.json", "pnpm-lock.yaml", ".gitignore", ".gitattributes", ".gitmodules", ".editorconfig", ".clang-format", ".git-blame-ignore-revs", ".envrc", ".npmignore", "eslint.config.*", ".eslintrc*", ".prettierrc*", ".prettierignore", ".tsqueryrc.json", "README*", "CHANGELOG*", "CONTRIBUTING*", "SECURITY*", "FUNDING*", "ISSUE_TEMPLATE*", "CODE_OF_CONDUCT*", "package.swift", "binding.gyp", "setup.py", "pyproject.toml", "cargo.toml", "cmakelists.txt", "makefile", "justfile", "build.zig", "build.zig.zon", "gemfile"}
	}
	if r.Queries.Rewrites == nil {
		r.Queries.Rewrites = []QueryRewrite{{Language: "ocaml-interface", File: "highlights.scm", Find: " (shebang)", Replace: ""}}
	}
	if r.SyncModules == nil {
		r.SyncModules = configsync.DefaultModuleMapping()
	}
	return r
}

func (r RuntimeConfig) Validate() error {
	for name, values := range map[string][]string{"wasm.backend": {"auto", "emcc", "docker", "podman"}, "node.install": {"auto", "never", "always"}, "cleanup.after_move": {"keep", "prune", "remove"}} {
		value := r.WASM.Backend
		if name == "node.install" {
			value = r.Node.Install
		}
		if name == "cleanup.after_move" {
			value = r.Cleanup.AfterMove
		}
		if !containsString(values, value) {
			return fmt.Errorf("runtime.%s: invalid value %q", name, value)
		}
	}
	if strings.TrimSpace(r.WASM.Image) == "" {
		return errors.New("runtime.wasm.image cannot be blank")
	}
	for _, tool := range []string{r.Tools.TreeSitter, r.Tools.Git, r.Tools.NPM, r.Tools.EMCC, r.Tools.Docker, r.Tools.Podman, r.Tools.Zig, r.Compiler.HostCC} {
		if strings.TrimSpace(tool) == "" || strings.ContainsAny(tool, "\n\r\x00") {
			return errors.New("runtime tools must contain executable names or paths")
		}
	}
	for osName, arches := range r.SupportedTargets {
		if !containsString([]string{"linux", "windows", "macos"}, osName) || len(arches) == 0 {
			return fmt.Errorf("runtime.supported_targets: invalid OS %q", osName)
		}
		for _, arch := range arches {
			if !containsString([]string{"amd64", "arm64"}, arch) {
				return fmt.Errorf("runtime.supported_targets: unsupported architecture %q", arch)
			}
		}
	}
	for key, pairs := range r.Compiler.Candidates {
		parts := strings.Split(key, "/")
		if len(parts) != 2 || !containsString([]string{"linux", "windows", "macos"}, parts[0]) || !containsString([]string{"amd64", "arm64"}, parts[1]) {
			return fmt.Errorf("runtime.compiler.candidates: unsupported target %q", key)
		}
		for _, pair := range pairs {
			if strings.TrimSpace(pair.CC) == "" {
				return fmt.Errorf("runtime.compiler.candidates[%q]: cc is required", key)
			}
		}
	}
	for _, platform := range r.Wizard.Platforms {
		if platform == "host" || platform == "wasm" {
			continue
		}
		parts := strings.Split(platform, "/")
		if len(parts) != 2 || !containsString(r.SupportedTargets[parts[0]], parts[1]) {
			return fmt.Errorf("runtime.wizard.platforms: invalid platform %q", platform)
		}
	}
	for _, artifact := range r.Wizard.Artifacts {
		if !containsString([]string{"manifest", "scm", "compact", "source", "js", "wasm"}, artifact) {
			return fmt.Errorf("runtime.wizard.artifacts: invalid artifact %q", artifact)
		}
	}
	p := r.Cleanup.Prune
	for _, patterns := range [][]string{p.Directories, p.Files, p.PreserveFiles, p.PreserveDirectories, p.ProtectedFiles} {
		for _, pattern := range patterns {
			if pattern == "" || filepath.IsAbs(pattern) || strings.Contains(pattern, "\\") {
				return fmt.Errorf("runtime.cleanup.prune: invalid pattern %q", pattern)
			}
			for _, part := range strings.Split(pattern, "/") {
				if part == ".." || part == "." {
					return fmt.Errorf("runtime.cleanup.prune: unsafe pattern %q", pattern)
				}
			}
			if _, err := filepath.Match(pattern, ""); err != nil {
				return fmt.Errorf("runtime.cleanup.prune: %w", err)
			}
		}
	}
	if r.Node.Install != "never" && (len(r.Node.CIArgs) == 0 || len(r.Node.InstallArgs) == 0) {
		return errors.New("runtime.node: install argument lists cannot be empty")
	}
	// Dependency lifecycle scripts stay disabled even when install arguments are customized.
	for _, args := range [][]string{r.Node.CIArgs, r.Node.InstallArgs} {
		if r.Node.Install == "never" {
			break
		}
		if !containsString(args, "--ignore-scripts") {
			return errors.New("runtime.node: install arguments must include --ignore-scripts")
		}
		for _, arg := range args {
			if strings.HasPrefix(arg, "--ignore-scripts=") || strings.HasPrefix(arg, "--no-ignore-scripts") {
				return errors.New("runtime.node: lifecycle scripts must remain disabled")
			}
		}
	}
	for lang, paths := range r.Queries.Paths {
		if !languageNamePattern.MatchString(lang) {
			return fmt.Errorf("runtime.queries.paths: invalid language %q", lang)
		}
		for _, path := range paths {
			if path == "." || !filepath.IsLocal(path) {
				return fmt.Errorf("runtime.queries.paths: unsafe path %q", path)
			}
		}
	}
	for _, rewrite := range r.Queries.Rewrites {
		if !languageNamePattern.MatchString(rewrite.Language) || filepath.Base(rewrite.File) != rewrite.File || !strings.HasSuffix(rewrite.File, ".scm") || rewrite.Find == "" {
			return errors.New("runtime.queries.rewrites: language, .scm filename, and nonempty find are required")
		}
	}
	return nil
}
func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func HostPlatform() string {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}
	return osName + "/" + runtime.GOARCH
}

// ValidateBuildPaths prevents build cleanup or publication from destroying the other tree.
func (c *Config) ValidateBuildPaths() error {
	build, err := canonicalPath(c.BuildDir)
	if err != nil {
		return err
	}
	output, err := canonicalPath(c.Output.GrammarBase)
	if err != nil {
		return err
	}
	if containsPath(build, output) || containsPath(output, build) {
		return errors.New("build_dir and output.grammar_base must not overlap")
	}
	return nil
}

// ValidateWorkspacePath confines destructive cleanup to the configured workspace.
func (c *Config) ValidateWorkspacePath(path string) error {
	workspace, err := canonicalPath(c.BuildDir)
	if err != nil {
		return err
	}
	root, err := canonicalPath(path)
	if err != nil {
		return err
	}
	if filepath.Dir(workspace) == workspace || root == workspace || !containsPath(workspace, root) {
		return fmt.Errorf("cleanup path %q must be strictly inside build_dir %q", path, c.BuildDir)
	}
	if c.Output.GrammarBase != "" {
		output, err := canonicalPath(c.Output.GrammarBase)
		if err != nil {
			return err
		}
		if containsPath(root, output) || containsPath(output, root) {
			return fmt.Errorf("cleanup path %q overlaps output.grammar_base", path)
		}
	}
	return nil
}
