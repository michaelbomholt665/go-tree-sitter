package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

type ManifestData struct {
	Grammar       string       `json:"grammar"`
	Version       string       `json:"version"`
	CompiledAt    string       `json:"compiled_at"`
	TreeSitterVer string       `json:"tree_sitter_version"`
	Binaries      []BinaryInfo `json:"binaries"`
	ABI           ABIInfo      `json:"abi"`
	Artifacts     ArtifactInfo `json:"artifacts"`
}

type BinaryInfo struct {
	Platform       string `json:"platform"`
	Arch           string `json:"arch"`
	Filename       string `json:"filename"`
	ChecksumSHA256 string `json:"checksum_sha256"`
}

type ABIInfo struct {
	MinVersion    int    `json:"min_version"`
	MaxVersion    int    `json:"max_version"`
	ParserVersion string `json:"parser_version"`
}

type ArtifactInfo struct {
	HasNodeTypes bool `json:"has_node_types"`
	HasQueries   bool `json:"has_queries"`
	HasWasm      bool `json:"has_wasm"`
}

func GenerateManifest(lang _jsii.Language, binaryPaths []string, cfg *_jsii.Config, hasNodeTypes, hasQueries bool, compiledAt time.Time) (*ManifestData, error) {
	abi, err := cfg.ABIFor(lang.TreeSitterVersion)
	if err != nil {
		return nil, err
	}

	checksums, err := CalculateChecksums(binaryPaths)
	if err != nil {
		return nil, err
	}

	binaries := make([]BinaryInfo, 0, len(binaryPaths))
	for _, path := range binaryPaths {
		filename := filepath.Base(path)
		platform, arch, err := parseBinaryMetadata(filename, lang)
		if err != nil {
			return nil, err
		}

		binaries = append(binaries, BinaryInfo{
			Platform:       platform,
			Arch:           arch,
			Filename:       filename,
			ChecksumSHA256: checksums[path],
		})
	}

	sort.Slice(binaries, func(i, j int) bool {
		return binaries[i].Filename < binaries[j].Filename
	})

	return &ManifestData{
		Grammar:       lang.Name,
		Version:       lang.Version,
		CompiledAt:    compiledAt.UTC().Format(time.RFC3339),
		TreeSitterVer: lang.TreeSitterVersion,
		Binaries:      binaries,
		ABI: ABIInfo{
			MinVersion:    abi.Min,
			MaxVersion:    abi.Max,
			ParserVersion: lang.TreeSitterVersion,
		},
		Artifacts: ArtifactInfo{
			HasNodeTypes: hasNodeTypes,
			HasQueries:   hasQueries,
			HasWasm:      false,
		},
	}, nil
}

func (m *ManifestData) WriteToFile(outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}

	payload, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	payload = append(payload, '\n')

	if err := os.WriteFile(outputPath, payload, 0o644); err != nil {
		return fmt.Errorf("write manifest %q: %w", outputPath, err)
	}
	return nil
}

func CalculateChecksums(filePaths []string) (map[string]string, error) {
	checksums := make(map[string]string, len(filePaths))
	for _, path := range filePaths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %q for checksum: %w", path, err)
		}
		sum := sha256.Sum256(content)
		checksums[path] = hex.EncodeToString(sum[:])
	}
	return checksums, nil
}

func parseBinaryMetadata(filename string, lang _jsii.Language) (string, string, error) {
	stem := strings.TrimSuffix(filename, filepath.Ext(filename))
	prefix := fmt.Sprintf("%s-%s-", lang.Name, lang.Version)
	if !strings.HasPrefix(stem, prefix) {
		return "", "", fmt.Errorf("binary %q does not match expected prefix %q", filename, prefix)
	}

	parts := strings.Split(strings.TrimPrefix(stem, prefix), "-")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("binary %q does not match expected platform/arch naming", filename)
	}

	return parts[0], parts[1], nil
}
