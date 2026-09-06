package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

// RawNodeTypeEntry represents an entry in tree-sitter node-types.json.
type RawNodeTypeEntry struct {
	Type     string                        `json:"type"`
	Named    bool                          `json:"named"`
	Subtypes []RawTypeChild                `json:"subtypes,omitempty"`
	Fields   map[string]RawFieldDescriptor `json:"fields,omitempty"`
	Children *RawChildrenDescriptor        `json:"children,omitempty"`
}

// RawTypeChild represents a referenced type inside subtypes, field types, or children types.
type RawTypeChild struct {
	Type  string `json:"type"`
	Named bool   `json:"named"`
}

// RawFieldDescriptor describes a field on a node type.
type RawFieldDescriptor struct {
	Multiple bool           `json:"multiple"`
	Required bool           `json:"required"`
	Types    []RawTypeChild `json:"types"`
}

// RawChildrenDescriptor describes allowed child nodes.
type RawChildrenDescriptor struct {
	Multiple bool           `json:"multiple"`
	Required bool           `json:"required"`
	Types    []RawTypeChild `json:"types"`
}

// CompactNode represents the compact schema for a named node in YAML.
type CompactNode struct {
	Fields   map[string]string `json:"fields,omitempty"`
	Children string            `json:"children,omitempty"`
}

// CompactMetrics aggregates metrics from comparing raw node-types.json and compact YAML.
type CompactMetrics struct {
	TotalNodes      int
	TotalFields     int
	TotalSupertypes int
	TotalChildren   int
	OriginalBytes   int64
	CompactBytes    int64
	ReductionPct    float64
}

// CompactNodeTypes transforms raw node-types.json bytes into a token-efficient compact YAML representation.
func CompactNodeTypes(rawJSON []byte) ([]byte, error) {
	var rawData []RawNodeTypeEntry
	if err := json.Unmarshal(rawJSON, &rawData); err != nil {
		return nil, fmt.Errorf("parse raw node-types.json: %w", err)
	}

	supertypes := make(map[string][]string)
	nodes := make(map[string]CompactNode)

	for _, entry := range rawData {
		if !entry.Named || entry.Type == "" {
			continue
		}

		if entry.Subtypes != nil {
			var subtypes []string
			for _, s := range entry.Subtypes {
				if s.Named && s.Type != "" {
					subtypes = append(subtypes, s.Type)
				}
			}
			sort.Strings(subtypes)
			supertypes[entry.Type] = subtypes
			continue
		}

		var nodeData CompactNode

		if len(entry.Fields) > 0 {
			nodeData.Fields = make(map[string]string, len(entry.Fields))
			for fname, fval := range entry.Fields {
				var types []string
				for _, t := range fval.Types {
					if t.Named && t.Type != "" {
						types = append(types, t.Type)
					}
				}
				sort.Strings(types)
				typeStr := "any"
				if len(types) > 0 {
					typeStr = strings.Join(types, " | ")
				}
				if fval.Multiple {
					typeStr = "[" + typeStr + "]"
				}
				fieldKey := fname
				if !fval.Required {
					fieldKey = fname + "?"
				}
				nodeData.Fields[fieldKey] = typeStr
			}
		}

		if entry.Children != nil {
			var types []string
			for _, t := range entry.Children.Types {
				if t.Named && t.Type != "" {
					types = append(types, t.Type)
				}
			}
			sort.Strings(types)
			typeStr := "any"
			if len(types) > 0 {
				typeStr = strings.Join(types, " | ")
			}
			if entry.Children.Multiple {
				typeStr = "[" + typeStr + "]"
			}
			nodeData.Children = typeStr
		}

		nodes[entry.Type] = nodeData
	}

	var b strings.Builder
	b.WriteString("# Compact Tree-sitter Node Types (Generated from node-types.json)\n")
	b.WriteString("# Legend:\n")
	b.WriteString("#   field_name? : optional field\n")
	b.WriteString("#   [Type]      : multiple occurrences allowed\n")
	b.WriteString("#   TypeA | B   : alternative node types\n\n")
	b.WriteString("supertypes:\n")

	supertypeKeys := make([]string, 0, len(supertypes))
	for k := range supertypes {
		supertypeKeys = append(supertypeKeys, k)
	}
	sort.Strings(supertypeKeys)
	for _, k := range supertypeKeys {
		b.WriteString(fmt.Sprintf("  %s: [%s]\n", k, strings.Join(supertypes[k], ", ")))
	}

	b.WriteString("\nnodes:\n")
	nodeKeys := make([]string, 0, len(nodes))
	for k := range nodes {
		nodeKeys = append(nodeKeys, k)
	}
	sort.Strings(nodeKeys)
	for _, k := range nodeKeys {
		data := nodes[k]
		if len(data.Fields) == 0 && data.Children == "" {
			b.WriteString(fmt.Sprintf("  %s: {}\n", k))
			continue
		}
		b.WriteString(fmt.Sprintf("  %s:\n", k))
		if len(data.Fields) > 0 {
			b.WriteString("    fields:\n")
			fkeys := make([]string, 0, len(data.Fields))
			for fk := range data.Fields {
				fkeys = append(fkeys, fk)
			}
			sort.Strings(fkeys)
			for _, fk := range fkeys {
				b.WriteString(fmt.Sprintf("      %s: %s\n", fk, data.Fields[fk]))
			}
		}
		if data.Children != "" {
			b.WriteString(fmt.Sprintf("    children: %s\n", data.Children))
		}
	}

	return []byte(b.String()), nil
}

