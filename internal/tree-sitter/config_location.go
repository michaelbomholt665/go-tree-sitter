package treesitter

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	bundledconfig "github.com/michaelbomholt665/go-tree-sitter"
	"gopkg.in/yaml.v3"
)

// ResolveConfigPath selects development configuration only through the current
// directory's .env marker. Otherwise it initializes and selects the OS user file.
// It does not load .env values into the process environment or search parent directories.
func ResolveConfigPath() (string, error) {
	return resolveConfigPath(true)
}

func resolveConfigPath(initialize bool) (string, error) {
	development, err := developmentMarker()
	if err != nil {
		return "", err
	}
	if development {
		path, err := filepath.Abs(DefaultConfigPath)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("development configuration %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("development configuration %s is not a regular file", path)
		}
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user configuration: %w", err)
	}
	path := filepath.Join(dir, "ts-build", DefaultConfigPath)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if !initialize {
		return path, nil
	}
	if err := initializeUserConfig(path); err != nil {
		return "", err
	}
	return path, nil
}

func developmentMarker() (bool, error) {
	file, err := os.Open(".env")
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read development marker: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	mode := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(key) != "TS_BUILD_ENV" {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
			quote := value[0]
			end := strings.IndexByte(value[1:], quote)
			if end < 0 {
				return false, errors.New(".env: unterminated TS_BUILD_ENV value")
			}
			tail := strings.TrimSpace(value[end+2:])
			if tail != "" && !strings.HasPrefix(tail, "#") {
				return false, errors.New(".env: invalid TS_BUILD_ENV value")
			}
			value = value[1 : end+1]
		} else if i := strings.Index(value, " #"); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}
		mode = value
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("read development marker: %w", err)
	}
	switch mode {
	case "", "PRODUCTION":
		return false, nil
	case "DEVELOPMENT":
		return true, nil
	default:
		return false, fmt.Errorf(".env: TS_BUILD_ENV must be DEVELOPMENT or PRODUCTION, got %q", mode)
	}
}

// defaultUserConfig prepares initialization content without creating files.
func defaultUserConfig() ([]byte, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("locate build cache: %w", err)
	}
	data, err := userDataDir()
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(bundledconfig.YAML), &doc); err != nil {
		return nil, fmt.Errorf("decode embedded configuration: %w", err)
	}
	build := filepath.Join(cache, "ts-build", "build")
	setYAMLPath(doc.Content[0], "build_dir", build)
	output := yamlMappingValue(doc.Content[0], "output")
	if output == nil {
		return nil, errors.New("embedded configuration has no output mapping")
	}
	setYAMLPath(output, "grammar_base", filepath.Join(data, "ts-build", "grammar"))
	setYAMLPath(output, "grammar_build", filepath.Join(build, "{lang}", "{arch}"))
	setYAMLPath(output, "grammar_compile", filepath.Join(build, "{lang}", "{arch}"))
	return yaml.Marshal(&doc)
}

// initializeUserConfig never replaces an existing file.
func initializeUserConfig(path string) error {
	content, err := defaultUserConfig()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("create user configuration: %w", err)
	}
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("write user configuration: %w", err)
	}
	return nil
}

func yamlMappingValue(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func setYAMLPath(node *yaml.Node, key, value string) {
	if entry := yamlMappingValue(node, key); entry != nil {
		entry.Value = value
	}
}

func userDataDir() (string, error) {
	if runtime.GOOS == "windows" {
		if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
			return dir, nil
		}
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return os.UserConfigDir()
	}
	if dir := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(dir) {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate grammar data directory: %w", err)
	}
	return filepath.Join(home, ".local", "share"), nil
}

// loadRuntimeConfig anchors path settings to the selected configuration, not the shell.
func loadRuntimeConfig(path string) (*Config, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	return anchorConfigPaths(cfg, path)
}

func anchorConfigPaths(cfg *Config, path string) (*Config, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	base := filepath.Dir(absolute)
	anchor := func(setting *string) {
		if !filepath.IsAbs(*setting) {
			*setting = filepath.Join(base, *setting)
		}
	}
	anchor(&cfg.BuildDir)
	for _, setting := range []*string{&cfg.Output.GrammarBase, &cfg.Output.GrammarBuild, &cfg.Output.GrammarCompile} {
		*setting = strings.ReplaceAll(*setting, "{build_dir}", cfg.BuildDir)
		anchor(setting)
	}
	anchorExecutable := func(setting *string) {
		if strings.ContainsAny(*setting, "/\\") && !filepath.IsAbs(*setting) {
			anchor(setting)
		}
	}
	tools := &cfg.Runtime.Tools
	for _, setting := range []*string{&tools.TreeSitter, &tools.Git, &tools.NPM, &tools.EMCC, &tools.Docker, &tools.Podman, &tools.Zig, &cfg.Runtime.Compiler.HostCC} {
		anchorExecutable(setting)
	}
	for key, target := range cfg.Targets {
		anchorExecutable(&target.Compiler)
		anchorExecutable(&target.CXX)
		cfg.Targets[key] = target
	}
	for key, pairs := range cfg.Runtime.Compiler.Candidates {
		for i := range pairs {
			anchorExecutable(&pairs[i].CC)
			anchorExecutable(&pairs[i].CXX)
		}
		cfg.Runtime.Compiler.Candidates[key] = pairs
	}
	return cfg, nil
}
