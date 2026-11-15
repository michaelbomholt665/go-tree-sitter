package configsync

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/michaelbomholt665/go-tree-sitter/internal/fileupdate"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultConfig = "tree-sitter-config.yaml"
const defaultGomod = "go.mod"

// moduleToLanguage maps Go module paths to the language name(s) used in the config.
// Each module path maps to one or more language names (e.g. tree-sitter-typescript
// covers both "typescript" and "tsx").
// Make and Zig intentionally retain legacy Go bindings after upstream removed Go
// support. Their native grammar versions and revisions are maintained in YAML.
var moduleToLanguage = map[string][]string{
	"github.com/joakker/tree-sitter-json5.git":             {"json5"},
	"github.com/UserNobody14/tree-sitter-dart":             {"dart"},
	"github.com/airbus-cert/tree-sitter-powershell":        {"powershell"},
	"github.com/tree-sitter/tree-sitter-swift":             {"swift"},
	"github.com/fwcd/tree-sitter-kotlin":                   {"kotlin"},
	"github.com/tree-sitter/tree-sitter-zsh":               {"zsh"},
	"github.com/gleam-lang/tree-sitter-gleam":              {"gleam"},
	"github.com/ionide/tree-sitter-fsharp":                 {"fsharp", "fsharp-signature"},
	"www.github.com/srazzak/tree-sitter-mdx":               {"mdx"},
	"github.com/tree-sitter-grammars/tree-sitter-hcl":      {"hcl"},
	"github.com/tree-sitter-grammars/tree-sitter-objc":     {"objc"},
	"github.com/tree-sitter-grammars/tree-sitter-svelte":   {"svelte"},
	"github.com/victorhqc/tree-sitter-prisma":              {"prisma"},
	"github.com/DerekStride/tree-sitter-sql":               {"sql"},
	"github.com/EmranMR/tree-sitter-blade":                 {"blade"},
	"github.com/JoranHonig/tree-sitter-solidity":           {"solidity"},
	"github.com/bajrangCoder/tree-sitter-sass":             {"sass"},
	"github.com/camdencheek/tree-sitter-dockerfile":        {"dockerfile"},
	"github.com/camdencheek/tree-sitter-go-mod":            {"go.mod"},
	"github.com/coder3101/tree-sitter-proto":               {"proto"},
	"github.com/elixir-lang/tree-sitter-elixir":            {"elixir"},
	"github.com/pupli/tree-sitter-cypher":                  {"cypher"},
	"github.com/tree-sitter-grammars/tree-sitter-go-sum":   {"go.sum"},
	"github.com/tree-sitter-grammars/tree-sitter-lua":      {"lua"},
	"github.com/tree-sitter-grammars/tree-sitter-markdown": {"markdown", "markdown-inline"},
	"github.com/tree-sitter-grammars/tree-sitter-scss":     {"scss"},
	"github.com/tree-sitter-grammars/tree-sitter-toml":     {"toml"},
	"github.com/tree-sitter-grammars/tree-sitter-yaml":     {"yaml"},
	"github.com/tree-sitter/tree-sitter-bash":              {"bash"},
	"github.com/tree-sitter/tree-sitter-c":                 {"c"},
	"github.com/tree-sitter/tree-sitter-c-sharp":           {"csharp"},
	"github.com/tree-sitter/tree-sitter-cpp":               {"cpp"},
	"github.com/tree-sitter/tree-sitter-css":               {"css"},
	"github.com/tree-sitter/tree-sitter-elixir":            {"elixir"},
	"github.com/tree-sitter/tree-sitter-go":                {"go"},
	"github.com/tree-sitter/tree-sitter-haskell":           {"haskell"},
	"github.com/tree-sitter/tree-sitter-html":              {"html"},
	"github.com/tree-sitter/tree-sitter-java":              {"java"},
	"github.com/tree-sitter/tree-sitter-javascript":        {"javascript"},
	"github.com/tree-sitter/tree-sitter-json":              {"json"},
	"github.com/tree-sitter/tree-sitter-julia":             {"julia"},
	"github.com/tree-sitter/tree-sitter-ocaml":             {"ocaml", "ocaml-interface"},
	"github.com/tree-sitter/tree-sitter-php":               {"php", "php_only"},
	"github.com/tree-sitter/tree-sitter-python":            {"python"},
	"github.com/tree-sitter/tree-sitter-regex":             {"regex"},
	"github.com/tree-sitter/tree-sitter-ruby":              {"ruby"},
	"github.com/tree-sitter/tree-sitter-rust":              {"rust"},
	"github.com/tree-sitter/tree-sitter-scala":             {"scala"},
	"github.com/tree-sitter/tree-sitter-typescript":        {"typescript", "tsx"},
	"github.com/uyha/tree-sitter-cmake":                    {"cmake"},
}

