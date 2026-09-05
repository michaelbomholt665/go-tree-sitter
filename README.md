# go-tree-sitter

A reproducible Tree-sitter grammar builder. It pins grammar revisions, the Tree-sitter generator, compiler targets, and flags; then validates metadata against the final native libraries before publishing the complete catalog atomically.

## Requirements

- Go 1.26+
- Tree-sitter CLI 0.26.8 (the configured version is checked from `tree-sitter --version`)
- The compiler and version selected by `tree-sitter-config.yaml`
- Network access for the initial pinned grammar checkouts and locked npm dependencies

Install the pinned CLI with `make install-tree-sitter-cli`.

## Release workflow

```bash
make release
```

This runs the three explicit phases:

```bash
go run ./cmd/tree-sitter build --force
go run ./cmd/tree-sitter compile
go run ./cmd/tree-sitter move --both --force
```

`build` checks out each full commit, invokes `tree-sitter generate --abi 15`, and records the invoked generator version and generated `node-types.json` hash. `compile` records the invoked compiler identity, target triple, flags, and source provenance. `move` stages the complete catalog, loads every final native library, measures its ABI, validates parsing and queries, computes checksums from the final bytes, and swaps the release directory only after every grammar passes.

Manifest-only regeneration and publication are intentionally unsupported. A manifest cannot be made authoritative without loading the exact native artifact it describes.

## Validation

The validation runtime is `github.com/tree-sitter/go-tree-sitter v0.25.0`. Compatibility uses that binding's `MIN_COMPATIBLE_LANGUAGE_VERSION` and `LANGUAGE_VERSION` constants (currently 13 and 15). Those constants are validation policy, not binary metadata.

For each native artifact the validator:

- checks its file format, architecture, and sole expected `tree_sitter_<grammar>` export;
- keeps the dynamic library loaded while calling its constructor and `Language.AbiVersion()`;
- assigns the language through `Parser.SetLanguage`, parses the configured sample, and rejects error trees;
- validates `node-types.json` against generation provenance and compiles every shipped query with located errors;
- verifies manifest ABI invariants, provenance, and final-byte SHA-256 values.

Cross-platform binaries must run this validation on their target platform. A Linux validation job will reject Windows or macOS binaries rather than infer their ABI.

## Manifest schema

Published manifests use schema version 2. See [docs/manifest-schema-v2.md](docs/manifest-schema-v2.md) for the field contract and migration notes.

## Development

```bash
make fmt
make test
make vet
```

The configuration and source revisions are in `tree-sitter-config.yaml`; implementation is under `internal/tree-sitter/`.

## License

MIT. See [LICENSE](LICENSE).
