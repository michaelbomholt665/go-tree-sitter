package build

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_jsii "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
)

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller path")
	}
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found above %q", filename)
		}
		dir = parent
	}
}

func TestCompactNodeTypesRealFixtures(t *testing.T) {
	t.Parallel()
	root := testRepoRoot(t)

	// Fixtures explicitly called out in task doc 010 plus all available grammars
	languages := []string{"go", "python", "rust", "typescript", "c", "cpp", "c-sharp", "java", "javascript", "json", "yaml"}

	for _, lang := range languages {
		lang := lang
		t.Run(lang, func(t *testing.T) {
			t.Parallel()
			nodeTypesPath := filepath.Join(root, "data", "tree-sitter", "grammar", lang, "node-types.json")
			rawJSON, err := os.ReadFile(nodeTypesPath)
			if err != nil {
				t.Fatalf("read node-types.json for %s: %v", lang, err)
			}

			compactYAML, err := CompactNodeTypes(rawJSON)
			if err != nil {
				t.Fatalf("CompactNodeTypes failed for %s: %v", lang, err)
			}

			// Validate lossless AST preservation
			discrepancies, metrics, err := ValidateCapture(rawJSON, string(compactYAML))
			if err != nil {
				t.Fatalf("ValidateCapture failed for %s: %v", lang, err)
			}
			if len(discrepancies) > 0 {
				t.Fatalf("expected 0 discrepancies for %s, got %d: %s", lang, len(discrepancies), strings.Join(discrepancies, "; "))
			}

			// Verify reduction criteria (> 65% reduction)
			if metrics.ReductionPct < 65.0 {
				t.Errorf("expected > 65%% reduction for %s, got %.2f%%", lang, metrics.ReductionPct)
			}

			// Verify metrics sanity
			if metrics.TotalNodes == 0 {
				t.Errorf("expected > 0 nodes for %s, got 0", lang)
			}
			if metrics.CompactBytes >= metrics.OriginalBytes {
				t.Errorf("expected compact bytes (%d) < original bytes (%d)", metrics.CompactBytes, metrics.OriginalBytes)
			}
		})
	}
}

func TestValidateCaptureDetectsDiscrepancies(t *testing.T) {
	t.Parallel()
	root := testRepoRoot(t)
	rawJSON, err := os.ReadFile(filepath.Join(root, "data", "tree-sitter", "grammar", "python", "node-types.json"))
	if err != nil {
		t.Fatalf("read python node-types.json: %v", err)
	}

	validYAML, err := CompactNodeTypes(rawJSON)
	if err != nil {
		t.Fatalf("generate compact YAML: %v", err)
	}

	tests := []struct {
		name           string
		mutateYAML     func(string) string
		expectedSubstr string
	}{
		{
			name: "missing supertype",
			mutateYAML: func(y string) string {
				return strings.Replace(y, "  _compound_statement: [", "  # _compound_statement: [", 1)
			},
			expectedSubstr: "Supertype '_compound_statement' missing from compact YAML",
		},
		{
			name: "mismatched supertype subtypes",
			mutateYAML: func(y string) string {
				return strings.Replace(y, "  _compound_statement: [class_definition,", "  _compound_statement: [bogus_definition,", 1)
			},
			expectedSubstr: "Supertype '_compound_statement' subtypes mismatch",
		},
		{
			name: "missing node",
			mutateYAML: func(y string) string {
				return strings.Replace(y, "  function_definition:\n", "  # function_definition:\n", 1)
			},
			expectedSubstr: "Node 'function_definition' missing from compact YAML",
		},
		{
			name: "missing field",
			mutateYAML: func(y string) string {
				// Remove return_type? from function_definition
				return strings.Replace(y, "      return_type?: type\n", "", 1)
			},
			expectedSubstr: "field 'return_type?' missing from compact YAML",
		},
		{
			name: "field type mismatch",
			mutateYAML: func(y string) string {
				return strings.Replace(y, "      return_type?: type\n", "      return_type?: wrong_type\n", 1)
			},
			expectedSubstr: "field 'return_type?' mismatch",
		},
		{
			name: "field plurality mismatch",
			mutateYAML: func(y string) string {
				return strings.Replace(y, "      return_type?: type\n", "      return_type?: [type]\n", 1)
			},
			expectedSubstr: "field 'return_type?' mismatch: expected 'type', got '[type]'",
		},
		{
			name: "unexpected extra field",
			mutateYAML: func(y string) string {
				return strings.Replace(y, "  function_definition:\n    fields:\n", "  function_definition:\n    fields:\n      extra_field: identifier\n", 1)
			},
			expectedSubstr: "has unexpected extra field 'extra_field' in compact YAML",
		},
		{
			name: "node should not have fields",
			mutateYAML: func(y string) string {
				return strings.Replace(y, "  pass_statement: {}\n", "  pass_statement:\n    fields:\n      bogus: identifier\n", 1)
			},
			expectedSubstr: "Node 'pass_statement' should not have fields in compact YAML",
		},
		{
			name: "node should not have children",
			mutateYAML: func(y string) string {
				return strings.Replace(y, "  pass_statement: {}\n", "  pass_statement:\n    children: [comment]\n", 1)
			},
			expectedSubstr: "Node 'pass_statement' should not have children in compact YAML",
		},
		{
			name: "children mismatch",
			mutateYAML: func(y string) string {
				// block has children: [_compound_statement | _simple_statement]
				return strings.Replace(y, "    children: [_compound_statement | _simple_statement]\n", "    children: [_compound_statement | bogus]\n", 1)
			},
			expectedSubstr: "children mismatch",
		},
		{
			name: "extra unknown type",
			mutateYAML: func(y string) string {
				return y + "\n  extraneous_node_type: {}\n"
			},
			expectedSubstr: "Compact YAML contains extra unknown types",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mutated := tc.mutateYAML(string(validYAML))
			discrepancies, _, err := ValidateCapture(rawJSON, mutated)
			if err != nil {
				t.Fatalf("ValidateCapture failed with unexpected error: %v", err)
			}
			if len(discrepancies) == 0 {
				t.Fatalf("expected discrepancy for %s, got 0", tc.name)
			}
			found := false
			for _, d := range discrepancies {
				if strings.Contains(d, tc.expectedSubstr) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected discrepancy containing %q, got: %v", tc.expectedSubstr, discrepancies)
			}
		})
	}
}

