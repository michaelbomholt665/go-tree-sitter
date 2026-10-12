# Compact Node Types and Token Efficiency

This document explains the rationale, theoretical foundation, and design trade-offs behind the **Compact Node Types** generator in `go-tree-sitter`, implemented in [`internal/tree-sitter/build/compact.go`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/compact.go).

---

## 1. The LLM Context Window Problem with AST Schemas

Modern AI coding agents and static code analyzers rely on Tree-sitter ASTs to inspect, generate, and transform code. To synthesize accurate Tree-sitter query expressions (`.scm`) or construct AST navigation routines, an LLM must understand the schema of target programming languages: which nodes exist, which fields they expose, and which child node types are legal.

Historically, Tree-sitter provides this schema via `node-types.json`. However, `node-types.json` presents a severe barrier for LLM context windows:

1. **Repetitive JSON Boilerplate**: For every field in every node, `node-types.json` nests multiple dictionary layers containing redundant boolean flags:
   ```json
   "fields": {
     "name": {
       "multiple": false,
       "required": true,
       "types": [
         { "type": "identifier", "named": true }
       ]
     }
   }
   ```
2. **Context Window Exhaustion & Context Rot**: In languages like C++, TypeScript, or Rust, `node-types.json` ranges from 80 KB to 250 KB in raw size. Passing an entire `node-types.json` file into an LLM prompt consumes between 25,000 and 60,000 tokens—often dominating the prompt, inducing severe **context rot**, and displacing actual user code or reasoning context.
3. **Low Information Density**: More than 70% of the byte count in `node-types.json` is structural ceremony (`{`, `}`, `"types"`, `"named"`, `"multiple"`), offering very little semantic signal per token.

