package build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	Root     bool                          `json:"root,omitempty"`
	Extra    bool                          `json:"extra,omitempty"`
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

// CompactSupertype represents the compact schema for a supertype.
type CompactSupertype struct {
	Subtypes []string `json:"subtypes"`
	Root     bool     `json:"root,omitempty"`
	Extra    bool     `json:"extra,omitempty"`
}

// CompactNode represents the compact schema for a node in YAML.
type CompactNode struct {
	Named            *bool             `json:"named,omitempty"`
	Root             bool              `json:"root,omitempty"`
	Extra            bool              `json:"extra,omitempty"`
	Fields           map[string]string `json:"fields,omitempty"`
	Children         string            `json:"children,omitempty"`
	ChildrenOptional bool              `json:"children_optional,omitempty"`
}

// CompactParsed represents the parsed compact YAML document.
type CompactParsed struct {
	Tokens     []string
	Supertypes map[string]CompactSupertype
	Nodes      map[string]CompactNode
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

func parseRawNodeTypesStrict(rawJSON []byte) ([]RawNodeTypeEntry, error) {
	dec := json.NewDecoder(bytes.NewReader(rawJSON))
	dec.DisallowUnknownFields()
	var rawData []RawNodeTypeEntry
	if err := dec.Decode(&rawData); err != nil {
		return nil, fmt.Errorf("parse raw node-types.json: unsupported or malformed property: %w", err)
	}
	return rawData, nil
}

func formatTypeChild(t RawTypeChild) string {
	if t.Named {
		return t.Type
	}
	return strconv.Quote(t.Type)
}

func formatTypeExpression(types []RawTypeChild, multiple bool) string {
	if len(types) == 0 {
		if multiple {
			return "[any]"
		}
		return "any"
	}
	formatted := make([]string, len(types))
	for i, t := range types {
		formatted[i] = formatTypeChild(t)
	}
	sort.Strings(formatted)
	typeStr := strings.Join(formatted, " | ")
	if multiple {
		typeStr = "[" + typeStr + "]"
	}
	return typeStr
}

func needsYAMLQuotes(s string) bool {
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	switch lower {
	case "true", "false", "yes", "no", "on", "off", "null", "~":
		return true
	}
	if strings.ContainsAny(s, "\",:{}\n\r\t") || strings.HasPrefix(s, "\"") || strings.HasPrefix(s, "'") || strings.HasPrefix(s, "@") || strings.HasPrefix(s, "`") {
		return true
	}
	return false
}

func formatYAMLVal(val string) string {
	if needsYAMLQuotes(val) {
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	}
	return val
}

func formatYAMLKey(key string) string {
	if needsYAMLQuotes(key) {
		return "'" + strings.ReplaceAll(key, "'", "''") + "'"
	}
	return key
}

func unformatYAMLVal(val string) string {
	trimmed := strings.TrimSpace(val)
	if strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") && len(trimmed) >= 2 {
		return strings.ReplaceAll(trimmed[1:len(trimmed)-1], "''", "'")
	}
	return trimmed
}

func unformatYAMLKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") && len(trimmed) >= 2 {
		unquoted := strings.ReplaceAll(trimmed[1:len(trimmed)-1], "''", "'")
		if strings.HasPrefix(unquoted, "\"") && strings.HasSuffix(unquoted, "\"") && len(unquoted) >= 2 {
			if s, err := strconv.Unquote(unquoted); err == nil {
				return s
			}
		}
		return unquoted
	}
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") && len(trimmed) >= 2 {
		if s, err := strconv.Unquote(trimmed); err == nil {
			return s
		}
	}
	return trimmed
}

func isQuotedAnonymousKey(key string) bool {
	trimmed := strings.TrimSpace(key)
	if strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") && len(trimmed) >= 2 {
		trimmed = strings.ReplaceAll(trimmed[1:len(trimmed)-1], "''", "'")
	}
	return strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") && len(trimmed) >= 2
}

func splitCommaSeparated(s string) []string {
	var items []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuotes {
			current.WriteByte(c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inQuotes = false
			}
		} else {
			if c == '"' {
				inQuotes = true
				current.WriteByte(c)
			} else if c == ',' {
				item := strings.TrimSpace(current.String())
				if item != "" {
					items = append(items, item)
				}
				current.Reset()
			} else {
				current.WriteByte(c)
			}
		}
	}
	item := strings.TrimSpace(current.String())
	if item != "" {
		items = append(items, item)
	}
	return items
}

