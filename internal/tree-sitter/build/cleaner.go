package build

import (
	"fmt"
	"os"
	"path/filepath"
)

type Cleaner struct{}

func NewCleaner() *Cleaner {
	return &Cleaner{}
}

func (c *Cleaner) CleanLanguage(buildDir, language string) error {
	target := filepath.Join(buildDir, language)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove %q: %w", target, err)
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return fmt.Errorf("ensure build directory %q exists: %w", buildDir, err)
	}
	return nil
}
