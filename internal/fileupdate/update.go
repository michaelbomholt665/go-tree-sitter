// Package fileupdate serializes cooperative configuration writers.
package fileupdate

import (
	"bytes"
	"errors"
	"github.com/michaelbomholt665/go-tree-sitter/internal/filelock"
	"os"
	"path/filepath"
)

func Lock(path string) (func(), error) {
	return filelock.Lock(path + ".support.lock")
}

func Replace(path string, original, updated []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("configuration must be a regular file, not a symlink")
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".support-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := file.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if _, err := file.Write(updated); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, original) {
		return errors.New("configuration changed during update; retry after reviewing it")
	}
	return os.Rename(file.Name(), path)
}