func splitTypeAlternatives(s string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuotes {
			current.WriteByte(c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inQuotes = false
			}
		} else {
			if c == '"' {
				inQuotes = true
				current.WriteByte(c)
			} else if c == '|' {
				part := strings.TrimSpace(current.String())
				if part != "" {
					parts = append(parts, part)
				}
				current.Reset()
			} else {
				current.WriteByte(c)
			}
		}
	}
	part := strings.TrimSpace(current.String())
	if part != "" {
		parts = append(parts, part)
	}
	return parts
}

func decodeTypeAlternatives(expr string) (multiple bool, types []RawTypeChild) {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "[") && strings.HasSuffix(expr, "]") {
		multiple = true
		expr = strings.TrimSpace(expr[1 : len(expr)-1])
	}
	if expr == "" || expr == "any" {
		return multiple, nil
	}
	parts := splitTypeAlternatives(expr)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "\"") && strings.HasSuffix(p, "\"") && len(p) >= 2 {
			unquoted, err := strconv.Unquote(p)
			if err == nil {
				types = append(types, RawTypeChild{Type: unquoted, Named: false})
				continue
			}
		}
		types = append(types, RawTypeChild{Type: p, Named: true})
	}
	return multiple, types
}

func compareTypeChildSlices(a, b []RawTypeChild) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA := make([]RawTypeChild, len(a))
	copy(sortedA, a)
	sort.Slice(sortedA, func(i, j int) bool {
		if sortedA[i].Named != sortedA[j].Named {
			return sortedA[i].Named
		}
		return sortedA[i].Type < sortedA[j].Type
	})

	sortedB := make([]RawTypeChild, len(b))
	copy(sortedB, b)
	sort.Slice(sortedB, func(i, j int) bool {
		if sortedB[i].Named != sortedB[j].Named {
			return sortedB[i].Named
		}
		return sortedB[i].Type < sortedB[j].Type
	})

	for i := range sortedA {
		if sortedA[i].Type != sortedB[i].Type || sortedA[i].Named != sortedB[i].Named {
			return false
		}
	}
	return true
}

func parseSubtypesList(val string) ([]string, error) {
	if !strings.HasPrefix(val, "[") || !strings.HasSuffix(val, "]") {
		return nil, fmt.Errorf("malformed subtypes list: expected [A, B], got %q", val)
	}
	inner := strings.TrimSpace(val[1 : len(val)-1])
	if inner == "" {
		return nil, nil
	}
	items := splitCommaSeparated(inner)
	result := make([]string, len(items))
	for i, it := range items {
		result[i] = it
	}
	return result, nil
}

