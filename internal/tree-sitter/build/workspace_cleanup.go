package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

// Plan cleanup before publication; never delete a shared or external template root.
func languageCleanupRoots(cfg *ts.Config, lang ts.Language) ([]string, error) {
	for _, template := range []string{cfg.Output.GrammarBuild, cfg.Output.GrammarCompile} {
		if template != "" && !strings.Contains(template, "{lang}") && !strings.Contains(template, "{language}") {
			return nil, fmt.Errorf("cleanup requires a language placeholder in template %q", template)
		}
	}
	paths := []string{buildRootDir(cfg, lang)}
	for platform, arches := range cfg.EffectiveRuntime().SupportedTargets {
		for _, arch := range arches {
			t, err := resolveTarget(platform, arch, cfg)
			if err != nil {
				return nil, err
			}
			paths = append(paths, binaryDirForTarget(cfg, lang, t))
		}
	}
	var roots []string
	for _, path := range paths {
		if err := cfg.ValidateWorkspacePath(path); err != nil {
			return nil, err
		}
		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		duplicate := false
		for _, existing := range roots {
			if path == existing {
				duplicate = true
				break
			}
		}
		if !duplicate {
			roots = append(roots, path)
		}
	}
	// Preserve the legacy per-language cleanup boundary for standard templates.
	parent, err := filepath.Abs(filepath.Join(cfg.BuildDir, lang.Name))
	if err != nil {
		return nil, err
	}
	allUnderParent := true
	for _, root := range roots {
		rel, err := filepath.Rel(parent, root)
		if err != nil || (rel != "." && !filepath.IsLocal(rel)) {
			allUnderParent = false
			break
		}
	}
	if allUnderParent {
		if err := cfg.ValidateWorkspacePath(parent); err != nil {
			return nil, err
		}
		return []string{parent}, nil
	}
	return roots, nil
}

func (c *Cleaner) cleanupLanguage(cfg *ts.Config, lang ts.Language, action string) error {
	if action == "keep" {
		return nil
	}
	if action != "prune" && action != "remove" {
		return fmt.Errorf("invalid cleanup action %q", action)
	}
	roots, err := languageCleanupRoots(cfg, lang)
	if err != nil {
		return err
	}
	for _, root := range roots {
		if action == "prune" {
			err = pruneDirectory(root, cfg.EffectiveRuntime().Cleanup.Prune)
		} else {
			err = os.RemoveAll(root)
		}
		if err != nil {
			return fmt.Errorf("%s workspace %q: %w", action, root, err)
		}
	}
	return nil
}
