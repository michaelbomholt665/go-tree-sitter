# How to Consume and Parse in Go Applications

This guide explains how to use compiled Tree-sitter grammars in Go applications to parse source code, inspect AST nodes, and execute tree-sitter queries.

---

## 1. Using Built-in Grammars via `internal/grammars`

The [`internal/grammars`](file:///home/michael/projects/go/tree-sitter/internal/grammars) package provides a centralized registry mapping grammar names to [`*sitter.Language`](file:///home/michael/projects/go/tree-sitter/internal/grammars/registry.go#L28) pointers:

```go
package main

import (
	"fmt"

	"github.com/michaelbomholt665/go-tree-sitter/internal/grammars"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func main() {
	// Retrieve language pointer (e.g. "go", "python", "json", "rust", etc.)
	lang := grammars.GetLanguage("python")
	if lang == nil {
		panic("language not supported")
	}

	// Create and configure a parser instance
	parser := sitter.NewParser()
	defer parser.Close()

	if err := parser.SetLanguage(lang); err != nil {
		panic(err)
	}

	// Parse source code
	source := []byte("def greet(name):\n    return f'Hello, {name}'\n")
	tree := parser.Parse(source, nil)
	defer tree.Close()

	root := tree.RootNode()
	fmt.Printf("Parsed %d bytes into AST: %s\n", len(source), root.Kind())
}
```

> [!NOTE]
> When compiling code that imports [`internal/grammars`](file:///home/michael/projects/go/tree-sitter/internal/grammars), pass the build tag `-tags grammars`:
> ```bash
> go run -tags grammars main.go
> ```

---

## 2. Loading Published Dynamic Shared Libraries at Runtime

To load published `.so`, `.dylib`, or `.dll` binaries dynamically at runtime without linking them at compile time:

```go
package main

import (
	"fmt"
	"path/filepath"
	"plugin" // Or dlopen on Unix
	"unsafe"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func LoadLanguageFromSO(soPath, constructorSymbol string) (*sitter.Language, error) {
	// Open dynamic shared library
	plug, err := plugin.Open(soPath)
	if err != nil {
		return nil, fmt.Errorf("open dynamic library: %w", err)
	}

	// Lookup grammar constructor symbol (e.g. "tree_sitter_python")
	sym, err := plug.Lookup(constructorSymbol)
	if err != nil {
		return nil, fmt.Errorf("lookup constructor %s: %w", constructorSymbol, err)
	}

	// Cast symbol to language constructor func() unsafe.Pointer
	constructor, ok := sym.(func() unsafe.Pointer)
	if !ok {
		return nil, fmt.Errorf("symbol %s does not match language constructor signature", constructorSymbol)
	}

	langPtr := constructor()
	return sitter.NewLanguage(langPtr), nil
}
```

---

## 3. Running Tree-Sitter Queries (`.scm`)

Published grammars in `data/tree-sitter/grammar/<language>/queries/` contain `.scm` query files (such as `highlights.scm` or `tags.scm`).

To execute a query against a parsed syntax tree:

```go
package main

import (
	"fmt"
	"os"

	"github.com/michaelbomholt665/go-tree-sitter/internal/grammars"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func main() {
	lang := grammars.GetLanguage("python")
	parser := sitter.NewParser()
	defer parser.Close()
	_ = parser.SetLanguage(lang)

	source := []byte("def calculate(x, y):\n    return x + y\n")
	tree := parser.Parse(source, nil)
	defer tree.Close()

	// 1. Read query string (or read from queries/highlights.scm)
	queryStr := `(function_definition name: (identifier) @func.name)`

	// 2. Compile query
	query, err := sitter.NewQuery(lang, queryStr)
	if err != nil {
		panic(err)
	}
	defer query.Close()

	// 3. Execute query cursor
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(query, tree.RootNode(), source)
	for match := matches.Next(); match != nil; match = matches.Next() {
		for _, capture := range match.Captures {
			captureName := query.CaptureNameForId(capture.Index)
			nodeText := string(source[capture.Node.StartByte():capture.Node.EndByte()])
			fmt.Printf("Matched @%s: %s\n", captureName, nodeText)
		}
	}
}
```

---

## 4. Reading `node-types.json` for AST Reflection

The `data/tree-sitter/grammar/<language>/node-types.json` artifact provides the complete structural schema of AST node types emitted by the grammar.

To inspect node definitions:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type NodeType struct {
	Type   string `json:"type"`
	Named  bool   `json:"named"`
	Fields map[string]struct {
		Multiple bool     `json:"multiple"`
		Required bool     `json:"required"`
		Types    []struct {
			Type  string `json:"type"`
			Named bool   `json:"named"`
		} `json:"types"`
	} `json:"fields,omitempty"`
}

func main() {
	data, err := os.ReadFile("data/tree-sitter/grammar/python/node-types.json")
	if err != nil {
		panic(err)
	}

	var schema []NodeType
	if err := json.Unmarshal(data, &schema); err != nil {
		panic(err)
	}

	fmt.Printf("Loaded %d Python AST node type definitions.\n", len(schema))
}
```

---

## 5. Using `compact-node-types.yaml` for LLM Context & Tooling

When `compact-node-types.yaml` is published in the catalog (`data/tree-sitter/grammar/<language>/compact-node-types.yaml`), it can be parsed with Go's `gopkg.in/yaml.v3` or directly embedded into LLM prompt contexts for query generation and code synthesis:

```go
package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type CompactSchema struct {
	Supertypes map[string][]string `yaml:"supertypes"`
	Nodes      map[string]struct {
		Fields   map[string]string `yaml:"fields,omitempty"`
		Children string            `yaml:"children,omitempty"`
	} `yaml:"nodes"`
}

func main() {
	yamlBytes, err := os.ReadFile("data/tree-sitter/grammar/python/compact-node-types.yaml")
	if err != nil {
		panic(err)
	}

	var schema CompactSchema
	if err := yaml.Unmarshal(yamlBytes, &schema); err != nil {
		panic(err)
	}

	fmt.Printf("Loaded compact schema: %d supertypes, %d node types.\n",
		len(schema.Supertypes), len(schema.Nodes))
}
```

Because `compact-node-types.yaml` achieves 60%–80% token reduction while preserving all field requirements (`?`), multiplicity (`[...]`), and union types (`|`), it is the recommended format when supplying grammar schemas to LLM prompts.