// CompactNodeTypes transforms raw node-types.json bytes into a token-efficient compact YAML representation.
func CompactNodeTypes(rawJSON []byte) ([]byte, error) {
	rawData, err := parseRawNodeTypesStrict(rawJSON)
	if err != nil {
		return nil, err
	}

	var bareTokens []string
	supertypes := make(map[string]CompactSupertype)
	nodes := make(map[string]CompactNode)

	for _, entry := range rawData {
		if entry.Type == "" {
			continue
		}

		if entry.Subtypes != nil {
			var subtypes []string
			for _, s := range entry.Subtypes {
				if s.Type != "" {
					subtypes = append(subtypes, formatTypeChild(s))
				}
			}
			sort.Strings(subtypes)
			supertypes[entry.Type] = CompactSupertype{
				Subtypes: subtypes,
				Root:     entry.Root,
				Extra:    entry.Extra,
			}
			continue
		}

		if !entry.Named {
			// Check if this anonymous entry has any fields, children, or metadata
			hasFields := len(entry.Fields) > 0
			hasChildren := entry.Children != nil
			hasMetadata := entry.Root || entry.Extra
			if !hasFields && !hasChildren && !hasMetadata {
				bareTokens = append(bareTokens, strconv.Quote(entry.Type))
				continue
			}
		}

		var nodeData CompactNode
		if !entry.Named {
			f := false
			nodeData.Named = &f
		}
		nodeData.Root = entry.Root
		nodeData.Extra = entry.Extra

		if len(entry.Fields) > 0 {
			nodeData.Fields = make(map[string]string, len(entry.Fields))
			for fname, fval := range entry.Fields {
				typeStr := formatTypeExpression(fval.Types, fval.Multiple)
				fieldKey := fname
				if !fval.Required {
					fieldKey = fname + "?"
				}
				nodeData.Fields[fieldKey] = typeStr
			}
		}

		if entry.Children != nil {
			nodeData.Children = formatTypeExpression(entry.Children.Types, entry.Children.Multiple)
			nodeData.ChildrenOptional = !entry.Children.Required
		}

		nodeKey := entry.Type
		if !entry.Named {
			nodeKey = strconv.Quote(entry.Type)
		}
		nodes[nodeKey] = nodeData
	}

	sort.Strings(bareTokens)

	var b strings.Builder
	b.WriteString("# Compact Tree-sitter Node Types (Generated from node-types.json)\n")
	b.WriteString("# Legend:\n")
	b.WriteString("#   tokens: [...]   : inventory of bare anonymous tokens\n")
	b.WriteString("#   name            : named node declaration or reference\n")
	b.WriteString("#   \"token\"         : anonymous token declaration or reference\n")
	b.WriteString("#   key?            : optional field or children (omitted ? means required)\n")
	b.WriteString("#   [T]             : multiple occurrences allowed (omitted brackets means single)\n")
	b.WriteString("#   T1 | T2         : alternative types\n")
	b.WriteString("#   root: true      : grammar root node\n")
	b.WriteString("#   extra: true     : extra node (can appear anywhere, e.g. comments, whitespace)\n\n")

	b.WriteString("tokens: [")
	b.WriteString(strings.Join(bareTokens, ", "))
	b.WriteString("]\n\n")

	b.WriteString("supertypes:\n")
	supertypeKeys := make([]string, 0, len(supertypes))
	for k := range supertypes {
		supertypeKeys = append(supertypeKeys, k)
	}
	sort.Strings(supertypeKeys)
	for _, k := range supertypeKeys {
		st := supertypes[k]
		kFormatted := formatYAMLKey(k)
		if !st.Root && !st.Extra {
			b.WriteString(fmt.Sprintf("  %s: [%s]\n", kFormatted, strings.Join(st.Subtypes, ", ")))
		} else {
			b.WriteString(fmt.Sprintf("  %s:\n", kFormatted))
			if st.Root {
				b.WriteString("    root: true\n")
			}
			if st.Extra {
				b.WriteString("    extra: true\n")
			}
			b.WriteString(fmt.Sprintf("    subtypes: [%s]\n", strings.Join(st.Subtypes, ", ")))
		}
	}

	b.WriteString("\nnodes:\n")
	nodeKeys := make([]string, 0, len(nodes))
	for k := range nodes {
		nodeKeys = append(nodeKeys, k)
	}
	sort.Strings(nodeKeys)
	for _, k := range nodeKeys {
		data := nodes[k]
		kFormatted := formatYAMLKey(k)

		hasFields := len(data.Fields) > 0
		hasChildren := data.Children != ""
		hasMetadata := data.Root || data.Extra || (data.Named != nil && !*data.Named)

		if !hasFields && !hasChildren && !hasMetadata {
			b.WriteString(fmt.Sprintf("  %s: {}\n", kFormatted))
			continue
		}

		b.WriteString(fmt.Sprintf("  %s:\n", kFormatted))
		if data.Named != nil && !*data.Named {
			b.WriteString("    named: false\n")
		}
		if data.Root {
			b.WriteString("    root: true\n")
		}
		if data.Extra {
			b.WriteString("    extra: true\n")
		}
		if hasFields {
			b.WriteString("    fields:\n")
			fkeys := make([]string, 0, len(data.Fields))
			for fk := range data.Fields {
				fkeys = append(fkeys, fk)
			}
			sort.Strings(fkeys)
			for _, fk := range fkeys {
				fval := formatYAMLVal(data.Fields[fk])
				b.WriteString(fmt.Sprintf("      %s: %s\n", fk, fval))
			}
		}
		if hasChildren {
			cval := formatYAMLVal(data.Children)
			ckey := "children"
			if data.ChildrenOptional {
				ckey = "children?"
			}
			b.WriteString(fmt.Sprintf("    %s: %s\n", ckey, cval))
		}
	}

	return []byte(b.String()), nil
}

