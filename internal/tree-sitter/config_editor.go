package treesitter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/michaelbomholt665/go-tree-sitter/internal/fileupdate"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Map and list settings are atomic leaves: dots inside module names are not key separators.
type configSetting struct {
	key       string
	valueType reflect.Type
}

func configSettings() []configSetting {
	var settings []configSetting
	var visit func(reflect.Type, string)
	visit = func(typ reflect.Type, prefix string) {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			name := strings.Split(field.Tag.Get("yaml"), ",")[0]
			if name == "" || name == "-" || name == "languages" {
				continue
			}
			key := prefix + name
			child := field.Type
			if child.Kind() == reflect.Pointer {
				child = child.Elem()
			}
			if child.Kind() == reflect.Struct {
				visit(child, key+".")
			} else {
				settings = append(settings, configSetting{key, field.Type})
			}
		}
	}
	visit(reflect.TypeOf(Config{}), "")
	sort.Slice(settings, func(i, j int) bool { return settings[i].key < settings[j].key })
	return settings
}

func findConfigSetting(key string) (configSetting, error) {
	for _, setting := range configSettings() {
		if setting.key == key {
			return setting, nil
		}
	}
	return configSetting{}, fmt.Errorf("unknown or restricted setting %q; use config show for settings, support update for assessments", key)
}