// ParseCompactYAML parses the compact node types YAML into supertypes and nodes maps.
func ParseCompactYAML(yamlText string) (supertypes map[string][]string, nodes map[string]CompactNode, err error) {
	supertypes = make(map[string][]string)
	nodes = make(map[string]CompactNode)
	var currentSection string
	var currentNode string
	var inFields bool

	normalized := strings.ReplaceAll(yamlText, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for lineIdx, line := range lines {
		stripped := strings.TrimSpace(line)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))

		if indent == 0 {
			if stripped == "supertypes:" {
				currentSection = "supertypes"
			} else if stripped == "nodes:" {
				currentSection = "nodes"
			} else {
				return nil, nil, fmt.Errorf("line %d: unexpected top-level section %q", lineIdx+1, stripped)
			}
			continue
		}

		if currentSection == "supertypes" && indent == 2 {
			key, rest, ok := strings.Cut(stripped, ":")
			if !ok {
				return nil, nil, fmt.Errorf("line %d: malformed supertype entry %q", lineIdx+1, stripped)
			}
			val := strings.TrimSpace(rest)
			var items []string
			if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
				inner := strings.TrimSpace(val[1 : len(val)-1])
				if inner != "" {
					for _, item := range strings.Split(inner, ",") {
						item = strings.TrimSpace(item)
						if item != "" {
							items = append(items, item)
						}
					}
				}
			}
			supertypes[strings.TrimSpace(key)] = items
		} else if currentSection == "nodes" {
			if indent == 2 {
				inFields = false
				if strings.HasSuffix(stripped, ": {}") {
					currentNode = strings.TrimSpace(strings.TrimSuffix(stripped, ": {}"))
					nodes[currentNode] = CompactNode{}
				} else if strings.HasSuffix(stripped, ":") {
					currentNode = strings.TrimSpace(strings.TrimSuffix(stripped, ":"))
					nodes[currentNode] = CompactNode{}
				} else {
					return nil, nil, fmt.Errorf("line %d: malformed node entry %q", lineIdx+1, stripped)
				}
			} else if indent == 4 && currentNode != "" {
				if stripped == "fields:" {
					inFields = true
					node := nodes[currentNode]
					if node.Fields == nil {
						node.Fields = make(map[string]string)
					}
					nodes[currentNode] = node
				} else if strings.HasPrefix(stripped, "children:") {
					inFields = false
					_, rest, ok := strings.Cut(stripped, ":")
					if !ok {
						return nil, nil, fmt.Errorf("line %d: malformed children entry %q", lineIdx+1, stripped)
					}
					node := nodes[currentNode]
					node.Children = strings.TrimSpace(rest)
					nodes[currentNode] = node
				} else {
					return nil, nil, fmt.Errorf("line %d: unexpected entry under node %q: %q", lineIdx+1, currentNode, stripped)
				}
			} else if indent == 6 && currentNode != "" && inFields {
				fkey, fval, ok := strings.Cut(stripped, ":")
				if !ok {
					return nil, nil, fmt.Errorf("line %d: malformed field entry %q", lineIdx+1, stripped)
				}
				node := nodes[currentNode]
				if node.Fields == nil {
					node.Fields = make(map[string]string)
				}
				node.Fields[strings.TrimSpace(fkey)] = strings.TrimSpace(fval)
				nodes[currentNode] = node
			} else {
				return nil, nil, fmt.Errorf("line %d: invalid indentation or structure %q", lineIdx+1, stripped)
			}
		} else {
			return nil, nil, fmt.Errorf("line %d: unexpected indentation or structure %q", lineIdx+1, stripped)
		}
	}

	return supertypes, nodes, nil
}

