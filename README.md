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

The builder supports **dual-mode validation**:

- **Host targets (e.g. Linux on Linux)**: Dynamically validated via `dlopen`:
  - checks file format, architecture, and sole expected `tree_sitter_<grammar>` export;
  - keeps the dynamic library loaded while calling its constructor and `Language.AbiVersion()`;
  - assigns the language through `Parser.SetLanguage`, parses the configured sample, and rejects error trees;
  - validates `node-types.json` against generation provenance and compiles every shipped query with located errors;
  - verifies manifest ABI invariants, provenance, and final-byte SHA-256 values.
- **Cross-platform targets (e.g. Windows `.dll`, macOS `.dylib` on Linux)**: Statically verified without `dlopen`:
  - **Windows PE**: inspects headers via `debug/pe`, verifies target machine (`IMAGE_FILE_MACHINE_AMD64` or `IMAGE_FILE_MACHINE_ARM64`), valid section headers, and verifies that the PE Export Directory Table exports the constructor symbol (`tree_sitter_<grammar>`);
  - **macOS Mach-O**: inspects headers via `debug/macho`, verifies CPU architecture (`CpuAmd64` or `CpuArm64`), `TypeDylib`, and symbol table presence of constructor symbols (`_tree_sitter_<grammar>` / `tree_sitter_<grammar>`);
  - validates `node-types.json` and query files;
  - obtains ABI version from source and binary build provenance.

## Cross-compilation

To compile Windows and macOS binaries from a Linux host:

### Prerequisites

You can use either target-specific toolchains or [Zig](https://ziglang.org/):

- **Windows (`.dll`)**: `x86_64-w64-mingw32-gcc` (from `mingw-w64`) or `zig` (`zig cc -target x86_64-windows-gnu`).
- **macOS (`.dylib`)**: `zig` (`zig cc -target aarch64-macos` / `zig cc -target x86_64-macos`) or an osxcross clang toolchain (`oa64-clang`, `o64-clang`).

If `zig` is installed and available in `$PATH`, the builder will automatically invoke it as a fallback toolchain when target-specific cross-compilers are not found.

### Compiling and Publishing Foreign Targets

```bash
# Generate grammar sources for the host
go run ./cmd/tree-sitter build --force

# Compile for Windows
go run ./cmd/tree-sitter compile --os windows --arch amd64

# Compile for macOS (Apple Silicon and Intel)
go run ./cmd/tree-sitter compile --os macos --arch arm64
go run ./cmd/tree-sitter compile --os macos --arch amd64

# Compile for Linux host
go run ./cmd/tree-sitter compile --os linux --arch amd64

# Statically validate foreign binaries, dynamically validate host binaries, and publish the release catalog
go run ./cmd/tree-sitter move --both --force
```

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