// ParseCompactDoc parses the compact node types YAML into a full CompactParsed structure.
func ParseCompactDoc(yamlText string) (*CompactParsed, error) {
	doc := &CompactParsed{
		Tokens:     make([]string, 0),
		Supertypes: make(map[string]CompactSupertype),
		Nodes:      make(map[string]CompactNode),
	}

	var currentSection string
	var currentSupertype string
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
			currentSupertype = ""
			currentNode = ""
			inFields = false

			if strings.HasPrefix(stripped, "tokens:") {
				_, rest, ok := strings.Cut(stripped, ":")
				if !ok {
					return nil, fmt.Errorf("line %d: malformed tokens entry %q", lineIdx+1, stripped)
				}
				val := strings.TrimSpace(rest)
				if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
					inner := strings.TrimSpace(val[1 : len(val)-1])
					if inner != "" {
						items := splitCommaSeparated(inner)
						for _, item := range items {
							tok, err := strconv.Unquote(item)
							if err != nil {
								tok = item
							}
							doc.Tokens = append(doc.Tokens, tok)
						}
					}
				}
				continue
			} else if stripped == "supertypes:" {
				currentSection = "supertypes"
				continue
			} else if stripped == "nodes:" {
				currentSection = "nodes"
				continue
			} else {
				return nil, fmt.Errorf("line %d: unexpected top-level section %q", lineIdx+1, stripped)
			}
		}

		if currentSection == "supertypes" {
			if indent == 2 {
				key, rest, ok := strings.Cut(stripped, ":")
				if !ok {
					return nil, fmt.Errorf("line %d: malformed supertype entry %q", lineIdx+1, stripped)
				}
				cleanKey := unformatYAMLKey(key)
				val := strings.TrimSpace(rest)
				if val == "" {
					currentSupertype = cleanKey
					doc.Supertypes[cleanKey] = CompactSupertype{}
				} else {
					currentSupertype = ""
					subtypes, err := parseSubtypesList(val)
					if err != nil {
						return nil, fmt.Errorf("line %d: %w", lineIdx+1, err)
					}
					doc.Supertypes[cleanKey] = CompactSupertype{Subtypes: subtypes}
				}
			} else if indent == 4 && currentSupertype != "" {
				st := doc.Supertypes[currentSupertype]
				if strings.HasPrefix(stripped, "subtypes:") {
					_, rest, ok := strings.Cut(stripped, ":")
					if !ok {
						return nil, fmt.Errorf("line %d: malformed subtypes list in supertype %q", lineIdx+1, currentSupertype)
					}
					subtypes, err := parseSubtypesList(strings.TrimSpace(rest))
					if err != nil {
						return nil, fmt.Errorf("line %d: %w", lineIdx+1, err)
					}
					st.Subtypes = subtypes
				} else if stripped == "root: true" {
					st.Root = true
				} else if stripped == "extra: true" {
					st.Extra = true
				} else {
					return nil, fmt.Errorf("line %d: unexpected entry under supertype %q: %q", lineIdx+1, currentSupertype, stripped)
				}
				doc.Supertypes[currentSupertype] = st
			} else {
				return nil, fmt.Errorf("line %d: invalid indentation in supertypes: %q", lineIdx+1, stripped)
			}
		} else if currentSection == "nodes" {
			if indent == 2 {
				inFields = false
				if strings.HasSuffix(stripped, ": {}") {
					rawKey := strings.TrimSpace(strings.TrimSuffix(stripped, ": {}"))
					cleanKey := unformatYAMLKey(rawKey)
					currentNode = cleanKey
					node := CompactNode{}
					if isQuotedAnonymousKey(rawKey) {
						f := false
						node.Named = &f
					}
					doc.Nodes[currentNode] = node
				} else if strings.HasSuffix(stripped, ":") {
					rawKey := strings.TrimSpace(strings.TrimSuffix(stripped, ":"))
					cleanKey := unformatYAMLKey(rawKey)
					currentNode = cleanKey
					node := CompactNode{}
					if isQuotedAnonymousKey(rawKey) {
						f := false
						node.Named = &f
					}
					doc.Nodes[currentNode] = node
				} else {
					return nil, fmt.Errorf("line %d: malformed node entry %q", lineIdx+1, stripped)
				}
			} else if indent == 4 && currentNode != "" {
				node := doc.Nodes[currentNode]
				if stripped == "fields:" {
					inFields = true
					if node.Fields == nil {
						node.Fields = make(map[string]string)
					}
					doc.Nodes[currentNode] = node
				} else if strings.HasPrefix(stripped, "children:") || strings.HasPrefix(stripped, "children?:") {
					inFields = false
					key, rest, ok := strings.Cut(stripped, ":")
					if !ok {
						return nil, fmt.Errorf("line %d: malformed children entry %q", lineIdx+1, stripped)
					}
					node.Children = unformatYAMLVal(strings.TrimSpace(rest))
					node.ChildrenOptional = strings.TrimSpace(key) == "children?"
					doc.Nodes[currentNode] = node
				} else if stripped == "root: true" {
					inFields = false
					node.Root = true
					doc.Nodes[currentNode] = node
				} else if stripped == "extra: true" {
					inFields = false
					node.Extra = true
					doc.Nodes[currentNode] = node
				} else if stripped == "named: false" {
					inFields = false
					f := false
					node.Named = &f
					doc.Nodes[currentNode] = node
				} else {
					return nil, fmt.Errorf("line %d: unexpected entry under node %q: %q", lineIdx+1, currentNode, stripped)
				}
			} else if indent == 6 && currentNode != "" && inFields {
				fkey, fval, ok := strings.Cut(stripped, ":")
				if !ok {
					return nil, fmt.Errorf("line %d: malformed field entry %q", lineIdx+1, stripped)
				}
				node := doc.Nodes[currentNode]
				if node.Fields == nil {
					node.Fields = make(map[string]string)
				}
				cleanVal := unformatYAMLVal(strings.TrimSpace(fval))
				node.Fields[strings.TrimSpace(fkey)] = cleanVal
				doc.Nodes[currentNode] = node
			} else {
				return nil, fmt.Errorf("line %d: invalid indentation or structure %q", lineIdx+1, stripped)
			}
		} else {
			return nil, fmt.Errorf("line %d: unexpected indentation or structure %q", lineIdx+1, stripped)
		}
	}

	return doc, nil
}

