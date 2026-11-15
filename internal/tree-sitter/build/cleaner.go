package build

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type Cleaner struct{}

func NewCleaner() *Cleaner {
	return &Cleaner{}
}

func (c *Cleaner) CleanLanguage(buildDir, language string) error {
	language = strings.TrimSpace(language)
	if language == "" {
		return errors.New("language cannot be empty")
	}
	target := filepath.Join(buildDir, language)
	if err := (&_jsii.Config{BuildDir: buildDir}).ValidateWorkspacePath(target); err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove %q: %w", target, err)
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return fmt.Errorf("ensure build directory %q exists: %w", buildDir, err)
	}
	return nil
}

func (c *Cleaner) PruneLanguage(buildDir, language string) error {
	return c.PruneLanguageWithRules(buildDir, language, (*_jsii.Config)(nil).EffectiveRuntime().Cleanup.Prune)
}

func (c *Cleaner) PruneLanguageWithRules(buildDir, language string, rules _jsii.PruneRules) error {
	language = strings.TrimSpace(language)
	if language == "" {
		return errors.New("language cannot be empty")
	}
	target := filepath.Join(buildDir, language)
	if err := (&_jsii.Config{BuildDir: buildDir}).ValidateWorkspacePath(target); err != nil {
		return err
	}
	return pruneDirectory(target, rules)
}

func pruneDirectory(target string, rules _jsii.PruneRules) error {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect %q: %w", target, err)
	}

	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if path == target {
			return nil
		}

		name := d.Name()
		rel, relErr := filepath.Rel(target, path)
		if relErr != nil {
			return relErr
		}
		slashRel := filepath.ToSlash(rel)

		if d.IsDir() {
			if preservedDirectory(slashRel, rules.PreserveDirectories) {
				return nil
			}

			if prunedDirectory(slashRel, rules.Directories) {
				if err := removeAllSafely(path); err != nil {
					return fmt.Errorf("prune directory %q: %w", path, err)
				}
				return filepath.SkipDir
			}
			return nil
		}

		if matchesPatterns(name, rules.PreserveFiles, false) {
			return nil
		}

		patterns := rules.Files
		if preservedDirectory(slashRel, rules.PreserveDirectories) {
			patterns = rules.ProtectedFiles
		}
		if matchesPatterns(name, patterns, true) {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("prune file %q: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("prune target %s: %w", target, err)
	}
	return nil
}

func matchesPatterns(name string, patterns []string, foldCase bool) bool {
	for _, pattern := range patterns {
		candidate := name
		if foldCase {
			candidate, pattern = strings.ToLower(candidate), strings.ToLower(pattern)
		}
		if matched, _ := filepath.Match(pattern, candidate); matched {
			return true
		}
	}
	return false
}

func preservedDirectory(relative string, patterns []string) bool {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for _, part := range parts[:len(parts)-1] {
		if matchesPatterns(part, patterns, false) {
			return true
		}
	}
	return matchesPatterns(filepath.Base(relative), patterns, false)
}

func prunedDirectory(relative string, patterns []string) bool {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for i := range parts {
		if matchesPatterns(strings.Join(parts[i:], "/"), patterns, false) {
			return true
		}
	}
	return false
}

func (c *Cleaner) Clean(ctx context.Context, cfg *_jsii.Config, req _jsii.CleanRequest) error {
	languages, err := cfg.ResolveLanguages(req.Language)
	if err != nil {
		return err
	}
	var cleanErrors []error
	for _, lang := range languages {
		if err := ctx.Err(); err != nil {
			return err
		}
		action := "remove"
		if req.Prune {
			action = "prune"
		}
		if err := c.cleanupLanguage(cfg, lang, action); err != nil {
			cleanErrors = append(cleanErrors, fmt.Errorf("%s %s: %w", action, lang.Name, err))
		}
	}
	return errors.Join(cleanErrors...)
}

func removeAllSafely(path string) error {
	err := os.RemoveAll(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	_ = filepath.WalkDir(path, func(p string, _ fs.DirEntry, walkErr error) error {
		if walkErr == nil {
			_ = os.Chmod(p, 0o777)
		}
		return nil
	})
	return os.RemoveAll(path)
}
