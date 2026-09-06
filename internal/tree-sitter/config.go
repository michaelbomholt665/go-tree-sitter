package treesitter

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var languageNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type Config struct {
	Version              string                 `yaml:"version"`
	BuildDir             string                 `yaml:"build_dir"`
	TreeSitterCLIVersion string                 `yaml:"tree_sitter_cli_version"`
	GenerateABI          int                    `yaml:"generate_abi"`
	ABIRange             *ABIRange              `yaml:"abi_range,omitempty"`
	BuildFlags           []string               `yaml:"build_flags"`
	OSTarget             string                 `yaml:"OS_TARGET"`
	Targets              map[string]BuildTarget `yaml:"targets"`
	ABIVersions          map[string]ABIRange    `yaml:"abi_versions,omitempty"` // Deprecated: never used for artifact ABI metadata.
	Languages            []Language             `yaml:"languages"`
	Output               Output                 `yaml:"output"`
	Prune                bool                   `yaml:"prune,omitempty"`
}

type BuildTarget struct {
	OS              string `yaml:"os"`
	Arch            string `yaml:"arch"`
	Triple          string `yaml:"triple"`
	Compiler        string `yaml:"compiler"`
	CXX             string `yaml:"cxx"`
	CompilerVersion string `yaml:"compiler_version"`
}

type ABIRange struct {
	Min int `yaml:"min" json:"min_version"`
	Max int `yaml:"max" json:"max_version"`
}

type Language struct {
	Name              string `yaml:"name"`
	Grammar           string `yaml:"grammar"`
	Constructor       string `yaml:"constructor"`
	Version           string `yaml:"version"`
	GenerateABI       *int   `yaml:"generate_abi,omitempty"`
	TreeSitterVersion string `yaml:"tree_sitter_version,omitempty"` // Deprecated compatibility input; never emitted as measured provenance.
	Repository        string `yaml:"repository"`
	Revision          string `yaml:"revision"`
	SourceSubdir      string `yaml:"source_subdir"`
	Sample            string `yaml:"sample"`
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
	return c.validateLanguages()
}