// ParseCompactYAML parses the compact node types YAML into supertypes and nodes maps.
func ParseCompactYAML(yamlText string) (supertypes map[string][]string, nodes map[string]CompactNode, err error) {
	doc, err := ParseCompactDoc(yamlText)
	if err != nil {
		return nil, nil, err
	}
	supertypes = make(map[string][]string, len(doc.Supertypes))
	for k, st := range doc.Supertypes {
		supertypes[k] = st.Subtypes
	}
	return supertypes, doc.Nodes, nil
}

// ValidateCapture performs strict lossless validation that compact YAML preserves 100% of raw JSON AST schema.
func ValidateCapture(rawJSON []byte, yamlText string) (discrepancies []string, metrics CompactMetrics, err error) {
	rawData, err := parseRawNodeTypesStrict(rawJSON)
	if err != nil {
		return nil, metrics, err
	}

	doc, err := ParseCompactDoc(yamlText)
	if err != nil {
		return nil, metrics, fmt.Errorf("parse compact YAML: %w", err)
	}

	metrics.TotalNodes = len(doc.Nodes)
	metrics.TotalSupertypes = len(doc.Supertypes)
	for _, n := range doc.Nodes {
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

	tokensInYAML := make(map[string]struct{}, len(doc.Tokens))
	for _, t := range doc.Tokens {
		tokensInYAML[t] = struct{}{}
	}

	namedTypesInJSON := make(map[string]struct{})
	anonTokensInJSON := make(map[string]struct{})

	for _, entry := range rawData {
		if entry.Type == "" {
			continue
		}

		if !entry.Named {
			anonTokensInJSON[entry.Type] = struct{}{}
			hasFields := len(entry.Fields) > 0
			hasChildren := entry.Children != nil
			hasSubtypes := len(entry.Subtypes) > 0
			hasMetadata := entry.Root || entry.Extra
			if !hasFields && !hasChildren && !hasSubtypes && !hasMetadata {
				if _, ok := tokensInYAML[entry.Type]; !ok {
					discrepancies = append(discrepancies, fmt.Sprintf("Anonymous token %q missing from compact YAML tokens", entry.Type))
				}
				continue
			}
			// Anonymous node with metadata or fields or children
			actNode, ok := doc.Nodes[entry.Type]
			if !ok {
				discrepancies = append(discrepancies, fmt.Sprintf("Anonymous node %q missing from compact YAML", entry.Type))
				continue
			}
			if actNode.Root != entry.Root {
				discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' root mismatch: expected %v, got %v", entry.Type, entry.Root, actNode.Root))
			}
			if actNode.Extra != entry.Extra {
				discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' extra mismatch: expected %v, got %v", entry.Type, entry.Extra, actNode.Extra))
			}
			continue
		}

		namedTypesInJSON[entry.Type] = struct{}{}

		if entry.Subtypes != nil {
			st, ok := doc.Supertypes[entry.Type]
			if !ok {
				discrepancies = append(discrepancies, fmt.Sprintf("Supertype '%s' missing from compact YAML", entry.Type))
				continue
			}
			if st.Root != entry.Root {
				discrepancies = append(discrepancies, fmt.Sprintf("Supertype '%s' root mismatch: expected %v, got %v", entry.Type, entry.Root, st.Root))
			}
			if st.Extra != entry.Extra {
				discrepancies = append(discrepancies, fmt.Sprintf("Supertype '%s' extra mismatch: expected %v, got %v", entry.Type, entry.Extra, st.Extra))
			}
			actSubtypes := make([]RawTypeChild, 0, len(st.Subtypes))
			for _, s := range st.Subtypes {
				_, types := decodeTypeAlternatives(s)
				actSubtypes = append(actSubtypes, types...)
			}
			if !compareTypeChildSlices(entry.Subtypes, actSubtypes) {
				expSubs := make([]string, len(entry.Subtypes))
				for i, s := range entry.Subtypes {
					expSubs[i] = formatTypeChild(s)
				}
				sort.Strings(expSubs)
				discrepancies = append(discrepancies, fmt.Sprintf("Supertype '%s' subtypes mismatch: expected %v, got %v", entry.Type, expSubs, st.Subtypes))
			}
		} else {
			actNode, ok := doc.Nodes[entry.Type]
			if !ok {
				discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' missing from compact YAML", entry.Type))
				continue
			}
			if actNode.Root != entry.Root {
				discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' root mismatch: expected %v, got %v", entry.Type, entry.Root, actNode.Root))
			}
			if actNode.Extra != entry.Extra {
				discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' extra mismatch: expected %v, got %v", entry.Type, entry.Extra, actNode.Extra))
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
					actMultiple, actTypes := decodeTypeAlternatives(actVal)
					if actMultiple != fval.Multiple || !compareTypeChildSlices(fval.Types, actTypes) {
						expStr := formatTypeExpression(fval.Types, fval.Multiple)
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
				if actNode.Children == "" {
					discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' children missing from compact YAML", entry.Type))
				} else {
					expOptional := !entry.Children.Required
					if actNode.ChildrenOptional != expOptional {
						discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' children mismatch: expected optional=%v, got optional=%v", entry.Type, expOptional, actNode.ChildrenOptional))
					}
					actMultiple, actTypes := decodeTypeAlternatives(actNode.Children)
					if actMultiple != entry.Children.Multiple || !compareTypeChildSlices(entry.Children.Types, actTypes) {
						expStr := formatTypeExpression(entry.Children.Types, entry.Children.Multiple)
						discrepancies = append(discrepancies, fmt.Sprintf("Node '%s' children mismatch: expected '%s', got '%s'", entry.Type, expStr, actNode.Children))
					}
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
	for st := range doc.Supertypes {
		if _, ok := namedTypesInJSON[st]; !ok {
			extraTypes = append(extraTypes, st)
		}
	}
	for nd := range doc.Nodes {
		if _, ok := namedTypesInJSON[nd]; !ok {
			if _, okAnon := anonTokensInJSON[nd]; !okAnon {
				extraTypes = append(extraTypes, nd)
			}
		}
	}
	sort.Strings(extraTypes)
	if len(extraTypes) > 0 {
		discrepancies = append(discrepancies, fmt.Sprintf("Compact YAML contains extra unknown types: %v", extraTypes))
	}

	var extraTokens []string
	for tok := range tokensInYAML {
		if _, ok := anonTokensInJSON[tok]; !ok {
			extraTokens = append(extraTokens, tok)
		}
	}
	sort.Strings(extraTokens)
	if len(extraTokens) > 0 {
		discrepancies = append(discrepancies, fmt.Sprintf("Compact YAML contains extra unknown tokens: %v", extraTokens))
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
