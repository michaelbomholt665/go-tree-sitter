package build

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type target struct {
	GOOS     string
	Platform string
	Arch     string
	Ext      string
}

var supportedTargets = map[string]map[string]struct{}{
	"linux": {
		"amd64": {},
	},
	"windows": {
		"amd64": {},
	},
	"darwin": {
		"amd64": {},
		"arm64": {},
	},
}

func resolveTarget(goos, arch string) (target, error) {
	if goos == "" {
		goos = runtime.GOOS
	}
	if arch == "" {
		arch = runtime.GOARCH
	}

	goos = normalizeGOOS(goos)
	arch = strings.ToLower(strings.TrimSpace(arch))

	arches, ok := supportedTargets[goos]
	if !ok {
		return target{}, fmt.Errorf("unsupported target OS %q", goos)
	}
	if _, ok := arches[arch]; !ok {
		return target{}, fmt.Errorf("unsupported architecture %q for target OS %q", arch, goos)
	}

	return target{
		GOOS:     goos,
		Platform: platformName(goos),
		Arch:     arch,
		Ext:      sharedLibraryExt(goos),
	}, nil
}

func resolveConfiguredTarget(cfg *_jsii.Config, goos, arch string) (target, error) {
	if goos != "" || arch != "" {
		return resolveTarget(goos, arch)
	}

	if cfg.OSTarget == "" {
		return resolveTarget("", "")
	}

	configuredTarget, ok := cfg.Targets[cfg.OSTarget]
	if !ok {
		return target{}, fmt.Errorf("OS_TARGET %q is not defined in targets", cfg.OSTarget)
	}

	resolved, err := resolveTarget(configuredTarget.OS, configuredTarget.Arch)
	if err != nil {
		return target{}, fmt.Errorf("targets[%q]: %w", cfg.OSTarget, err)
	}
	return resolved, nil
}

func normalizeGOOS(goos string) string {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "darwin", "macos", "mac":
		return "darwin"
	case "linux":
		return "linux"
	case "windows", "win":
		return "windows"
	default:
		return strings.ToLower(strings.TrimSpace(goos))
	}
}

func platformName(goos string) string {
	if goos == "darwin" {
		return "macos"
	}
	return goos
}

func sharedLibraryExt(goos string) string {
	switch goos {
	case "windows":
		return ".dll"
	case "darwin":
		return ".dylib"
	default:
		return ".so"
	}
}

func buildRootDir(cfg *_jsii.Config, lang _jsii.Language) string {
	target, err := resolveConfiguredTarget(cfg, "", "")
	if err != nil {
		return filepath.Join(cfg.BuildDir, lang.Name)
	}
	template := cfg.Output.GrammarBuild
	if strings.TrimSpace(template) == "" {
		return filepath.Join(cfg.BuildDir, lang.Name)
	}
	return renderPathTemplate(template, lang, target)
}

func sourceDir(cfg *_jsii.Config, lang _jsii.Language) string {
	if lang.SourceSubdir == "" {
		return buildRootDir(cfg, lang)
	}
	return filepath.Join(buildRootDir(cfg, lang), filepath.FromSlash(lang.SourceSubdir))
}

func binaryDir(cfg *_jsii.Config, lang _jsii.Language) string {
	target, err := resolveConfiguredTarget(cfg, "", "")
	if err != nil {
		return filepath.Join(buildRootDir(cfg, lang), "bin")
	}
	template := cfg.Output.GrammarCompile
	if strings.TrimSpace(template) == "" {
		return filepath.Join(buildRootDir(cfg, lang), "bin")
	}
	return renderPathTemplate(template, lang, target)
}

func binaryFilename(lang _jsii.Language, target target) string {
	return fmt.Sprintf("%s-%s-%s-%s%s", lang.Name, lang.Version, target.Platform, target.Arch, target.Ext)
}

func renderPathTemplate(template string, lang _jsii.Language, target target) string {
	replacer := strings.NewReplacer(
		"{lang}", lang.Name,
		"{language}", lang.Name,
		"{os}", target.Platform,
		"{goos}", target.GOOS,
		"{arch}", target.Arch,
		"{version}", lang.Version,
	)
	return filepath.Clean(filepath.FromSlash(replacer.Replace(template)))
}
