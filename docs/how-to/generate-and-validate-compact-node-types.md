# How to Generate and Validate Compact Node Types

This guide explains how to generate, audit, and publish token-efficient `compact-node-types.yaml` specifications from `node-types.json` using [`ts-build`](file:///home/michael/projects/go/tree-sitter/cmd/ts-build/main.go).

---

## 1. Why Compact Node Types?

Tree-sitter's standard `node-types.json` format is highly verbose and deeply nested. For large grammars (such as TypeScript, C++, or Rust), `node-types.json` can exceed 100 KB and consume tens of thousands of tokens when passed into Large Language Model (LLM) context windows for code generation or AST analysis. This causes severe **context rot** and rapidly exhausts LLM context windows in downstream tools (like the author's **graph builder**).

Like `manifest.json`, `compact-node-types.yaml` was **not built for this app**—it was built for external downstream applications that consume these grammars. [`CompactNodeTypes`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/compact.go#L65) transforms the massive ~100 KB JSON schema into an AI-friendly ~20 KB YAML file:
- **75% to 80% token reduction** compared to raw JSON, preventing context rot in LLMs.
- **100% bidirectional lossless capture**: Every supertype, subtype, named node type, field name, field requirement (`?`), multiplicity (`[...]`), and child union is preserved.
- **Built at release time**: Generating it during `ts-build move` ensures downstream AI tools receive verified, pre-compacted AST schemas directly from the release catalog.

---

## 2. Generating Compact Node Types (`ts-build compact`)

### Generate for a Single Language

Run the `compact` subcommand specifying the language:

```bash
go run ./cmd/ts-build compact --language python
```

Output:
```text
[✓] compact   python [122 nodes, 95 fields, 6 supertypes, 83.4% reduction] (2ms)
  -> Wrote: data/tree-sitter/grammar/python/compact-node-types.yaml
  -> Reduced: 3,746 lines (64,995 bytes) to 351 lines (10,777 bytes) [83.4% reduction]
  -> VALIDATION PASSED: 100% coverage verified (122 nodes, 95 fields, 6 supertypes, 70 children, 0 errors)
```

The generator produces:
- `data/tree-sitter/grammar/python/compact-node-types.yaml` (if already published), or
- `build/python/amd64/compact-node-types.yaml` (if only built).

### Generate for All Configured Languages

To generate or refresh compact node types across all 38 languages:

```bash
go run ./cmd/ts-build compact
```

### Direct File Path Invocation

You can also pass a direct path to any `node-types.json` file or directory containing one:

```bash
go run ./cmd/ts-build compact ./build/rust/amd64/src/node-types.json
# Or specify a directory:
go run ./cmd/ts-build compact data/tree-sitter/grammar/python/
```

---

## 3. Auditing for Lossless Integrity (`--check`)

To audit that an existing `compact-node-types.yaml` file is completely synchronized with its source `node-types.json` without modifying any files, pass `--check`:

```bash
go run ./cmd/ts-build compact --language python --check
```

Output when valid:
```text
[✓] compact   python [122 nodes, 95 fields, 6 supertypes, 83.4% reduction] (1ms)
  -> VALIDATION PASSED: 100% coverage verified (122 nodes, 95 fields, 6 supertypes, 70 children, 0 errors)
  -> Reduced: 3,746 lines (64,995 bytes) to 351 lines (10,777 bytes) [83.4% reduction]
```

### CI / Release Audit

In CI workflows or pre-release checks, audit all 38 grammars to ensure no schema drift or manual edits corrupted the compact definitions:

```bash
go run ./cmd/ts-build compact --check
```

If any discrepancies exist (such as missing fields, dropped subtypes, or syntax errors), the command exits with code 1 and prints the detailed discrepancy report.

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
#   field_name? : optional field
#   [Type]      : multiple occurrences allowed
#   TypeA | B   : alternative node types

supertypes:
  _compound_statement: [class_definition, decorated_definition, for_statement, function_definition, if_statement, match_statement, try_statement, while_statement, with_statement]
  _simple_statement: [assert_statement, break_statement, continue_statement, delete_statement, exec_statement, expression_statement, future_import_statement, global_statement, import_from_statement, import_statement, nonlocal_statement, pass_statement, print_statement, raise_statement, return_statement, type_alias_statement]
  expression: [as_pattern, boolean_operator, comparison_operator, conditional_expression, lambda, named_expression, not_operator, primary_expression]

nodes:
  call:
    fields:
      arguments: argument_list
      function: _expression
  for_statement:
    fields:
      body: block
      left: _expression
      right: _expression
  binary_operator:
    fields:
      left: _expression
      operator: any
      right: _expression
```

For complete schema semantics, see the [Compact Node Types Schema Reference](file:///home/michael/projects/go/tree-sitter/docs/reference/compact-node-types-schema.md).