func parseConfigValue(setting configSetting, input string) (*yaml.Node, error) {
	typ := setting.valueType
	trimmed := strings.TrimSpace(input)
	if typ.Kind() == reflect.String && !strings.HasPrefix(trimmed, "\"") && !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "{") {
		inputBytes, _ := json.Marshal(input)
		input = string(inputBytes)
	}
	if !json.Valid([]byte(input)) {
		return nil, fmt.Errorf("%s requires a string or correctly typed JSON value", setting.key)
	}
	// YAML uses the schema's yaml tags, unlike encoding/json's Go field-name matching.
	target := reflect.New(typ)
	decoder := yaml.NewDecoder(strings.NewReader(input))
	decoder.KnownFields(true)
	if err := decoder.Decode(target.Interface()); err != nil {
		return nil, fmt.Errorf("%s: %w", setting.key, err)
	}
	var node yaml.Node
	if err := node.Encode(target.Elem().Interface()); err != nil {
		return nil, err
	}
	// Reject implicit coercions such as JSON numbers into strings or fractional integers.
	var supplied, converted any
	if err := json.Unmarshal([]byte(input), &supplied); err != nil {
		return nil, err
	}
	if err := node.Decode(&converted); err != nil {
		return nil, err
	}
	convertedJSON, err := json.Marshal(converted)
	if err != nil {
		return nil, err
	}
	var normalized any
	if err := json.Unmarshal(convertedJSON, &normalized); err != nil {
		return nil, err
	}
	if !sameConfigInput(supplied, normalized) {
		return nil, fmt.Errorf("%s: value does not match the setting's type (including required object fields)", setting.key)
	}
	return &node, nil
}
func sameConfigInput(input, typed any) bool {
	switch value := input.(type) {
	case map[string]any:
		converted, ok := typed.(map[string]any)
		if !ok {
			return false
		}
		for key, item := range value {
			if !sameConfigInput(item, converted[key]) {
				return false
			}
		}
		return true
	case []any:
		converted, ok := typed.([]any)
		if !ok || len(value) != len(converted) {
			return false
		}
		for i, item := range value {
			if !sameConfigInput(item, converted[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(input, typed)
	}
}

type configDocument struct {
	path     string
	original []byte
	node     yaml.Node
	cfg      *Config
}

func parseConfigDocument(path string, content []byte) (*configDocument, error) {
	doc := &configDocument{path: path, original: content}
	if err := yaml.Unmarshal(content, &doc.node); err != nil {
		return nil, err
	}
	cfg, err := decodeEditedConfig(path, content)
	if err != nil {
		return nil, err
	}
	doc.cfg = cfg
	return doc, nil
}

func decodeEditedConfig(path string, content []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}
	applyDefaults(&cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	anchored := cfg
	if !filepath.IsAbs(anchored.BuildDir) {
		anchored.BuildDir = filepath.Join(filepath.Dir(absolute), anchored.BuildDir)
	}
	anchored.Output.GrammarBase = strings.ReplaceAll(anchored.Output.GrammarBase, "{build_dir}", anchored.BuildDir)
	if !filepath.IsAbs(anchored.Output.GrammarBase) {
		anchored.Output.GrammarBase = filepath.Join(filepath.Dir(absolute), anchored.Output.GrammarBase)
	}
	if err := anchored.ValidateBuildPaths(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func configValue(cfg *Config, key string) (any, error) {
	current := reflect.ValueOf(cfg).Elem()
	for _, part := range strings.Split(key, ".") {
		if current.Kind() == reflect.Pointer {
			if current.IsNil() {
				return nil, nil
			}
			current = current.Elem()
		}
		found := false
		for i := 0; i < current.NumField(); i++ {
			if strings.Split(current.Type().Field(i).Tag.Get("yaml"), ",")[0] == part {
				current = current.Field(i)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown setting %q", key)
		}
	}
	var node yaml.Node
	if err := node.Encode(current.Interface()); err != nil {
		return nil, err
	}
	var value any
	if err := node.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

// set changes the document only after the entire candidate configuration validates.
func (doc *configDocument) set(key, input string) error {
	setting, err := findConfigSetting(key)
	if err != nil {
		return err
	}
	replacement, err := parseConfigValue(setting, input)
	if err != nil {
		return err
	}
	originalNode, err := encodeConfigNode(&doc.node)
	if err != nil {
		return err
	}
	var candidate yaml.Node
	if err := yaml.Unmarshal(originalNode, &candidate); err != nil {
		return err
	}
	current := candidate.Content[0]
	parts := strings.Split(key, ".")
	for depth, part := range parts {
		if current.Kind != yaml.MappingNode {
			return fmt.Errorf("%s: cannot edit through a YAML alias or non-mapping value", key)
		}
		index := -1
		for i := 0; i < len(current.Content); i += 2 {
			if current.Content[i].Value == part {
				index = i
				break
			}
		}
		last := depth == len(parts)-1
		value := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		if last {
			value = replacement
		}
		if index < 0 {
			pair := []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: part}, value}
			// Keep the catalog as the final root block when adding an omitted setting.
			index = len(current.Content)
			if depth == 0 {
				for i := 0; i < len(current.Content); i += 2 {
					if current.Content[i].Value == "languages" {
						index = i
						break
					}
				}
			}
			current.Content = append(current.Content[:index], append(pair, current.Content[index:]...)...)
		} else if last {
			old := current.Content[index+1]
			replacement.HeadComment, replacement.LineComment, replacement.FootComment = old.HeadComment, old.LineComment, old.FootComment
			current.Content[index+1] = replacement
		} else if current.Content[index+1].Tag == "!!null" {
			current.Content[index+1] = value
		}
		current = current.Content[index+1]
	}
	updated, err := encodeConfigNode(&candidate)
	if err != nil {
		return err
	}
	cfg, err := decodeEditedConfig(doc.path, updated)
	if err != nil {
		return err
	}
	// Detect changes to unrelated values through YAML anchors, aliases, or merges.
	expected, err := configWithoutSetting(&doc.node, key)
	if err != nil {
		return err
	}
	actual, err := configWithoutSetting(&candidate, key)
	if err != nil {
		return err
	}
	if !bytes.Equal(expected, actual) || !reflect.DeepEqual(doc.cfg.Languages, cfg.Languages) {
		return errors.New("setting change affects other settings through YAML aliases")
	}
	doc.node, doc.cfg = candidate, cfg
	return nil
}

func configWithoutSetting(node *yaml.Node, key string) ([]byte, error) {
	var values map[string]any
	if err := node.Decode(&values); err != nil {
		return nil, err
	}
	var remove func(map[string]any, []string)
	remove = func(parent map[string]any, parts []string) {
		if len(parts) == 1 {
			delete(parent, parts[0])
			return
		}
		child, ok := parent[parts[0]].(map[string]any)
		if !ok {
			return
		}
		remove(child, parts[1:])
		if len(child) == 0 {
			delete(parent, parts[0])
		}
	}
	remove(values, strings.Split(key, "."))
	return yaml.Marshal(values)
}

func encodeConfigNode(node *yaml.Node) ([]byte, error) {
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(node); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func lockConfigUpdate(path string) (func(), error) {
	return fileupdate.Lock(path)
}

func (doc *configDocument) save() error {
	release, err := lockConfigUpdate(doc.path)
	if err != nil {
		return err
	}
	defer release()
	updated, err := encodeConfigNode(&doc.node)
	if err != nil {
		return err
	}
	if _, err := decodeEditedConfig(doc.path, updated); err != nil {
		return err
	}
	return writeConfigReplacement(doc.path, doc.original, updated)
}

func writeConfigValue(out io.Writer, value any, jsonOutput bool) error {
	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(value)
	}
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}
