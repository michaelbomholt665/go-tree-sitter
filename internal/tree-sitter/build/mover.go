package build

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type Mover struct {
	clock   Clock
	cleaner *Cleaner
	stdout  io.Writer
	stderr  io.Writer
}

func NewMover(clock Clock, cleaner *Cleaner, stdout, stderr io.Writer) *Mover {
	if clock == nil {
		clock = RealClock{}
	}
	if cleaner == nil {
		cleaner = NewCleaner()
	}
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return &Mover{
		clock:   clock,
		cleaner: cleaner,
		stdout:  stdout,
		stderr:  stderr,
	}
}

func (m *Mover) Move(_ context.Context, cfg *_jsii.Config, req _jsii.MoveRequest) error {
	languages, err := cfg.ResolveLanguages(req.Language)
	if err != nil {
		return err
	}

	for _, lang := range languages {
		if err := m.moveLanguage(cfg, lang, req); err != nil {
			return fmt.Errorf("move %s: %w", lang.Name, err)
		}
	}

	return nil
}

func (m *Mover) moveLanguage(cfg *_jsii.Config, lang _jsii.Language, req _jsii.MoveRequest) error {
	srcBinDir := binaryDir(cfg, lang)
	outputDir := filepath.Join(cfg.Output.GrammarBase, lang.Name)
	binaries, err := collectBinaries(srcBinDir, lang)
	if err != nil {
		if hasMovedArtifacts(outputDir, lang) {
			fmt.Fprintf(m.stdout, "skipping %s; artifacts already exist in %s\n", lang.Name, outputDir)
			return nil
		}
		return fmt.Errorf("compiled binaries not found in %q; run `tree-sitter compile` first", srcBinDir)
	}
	if len(binaries) == 0 {
		if hasMovedArtifacts(outputDir, lang) {
			fmt.Fprintf(m.stdout, "skipping %s; artifacts already exist in %s\n", lang.Name, outputDir)
			return nil
		}
		return fmt.Errorf("compiled binaries not found in %q; run `tree-sitter compile` first", srcBinDir)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", outputDir, err)
	}

	copiedBinaries := make([]string, 0, len(binaries))
	for _, sourcePath := range binaries {
		targetPath := filepath.Join(outputDir, filepath.Base(sourcePath))
		if err := copyFile(sourcePath, targetPath, req.Force); err != nil {
			return err
		}
		copiedBinaries = append(copiedBinaries, targetPath)
	}

	src := sourceDir(cfg, lang)
	repoRoot := buildRootDir(cfg, lang)
	hasNodeTypes := false
	if req.Mode.IncludesNodeTypes() {
		hasNodeTypes, err = m.copyNodeTypes(src, outputDir, req.Force)
		if err != nil {
			return err
		}
		if !hasNodeTypes {
			fmt.Fprintf(m.stderr, "warning: node-types.json missing for %s\n", lang.Name)
		}
	}

	hasQueries := false
	if req.Mode.IncludesQueries() {
		hasQueries, err = m.copyQueries(src, repoRoot, outputDir, req.Force)
		if err != nil {
			return err
		}
		if !hasQueries {
			fmt.Fprintf(m.stderr, "warning: queries directory missing for %s\n", lang.Name)
		}
	}

	if cfg.Output.GenerateManifest && !req.NoManifest {
		manifest, err := GenerateManifest(lang, copiedBinaries, cfg, hasNodeTypes, hasQueries, m.clock.Now())
		if err != nil {
			return err
		}
		if err := manifest.WriteToFile(filepath.Join(outputDir, "manifest.json")); err != nil {
			return err
		}
	}

	if req.Clean {
		if err := m.cleaner.CleanLanguage(cfg.BuildDir, lang.Name); err != nil {
			return err
		}
	}

	fmt.Fprintf(m.stdout, "moved %s artifacts to %s\n", lang.Name, outputDir)
	return nil
}

func (m *Mover) copyNodeTypes(sourceDir, outputDir string, force bool) (bool, error) {
	candidates := []string{
		filepath.Join(sourceDir, "src", "node-types.json"),
		filepath.Join(sourceDir, "node-types.json"),
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			target := filepath.Join(outputDir, "node-types.json")
			return true, copyFile(candidate, target, force)
		}
	}
	return false, nil
}

func (m *Mover) copyQueries(sourceDir, repoRoot, outputDir string, force bool) (bool, error) {
	candidates := []string{
		filepath.Join(sourceDir, "queries"),
		filepath.Join(repoRoot, "queries"),
	}

	for _, queriesDir := range candidates {
		info, err := os.Stat(queriesDir)
		if err != nil || !info.IsDir() {
			continue
		}

		target := filepath.Join(outputDir, "queries")
		if err := copyDir(queriesDir, target, force); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func collectBinaries(root string, lang _jsii.Language) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, lang.Name+"-"+lang.Version+"-") {
			continue
		}
		switch filepath.Ext(name) {
		case ".so", ".dylib", ".dll":
			files = append(files, filepath.Join(root, name))
		}
	}
	return files, nil
}

func hasMovedArtifacts(outputDir string, lang _jsii.Language) bool {
	binaries, err := collectBinaries(outputDir, lang)
	return err == nil && len(binaries) > 0
}

func copyFile(sourcePath, targetPath string, force bool) error {
	if !force {
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("target file %q already exists; rerun with --force to overwrite", targetPath)
		}
	}

	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read %q: %w", sourcePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", targetPath, err)
	}

	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", targetPath, err)
	}

	return nil
}

func copyDir(sourceDir, targetDir string, force bool) error {
	if force {
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("remove existing directory %q: %w", targetDir, err)
		}
	} else if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("target directory %q already exists; rerun with --force to overwrite", targetDir)
	}

	return filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(targetDir, relative)

		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		return copyFile(path, destination, true)
	})
}
