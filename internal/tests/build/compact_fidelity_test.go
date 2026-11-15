package build_test

import (
	build "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompactPublishedCatalogFidelity(t *testing.T) {
	t.Parallel()
	paths, err := filepath.Glob(filepath.Join(testRepoRoot(t), "data", "tree-sitter", "grammar", "*", "node-types.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no published node-types.json fixtures found")
	}
	for _, path := range paths {
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			published, err := os.ReadFile(filepath.Join(filepath.Dir(path), "compact-node-types.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			generated, err := build.CompactNodeTypes(raw)
			if err != nil {
				t.Fatal(err)
			}
			if string(generated) != string(published) {
				t.Fatal("published compact schema differs from fresh generation")
			}
			discrepancies, _, err := build.ValidateCapture(raw, string(published))
			if err != nil || len(discrepancies) != 0 {
				t.Fatalf("published schema failed fidelity validation: %v %v", discrepancies, err)
			}
		})
	}
}

func TestCompactStructuredIdentities(t *testing.T) {
	raw := []byte(`[
 {"type":"same","named":true,"fields":{"value":{"required":true,"multiple":false,"types":[{"type":"id","named":true}]}}},
 {"type":"same","named":false,"extra":true,"fields":{"value":{"required":false,"multiple":true,"types":[{"type":"id","named":true}]}},"children":{"required":true,"multiple":false,"types":[{"type":"id","named":true}]}},
 {"type":"id","named":true}
 ]`)
	compact, err := build.CompactNodeTypes(raw)
	if err != nil {
		t.Fatal(err)
	}
	ds, _, err := build.ValidateCapture(raw, string(compact))
	if err != nil || len(ds) > 0 {
		t.Fatalf("distinct named/anonymous identities failed: %v %v", ds, err)
	}
	for _, tc := range []struct{ name, from, to string }{
		{"anonymous field types", "value?: [id]", "value?: [wrong]"},
		{"anonymous field requiredness", "value?: [id]", "value: [id]"},
		{"anonymous field multiplicity", "value?: [id]", "value?: id"},
		{"anonymous children", "children: id", "children: wrong"},
		{"anonymous children requiredness", "children: id", "children?: id"},
		{"anonymous children multiplicity", "children: id", "children: [id]"},
		{"anonymous named flag", "    named: false\n", "    named: true\n"},
		{"named node flag", "  same:\n", "  same:\n    named: false\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.Replace(string(compact), tc.from, tc.to, 1)
			if changed == string(compact) {
				t.Fatal("mutation did not apply")
			}
			ds, _, err := build.ValidateCapture(raw, changed)
			if err == nil && len(ds) == 0 {
				t.Fatal("corrupted schema passed validation")
			}
		})
	}
}

func TestCompactRejectsUnsupportedRawDeclarations(t *testing.T) {
	for _, tc := range []struct{ name, raw string }{
		{"duplicate identity", `[{"type":"x","named":true},{"type":"x","named":true}]`},
		{"supertype fields", `[{"type":"x","named":true,"subtypes":[],"fields":{"v":{"required":true,"multiple":false,"types":[]}}}]`},
		{"supertype children", `[{"type":"x","named":true,"subtypes":[],"children":{"required":false,"multiple":true,"types":[]}}]`},
		{"anonymous supertype", `[{"type":"x","named":false,"subtypes":[]}]`},
		{"empty declaration", `[{"type":"","named":true}]`},
		{"empty subtype", `[{"type":"x","named":true,"subtypes":[{"type":"","named":true}]}]`},
		{"trailing JSON", `[{"type":"x","named":true}] []`},
		{"quoted named identity collision", `[{"type":"\"x\"","named":true},{"type":"x","named":false,"extra":true}]`},
		{"reserved named type", `[{"type":"any","named":true}]`},
		{"ambiguous named reference", `[{"type":"x","named":true,"subtypes":[{"type":"a | b","named":true}]}]`},
		{"duplicate node property", `[{"type":"x","type":"y","named":true}]`},
		{"duplicate field property", `[{"type":"x","named":true,"fields":{"v":{"types":[]},"v":{"types":[]}}}]`},
		{"duplicate descriptor property", `[{"type":"x","named":true,"children":{"required":true,"required":false,"types":[]}}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := build.CompactNodeTypes([]byte(tc.raw)); err == nil {
				t.Fatal("unsupported raw schema accepted")
			}
			if _, _, err := build.ValidateCapture([]byte(tc.raw), "tokens: []\nsupertypes:\nnodes:\n"); err == nil {
				t.Fatal("validator accepted unsupported raw schema")
			}
		})
	}
}

func TestCompactRejectsDuplicateDeclarations(t *testing.T) {
	for _, tc := range []struct{ name, yaml string }{
		{"tokens", "tokens: [\"x\", \"x\"]\n"},
		{"nodes", "nodes:\n  x: {}\n  x: {}\n"},
		{"supertypes", "supertypes:\n  x: [id]\n  x: [id]\n"},
		{"sections", "nodes:\n  x: {}\nnodes:\n  y: {}\n"},
		{"fields", "nodes:\n  x:\n    fields:\n      v: id\n      v: id\n"},
		{"required optional field collision", "nodes:\n  x:\n    fields:\n      v: id\n      v?: id\n"},
		{"children variants", "nodes:\n  x:\n    children: id\n    children?: id\n"},
		{"metadata", "nodes:\n  x:\n    extra: true\n    extra: true\n"},
		{"token and node", "tokens: [\"x\"]\nnodes:\n  '\"x\"':\n    named: false\n"},
		{"supertype and node", "supertypes:\n  x: [id]\nnodes:\n  x: {}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := build.ParseCompactDoc(tc.yaml); err == nil {
				t.Fatal("duplicate compact declaration accepted")
			}
		})
	}
}

func TestCompactCSharpNamedFlagRegression(t *testing.T) {
	root := filepath.Join(testRepoRoot(t), "data", "tree-sitter", "grammar", "csharp")
	raw, err := os.ReadFile(filepath.Join(root, "node-types.json"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := build.CompactNodeTypes(raw)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(c), "  method_declaration:\n", "  method_declaration:\n    named: false\n", 1)
	if changed == string(c) {
		t.Fatal("mutation did not apply")
	}
	ds, _, err := build.ValidateCapture(raw, changed)
	if err == nil && len(ds) == 0 {
		t.Fatal("C# method incorrectly marked anonymous passed validation")
	}
}