// ValidateCapture performs strict lossless validation that compact YAML preserves 100% of raw JSON AST schema.
func ValidateCapture(rawJSON []byte, yamlText string) (discrepancies []string, metrics CompactMetrics, err error) {
	supertypes, nodes, err := ParseCompactYAML(yamlText)
	if err != nil {
		return nil, metrics, fmt.Errorf("parse compact YAML: %w", err)
	}

	var rawData []RawNodeTypeEntry
	if err := json.Unmarshal(rawJSON, &rawData); err != nil {
		return nil, metrics, fmt.Errorf("parse raw node-types.json: %w", err)
	}

	metrics.TotalNodes = len(nodes)
	metrics.TotalSupertypes = len(supertypes)
	for _, n := range nodes {
		metrics.TotalFields += len(n.Fields)
		if n.Children != "" {
			metrics.TotalChildren++
		}
	}
	metrics.OriginalBytes = int64(len(rawJSON))
	metrics.CompactBytes = int64(len(yamlText))
	if metrics.OriginalBytes > 0 {
		metrics.ReductionPct = (float64(metrics.OriginalBytes-metrics.CompactBytes) / float64(metrics.OriginalBytes)) * 100.0
	}

	namedTypesInJSON := make(map[string]struct{})

	for _, entry := range rawData {
		if !entry.Named || entry.Type == "" {
			continue
		}

		namedTypesInJSON[entry.Type] = struct{}{}

		if entry.Subtypes != nil {
			actSubs, ok := supertypes[entry.Type]
			if !ok {
				discrepancies = append(discrepancies, fmt.Sprintf("Supertype '%s' missing from compact YAML", entry.Type))
				continue
			}
			var expSubs []string
			for _, s := range entry.Subtypes {
				if s.Named && s.Type != "" {
					expSubs = append(expSubs, s.Type)
				}
			}
			sort.Strings(expSubs)
			sortedActSubs := make([]string, len(actSubs))
			copy(sortedActSubs, actSubs)
			sort.Strings(sortedActSubs)
			if !slices.Equal(expSubs, sortedActSubs) {
				discrepancies = append(discrepancies, fmt.Sprintf("Supertype '%s' subtypes mismatch: expected %v, got %v", entry.Type, expSubs, sortedActSubs))
			}
		} else {
			actNode, ok := nodes[entry.Type]
			if !ok {
				discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' missing from compact YAML", entry.Type))
				continue
			}

			// Validate fields
			if len(entry.Fields) > 0 {
				actFields := actNode.Fields
				expKeys := make(map[string]struct{}, len(entry.Fields))
				fieldNames := make([]string, 0, len(entry.Fields))
				for fname := range entry.Fields {
					fieldNames = append(fieldNames, fname)
				}
				sort.Strings(fieldNames)

				for _, fname := range fieldNames {
					fval := entry.Fields[fname]
					fkey := fname
					if !fval.Required {
						fkey = fname + "?"
					}
					expKeys[fkey] = struct{}{}
					actVal, exists := actFields[fkey]
					if !exists {
						discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' field '%s' missing from compact YAML", entry.Type, fkey))
						continue
					}
					var expTypes []string
					for _, t := range fval.Types {
						if t.Named && t.Type != "" {
							expTypes = append(expTypes, t.Type)
						}
					}
					sort.Strings(expTypes)
					expStr := "any"
					if len(expTypes) > 0 {
						expStr = strings.Join(expTypes, " | ")
					}
					if fval.Multiple {
						expStr = "[" + expStr + "]"
					}
					if actVal != expStr {
						discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' field '%s' mismatch: expected '%s', got '%s'", entry.Type, fkey, expStr, actVal))
					}
				}

				// Assert no extra fields in YAML
				var extraFieldKeys []string
				for actK := range actFields {
					if _, ok := expKeys[actK]; !ok {
						extraFieldKeys = append(extraFieldKeys, actK)
					}
				}
				sort.Strings(extraFieldKeys)
				for _, actK := range extraFieldKeys {
					discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' has unexpected extra field '%s' in compact YAML", entry.Type, actK))
				}
			} else {
				if len(actNode.Fields) > 0 {
					discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' should not have fields in compact YAML", entry.Type))
				}
			}

			// Validate children
			if entry.Children != nil {
				var expTypes []string
				for _, t := range entry.Children.Types {
					if t.Named && t.Type != "" {
						expTypes = append(expTypes, t.Type)
					}
				}
				sort.Strings(expTypes)
				expStr := "any"
				if len(expTypes) > 0 {
					expStr = strings.Join(expTypes, " | ")
				}
				if entry.Children.Multiple {
					expStr = "[" + expStr + "]"
				}
				if actNode.Children != expStr {
					discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' children mismatch: expected '%s', got '%s'", entry.Type, expStr, actNode.Children))
				}
			} else {
				if actNode.Children != "" {
					discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' should not have children in compact YAML", entry.Type))
				}
			}
		}
	}

	// Assert no unknown types in YAML
	var extraTypes []string
	for st := range supertypes {
		if _, ok := namedTypesInJSON[st]; !ok {
			extraTypes = append(extraTypes, st)
		}
	}
	for nd := range nodes {
		if _, ok := namedTypesInJSON[nd]; !ok {
			extraTypes = append(extraTypes, nd)
		}
	}
	sort.Strings(extraTypes)
	if len(extraTypes) > 0 {
		discrepancies = append(discrepancies, fmt.Sprintf("Compact YAML contains extra unknown types: %v", extraTypes))
	}

	return discrepancies, metrics, nil
}

