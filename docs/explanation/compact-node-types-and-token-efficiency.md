# Compact node types and token efficiency

Tree-sitter's `node-types.json` describes AST identities, fields, child constraints, and subtype relationships. Its nested JSON repeats structural keys for each descriptor.

Downstream AI agents and query tools often need the schema rather than the repeated serialization structure. Compact YAML reduces that repetition without intentionally removing schema meaning.

## A downstream artifact

Like `manifest.json`, `compact-node-types.yaml` serves external consumers rather than the builder itself.

Publication is a convenient generation point because staging already contains the selected AST schema. Standalone `compact` generation also works without recompiling grammar binaries.

A compact file is not a parser, a source-language support declaration, or a replacement for binary validation.

## Identity is more than spelling

A Tree-sitter type identity contains both its spelling and its `named` value.

A named node called `await` and an anonymous token spelled `await` are different identities. The compact representation preserves that distinction in declarations and references.

Bare anonymous tokens without fields, children, subtypes, or metadata appear in the top-level token inventory. Anonymous entries with structure remain explicit declarations.

Named identifiers and quoted anonymous-token markers must not be collapsed during parsing. YAML quoting also protects names that resemble booleans or other scalar values.

## Preserve independent constraints

Requiredness and multiplicity are independent:

| Descriptor | Required | Multiple |
| --- | --- | --- |
| `key: T` | Yes | No |
| `key?: T` | No | No |
| `key: [T]` | Yes | Yes |
| `key?: [T]` | No | Yes |

The same distinction applies to `children` and `children?`.

Alternatives remain explicit, including unions of named types and anonymous operators. Replacing an operator union with `any` loses information.

Supertypes preserve subtype relationships. Root and extra metadata also remain part of the comparison.

The generator rejects unsupported source properties instead of silently discarding them.

## Semantic round-trip validation

The generator parses JSON, writes compact YAML, parses the YAML back, and reconciles both representations.

Validation checks declarations and referenced identities in both directions. It also compares fields, alternatives, requiredness, multiplicity, child descriptors, subtypes, and supported metadata.

This is semantic preservation, not byte-for-byte JSON reproduction. Formatting and ordering can differ without changing the schema.

The `--check` path compares an existing compact file without rewriting it. A manual edit that changes meaning fails that check.

## Size and model limits

Size reduction depends on the grammar's structure. A schema dominated by repetitive descriptors compresses differently from one dominated by unique names.

Byte savings are not tokenizer-independent token savings. A shorter schema also does not prove that a model reasons more accurately or avoids context degradation.

The stable contract is preservation of supported schema semantics. Measure size and model behavior separately for the intended consumer.

See [Generation and audit instructions](../how-to/generate-and-validate-compact-node-types.md) and the [schema reference](../reference/compact-node-types-schema.md).