### Built for Downstream Consumers
Like `manifest.json`, `compact-node-types.yaml` was **not built for this app**—it was created specifically for the external applications that consume these compiled grammars (such as the author's **graph builder** and autonomous AI agents). Generating and losslessly validating it during `ts-build move` is simply the easiest, most reliable place in the lifecycle to produce it. Instead of forcing downstream tools to parse and ingest massive ~100 KB JSON files, the release catalog provides clean ~20 KB YAML files out of the box.

---

## 2. The Compact Schema Design

[`CompactNodeTypes`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/compact.go#L65) reformulates the AST schema using human- and machine-readable YAML conventions inspired by modern type systems (such as TypeScript and Rust):

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

tokens: ["!=", "%", "*", "+", "-", "/", "and", "or", ...]

supertypes:
  expression: [as_pattern, boolean_operator, comparison_operator, ...]

nodes:
  boolean_operator:
    fields:
      left: expression
      operator: '"and" | "or"'
      right: expression
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

### Design Principles

1. **Bare Anonymous Token Inventory (`tokens: [...]`)**:
   Purely anonymous structural and punctuation tokens (such as `+`, `-`, `and`, `or`, `(`, `)`) that carry no additional fields, children, subtypes, or metadata are gathered into a compact, sorted top-level sequence. This avoids declaring hundreds of empty, repetitive node blocks. If an anonymous token carries fields, children, or metadata, it is declared explicitly under `nodes:` as a double-quoted key (e.g., `";":`).
2. **Unambiguous Named vs. Anonymous Notation**:
   Named node declarations and references use bare identifiers (or single-quoted identifiers when conflicting with YAML reserved keywords like `'true'`, `'false'`). Anonymous tokens are explicitly double-quoted (e.g. `"and"`, `"+"`). This guarantees that named nodes and anonymous tokens with the same spelling (e.g., named `await` vs. anonymous `"await"`) are never conflated.
3. **Full Operator Alternatives & Mixed Unions**:
   Every type alternative in field unions, children descriptors, and supertypes is preserved. Operator fields (such as `augmented_assignment.operator`, `binary_operator.operator`, and `boolean_operator.operator`) retain their explicit anonymous token alternatives (e.g. `'"and" | "or"'`) rather than being approximated or replaced with `any`. Mixed unions (e.g. `for_in_clause.right` with named `expression` alongside anonymous `","`) survive cleanly with bracketed/quoted formatting.
4. **Independent Requiredness (`?`) and Multiplicity (`[...]`)**:
   Both fields and children descriptors independently preserve `required` and `multiple` across all four combinations:
   - `key: T`: Required single instance (`required: true, multiple: false`)
   - `key?: T`: Optional single instance (`required: false, multiple: false`)
   - `key: [T]`: Required repeated group (`required: true, multiple: true`)
   - `key?: [T]`: Optional repeated group (`required: false, multiple: true`)
   This applies equally to `children:` (required child group) and `children?:` (optional child group).
5. **Supertype Segregation**:
   Abstract supertypes (such as `expression` or `statement`) are isolated into a top-level `supertypes:` mapping. Nodes reference the supertype directly, preventing combinatorial explosion across individual node fields.
6. **Semantic Metadata Preservation (`root` and `extra`)**:
   `root: true` is preserved on root nodes (such as `module`), and `extra: true` is preserved on nodes that can appear anywhere in the grammar (such as `comment` or `line_continuation`). Any unrecognized source properties trigger an immediate failure rather than being silently discarded.

---

## 3. Lossless Capture Theory

Token compression is counterproductive if it discards semantic information required to parse or validate syntax trees. `go-tree-sitter` enforces **Lossless Capture**: the compact YAML must represent an exact, mathematically equivalent projection of the AST structure.

### Bidirectional Validation (`ValidateCapture`)

The system does not trust the serializer blindly. After generating compact YAML, [`ValidateCapture`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/compact.go#L350) parses the YAML back into abstract structures and performs rigorous bidirectional reconciliation against the original `node-types.json`:

```
┌──────────────────┐               ┌───────────────────────────┐
│ node-types.json  │ ────────────► │  CompactNodeTypes(bytes)  │
└──────────────────┘               └───────────────────────────┘
         │                                       │
         │                                       ▼
         │                          compact-node-types.yaml
         │                                       │
         │         ┌─────────────────────────────┤
         ▼         ▼                             ▼
┌──────────────────────────────────────────────────────────────┐
│                  ValidateCapture() Invariants                │
│                                                              │
│  1. ∀ (type, named) ∈ JSON ⟺  node / token ∈ YAML            │
│  2. ∀ supertype ∈ JSON     ⟺  supertype ∈ YAML               │
│  3. ∀ field ∈ JSON         ⟺  field ∈ YAML                   │
│  4. field.Required         ⟺  !(fieldKey ends with '?')      │
│  5. field.Multiple         ⟺  typeExpr is enclosed in '[...]'│
│  6. children.Required      ⟺  children: vs children?:        │
│  7. children.Multiple      ⟺  typeExpr is enclosed in '[...]'│
│  8. All type alternatives  ⟺  Union components match exactly │
│  9. Metadata (root/extra)  ⟺  Flags match source values      │
│ 10. No unrecognized props  ⟺  0 unhandled properties         │
└──────────────────────────────────────────────────────────────┘
```

If even a single optional flag is flipped, a single operator alternative is dropped, or an unrecognized property is encountered, `ValidateCapture` aborts the build and returns a detailed discrepancy report.

---

## 4. Empirical Compression Metrics

Across all 42 supported grammars, the compact generator consistently yields dramatic savings:

| Language | Raw `node-types.json` | `compact-node-types.yaml` | Byte Reduction | Estimated Token Reduction |
| :--- | :--- | :--- | :--- | :--- |
| `c` | 54.8 KB | 12.1 KB | **77.9%** | ~78% |
| `bash` | 62.1 KB | 14.2 KB | **77.1%** | ~77% |
| `python` | 67.6 KB (3,861 lines) | 12.7 KB (361 lines) | **81.2%** | **~81%** |
| `rust` | 134.5 KB | 31.8 KB | **76.4%** | ~76% |
| `typescript` | 115.0 KB | 28.2 KB | **75.5%** | ~75% |
| `cpp` | 112.4 KB | 26.3 KB | **76.6%** | ~77% |

### Why Token Reductions Exceed Raw Byte Reductions
Because JSON tokenizers must break down repetitive punctuation (`"`, `:`, `,`, `{`, `}`) into individual BPE (Byte Pair Encoding) tokens, eliminating structural boilerplate yields an even greater token efficiency multiplier than raw byte comparisons suggest.

---

## 5. Usage in Downstream AI Tooling

With `compact-node-types.yaml` published in each grammar's catalog directory (`data/tree-sitter/grammar/<lang>/compact-node-types.yaml`), downstream applications can:
1. Provide comprehensive grammar rules to LLMs within a compact ~3,000-token budget instead of 25,000+ tokens.
2. Prompt LLMs to author Tree-sitter queries (`.scm`) with guaranteed node name, field name, operator, and token accuracy.
3. Automatically validate generated queries against known node fields before executing them against syntax trees.

