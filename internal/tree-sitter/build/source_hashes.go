package build

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
)

func hashSourceInputs(root string) (map[string]string, error) {
	dir := filepath.Join(root, "src")
	if _, err := os.Stat(filepath.Join(dir, "parser.c")); err != nil {
		return nil, err
	}
	hashes := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("source input must be a regular file: %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sum, err := checksumFile(path)
		if err != nil {
			return err
		}
		hashes[filepath.ToSlash(relative)] = sum
		return nil
	})
	return hashes, err
}

func verifySourceInputs(root string, provenance SourceProvenance) error {
	actual, err := hashSourceInputs(root)
	if err != nil {
		return err
	}
	if len(provenance.SourceHashes) == 0 || !maps.Equal(actual, provenance.SourceHashes) {
		return fmt.Errorf("source input hashes changed or are missing; run tree-sitter build again")
	}
	return nil
}
