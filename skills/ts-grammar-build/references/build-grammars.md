# Build and release grammars

Use this guide after installing the configured toolchain. For a first isolated Python build, follow the tutorial.

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

For other targets, use Cross-compile grammars. For WASM, use Compile WebAssembly parsers.

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


---

# Cross-compile grammars

Cross-compilation produces artifacts for another operating system or architecture. It does not prove that those artifacts execute correctly on that target.

## 1. Choose the release contract

| Goal | Required validation |
| --- | --- |
| Publish a normal manifest with measured parser ABIs | Execute validation on a compatible target host |
| Produce foreign artifacts for later target-host validation | Compile and retain artifacts plus provenance |
| Publish raw foreign artifacts without the custom manifest | Explicit static validation and `--manifest=false` |

Do not combine foreign artifacts into a normal manifest by copying the configured generation ABI into metadata.

## 2. Configure the cross toolchain

Inspect target settings:

```bash
ts-build config get targets
ts-build config get runtime.compiler.allow_fallback
```

Set each target's OS, architecture, triple, compiler, and compiler version. If the grammar has a C++ scanner, configure a compatible C++ compiler.

The target compiler must actually support the selected triple. A configured name or a Zig installation alone does not establish a usable cross toolchain.

Use explicit compiler paths and matching version pins for repeatable builds. Windows hosts do not support the current Zig/Emscripten shell-wrapper paths.

## 3. Generate once and compile a foreign target

For example, from Linux with a configured Windows amd64 cross compiler:

```bash
ts-build build --language python
ts-build compile --language python --os windows --arch amd64 --allow-cross-validation
```

Other configured targets use the same selection pattern:

```bash
ts-build compile --language python --os macos --arch arm64 --allow-cross-validation
```

These commands require working cross compilers. They are not toolchain installation commands.

Keep generated sources and provenance until target validation completes.

## 4. Validate releases on their target hosts

For normal downstream manifests, use a runner for each supported target. On each runner, select matching configuration and perform the host pipeline:

```bash
ts-build build --language python
ts-build compile --language python
ts-build move --language python --compact --clean=false
```

Ensure `OS_TARGET` and the target architecture match that runner. Choose a fresh release destination if catalog replacement is unsupported.

A target runner must load the binary and measure its ABI. A cross-build or mocked test does not provide native execution evidence.

## 5. Publish without a manifest only when intentional

WARNING: `--manifest=false` removes measured-ABI downstream metadata. Use it only if the consumer accepts that contract.

For a foreign-only workspace with static validation explicitly allowed:

```bash
ts-build move --language python --allow-cross-validation --manifest=false --clean=false
```

The resulting output is raw distribution material, not a measured-ABI release. Static validation checks binary structure and expected exports; it does not parse samples on the foreign target.

Existing selected grammar output still requires `--force`. Replacement still requires a supported atomic directory exchange.

## Windows setup and testing

Use TEST-GUIDE.md and the root PowerShell installer for native Windows development setup.

The optional `-RunGrammarTest` path builds only Python for Windows amd64. It does not run the full Go suite, other grammars, or WASM tests. The script refuses an existing test catalog because Windows replacement is unsupported.

WARNING: The installer's uninstall action removes its listed tools, including tools installed before the script. Inspect the uninstall scope before confirming.

Native Windows and macOS runtime validation remains outstanding for this release.

See Dual-mode validation.


---

# Compile WebAssembly parsers

Use this guide to add a WASM parser to a grammar release. A working native host build remains necessary for the normal measured-ABI manifest.

## 1. Select a backend

```bash
ts-build config get runtime.wasm.backend
ts-build config get runtime.tools.emcc
```

Configure the available Emscripten or container backend and its executable paths. A container backend also requires a running container engine and a usable image.

The current Emscripten shell-wrapper path is unsupported on Windows. Use a supported Unix build environment for that backend. The Windows setup test does not validate WASM.

## 2. Generate sources

```bash
ts-build build --language python
```

Use an explicit `--config` on each command if automatic selection is not appropriate.

## 3. Compile the artifacts

Build the native host artifact for manifest measurement:

```bash
ts-build compile --language python
```

Build WASM separately:

```bash
ts-build compile --language python --wasm
```

The configured paths determine the output locations. Do not assume a checkout under the current directory or a fixed developer path.

## 4. Publish with WASM selected

```bash
ts-build move --language python --wasm --compact --clean=false
```

The mover includes the WASM file and checksum when WASM publication is selected. Host native validation supplies the normal manifest's measured ABI.

WASM integrity checks do not replace browser or WASM-runtime execution tests. Test the artifact in its intended consumer before distribution.

WARNING: `move --force` replaces the complete selected grammar directory. Include every artifact you intend to retain when publishing again.

For an existing output:

```bash
ts-build move --language python --wasm --compact --clean=false --force
```

Catalog replacement still requires a supported atomic exchange. See Build and release.

## Diagnose backend failures

- If a tool is not found, inspect `runtime.tools` and the command's executable path.
- If a container cannot start, verify the engine and selected image independently.
- If generation fails, resolve the pinned generator mismatch before compiling.
- If publication reports missing provenance or checksums, rebuild the artifact instead of editing metadata.

Use `--verbose` for streaming diagnostics. Captured command output is bounded; preserve external build logs when complete output is required.
