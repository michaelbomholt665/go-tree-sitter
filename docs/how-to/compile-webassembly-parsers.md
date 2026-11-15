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

Catalog replacement still requires a supported atomic exchange. See [Build and release](build-and-release-grammars.md).

## Diagnose backend failures

- If a tool is not found, inspect `runtime.tools` and the command's executable path.
- If a container cannot start, verify the engine and selected image independently.
- If generation fails, resolve the pinned generator mismatch before compiling.
- If publication reports missing provenance or checksums, rebuild the artifact instead of editing metadata.

Use `--verbose` for streaming diagnostics. Captured command output is bounded; preserve external build logs when complete output is required.
