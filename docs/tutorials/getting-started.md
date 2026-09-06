# Getting Started with go-tree-sitter

This tutorial guides you through setting up your environment, building your first Tree-sitter grammar, compiling it into a shared library, and parsing code with Go.

By the end of this lesson, you will have:
1. Built the `c` grammar from its upstream repository.
2. Compiled it into a native shared library for your host operating system.
3. Published the grammar and validated its metadata with `manifest.json`.
4. Run a Go program that parses a C snippet into a syntax tree using [`go-tree-sitter`](file:///home/michael/projects/go/tree-sitter/go.mod#L11).

---

## Prerequisites

Before starting, ensure the following tools are available on your system:

1. **Go 1.26 or higher**:
   ```bash
   go version
   ```
2. **Tree-sitter CLI 0.26.8**:
   Install via npm if not already installed:
   ```bash
   npm install -g tree-sitter-cli@0.26.8
   ```
   Verify the installation:
   ```bash
   tree-sitter --version
   # Expected: tree-sitter 0.26.8
   ```
3. **C Compiler (GCC or Clang)**:
   Ensure `gcc` or `clang` is installed and on your `$PATH`:
   ```bash
   gcc --version
   ```
4. **Git**:
   ```bash
   git --version
   ```

---

## Step 1: Clone the Repository and Check Configuration

Navigate to the `go-tree-sitter` repository root:

```bash
cd /home/michael/projects/go/tree-sitter
```

Open [tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml). The configuration file contains pinned repositories, revisions, compiler targets, and output rules.

Notice the definition for the C language grammar:

```yaml
languages:
  - name: "c"
    grammar: "c"
    constructor: "tree_sitter_c"
    version: "v0.24.2"
    repository: "https://github.com/tree-sitter/tree-sitter-c.git"
    revision: "b780e47fc780ddc8da13afa35a3f4ed5c157823d"
    sample: "int main(void) { return 0; }\n"
```

The configuration pins an exact 40-character commit hash (`revision`) to guarantee reproducible builds.

---

## Step 2: Build the Grammar Source

Run the `build` subcommand to clone the upstream grammar repository and generate the C parser source files:

```bash
go run ./cmd/ts-build build --language c
```

You should see concise output confirming the build step completed:

```text
[✓] build     c                              (1.2s)
```

### What Happened Behind the Scenes?
1. The builder cloned `tree-sitter-c` at pinned commit `b780e47...` into `build/c/`.
2. Extracted the commit timestamp for deterministic builds (`SOURCE_DATE_EPOCH`).
3. Ran `tree-sitter generate --abi 15`.
4. Stored generation metadata and `node-types.json` checksum in `build/c/.source-provenance.json`.

---

## Step 3: Compile the Native Shared Library

Now, compile the generated C source code into a native dynamic shared library for your host platform:

```bash
go run ./cmd/ts-build compile --language c
```

Output:

```text
[✓] compile   c [linux/amd64]                (0.6s)
```

Inspect the intermediate build directory:

```bash
ls -l build/c/amd64/
```

You will find the compiled shared library:
- Linux: `c-v0.24.2-linux-amd64.so`
- macOS: `c-v0.24.2-macos-amd64.dylib` (or `macos-arm64`)
- Windows: `c-v0.24.2-windows-amd64.dll`

Alongside the library, `ts-build` generated a binary provenance record:
- `c-v0.24.2-linux-amd64.so.provenance.json`

---

## Step 4: Validate and Publish Artifacts

Move the compiled artifacts into the canonical catalog under `data/tree-sitter/grammar/c/`:

```bash
go run ./cmd/ts-build move --language c --force
```

Output:

```text
[✓] move      data/tree-sitter/grammar       [1 grammars, 1 binaries] (0.1s)
```

Check the published output directory:

```bash
ls -l data/tree-sitter/grammar/c/
```

The published catalog now contains:
- The compiled native library (`c-v0.24.2-<os>-<arch>.so`)
- AST type definitions (`node-types.json`)
- Tree-sitter query files (`queries/highlights.scm`, `queries/tags.scm`)
- Validated catalog metadata (`manifest.json`)

Inspect `data/tree-sitter/grammar/c/manifest.json`:

```json
{
  "schema_version": 2,
  "grammar": "c",
  "version": "v0.24.2",
  "compiled_at": "2024-11-20T00:54:15Z",
  "tree_sitter_version": "0.26.8",
  "parser_abi": 15,
  "binaries": [
    {
      "platform": "linux",
      "arch": "amd64",
      "filename": "c-v0.24.2-linux-amd64.so",
      "checksum_sha256": "..."
    }
  ],
  "abi": {
    "min_version": 15,
    "max_version": 15,
    "parser_version": "0.26.8"
  },
  "artifacts": {
    "has_node_types": true,
    "has_queries": true,
    "has_wasm": false
  }
}
```

Notice that `parser_abi` was measured directly from the compiled binary by loading it and checking its runtime symbol.

---

## Step 5: Parse Code in Go

Now verify your grammar by writing a small Go program that parses a C function.

Create a temporary file `main.go`:

```go
package main

import (
	"fmt"

	"github.com/michaelbomholt665/go-tree-sitter/internal/grammars"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func main() {
	// 1. Get the C language binding
	cLang := grammars.GetLanguage("c")
	if cLang == nil {
		panic("c language not available")
	}

	// 2. Initialize a new parser
	parser := sitter.NewParser()
	defer parser.Close()

	if err := parser.SetLanguage(cLang); err != nil {
		panic(err)
	}

	// 3. Parse a C code snippet
	code := []byte("int add(int a, int b) { return a + b; }")
	tree := parser.Parse(code, nil)
	defer tree.Close()

	root := tree.RootNode()
	fmt.Printf("Root node type: %s\n", root.Kind())
	fmt.Printf("Syntax tree sexp: %s\n", root.ToSexp())
}
```

Run the program with the `grammars` build tag enabled:

```bash
go run -tags grammars main.go
```

Output:

```text
Root node type: translation_unit
Syntax tree sexp: (translation_unit (function_definition type: (primitive_type) declarator: (function_declarator declarator: (identifier) parameters: (parameter_list (parameter_declaration type: (primitive_type) declarator: (identifier)) (parameter_declaration type: (primitive_type) declarator: (identifier)))) body: (compound_statement (return_statement (binary_expression left: (identifier) right: (identifier))))))
```

---

## Next Steps

Congratulations! You have successfully built, compiled, validated, published, and parsed code with a Tree-sitter grammar.

To explore further:
- Try `go run ./cmd/ts-build wizard` for an interactive guided experience — the recommended way to run the pipeline day-to-day without memorising flags.
- Learn how to run full releases for all 38 languages in [How to Build and Release Grammars](file:///home/michael/projects/go/tree-sitter/docs/how-to/build-and-release-grammars.md).
- Learn how to compile Windows and macOS binaries in [How to Cross-Compile for Windows and macOS](file:///home/michael/projects/go/tree-sitter/docs/how-to/cross-compile-grammars.md).
- Understand how validation prevents corrupted releases in [Dual-Mode Validation Mechanics](file:///home/michael/projects/go/tree-sitter/docs/explanation/dual-mode-validation.md).
