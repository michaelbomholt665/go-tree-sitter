package artifacts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCypherNodeTypesMatchCypherGrammar(t *testing.T) {
	t.Parallel()

	nodeTypes := readNodeTypes(t, "cypher")

	for _, expected := range []string{
		"cypher",
		"match",
		"node_pattern",
		"regular_query",
		"relationship_pattern",
		"return",
	} {
		if !nodeTypes[expected] {
			t.Fatalf("expected cypher node-types.json to include %q", expected)
		}
	}

	for _, unexpected := range []string{
		"class_declaration",
		"datatype_declaration",
		"import_declaration",
		"newtype_declaration",
	} {
		if nodeTypes[unexpected] {
			t.Fatalf("cypher node-types.json contains non-Cypher node %q", unexpected)
		}
	}
}

func readNodeTypes(t *testing.T, language string) map[string]bool {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(repoRoot(t), "data", "tree-sitter", "grammar", language, "node-types.json"))
	if err != nil {
		t.Fatalf("read node-types.json: %v", err)
	}

	var nodes []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(content, &nodes); err != nil {
		t.Fatalf("parse node-types.json: %v", err)
	}

	nodeTypes := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		nodeTypes[node.Type] = true
	}
	return nodeTypes
}

func repoRoot(t *testing.T) string {
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
