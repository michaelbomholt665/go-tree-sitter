package treesitter

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

var languageNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type Config struct {
	Version     string                 `yaml:"version"`
	BuildDir    string                 `yaml:"build_dir"`
	OSTarget    string                 `yaml:"OS_TARGET"`
	Targets     map[string]BuildTarget `yaml:"targets"`
	ABIVersions map[string]ABIRange    `yaml:"abi_versions"`
	Languages   []Language             `yaml:"languages"`
	Output      Output                 `yaml:"output"`
}

type BuildTarget struct {
	OS   string `yaml:"os"`
	Arch string `yaml:"arch"`
}

type ABIRange struct {
	Min int `yaml:"min" json:"min_version"`
	Max int `yaml:"max" json:"max_version"`
}

type Language struct {
	Name              string `yaml:"name"`
	Version           string `yaml:"version"`
	TreeSitterVersion string `yaml:"tree_sitter_version"`
	Repository        string `yaml:"repository"`
	SourceSubdir      string `yaml:"source_subdir"`
	Note              string `yaml:"note,omitempty"`
}

type Output struct {
	GrammarBase        string   `yaml:"grammar_base"`
	GrammarBuild       string   `yaml:"grammar_build"`
	GrammarCompile     string   `yaml:"grammar_compile"`
	GenerateManifest   bool     `yaml:"generate_manifest"`
	DefaultMoveMode    string   `yaml:"default_move_mode"`
	SupportedMoveModes []string `yaml:"supported_move_modes"`
}

func LoadConfig(filePath string) (*Config, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", filePath, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", filePath, err)
	}

	applyDefaults(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %q: %w", filePath, err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}

	if err := c.validateRequiredFields(); err != nil {
		return err
	}
	if err := c.validateABIVersions(); err != nil {
		return err
	}
	return c.validateLanguages()
}