// requireDirective matches a line like:
//
//	github.com/foo/bar v1.2.3
//	github.com/foo/bar v1.2.3 // indirect
var requireDirective = regexp.MustCompile(`^\s*([\w./-]+)\s+(v[\w.\-+]+)`)

// DefaultModuleMapping returns a copy of the compatibility mapping for configuration initialization.
func DefaultModuleMapping() map[string][]string {
	out := make(map[string][]string, len(moduleToLanguage))
	for module, names := range moduleToLanguage {
		out[module] = append([]string(nil), names...)
	}
	return out
}

// Run synchronizes grammar package versions without changing revision or assessment pins.
func Run(configPath, gomodPath string, dryRun bool, stdout io.Writer) error {
	if !dryRun {
		release, err := fileupdate.Lock(configPath)
		if err != nil {
			return err
		}
		defer release()
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	return run(configPath, gomodPath, raw, dryRun, stdout)
}

// Preview computes version changes from an in-memory config without writing files.
func Preview(raw []byte, gomodPath string, stdout io.Writer) error {
	return run("", gomodPath, raw, true, stdout)
}

func run(configPath, gomodPath string, raw []byte, dryRun bool, stdout io.Writer) error {
	versions, err := parseGomod(gomodPath)
	if err != nil {
		return fmt.Errorf("read go.mod: %w", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return errors.New("unexpected YAML structure")
	}
	root := doc.Content[0]

	var settings struct {
		Runtime struct {
			SyncModules map[string][]string `yaml:"sync_modules"`
		} `yaml:"runtime"`
	}
	if err := root.Decode(&settings); err != nil {
		return fmt.Errorf("decode sync settings: %w", err)
	}
	modules := settings.Runtime.SyncModules
	if modules == nil {
		modules = moduleToLanguage
	}
	// 3. Build a version-update map: language name → new version from go.mod.
	langUpdates := make(map[string]string)
	var unmapped []string
	for module, ver := range versions {
		names, ok := modules[module]
		if !ok {
			// Not a grammar module — skip silently.
			continue
		}
		for _, name := range names {
			langUpdates[name] = ver
		}
	}
	_ = unmapped

	// 4. Walk the YAML tree and patch version fields in the languages list.
	changed, report, err := patchLanguages(root, langUpdates)
	if err != nil {
		return err
	}

	if len(report) == 0 {
		fmt.Fprintln(stdout, "tree-sitter-config.yaml is already up to date.")
		return nil
	}

	for _, line := range report {
		fmt.Fprintln(stdout, line)
	}

	if dryRun {
		fmt.Fprintln(stdout, "\n(dry-run: no files written)")
		return nil
	}

	if !changed {
		return nil
	}

	// 5. Serialise back, preserving comments.
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("serialise config: %w", err)
	}
	if err := fileupdate.Replace(configPath, raw, out); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	fmt.Fprintf(stdout, "\nUpdated %s\n", configPath)
	return nil
}

// parseGomod returns a map of module path → version string for all require directives.
func parseGomod(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	inRequire := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "require (" {
			inRequire = true
			continue
		}
		if inRequire && trimmed == ")" {
			inRequire = false
			continue
		}
		if inRequire {
			m := requireDirective.FindStringSubmatch(line)
			if len(m) == 3 {
				result[m[1]] = m[2]
			}
		}
		// Single-line require
		if strings.HasPrefix(trimmed, "require ") && !strings.HasSuffix(trimmed, "(") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				result[parts[1]] = parts[2]
			}
		}
	}
	return result, scanner.Err()
}

// patchLanguages walks the YAML mapping node looking for a "languages" sequence
// and updates each language entry's "version" field based on langUpdates.
// Returns (changed, report lines, error).
func patchLanguages(root *yaml.Node, updates map[string]string) (bool, []string, error) {
	// Find the "languages" key in the root mapping.
	langSeq := findMappingValue(root, "languages")
	if langSeq == nil || langSeq.Kind != yaml.SequenceNode {
		return false, nil, errors.New("config has no 'languages' sequence")
	}

	changed := false
	var report []string

	for _, item := range langSeq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		name := findMappingScalar(item, "name")
		if name == nil {
			continue
		}
		langName := name.Value

		newVersion, ok := updates[langName]
		if !ok {
			continue
		}

		versionNode := findMappingValue(item, "version")
		if versionNode == nil {
			continue
		}

		oldVersion := versionNode.Value
		if oldVersion == newVersion {
			continue
		}

		report = append(report, fmt.Sprintf("  %-20s %s  →  %s", langName, oldVersion, newVersion))
		versionNode.Value = newVersion
		changed = true
	}

	return changed, report, nil
}

// findMappingValue finds the value node for a given key in a YAML mapping node.
func findMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// findMappingScalar finds the scalar key node for a given key in a YAML mapping.
func findMappingScalar(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}
