# How to Compile WebAssembly (WASM) Parsers

This guide explains how to compile Tree-sitter grammars into WebAssembly (`.wasm`) parser binaries using [`ts-build`](file:///home/michael/projects/go/tree-sitter/cmd/ts-build/main.go) and publish them alongside native shared libraries.

---

## 1. Overview & Prerequisites

[`ts-build`](file:///home/michael/projects/go/tree-sitter/cmd/ts-build/main.go) supports compiling grammars into standalone `.wasm` files (`tree-sitter-<lang>.wasm`) suitable for execution in browser runtimes, Node.js, or Go WebAssembly runtimes (such as `wazero`).

### Toolchain Discovery

The compiler ([`CompileWasm`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/compiler.go#L400)) automatically detects available WebAssembly toolchains on your `$PATH` in the following order of preference:

1. **Native Emscripten (`emcc`)**: Used directly if `emcc` is found on `$PATH`.
2. **Docker**: If `emcc` is not installed but `docker` is available, `ts-build` generates an ephemeral toolchain wrapper script that executes `emcc` inside the official `emscripten/emsdk` container image with volume mounts.
3. **Podman**: If `docker` is not available but `podman` is present, `ts-build` creates a container wrapper using `podman`.

If none of these three tools are detected on `$PATH`, WASM compilation fails with an explanatory error.

### Verification Checklist
Ensure at least one of the following commands succeeds:
```bash
# Option A: Native Emscripten
emcc --version

# Option B: Docker
docker --version

# Option C: Podman
podman --version
```

---

## 2. Step 1: Generate Grammar Sources

Before compiling to WASM, generate the grammar's C/C++ parser files:

```bash
go run ./cmd/ts-build build --language python
```

Output:
```text
[✓] build     python                         (1.2s)
```

This clones the upstream repository at its pinned commit and runs `tree-sitter generate` under `build/python/`.

---

## 3. Step 2: Compile the WASM Parser Artifact

Invoke the `compile` subcommand with the `--wasm` flag:

```bash
go run ./cmd/ts-build compile --language python --wasm
```

### What Happens Behind the Scenes
1. `ts-build` resolves the grammar directory under `build/python/` (accounting for any `source_subdir` if configured).
2. If `emcc` is not found natively, it creates an ephemeral wrapper directory containing an `emcc` shim that delegates to `docker` or `podman run --rm -v ... emscripten/emsdk emcc "$@"`.
3. Invokes `tree-sitter build --wasm --output <absPath>` inside the grammar directory.
4. Verifies that `tree-sitter-python.wasm` was generated as a regular, non-empty file.

Output:
```text
[✓] compile wasm python [wasm]               (4.8s)
```

The compiled WebAssembly binary is staged in the intermediate build directory:
- `build/python/tree-sitter-python.wasm`

---

## 4. Step 3: Publish WASM Artifacts to the Catalog

To move the compiled `.wasm` file into the publication directory [`data/tree-sitter/grammar/`](file:///home/michael/projects/go/tree-sitter/data/tree-sitter/grammar/) and record its presence in `manifest.json`, pass `--wasm` to the `move` subcommand:

```bash
go run ./cmd/ts-build move --language python --wasm --force
```

Output:
```text
[✓] move      data/tree-sitter/grammar       [1 grammars, 1 binaries] (0.1s)
```

### Published Artifacts
Inspect the published catalog directory:
```bash
ls -l data/tree-sitter/grammar/python/
```

The directory will contain:
- `python-v0.25.0-linux-amd64.so` (native shared library)
- `tree-sitter-python.wasm` (WebAssembly parser)
- `node-types.json`
- `queries/`
- `manifest.json`

---

## 5. Step 4: Verify `manifest.json` Metadata

Inspect [`data/tree-sitter/grammar/python/manifest.json`](file:///home/michael/projects/go/tree-sitter/data/tree-sitter/grammar/python/manifest.json):

```json
{
  "schema_version": 2,
  "grammar": "python",
  "version": "v0.25.0",
  "artifacts": {
    "has_node_types": true,
    "has_compact_node_types": false,
    "has_queries": true,
    "has_wasm": true,
    "has_c_source": false,
    "has_js": false
  }
}
```

Notice that `"has_wasm": true` is recorded in the `artifacts` block, allowing downstream release tooling or client applications to determine whether WASM artifacts are available without probing the filesystem.

---

## 6. Combining WASM with Native Multi-Platform Releases

You can combine native cross-compilation with WASM in your build scripts:

```bash
# 1. Generate grammar sources
go run ./cmd/ts-build build -l python

# 2. Compile host binary + WASM binary
go run ./cmd/ts-build compile -l python --os linux --arch amd64 --wasm

# 3. Publish native binary, WASM artifact, and queries
go run ./cmd/ts-build move -l python --wasm --force
```

---

## 7. Troubleshooting

### Error: "no WASM toolchain available"
```text
compile wasm python: no WASM toolchain available; install emcc or a container engine (docker or podman) with emscripten/emsdk
```
**Cause:** Neither `emcc`, `docker`, nor `podman` was located on `$PATH`.  
**Resolution:**
- Install Docker or Podman and verify with `docker ps` or `podman ps`.
- Or install the Emscripten SDK directly:
  ```bash
  git clone https://github.com/emscripten-core/emsdk.git
  cd emsdk
  ./emsdk install latest
  ./emsdk activate latest
  source ./emsdk_env.sh
  ```

### Docker Daemon Permission Denied
If using Docker and you see `permission denied while trying to connect to the Docker daemon socket`:
- Ensure your user is in the `docker` group (`sudo usermod -aG docker $USER`), or
- Use rootless `podman`, which requires no root daemon.