func TestParseCompactYAMLSyntaxErrors(t *testing.T) {
	t.Parallel()

	invalidInputs := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name:        "unexpected top level section",
			input:       "unknown_section:\n  foo: bar\n",
			errContains: "unexpected top-level section",
		},
		{
			name:        "malformed supertype",
			input:       "supertypes:\n  no_colon\n",
			errContains: "malformed supertype entry",
		},
		{
			name:        "malformed node entry",
			input:       "nodes:\n  bad_node\n",
			errContains: "malformed node entry",
		},
		{
			name:        "malformed field entry",
			input:       "nodes:\n  my_node:\n    fields:\n      no_colon_field\n",
			errContains: "malformed field entry",
		},
	}

	for _, tc := range invalidInputs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := ParseCompactYAML(tc.input)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Fatalf("expected error containing %q, got: %v", tc.errContains, err)
			}
		})
	}
}

func TestCompactorService(t *testing.T) {
	t.Parallel()
	root := testRepoRoot(t)

	cfg, err := _jsii.LoadConfig(filepath.Join(root, "tree-sitter-config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	dir := t.TempDir()
	// Create mock grammar directory with node-types.json
	pythonDir := filepath.Join(dir, "python")
	if err := os.MkdirAll(pythonDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	srcJSON, err := os.ReadFile(filepath.Join(root, "data", "tree-sitter", "grammar", "python", "node-types.json"))
	if err != nil {
		t.Fatalf("read python node-types.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pythonDir, "node-types.json"), srcJSON, 0o644); err != nil {
		t.Fatalf("write node-types.json: %v", err)
	}

	compactor := NewCompactor()

	// 1. Check before generation should fail (file missing)
	err = compactor.Compact(context.Background(), cfg, _jsii.CompactRequest{
		Language: pythonDir,
		Check:    true,
	})
	if err == nil {
		t.Fatalf("expected error for missing compact-node-types.yaml, got nil")
	}

	// 2. Generate compact YAML
	err = compactor.Compact(context.Background(), cfg, _jsii.CompactRequest{
		Language: pythonDir,
		Check:    false,
	})
	if err != nil {
		t.Fatalf("Compact generation failed: %v", err)
	}

	// Verify compact file exists
	compactPath := filepath.Join(pythonDir, "compact-node-types.yaml")
	if _, err := os.Stat(compactPath); err != nil {
		t.Fatalf("expected compact-node-types.yaml to exist: %v", err)
	}

	// 3. Check mode should now succeed
	err = compactor.Compact(context.Background(), cfg, _jsii.CompactRequest{
		Language: pythonDir,
		Check:    true,
	})
	if err != nil {
		t.Fatalf("Compact check failed: %v", err)
	}

	// 4. Corrupt compact file -> check mode should fail
	corruptYAML := []byte("supertypes:\n\nnodes:\n  corrupt_node: {}\n")
	if err := os.WriteFile(compactPath, corruptYAML, 0o644); err != nil {
		t.Fatalf("write corrupt yaml: %v", err)
	}
	err = compactor.Compact(context.Background(), cfg, _jsii.CompactRequest{
		Language: pythonDir,
		Check:    true,
	})
	if err == nil {
		t.Fatalf("expected check to fail on corrupted compact YAML, got nil")
	}
}

func BenchmarkCompactNodeTypes(b *testing.B) {
	root := testRepoRoot(&testing.T{})
	rawJSON, err := os.ReadFile(filepath.Join(root, "data", "tree-sitter", "grammar", "python", "node-types.json"))
	if err != nil {
		b.Fatalf("read fixture: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := CompactNodeTypes(rawJSON)
		if err != nil {
			b.Fatalf("CompactNodeTypes: %v", err)
		}
	}
}

func BenchmarkValidateCapture(b *testing.B) {
	root := testRepoRoot(&testing.T{})
	rawJSON, err := os.ReadFile(filepath.Join(root, "data", "tree-sitter", "grammar", "python", "node-types.json"))
	if err != nil {
		b.Fatalf("read fixture: %v", err)
	}
	compactYAML, err := CompactNodeTypes(rawJSON)
	if err != nil {
		b.Fatalf("CompactNodeTypes: %v", err)
	}
	yamlStr := string(compactYAML)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := ValidateCapture(rawJSON, yamlStr)
		if err != nil {
			b.Fatalf("ValidateCapture: %v", err)
		}
	}
}
