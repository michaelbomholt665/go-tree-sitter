# Generate and validate compact node types

Generate `compact-node-types.yaml` when downstream tools need the AST schema without the repeated structure of `node-types.json`.

Compact schemas serve external consumers, including AI agents. They are not required to compile a parser.

## Generate for one grammar

```bash
ts-build compact --language python
```

The command resolves the grammar through the selected configuration. Use `--config` if the default selection is not appropriate.

For a direct file or directory:

```bash
ts-build compact ./schemas/python/node-types.json
ts-build compact ./schemas/python
```

The directory must contain `node-types.json`. Generation writes the adjacent compact YAML.

Use either a positional path or `--language`, not both.

## Check existing output without rewriting

```bash
ts-build compact --language python --check
```

For a direct schema file:

```bash
ts-build compact ./schemas/python/node-types.json --check
```

The check compares the existing YAML with the JSON. Missing output or a semantic mismatch returns an error.

Checks preserve named versus anonymous identities, alternatives, fields, child constraints, subtypes, and supported metadata.

## Audit a complete configured catalog

```bash
ts-build compact --check
```

Omitting `--language` selects every configured entry. The current repository catalog contains 65 entries; a user's selected configuration can differ.

A complete audit requires the corresponding schema files. It is not a command to clone or build missing grammars.

## Generate during publication

```bash
ts-build move --language python --compact --clean=false
```

Generation and validation occur in staging before publication.

If existing YAML must pass without regeneration:

```bash
ts-build move --language python --check-compact --clean=false
```

Existing selected grammar output requires `--force`. That option replaces its entire directory, not just the compact schema. Catalog replacement also requires supported atomic exchange operations.

## Handle validation failures

If YAML differs from JSON, regenerate from the intended JSON source and repeat the check.

If JSON contains unsupported properties, preserve the original file and report the schema difference. Do not remove properties to make validation pass.

Do not hand-edit generated compact YAML and assume it remains equivalent.

## Measure size separately

The command reports byte-size reduction. Token savings depend on the tokenizer and grammar. A smaller file does not guarantee better model behavior.

See the [schema reference](../reference/compact-node-types-schema.md) for notation and [design explanation](../explanation/compact-node-types-and-token-efficiency.md) for preservation rules.
