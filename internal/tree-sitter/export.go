package treesitter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// canonicalPath resolves existing ancestors, including when the destination does not yet exist.
func canonicalPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	var missing []string
	for {
		real, err := filepath.EvalSymlinks(absolute)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				real = filepath.Join(real, missing[i])
			}
			return real, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return "", err
		}
		missing = append(missing, filepath.Base(absolute))
		absolute = parent
	}
}

func containsPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func exportCatalog(ctx context.Context, cfg *Config, source, destination, language string, force bool) error {
	if strings.TrimSpace(destination) == "" {
		return errors.New("export destination is required")
	}
	source, err := canonicalPath(source)
	if err != nil {
		return err
	}
	destination, err = canonicalPath(destination)
	if err != nil {
		return err
	}
	if containsPath(source, destination) || containsPath(destination, source) {
		return errors.New("export source and destination must not overlap")
	}
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("read published catalog: %w", err)
	}
	if !info.IsDir() {
		return errors.New("published catalog is not a directory")
	}
	langs, err := cfg.ResolveLanguages(language)
	if err != nil {
		return err
	}
	var names []string
	for _, lang := range langs {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Lstat(filepath.Join(source, lang.Name))
		if errors.Is(err, fs.ErrNotExist) && language == "" {
			continue
		}
		if err != nil {
			return fmt.Errorf("published grammar %s: %w", lang.Name, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("published grammar %s must be a directory, not a symlink or file", lang.Name)
		}
		target := filepath.Join(destination, lang.Name)
		existing, err := os.Lstat(target)
		if err == nil {
			if !existing.IsDir() {
				return fmt.Errorf("destination %s must be a directory, not a symlink or file", target)
			}
			if !force {
				return fmt.Errorf("destination %s exists; use --force to replace this grammar directory", target)
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		names = append(names, lang.Name)
	}
	if len(names) == 0 {
		return errors.New("no configured published grammars to export")
	}
	if dryRunFrom(ctx) != nil {
		return previewExport(ctx, source, destination, names, force)
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(destination, ".ts-export-")
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(stage)
		}
	}()
	if err := os.MkdirAll(filepath.Join(stage, "backup"), 0o700); err != nil {
		return err
	}
	for _, name := range names {
		if err := copyGrammar(ctx, filepath.Join(source, name), filepath.Join(stage, "new", name)); err != nil {
			return fmt.Errorf("stage %s: %w", name, err)
		}
	}
	// Stage every grammar before changing any selected destination directory.
	var committed []string
	backedUp := make(map[string]bool)
	rollback := func(cause error) error {
		var failures []error
		for i := len(committed) - 1; i >= 0; i-- {
			name := committed[i]
			if err := os.RemoveAll(filepath.Join(destination, name)); err != nil {
				failures = append(failures, err)
				continue
			}
			if backedUp[name] {
				if err := os.Rename(filepath.Join(stage, "backup", name), filepath.Join(destination, name)); err != nil {
					failures = append(failures, err)
				}
			}
		}
		if len(failures) > 0 {
			cleanup = false // Keep backups if restoration fails.
			return errors.Join(cause, fmt.Errorf("rollback incomplete; backups retained at %s: %w", stage, errors.Join(failures...)))
		}
		return cause
	}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return rollback(err)
		}
		target := filepath.Join(destination, name)
		if info, err := os.Lstat(target); err == nil {
			if !force || !info.IsDir() {
				return rollback(fmt.Errorf("destination %s changed or cannot be replaced", target))
			}
			if err := os.Rename(target, filepath.Join(stage, "backup", name)); err != nil {
				return rollback(err)
			}
			backedUp[name] = true
		} else if !errors.Is(err, fs.ErrNotExist) {
			return rollback(err)
		}
		// Include the current entry so rollback also restores its backup if installation fails.
		committed = append(committed, name)
		if err := os.Rename(filepath.Join(stage, "new", name), target); err != nil {
			return rollback(err)
		}
	}
	return nil
}

func copyGrammar(ctx context.Context, source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing non-regular file %s", path)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		return errors.Join(copyErr, out.Close())
	})
}
