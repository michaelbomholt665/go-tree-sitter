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
	"go.mod",
	"LICENSE*",
	"LICENCE*",
	"NOTICE*",
	"grammar.js",
	"node-types.json",
	".source-provenance.json",
}

var prunedDirNames = map[string]bool{
	".git":         true,
	".github":      true,
	".vscode":      true,
	".idea":        true,
	"node_modules": true,
	"bindings":     true,
	"test":         true,
	"tests":        true,
	"corpus":       true,
	"examples":     true,
	"target":       true,
	"docs":         true,
	"doc":          true,
	"script":       true,
	"scripts":      true,
	"tools":        true,
	"benchmark":    true,
	"benchmarks":   true,
	"_layouts":     true,
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

		if isProtectedFile(name) {
			return nil
		}

		inProtectedDir := isProtectedDir(slashRel, filepath.Base(filepath.Dir(slashRel)))
		if isPrunedFile(name, inProtectedDir) {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("prune file %q: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("prune language %s: %w", language, err)
	}
	return nil
}

func isPrunedFile(name string, inProtectedDir bool) bool {
	lower := strings.ToLower(name)

	// Object and static library archives
	if strings.HasSuffix(lower, ".o") || strings.HasSuffix(lower, ".obj") ||
		strings.HasSuffix(lower, ".a") || strings.HasSuffix(lower, ".lib") {
		return true
	}

	// Inside protected directories (src, queries), only object/archive files are pruned
	if inProtectedDir {
		return false
	}

	// Lock files
	if strings.HasSuffix(lower, ".lock") || strings.HasSuffix(lower, ".resolved") ||
		lower == "go.sum" || lower == "package-lock.json" || lower == "yarn.lock" || lower == "pnpm-lock.yaml" {
		return true
	}

	// Git, VCS, and editor/tool configs
	if lower == ".gitignore" || lower == ".gitattributes" || lower == ".gitmodules" ||
		lower == ".editorconfig" || lower == ".clang-format" || lower == ".git-blame-ignore-revs" ||
		lower == ".envrc" || lower == ".npmignore" {
		return true
	}

	// Linter configs
	if strings.HasPrefix(lower, "eslint.config.") || strings.HasPrefix(lower, ".eslintrc") ||
		strings.HasPrefix(lower, ".prettierrc") || lower == ".prettierignore" || lower == ".tsqueryrc.json" {
		return true
	}

	// Documentation & repository metadata
	if strings.HasPrefix(name, "README") || strings.HasPrefix(name, "CHANGELOG") ||
		strings.HasPrefix(name, "CONTRIBUTING") || strings.HasPrefix(name, "SECURITY") ||
		strings.HasPrefix(name, "FUNDING") || strings.HasPrefix(name, "ISSUE_TEMPLATE") ||
		strings.HasPrefix(name, "CODE_OF_CONDUCT") {
		return true
	}

	// Unused package / build manifests for other ecosystems
	if lower == "package.swift" || lower == "binding.gyp" || lower == "setup.py" ||
		lower == "pyproject.toml" || lower == "cargo.toml" || lower == "cmakelists.txt" ||
		lower == "makefile" || lower == "justfile" || lower == "build.zig" ||
		lower == "build.zig.zon" || lower == "gemfile" || lower == "gemfile.lock" {
		return true
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
