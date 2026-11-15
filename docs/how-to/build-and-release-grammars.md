# Build and release grammars

Use this guide after installing the configured toolchain. For a first isolated Python build, follow the [tutorial](../tutorials/getting-started.md).

## 1. Confirm configuration and pins

```bash
ts-build config path
ts-build config validate
ts-build list --language python
ts-build config get tree_sitter_cli_version
ts-build config get targets
```

Use `--config ./tree-sitter-config.yaml` on each command if you require the repository configuration.

Every grammar must pin a full, 40-character commit SHA. The generator and compiler versions must match the selected configuration.

## 2. Generate sources

```bash
ts-build build --language python
```

Generation writes parser sources and source provenance into the configured build workspace.

If checkout fails, inspect the preserved workspace before rebuilding.

WARNING: `build --force` removes the existing intermediate workspace for the selected grammar. Preserve manual changes before running it.

```bash
ts-build build --language python --force
```

## 3. Compile the host artifact

```bash
ts-build compile --language python
```

Compilation checks source provenance, builds fresh output, and records the artifact checksum. Do not edit source or binary provenance to bypass a mismatch.

For other targets, use [Cross-compile grammars](cross-compile-grammars.md). For WASM, use [Compile WebAssembly parsers](compile-webassembly-parsers.md).

## 4. Publish a new grammar output

```bash
ts-build move --language python --compact --clean=false
```

The mover locks the catalog, copies it to staging, validates selected artifacts, writes downstream metadata, and publishes only after validation succeeds.

The normal manifest requires a measured ABI for every included native binary. Static foreign validation alone cannot supply that measurement.

`--clean=false` retains intermediate files. Omitting it uses the selected cleanup behavior.

## 5. Replace existing grammar output

WARNING: `move --force` replaces the complete selected grammar directory. Artifacts absent from the new move are not retained.

Compile every artifact you intend to keep before publishing:

```bash
ts-build move --language python --compact --clean=false --force
```

The mover preserves unrelated grammar directories.

Existing catalog replacement requires supported atomic exchange operations. Linux and macOS support this on compatible filesystems. Windows replacement is not implemented.

If exchange is unsupported, choose a new catalog root in a separate release configuration. Do not delete the live catalog to bypass this protection.

## 6. Select downstream artifacts

| Option | Purpose |
| --- | --- |
| `--compact` | Generate and validate compact AST YAML |
| `--check-compact` | Audit existing compact YAML |
| `--source` | Include parser C/C++ sources |
| `--js` | Include grammar JavaScript |
| `--wasm` | Include the compiled WASM artifact |
| `--scm=false` | Omit query files |
| `--manifest=false` | Omit the custom downstream manifest |

`manifest.json` serves consumers such as the graph builder. `compact-node-types.yaml` serves AI agents and other AST consumers.

Manifest-only regeneration is blocked. Rebuild, compile, and publish validated artifacts instead.

## 7. Audit and export

```bash
ts-build compact --language python --check
ts-build export --language python --to ./consumer-grammars
```

Export copies already published output. It does not build parsers or measure ABIs.

If the selected destination grammar already exists, export requires explicit `--force`. Other destination directories remain untouched.

WARNING: `clean` removes intermediate build files. Published grammar output is separate.

```bash
ts-build clean --language python
```

Do not use `clean python`; positional grammar arguments are rejected. Omit `--language` only when you intend to select the entire configured catalog.