func (c *Config) validateRequiredFields() error {
	if strings.TrimSpace(c.Version) == "" {
		return errors.New("version is required")
	}
	if c.Version == "2.0" {
		if strings.TrimSpace(c.TreeSitterCLIVersion) == "" {
			return errors.New("tree_sitter_cli_version is required for config version 2.0")
		}
		if c.ABIRange != nil {
			if c.ABIRange.Min <= 0 || c.ABIRange.Max <= 0 {
				return errors.New("abi_range must specify positive min and max versions")
			}
			if c.ABIRange.Min > c.ABIRange.Max {
				return errors.New("abi_range min must be <= max")
			}
		}
		if c.GenerateABI > 0 && c.ABIRange != nil {
			if c.GenerateABI < c.ABIRange.Min || c.GenerateABI > c.ABIRange.Max {
				return fmt.Errorf("generate_abi %d must be within supported abi_range %d-%d for config version 2.0", c.GenerateABI, c.ABIRange.Min, c.ABIRange.Max)
			}
		}
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
	if err := c.validateABIVersions(); err != nil {
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
		target.Triple = strings.TrimSpace(target.Triple)
		target.Compiler = strings.TrimSpace(target.Compiler)
		target.CXX = strings.TrimSpace(target.CXX)
		target.CompilerVersion = strings.TrimSpace(target.CompilerVersion)

		if target.OS == "" {
			return fmt.Errorf("targets[%q].os is required", name)
		}
		if target.Arch == "" {
			return fmt.Errorf("targets[%q].arch is required", name)
		}
		if c.Version == "2.0" && (target.Triple == "" || target.Compiler == "" || target.CompilerVersion == "") {
			return fmt.Errorf("targets[%q] must pin triple, compiler, and compiler_version", name)
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
	for rangeExpr, abi := range c.ABIVersions {
		if strings.TrimSpace(rangeExpr) == "" {
			return errors.New("abi_versions keys must not be empty")
		}
		if _, err := parseRangeExpression(rangeExpr); err != nil {
			return fmt.Errorf("abi_versions[%q]: invalid range expression: %w", rangeExpr, err)
		}
		if abi.Min <= 0 || abi.Max <= 0 {
			return fmt.Errorf("abi_versions[%q] must use positive min/max versions", rangeExpr)
		}
		if abi.Min > abi.Max {
			return fmt.Errorf("abi_versions[%q] min must be <= max", rangeExpr)
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
	lang.Grammar = strings.TrimSpace(lang.Grammar)
	if lang.Grammar == "" {
		lang.Grammar = strings.NewReplacer(".", "", "-", "_").Replace(lang.Name)
	}
	lang.Constructor = strings.TrimSpace(lang.Constructor)
	if lang.Constructor == "" {
		lang.Constructor = "tree_sitter_" + lang.Grammar
	}
	lang.Version = strings.TrimSpace(lang.Version)
	lang.TreeSitterVersion = strings.TrimSpace(lang.TreeSitterVersion)
	lang.Repository = strings.TrimSpace(lang.Repository)
	lang.Revision = strings.TrimSpace(lang.Revision)
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
	if !regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(lang.Grammar) {
		return fmt.Errorf("languages[%d].grammar %q must match an exported constructor suffix", index, lang.Grammar)
	}
	if lang.Constructor != "tree_sitter_"+lang.Grammar {
		return fmt.Errorf("languages[%d].constructor %q must equal tree_sitter_<grammar> (%q)", index, lang.Constructor, "tree_sitter_"+lang.Grammar)
	}

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
	if lang.Revision != "" && !regexp.MustCompile(`^[0-9a-fA-F]{40}$`).MatchString(lang.Revision) {
		return fmt.Errorf("languages[%d].revision must be a full 40-character commit", index)
	}
	if lang.GenerateABI != nil {
		if *lang.GenerateABI <= 0 {
			return fmt.Errorf("languages[%d].generate_abi must be a positive integer", index)
		}
		minABI := 13
		maxABI := 15
		if c.ABIRange != nil {
			minABI = c.ABIRange.Min
			maxABI = c.ABIRange.Max
		}
		if *lang.GenerateABI < minABI || *lang.GenerateABI > maxABI {
			return fmt.Errorf("languages[%d].generate_abi %d must be within supported range %d-%d", index, *lang.GenerateABI, minABI, maxABI)
		}
	}
	if c.Version == "2.0" {
		if lang.Revision == "" {
			return fmt.Errorf("languages[%d].revision is required for config version 2.0", index)
		}
		if strings.TrimSpace(lang.Sample) == "" {
			return fmt.Errorf("languages[%d].sample is required for config version 2.0", index)
		}
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
	ver, err := parseSemVersion(version)
	if err != nil {
		return ABIRange{}, fmt.Errorf("invalid tree-sitter version %q: %w", version, err)
	}

	for rangeExpr, abi := range c.ABIVersions {
		constraints, err := parseRangeExpression(rangeExpr)
		if err != nil {
			continue // already validated at load time
		}
		matched := true
		for _, constraint := range constraints {
			if !constraint.matches(ver.parts) {
				matched = false
				break
			}
		}
		if matched {
			return abi, nil
		}
	}

	return ABIRange{}, fmt.Errorf("tree-sitter version %q has no ABI mapping", version)
}

// -- semver range helpers --

// semVersion holds a parsed major.minor[.patch] version.
type semVersion struct {
	parts [3]int
	n     int // number of significant parts (2 or 3)
}

// parseSemVersion parses a "major.minor" or "major.minor.patch" string.
func parseSemVersion(s string) (semVersion, error) {
	s = strings.TrimSpace(s)
	fields := strings.Split(s, ".")
	if len(fields) < 2 || len(fields) > 3 {
		return semVersion{}, fmt.Errorf("version %q must have 2 or 3 dot-separated parts", s)
	}
	var sv semVersion
	sv.n = len(fields)
	for i, f := range fields {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil || n < 0 {
			return semVersion{}, fmt.Errorf("version %q: part %q is not a non-negative integer", s, f)
		}
		sv.parts[i] = n
	}
	return sv, nil
}

// compareToConstraint compares ver against the constraint version using only the
// constraint's significant parts (2 or 3). Returns -1, 0, or 1 (ver vs constraint).
func (sv semVersion) compareToConstraint(ver [3]int) int {
	for i := 0; i < sv.n; i++ {
		switch {
		case ver[i] < sv.parts[i]:
			return -1
		case ver[i] > sv.parts[i]:
			return 1
		}
	}
	return 0
}

type versionConstraint struct {
	op  string // one of: >=, <=, >, <, =
	ver semVersion
}

// matches reports whether ver satisfies this constraint.
func (vc versionConstraint) matches(ver [3]int) bool {
	cmp := vc.ver.compareToConstraint(ver)
	switch vc.op {
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case "=":
		return cmp == 0
	}
	return false
}

// parseVersionConstraint parses a single constraint like ">=0.25" or "<=0.24".
func parseVersionConstraint(s string) (versionConstraint, error) {
	s = strings.TrimSpace(s)
	var op, verStr string
	for _, prefix := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(s, prefix) {
			op = prefix
			verStr = strings.TrimSpace(s[len(prefix):])
			break
		}
	}
	if op == "" {
		return versionConstraint{}, fmt.Errorf("constraint %q must start with an operator (>=, <=, >, <, =)", s)
	}
	ver, err := parseSemVersion(verStr)
	if err != nil {
		return versionConstraint{}, fmt.Errorf("constraint %q: %w", s, err)
	}
	return versionConstraint{op: op, ver: ver}, nil
}

// parseRangeExpression parses a comma-separated list of constraints,
// e.g. ">=0.20.3, <=0.24" or ">=0.25".
func parseRangeExpression(expr string) ([]versionConstraint, error) {
	parts := strings.Split(expr, ",")
	constraints := make([]versionConstraint, 0, len(parts))
	for _, part := range parts {
		c, err := parseVersionConstraint(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("range %q: %w", expr, err)
		}
		constraints = append(constraints, c)
	}
	if len(constraints) == 0 {
		return nil, fmt.Errorf("range %q: no constraints specified", expr)
	}
	return constraints, nil
}

func applyDefaults(cfg *Config) {
	if cfg.ABIRange == nil {
		cfg.ABIRange = &ABIRange{Min: 13, Max: 15}
	}
	if cfg.GenerateABI == 0 {
		cfg.GenerateABI = 15
	}
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
