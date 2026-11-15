# Use Tree-sitter grammars from Go

Choose between an upstream Go binding and a published native C library. They are different integration paths.

## Use an upstream Go binding

A binding compiles grammar sources into the consumer through CGo. It does not load the library published by `ts-build move`.

Create a consumer module with a working C/C++ toolchain:

```bash
mkdir python-parser-example
cd python-parser-example
go mod init example.com/python-parser-example
go get github.com/tree-sitter/go-tree-sitter@v0.25.0
go get github.com/tree-sitter/tree-sitter-python@v0.25.1-0.20250915213335-26855eabccb1
```

Save this example:

```go
// main.go
package main

import (
    "fmt"

    sitter "github.com/tree-sitter/go-tree-sitter"
    python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

func main() {
    parser := sitter.NewParser()
    defer parser.Close()

    language := sitter.NewLanguage(python.Language())
    if err := parser.SetLanguage(language); err != nil {
        panic(err)
    }

    tree := parser.Parse([]byte("print('hello')\n"), nil)
    if tree == nil {
        panic("parse failed")
    }
    defer tree.Close()

    root := tree.RootNode()
    if root.HasError() {
        panic("sample contains a parse error")
    }
    fmt.Println(root.ToSexp())
}
```

Run it:

```bash
go run .
```

The example uses the pinned Python binding and closes C-allocated parser and tree objects.

External modules cannot import this project's `internal/grammars` registry. That tagged registry is a repository-internal convenience, not the public distribution API.

## Use a published native library

The published `.so`, `.dylib`, and `.dll` files expose a C ABI. Go's `plugin.Open` loads Go plugins, not these parser libraries.

A native-library integration must:

1. Select an artifact for the consumer's OS and architecture.
2. Verify its checksum against trusted release metadata.
3. Compare its measured parser ABI with the consumer runtime's supported range.
4. Load it with the platform library loader.
5. Resolve the configured C constructor, such as `tree_sitter_python`.
6. Pass the returned language pointer to a compatible Tree-sitter runtime.
7. Keep the library loaded while any parser, tree, query, or language reference uses its data.

Use a maintained C interop loader or implement a platform-specific CGo bridge. The project's native validator demonstrates Unix and Windows loading, but it is an internal implementation, not an exported loader API.

A manifest describes the artifact; it does not make an incompatible architecture or ABI loadable.

## Export artifacts for a consumer

From a configured builder installation:

```bash
ts-build export --language python --to ./consumer-grammars
```

This copies the published `python/` directory. Export does not compile the consumer or add a Go module dependency.

If the consumer needs AST field names, use `compact-node-types.yaml` or `node-types.json`. If it needs queries, retain the published `.scm` files.

## Check support and licensing separately

A successful sample parse does not establish full source-language version support. Review revision-specific evidence.

A Go module license report also does not verify every native source license. See [NOTICE](../../NOTICE.md) and [Dual-mode validation](../explanation/dual-mode-validation.md).
