package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const sourceProvenanceFilename = ".source-provenance.json"

type SourceProvenance struct {
	SourceRepository string `json:"source_repository"`
	SourceRevision   string `json:"source_revision"`
	GeneratorVersion string `json:"generator_version"`
	GenerateABI      *int   `json:"generate_abi"`
	SourceDateEpoch  int64  `json:"source_date_epoch"`
	NodeTypesSHA256  string `json:"node_types_sha256"`
}

type BinaryProvenance struct {
	Filename         string   `json:"filename"`
	SourceRepository string   `json:"source_repository"`
	SourceRevision   string   `json:"source_revision"`
	GeneratorVersion string   `json:"generator_version"`
	GenerateABI      *int     `json:"generate_abi"`
	SourceDateEpoch  int64    `json:"source_date_epoch"`
	NodeTypesSHA256  string   `json:"node_types_sha256"`
	TargetTriple     string   `json:"target_triple"`
	Compiler         string   `json:"compiler"`
	CompilerVersion  string   `json:"compiler_version"`
	BuildFlags       []string `json:"build_flags"`
}

func binaryProvenancePath(binaryPath string) string {
	return binaryPath + ".provenance.json"
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", path, err)
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %q: %w", path, err)
	}
	payload = append(payload, '\n')
	temp, err := os.CreateTemp(filepath.Dir(path), ".metadata-*")
	if err != nil {
		return fmt.Errorf("create temporary metadata for %q: %w", path, err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o644); err != nil {
		temp.Close()
		return fmt.Errorf("set permissions on temporary metadata for %q: %w", path, err)
	}
	if _, err := temp.Write(payload); err != nil {
		temp.Close()
		return fmt.Errorf("write temporary metadata for %q: %w", path, err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync temporary metadata for %q: %w", path, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary metadata for %q: %w", path, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("publish metadata %q: %w", path, err)
	}
	return nil
}

func readJSON(path string, value any) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %q: %w", path, err)
	}
	if err := json.Unmarshal(payload, value); err != nil {
		return fmt.Errorf("parse %q: %w", path, err)
	}
	return nil
}

func checksumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %q: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func parseToolVersion(output, tool string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) < 2 || fields[0] != tool {
		return "", fmt.Errorf("unexpected %s --version output %q", tool, output)
	}
	return strings.TrimPrefix(fields[len(fields)-1], "v"), nil
}
