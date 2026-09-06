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

var protectedManifestPatterns = []string{
	"tree-sitter.json",
	"package.json",
	"Cargo.toml",
	"pyproject.toml",
	"setup.py",
	"Package.swift",
	"go.mod",
	"Makefile",
	"binding.gyp",
	"CMakeLists.txt",
	"build.zig",
	"LICENSE*",
	"LICENCE*",
	"grammar.js",
	"node-types.json",
}

var prunedDirNames = map[string]bool{
	".git":         true,
	"node_modules": true,
	"test":         true,
	"corpus":       true,
	"examples":     true,
	"target":       true,
}

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
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove %q: %w", target, err)
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return fmt.Errorf("ensure build directory %q exists: %w", buildDir, err)
	}
	return nil
}

func (c *Cleaner) PruneLanguage(buildDir, language string) error {
	language = strings.TrimSpace(language)
	if language == "" {
		return errors.New("language cannot be empty")
	}
	target := filepath.Join(buildDir, language)
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
			if isProtectedDir(slashRel, name) {
				return nil
			}

			if prunedDirNames[name] || slashRel == "build/Release" || strings.HasSuffix(slashRel, "/build/Release") {
				if err := removeAllSafely(path); err != nil {
					return fmt.Errorf("prune directory %q: %w", path, err)
				}
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(name, ".o") || strings.HasSuffix(name, ".obj") {
			if !isProtectedFile(name) {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("prune object file %q: %w", path, err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("prune language %s: %w", language, err)
	}
	return nil
}

func (c *Cleaner) Clean(ctx context.Context, cfg *_jsii.Config, req _jsii.CleanRequest) error {
	languages, err := cfg.ResolveLanguages(req.Language)
	if err != nil {
		return err
	}
	var cleanErrors []error
	for _, lang := range languages {
		if req.Prune {
			if err := c.PruneLanguage(cfg.BuildDir, lang.Name); err != nil {
				cleanErrors = append(cleanErrors, fmt.Errorf("prune %s: %w", lang.Name, err))
			}
		} else {
			if err := c.CleanLanguage(cfg.BuildDir, lang.Name); err != nil {
				cleanErrors = append(cleanErrors, fmt.Errorf("clean %s: %w", lang.Name, err))
			}
		}
	}
	return errors.Join(cleanErrors...)
}

func isProtectedFile(filename string) bool {
	for _, pattern := range protectedManifestPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}
	return false
}

func isProtectedDir(slashRel, name string) bool {
	if name == "src" || name == "queries" {
		return true
	}
	if strings.HasPrefix(slashRel, "src/") || strings.HasPrefix(slashRel, "queries/") {
		return true
	}
	if strings.Contains("/"+slashRel+"/", "/src/") || strings.Contains("/"+slashRel+"/", "/queries/") {
		return true
	}
	return false
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
