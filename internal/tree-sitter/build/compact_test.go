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

			// Verify reduction criteria (> 60% reduction)
			if metrics.ReductionPct < 60.0 {
				t.Errorf("expected > 60%% reduction for %s, got %.2f%%", lang, metrics.ReductionPct)
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

func TestSemanticPreservationPythonRegression(t *testing.T) {
	t.Parallel()
	root := testRepoRoot(t)
	rawJSON, err := os.ReadFile(filepath.Join(root, "data", "tree-sitter", "grammar", "python", "node-types.json"))
	if err != nil {
		t.Fatalf("read python node-types.json: %v", err)
	}

	compactYAML, err := CompactNodeTypes(rawJSON)
	if err != nil {
		t.Fatalf("CompactNodeTypes: %v", err)
	}
	yamlStr := string(compactYAML)

	// 1. Operator alternatives are preserved (not replaced with 'any')
	t.Run("operator alternatives preserved", func(t *testing.T) {
		if strings.Contains(yamlStr, "operator: any") {
			t.Errorf("found 'operator: any' in YAML; operator alternatives were discarded")
		}
		if !strings.Contains(yamlStr, `"and"`) || !strings.Contains(yamlStr, `"or"`) {
			t.Errorf("boolean_operator missing \"and\" or \"or\" alternatives")
		}
		if !strings.Contains(yamlStr, `"!="`) || !strings.Contains(yamlStr, `"=="`) {
			t.Errorf("comparison_operator missing \"!=\" or \"==\" alternatives")
		}
		if !strings.Contains(yamlStr, `"+="`) || !strings.Contains(yamlStr, `"-="`) {
			t.Errorf("augmented_assignment missing \"+=\" or \"-=\" alternatives")
		}
		if !strings.Contains(yamlStr, `"~"`) {
			t.Errorf("unary_operator missing \"~\" alternative")
		}
	})

	// 2. Mixed named/anonymous alternatives survive
	t.Run("mixed alternatives survive", func(t *testing.T) {
		if !strings.Contains(yamlStr, `"-"`) || !strings.Contains(yamlStr, `expression`) {
			t.Errorf("dict_pattern.key missing anonymous \"-\" or named expression")
		}
		if !strings.Contains(yamlStr, `","`) {
			t.Errorf("for_in_clause.right missing anonymous \",\"")
		}
	})

	// 3. Child requiredness is distinguishable
	t.Run("child requiredness distinguishable", func(t *testing.T) {
		// lambda_parameters requires children
		if !strings.Contains(yamlStr, "lambda_parameters:\n    children:") {
			t.Errorf("expected lambda_parameters to have required children (children:)")
		}
		// parameters has optional children
		if !strings.Contains(yamlStr, "parameters:\n    children?:") {
			t.Errorf("expected parameters to have optional children (children?:)")
		}
		// return_statement has optional children (and multiple: false)
		if !strings.Contains(yamlStr, "return_statement:\n    children?:") {
			t.Errorf("expected return_statement to have optional children (children?:)")
		}
	})

	// 4. Root and extra metadata preserved
	t.Run("root and extra metadata preserved", func(t *testing.T) {
		if !strings.Contains(yamlStr, "module:\n    root: true") {
			t.Errorf("module missing root: true")
		}
		if !strings.Contains(yamlStr, "comment:\n    extra: true") {
			t.Errorf("comment missing extra: true")
		}
		if !strings.Contains(yamlStr, "line_continuation:\n    extra: true") {
			t.Errorf("line_continuation missing extra: true")
		}
	})

	// 5. Separate identities for named vs anonymous await; named true/false strings
	t.Run("identities and scalar quoting", func(t *testing.T) {
		if !strings.Contains(yamlStr, "tokens:") || !strings.Contains(yamlStr, `"await"`) {
			t.Errorf("anonymous await missing from tokens list")
		}
		if !strings.Contains(yamlStr, "  await:\n") {
			t.Errorf("named await missing from nodes section")
		}
		if !strings.Contains(yamlStr, "'true':") && !strings.Contains(yamlStr, "'true': {}") {
			t.Errorf("named 'true' missing or unquoted in nodes")
		}
		if !strings.Contains(yamlStr, "'false':") && !strings.Contains(yamlStr, "'false': {}") {
			t.Errorf("named 'false' missing or unquoted in nodes")
		}
	})
}

func TestSyntheticEdgeCases(t *testing.T) {
	t.Parallel()

	// All 4 required/multiple combinations for fields and children
	t.Run("four required multiple combinations", func(t *testing.T) {
		syntheticJSON := []byte(`[
			{
				"type": "synth_node",
				"named": true,
				"fields": {
					"req_single": {"multiple": false, "required": true, "types": [{"type": "t1", "named": true}]},
					"opt_single": {"multiple": false, "required": false, "types": [{"type": "t2", "named": true}]},
					"req_multi":  {"multiple": true,  "required": true, "types": [{"type": "t3", "named": true}]},
					"opt_multi":  {"multiple": true,  "required": false, "types": [{"type": "t4", "named": true}]}
				},
				"children": {
					"multiple": true,
					"required": false,
					"types": [{"type": "child_type", "named": true}]
				}
			}
		]`)

		compact, err := CompactNodeTypes(syntheticJSON)
		if err != nil {
			t.Fatalf("CompactNodeTypes: %v", err)
		}
		s := string(compact)
		if !strings.Contains(s, "req_single: t1") {
			t.Errorf("expected 'req_single: t1', got:\n%s", s)
		}
		if !strings.Contains(s, "opt_single?: t2") {
			t.Errorf("expected 'opt_single?: t2', got:\n%s", s)
		}
		if !strings.Contains(s, "req_multi: [t3]") {
			t.Errorf("expected 'req_multi: [t3]', got:\n%s", s)
		}
		if !strings.Contains(s, "opt_multi?: [t4]") {
			t.Errorf("expected 'opt_multi?: [t4]', got:\n%s", s)
		}
		if !strings.Contains(s, "children?: [child_type]") {
			t.Errorf("expected 'children?: [child_type]', got:\n%s", s)
		}

		discrepancies, _, err := ValidateCapture(syntheticJSON, s)
		if err != nil {
			t.Fatalf("ValidateCapture: %v", err)
		}
		if len(discrepancies) > 0 {
			t.Errorf("unexpected discrepancies: %v", discrepancies)
		}
	})

	// Notation sensitive literals
	t.Run("notation sensitive literals", func(t *testing.T) {
		sensitiveJSON := []byte(`[
			{
				"type": "sensitive_node",
				"named": true,
				"fields": {
					"operator": {
						"multiple": false,
						"required": true,
						"types": [
							{"type": "|", "named": false},
							{"type": "[", "named": false},
							{"type": "]", "named": false},
							{"type": "?", "named": false},
							{"type": ":", "named": false},
							{"type": ",", "named": false},
							{"type": "\"", "named": false},
							{"type": "'", "named": false}
						]
					}
				}
			}
		]`)

		compact, err := CompactNodeTypes(sensitiveJSON)
		if err != nil {
			t.Fatalf("CompactNodeTypes: %v", err)
		}
		discrepancies, _, err := ValidateCapture(sensitiveJSON, string(compact))
		if err != nil {
			t.Fatalf("ValidateCapture: %v", err)
		}
		if len(discrepancies) > 0 {
			t.Errorf("unexpected discrepancies for sensitive literals: %v", discrepancies)
		}
	})

	// Unsupported properties fail with clear message
	t.Run("unsupported property detection", func(t *testing.T) {
		invalidJSON := []byte(`[
			{
				"type": "bad_node",
				"named": true,
				"unsupported_key": "some_value"
			}
		]`)

		_, err := CompactNodeTypes(invalidJSON)
		if err == nil {
			t.Fatalf("expected error for unsupported property, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported") && !strings.Contains(err.Error(), "unknown") {
			t.Errorf("expected error message to mention unsupported/unknown property, got: %v", err)
		}
	})
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
				return strings.Replace(y, "  expression: [", "  # expression: [", 1)
			},
			expectedSubstr: "Supertype 'expression' missing from compact YAML",
		},
		{
			name: "mismatched supertype subtypes",
			mutateYAML: func(y string) string {
				return strings.Replace(y, "  expression: [as_pattern,", "  expression: [bogus_definition,", 1)
			},
			expectedSubstr: "Supertype 'expression' subtypes mismatch",
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
				return strings.Replace(y, "  assert_statement:\n    children: [expression]\n", "  assert_statement:\n    children: [bogus]\n", 1)
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