// Compactor handles CLI generation and validation of compact node types.
type Compactor struct {
	reporter _jsii.Reporter
}

// NewCompactor constructs a default Compactor.
func NewCompactor() *Compactor {
	return NewCompactorWithReporter(nil)
}

// NewCompactorWithReporter constructs a Compactor with progress reporting.
func NewCompactorWithReporter(r _jsii.Reporter) *Compactor {
	if r == nil {
		r = _jsii.SilentReporter{}
	}
	return &Compactor{reporter: r}
}

// Compact generates or checks compact node types according to req.
func (c *Compactor) Compact(_ context.Context, cfg *_jsii.Config, req _jsii.CompactRequest) error {
	var targetLanguages []_jsii.Language

	if req.Language != "" {
		if _, statErr := os.Stat(req.Language); statErr == nil {
			// Direct file or directory path passed
			return c.processDirectPath(req.Language, req.Check)
		}
		langs, err := cfg.ResolveLanguages(req.Language)
		if err != nil {
			return err
		}
		targetLanguages = langs
	} else {
		targetLanguages = cfg.Languages
	}

	var errs []error
	for _, lang := range targetLanguages {
		start := time.Now()
		c.reporter.Start("compact", lang.Name)

		var nodeTypesPath string
		var outputDir string

		publishedPath := filepath.Join(cfg.Output.GrammarBase, lang.Name, "node-types.json")
		if info, statErr := os.Stat(publishedPath); statErr == nil && info.Mode().IsRegular() {
			nodeTypesPath = publishedPath
			outputDir = filepath.Dir(publishedPath)
		} else {
			source := sourceDir(cfg, lang)
			found, findErr := findNodeTypes(source)
			if findErr == nil {
				nodeTypesPath = found
				outputDir = source
			}
		}

		if nodeTypesPath == "" {
			err := fmt.Errorf("node-types.json not found for %s; build or move first", lang.Name)
			c.reporter.Failure("compact", lang.Name, err, "")
			errs = append(errs, err)
			continue
		}

		targetCompact := filepath.Join(outputDir, "compact-node-types.yaml")

		rawJSON, err := os.ReadFile(nodeTypesPath)
		if err != nil {
			err = fmt.Errorf("read %s: %w", nodeTypesPath, err)
			c.reporter.Failure("compact", lang.Name, err, "")
			errs = append(errs, err)
			continue
		}

		if req.Check {
			yamlBytes, err := os.ReadFile(targetCompact)
			if err != nil {
				err = fmt.Errorf("compact-node-types.yaml not found at %s: %w", targetCompact, err)
				c.reporter.Failure("compact", lang.Name, err, "")
				errs = append(errs, err)
				continue
			}

			discrepancies, metrics, valErr := ValidateCapture(rawJSON, string(yamlBytes))
			if valErr != nil {
				c.reporter.Failure("compact", lang.Name, valErr, "")
				errs = append(errs, fmt.Errorf("validate %s: %w", lang.Name, valErr))
				continue
			}

			if len(discrepancies) > 0 {
				diag := strings.Join(discrepancies, "\n")
				err := fmt.Errorf("compact validation failed for %s (%d discrepancies)", lang.Name, len(discrepancies))
				c.reporter.Failure("compact", lang.Name, err, diag)
				errs = append(errs, err)
				continue
			}

			detail := fmt.Sprintf("%d nodes, %d fields, %d supertypes, %.1f%% reduction",
				metrics.TotalNodes, metrics.TotalFields, metrics.TotalSupertypes, metrics.ReductionPct)
			c.reporter.Success("compact", lang.Name, detail, time.Since(start))
			c.reporter.Info("  -> VALIDATION PASSED: 100%% coverage verified (%d nodes, %d fields, %d supertypes, %d children, 0 errors)",
				metrics.TotalNodes, metrics.TotalFields, metrics.TotalSupertypes, metrics.TotalChildren)
			c.reporter.Info("  -> Reduced: %s lines (%s bytes) to %s lines (%s bytes) [%.1f%% reduction]",
				formatIntWithCommas(int64(countLines(rawJSON))),
				formatIntWithCommas(metrics.OriginalBytes),
				formatIntWithCommas(int64(countLines(yamlBytes))),
				formatIntWithCommas(metrics.CompactBytes),
				metrics.ReductionPct)
		} else {
			compactYAML, err := CompactNodeTypes(rawJSON)
			if err != nil {
				c.reporter.Failure("compact", lang.Name, err, "")
				errs = append(errs, fmt.Errorf("generate compact node types for %s: %w", lang.Name, err))
				continue
			}

			discrepancies, metrics, valErr := ValidateCapture(rawJSON, string(compactYAML))
			if valErr != nil {
				c.reporter.Failure("compact", lang.Name, valErr, "")
				errs = append(errs, fmt.Errorf("validate generated compact node types for %s: %w", lang.Name, valErr))
				continue
			}

			if len(discrepancies) > 0 {
				diag := strings.Join(discrepancies, "\n")
				err := fmt.Errorf("compact validation failed for %s (%d discrepancies)", lang.Name, len(discrepancies))
				c.reporter.Failure("compact", lang.Name, err, diag)
				errs = append(errs, err)
				continue
			}

			if err := os.WriteFile(targetCompact, compactYAML, 0o644); err != nil {
				c.reporter.Failure("compact", lang.Name, err, "")
				errs = append(errs, fmt.Errorf("write %s: %w", targetCompact, err))
				continue
			}

			detail := fmt.Sprintf("%d nodes, %d fields, %d supertypes, %.1f%% reduction",
				metrics.TotalNodes, metrics.TotalFields, metrics.TotalSupertypes, metrics.ReductionPct)
			c.reporter.Success("compact", lang.Name, detail, time.Since(start))
			c.reporter.Info("  -> Wrote: %s", targetCompact)
			c.reporter.Info("  -> Reduced: %s lines (%s bytes) to %s lines (%s bytes) [%.1f%% reduction]",
				formatIntWithCommas(int64(countLines(rawJSON))),
				formatIntWithCommas(metrics.OriginalBytes),
				formatIntWithCommas(int64(countLines(compactYAML))),
				formatIntWithCommas(metrics.CompactBytes),
				metrics.ReductionPct)
			c.reporter.Info("  -> VALIDATION PASSED: 100%% coverage verified (%d nodes, %d fields, %d supertypes, %d children, 0 errors)",
				metrics.TotalNodes, metrics.TotalFields, metrics.TotalSupertypes, metrics.TotalChildren)
		}
	}

	return errors.Join(errs...)
}

