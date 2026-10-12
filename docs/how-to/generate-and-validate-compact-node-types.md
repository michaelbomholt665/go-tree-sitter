# How to Generate and Validate Compact Node Types

This guide explains how to generate, audit, and publish token-efficient `compact-node-types.yaml` specifications from `node-types.json` using [`ts-build`](file:///home/michael/projects/go/tree-sitter/cmd/ts-build/main.go) and convenience Makefile commands.

---

## 1. Why Compact Node Types?

Tree-sitter's standard `node-types.json` format is highly verbose and deeply nested. For large grammars (such as TypeScript, C++, or Rust), `node-types.json` can exceed 100 KB and consume tens of thousands of tokens when passed into Large Language Model (LLM) context windows for code generation or AST analysis. This causes severe **context rot** and rapidly exhausts LLM context windows in downstream tools (like the author's **graph builder**).

Like `manifest.json`, `compact-node-types.yaml` was **not built for this app**—it was built for external downstream applications that consume these grammars. [`CompactNodeTypes`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/compact.go) transforms the massive ~100 KB JSON schema into an AI-friendly ~20 KB YAML file:
- **75% to 83% token/size reduction** compared to raw JSON, preventing context rot in LLMs.
- **100% bidirectional lossless semantic capture**: Every declared identity `(type, named)`, supertype, subtype, named node type, field name, field requirement (`?`), multiplicity (`[...]`), child requirement (`children:` vs `children?:`), operator alternatives, and metadata (`root`, `extra`) is preserved.
- **Built at release time or on-demand**: Generating it during `ts-build move` or standalone via `ts-build compact` / `make compact` allows updating compact schemas without rebuilding grammar binaries.

---

## 2. Generating Compact Node Types

### Option A: Using the CLI (`ts-build compact`)

#### Generate for a Single Language
```bash
go run ./cmd/ts-build compact --language python
```

Output:
```text
[✓] compact   python [123 nodes, 95 fields, 4 supertypes, 81.2% reduction] (1ms)
  -> Wrote: data/tree-sitter/grammar/python/compact-node-types.yaml
  -> Reduced: 3,861 lines (67,601 bytes) to 361 lines (12,704 bytes) [81.2% reduction]
  -> VALIDATION PASSED: 100% coverage verified (123 nodes, 95 fields, 4 supertypes, 71 children, 0 errors)
```

The generator writes:
- `data/tree-sitter/grammar/<lang>/compact-node-types.yaml` (if already published), or
- `build/<lang>/<arch>/compact-node-types.yaml` (if only built).

#### Generate for All Configured Languages
To generate or refresh compact node types across all 42 languages:
```bash
go run ./cmd/ts-build compact
```

#### Direct File Path Invocation
You can also pass a direct path to any `node-types.json` file or directory containing one:
```bash
go run ./cmd/ts-build compact ./build/rust/amd64/src/node-types.json
# Or specify a directory:
go run ./cmd/ts-build compact data/tree-sitter/grammar/python/
```

### Option B: Using the Makefile (`make compact`)

```bash
# Generate compact node types for python
make compact LANG=python

# Generate compact node types for all languages
make compact
```

---

## 3. Auditing for Lossless Integrity (`--check` / `make check-compact`)

To audit that an existing `compact-node-types.yaml` file is completely synchronized with its source `node-types.json` without modifying any files, pass `--check`:

```bash
# Using ts-build CLI:
go run ./cmd/ts-build compact --language python --check

# Or using Makefile:
make check-compact LANG=python
```

Output when valid:
```text
[✓] compact   python [123 nodes, 95 fields, 4 supertypes, 81.2% reduction] (1ms)
  -> VALIDATION PASSED: 100% coverage verified (123 nodes, 95 fields, 4 supertypes, 71 children, 0 errors)
  -> Reduced: 3,861 lines (67,601 bytes) to 361 lines (12,704 bytes) [81.2% reduction]
```

### CI / Release Audit Across All Languages

In CI workflows or pre-release checks, audit all 42 grammars to ensure no schema drift or manual edits corrupted the compact definitions:

```bash
make check-compact
# Or:
go run ./cmd/ts-build compact --check
```

If any discrepancies exist (such as missing fields, dropped subtypes, or syntax errors), the command exits with code 1 and prints a detailed discrepancy report.

---

## 4. Publishing with `ts-build move`

When publishing grammar releases with `ts-build move`, you can integrate compact node types into the release pipeline using either `--compact` or `--check-compact`.

### Option A: Generate and Stage on Move (`--compact`)

Passing `--compact` causes `ts-build move` to generate `compact-node-types.yaml`, validate it losslessly against `node-types.json`, stage it in the release directory, and update `manifest.json`:

```bash
go run ./cmd/ts-build move --language python --compact --force
```

### Option B: Audit Existing on Move (`--check-compact`)

If you want to enforce that a pre-generated `compact-node-types.yaml` already exists and is bit-for-bit lossless without auto-regenerating:

```bash
go run ./cmd/ts-build move --language python --check-compact --force
```

---

## 5. Verifying `manifest.json`

Inspect the generated [`manifest.json`](file:///home/michael/projects/go/tree-sitter/data/tree-sitter/grammar/python/manifest.json):

```json
{
  "schema_version": 2,
  "grammar": "python",
  "version": "v0.25.0",
  "artifacts": {
    "has_node_types": true,
    "has_compact_node_types": true,
    "has_queries": true,
    "has_wasm": false,
    "has_c_source": false,
    "has_js": false
  }
}
```

When published with `--compact`, `"has_compact_node_types"` is set to `true`.

---

## 6. Understanding the Generated YAML Format

Here is an excerpt from `data/tree-sitter/grammar/python/compact-node-types.yaml`:

```yaml
# Compact Tree-sitter Node Types (Generated from node-types.json)
# Legend:
#   tokens: [...]   : inventory of bare anonymous tokens
#   name            : named node declaration or reference
#   "token"         : anonymous token declaration or reference
#   key?            : optional field or children (omitted ? means required)
#   [T]             : multiple occurrences allowed (omitted brackets means single)
#   T1 | T2         : alternative types
#   root: true      : grammar root node
#   extra: true     : extra node (can appear anywhere, e.g. comments, whitespace)

tokens: ["!=", "%", "%=", "&", "&=", "(", ")", "*", "**", "+", "-", "/", "and", "or", ...]

supertypes:
  expression: [as_pattern, boolean_operator, comparison_operator, conditional_expression, lambda, named_expression, not_operator, primary_expression]

nodes:
  boolean_operator:
    fields:
      left: expression
      operator: '"and" | "or"'
      right: expression
  for_in_clause:
    fields:
      left: pattern | pattern_list
      right: '["," | expression]'
  lambda_parameters:
    children: [parameter]
  parameters:
    children?: [parameter]
  module:
    root: true
    children?: [...]
  comment:
    extra: true
```

For complete schema semantics, see the [Compact Node Types Schema Reference](file:///home/michael/projects/go/tree-sitter/docs/reference/compact-node-types-schema.md).