func (c *Config) validateRequiredFields() error {
	if strings.TrimSpace(c.Version) == "" {
		return errors.New("version is required")
	}

	c.BuildDir = cleanConfigPath(c.BuildDir)
	if c.BuildDir == "." {
		return errors.New("build_dir is required")
	}

	if len(c.Languages) == 0 {
		return errors.New("at least one language is required")
	}
	if err := c.validateTargets(); err != nil {
		return err
	}

	c.Output.GrammarBase = cleanConfigPath(c.Output.GrammarBase)
	if c.Output.GrammarBase == "." {
		return errors.New("output.grammar_base is required")
	}
	c.Output.GrammarBuild = normalizeTemplatePath(c.Output.GrammarBuild)
	if c.Output.GrammarBuild == "." {
		return errors.New("output.grammar_build is required")
	}
	c.Output.GrammarCompile = normalizeTemplatePath(c.Output.GrammarCompile)
	if c.Output.GrammarCompile == "." {
		return errors.New("output.grammar_compile is required")
	}
	if err := c.validateMoveModeConfig(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateTargets() error {
	c.OSTarget = strings.TrimSpace(c.OSTarget)
	if c.OSTarget == "" && len(c.Targets) > 0 {
		return errors.New("OS_TARGET is required when targets are configured")
	}
	if c.OSTarget != "" && len(c.Targets) == 0 {
		return errors.New("targets are required when OS_TARGET is configured")
	}

	for name := range c.Targets {
		target := c.Targets[name]
		targetName := strings.TrimSpace(name)
		if targetName == "" {
			return errors.New("targets keys must not be empty")
		}
		target.OS = strings.TrimSpace(target.OS)
		target.Arch = strings.TrimSpace(target.Arch)

		if target.OS == "" {
			return fmt.Errorf("targets[%q].os is required", name)
		}
		if target.Arch == "" {
			return fmt.Errorf("targets[%q].arch is required", name)
		}
		c.Targets[name] = target
	}

	if c.OSTarget != "" {
		if _, ok := c.Targets[c.OSTarget]; !ok {
			return fmt.Errorf("OS_TARGET %q is not defined in targets", c.OSTarget)
		}
	}

	return nil
}

func (c *Config) validateMoveModeConfig() error {
	mode, err := ParseConfiguredMoveMode(c.Output.DefaultMoveMode)
	if err != nil {
		return fmt.Errorf("output.default_move_mode: %w", err)
	}
	c.Output.DefaultMoveMode = "--" + string(mode)

	if len(c.Output.SupportedMoveModes) == 0 {
		return nil
	}

	seen := map[MoveMode]struct{}{}
	for index, rawMode := range c.Output.SupportedMoveModes {
		mode, err := ParseConfiguredMoveMode(rawMode)
		if err != nil {
			return fmt.Errorf("output.supported_move_modes[%d]: %w", index, err)
		}
		c.Output.SupportedMoveModes[index] = "--" + string(mode)
		seen[mode] = struct{}{}
	}

	if _, ok := seen[MoveModeJSON]; !ok {
		return errors.New("output.supported_move_modes must include --json")
	}
	if _, ok := seen[MoveModeSCM]; !ok {
		return errors.New("output.supported_move_modes must include --scm")
	}
	if _, ok := seen[MoveModeBoth]; !ok {
		return errors.New("output.supported_move_modes must include --both")
	}
	if _, ok := seen[mode]; !ok {
		return errors.New("output.default_move_mode must be included in output.supported_move_modes")
	}
	return nil
}

func (c *Config) validateABIVersions() error {
	for version, abi := range c.ABIVersions {
		if strings.TrimSpace(version) == "" {
			return errors.New("abi_versions keys must not be empty")
		}
		if abi.Min <= 0 || abi.Max <= 0 {
			return fmt.Errorf("abi_versions[%q] must use positive min/max versions", version)
		}
		if abi.Min > abi.Max {
			return fmt.Errorf("abi_versions[%q] min must be <= max", version)
		}
	}

	return nil
}

func (c *Config) validateLanguages() error {
	seen := map[string]struct{}{}
	for index := range c.Languages {
		lang := &c.Languages[index]
		normalizeLanguage(lang)
		if err := c.validateLanguage(index, lang, seen); err != nil {
			return err
		}
	}

	return nil
}

func normalizeLanguage(lang *Language) {
	lang.Name = strings.TrimSpace(lang.Name)
	lang.Version = strings.TrimSpace(lang.Version)
	lang.TreeSitterVersion = strings.TrimSpace(lang.TreeSitterVersion)
	lang.Repository = strings.TrimSpace(lang.Repository)
	lang.SourceSubdir = normalizeSubdir(lang.SourceSubdir)
}

func (c *Config) validateLanguage(index int, lang *Language, seen map[string]struct{}) error {
	if lang.Name == "" {
		return fmt.Errorf("languages[%d].name is required", index)
	}
	if !languageNamePattern.MatchString(lang.Name) {
		return fmt.Errorf("languages[%d].name %q contains unsupported characters", index, lang.Name)
	}
	if _, exists := seen[lang.Name]; exists {
		return fmt.Errorf("duplicate language name %q", lang.Name)
	}
	seen[lang.Name] = struct{}{}

	if lang.Version == "" {
		return fmt.Errorf("languages[%d].version is required", index)
	}
	if lang.Repository == "" {
		return fmt.Errorf("languages[%d].repository is required", index)
	}
	if err := validateRepositoryURL(lang.Repository); err != nil {
		return fmt.Errorf("languages[%d].repository: %w", index, err)
	}
	if lang.SourceSubdir != "" && (filepath.IsAbs(lang.SourceSubdir) || strings.HasPrefix(lang.SourceSubdir, "..")) {
		return fmt.Errorf("languages[%d].source_subdir must stay within the cloned repository", index)
	}
	if lang.TreeSitterVersion == "" {
		return fmt.Errorf("languages[%d].tree_sitter_version is required", index)
	}
	if _, ok := c.ABIVersions[lang.TreeSitterVersion]; !ok {
		return fmt.Errorf("languages[%d].tree_sitter_version %q has no ABI mapping", index, lang.TreeSitterVersion)
	}
	return nil
}

func (c *Config) ResolveLanguages(name string) ([]Language, error) {
	if name == "" {
		result := slices.Clone(c.Languages)
		slices.SortFunc(result, func(a, b Language) int {
			return strings.Compare(a.Name, b.Name)
		})
		return result, nil
	}

	for _, lang := range c.Languages {
		if lang.Name == name {
			return []Language{lang}, nil
		}
	}

	return nil, fmt.Errorf("language %q is not defined in the config", name)
}

func (c *Config) ABIFor(version string) (ABIRange, error) {
	abi, ok := c.ABIVersions[version]
	if !ok {
		return ABIRange{}, fmt.Errorf("tree-sitter version %q has no ABI mapping", version)
	}
	return abi, nil
}

func applyDefaults(cfg *Config) {
	if strings.TrimSpace(cfg.BuildDir) == "" {
		cfg.BuildDir = "build"
	}
	if strings.TrimSpace(cfg.Output.GrammarBase) == "" {
		cfg.Output.GrammarBase = filepath.Join("data", "tree-sitter", "grammar")
	}
	if strings.TrimSpace(cfg.Output.GrammarBuild) == "" {
		cfg.Output.GrammarBuild = filepath.Join(cfg.BuildDir, "{lang}", "{arch}")
	}
	if strings.TrimSpace(cfg.Output.GrammarCompile) == "" {
		cfg.Output.GrammarCompile = filepath.Join(cfg.BuildDir, "{lang}", "{arch}")
	}
	if strings.TrimSpace(cfg.Output.DefaultMoveMode) == "" {
		cfg.Output.DefaultMoveMode = "--both"
	}
	if len(cfg.Output.SupportedMoveModes) == 0 {
		cfg.Output.SupportedMoveModes = []string{"--json", "--scm", "--both"}
	}
}

func cleanConfigPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "."
	}
	return filepath.Clean(value)
}

func normalizeTemplatePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "."
	}
	return filepath.ToSlash(filepath.Clean(value))
}

func normalizeSubdir(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(value))
}

func validateRepositoryURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("must be an absolute repository URL")
	}
	return nil
}