func (c *Compactor) processDirectPath(targetPath string, checkOnly bool) error {
	info, err := os.Stat(targetPath)
	if err != nil {
		return err
	}

	var nodeTypesPath string
	var outputCompactPath string
	if info.IsDir() {
		nodeTypesPath = filepath.Join(targetPath, "node-types.json")
		outputCompactPath = filepath.Join(targetPath, "compact-node-types.yaml")
	} else {
		nodeTypesPath = targetPath
		outputCompactPath = filepath.Join(filepath.Dir(targetPath), "compact-node-types.yaml")
	}

	rawJSON, err := os.ReadFile(nodeTypesPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", nodeTypesPath, err)
	}

	targetName := filepath.Base(filepath.Dir(nodeTypesPath))
	start := time.Now()
	c.reporter.Start("compact", targetName)

	if checkOnly {
		yamlBytes, err := os.ReadFile(outputCompactPath)
		if err != nil {
			err = fmt.Errorf("compact-node-types.yaml not found at %s: %w", outputCompactPath, err)
			c.reporter.Failure("compact", targetName, err, "")
			return err
		}

		discrepancies, metrics, valErr := ValidateCapture(rawJSON, string(yamlBytes))
		if valErr != nil {
			c.reporter.Failure("compact", targetName, valErr, "")
			return valErr
		}
		if len(discrepancies) > 0 {
			diag := strings.Join(discrepancies, "\n")
			err := fmt.Errorf("compact validation failed for %s (%d discrepancies)", targetName, len(discrepancies))
			c.reporter.Failure("compact", targetName, err, diag)
			return err
		}

		detail := fmt.Sprintf("%d nodes, %d fields, %d supertypes, %.1f%% reduction",
			metrics.TotalNodes, metrics.TotalFields, metrics.TotalSupertypes, metrics.ReductionPct)
		c.reporter.Success("compact", targetName, detail, time.Since(start))
		c.reporter.Info("  -> VALIDATION PASSED: 100%% coverage verified (%d nodes, %d fields, %d supertypes, %d children, 0 errors)",
			metrics.TotalNodes, metrics.TotalFields, metrics.TotalSupertypes, metrics.TotalChildren)
		c.reporter.Info("  -> Reduced: %s lines (%s bytes) to %s lines (%s bytes) [%.1f%% reduction]",
			formatIntWithCommas(int64(countLines(rawJSON))),
			formatIntWithCommas(metrics.OriginalBytes),
			formatIntWithCommas(int64(countLines(yamlBytes))),
			formatIntWithCommas(metrics.CompactBytes),
			metrics.ReductionPct)
		return nil
	}

	compactYAML, err := CompactNodeTypes(rawJSON)
	if err != nil {
		c.reporter.Failure("compact", targetName, err, "")
		return err
	}
	discrepancies, metrics, valErr := ValidateCapture(rawJSON, string(compactYAML))
	if valErr != nil {
		c.reporter.Failure("compact", targetName, valErr, "")
		return valErr
	}
	if len(discrepancies) > 0 {
		diag := strings.Join(discrepancies, "\n")
		err := fmt.Errorf("compact validation failed for %s (%d discrepancies)", targetName, len(discrepancies))
		c.reporter.Failure("compact", targetName, err, diag)
		return err
	}

	if err := os.WriteFile(outputCompactPath, compactYAML, 0o644); err != nil {
		c.reporter.Failure("compact", targetName, err, "")
		return fmt.Errorf("write %s: %w", outputCompactPath, err)
	}

	detail := fmt.Sprintf("%d nodes, %d fields, %d supertypes, %.1f%% reduction",
		metrics.TotalNodes, metrics.TotalFields, metrics.TotalSupertypes, metrics.ReductionPct)
	c.reporter.Success("compact", targetName, detail, time.Since(start))
	c.reporter.Info("  -> Wrote: %s", outputCompactPath)
	c.reporter.Info("  -> Reduced: %s lines (%s bytes) to %s lines (%s bytes) [%.1f%% reduction]",
		formatIntWithCommas(int64(countLines(rawJSON))),
		formatIntWithCommas(metrics.OriginalBytes),
		formatIntWithCommas(int64(countLines(compactYAML))),
		formatIntWithCommas(metrics.CompactBytes),
		metrics.ReductionPct)
	c.reporter.Info("  -> VALIDATION PASSED: 100% coverage verified (%d nodes, %d fields, %d supertypes, %d children, 0 errors)",
		metrics.TotalNodes, metrics.TotalFields, metrics.TotalSupertypes, metrics.TotalChildren)
	return nil
}

func countLines(b []byte) int {
	trimmed := strings.TrimRight(string(b), "\r\n")
	if len(trimmed) == 0 {
		return 0
	}
	return len(strings.Split(trimmed, "\n"))
}

func formatIntWithCommas(n int64) string {
	in := strconv.FormatInt(n, 10)
	var out []byte
	l := len(in)
	for i, c := range in {
		if i > 0 && (l-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}
