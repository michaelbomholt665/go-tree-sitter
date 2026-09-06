# CLI Command Reference (`ts-build`)

`ts-build` is the command-line interface for cloning, generating, cross-compiling, validating, and publishing Tree-sitter grammars.

---

## Synopsis

```bash
ts-build [command] [flags]
```

Or via Go:

```bash
go run ./cmd/ts-build [command] [flags]
```

---

## Persistent Global Flags

These flags are available on all subcommands:

| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--config` | `-c` | string | `tree-sitter-config.yaml` | Path to the YAML configuration file. |
| `--verbose` | `-v` | bool | `false` | Stream raw child-process stdout/stderr in real time. |
| `--quiet` | `-q` | bool | `false` | Silence all non-error output. |
| `--help` | `-h` | bool | | Display help information for any command. |

---

## Subcommands

### `ts-build wizard` *(recommended for human use)*

Launches a four-step interactive wizard that guides you through every configuration choice before running the build pipeline. No flags are required.

```bash
ts-build wizard
```

The wizard walks through the following steps in order:

| Step | Prompt | Description |
| :--- | :--- | :--- |
| 1 | **Pipeline** | Choose which phases to run: build only, build+compile, full end-to-end, compile only, or publish only. |
| 2 | **Target Platforms** | Multi-select from Linux amd64, macOS arm64, macOS amd64, Windows amd64, and WebAssembly. |
| 3 | **Artifacts & Options** | Toggle `manifest.json`, `.scm` query files, `compact-node-types.yaml`, C/C++ sources, `grammar.js`, WebAssembly staging, force-overwrite, and build-cache pruning. |
| 4 | **Grammars** | Select any subset of the 38 configured languages, or choose **[ ALL ]** to process everything. |

Before executing, the wizard displays a compact summary and prompts for final confirmation. Pressing `N` aborts cleanly without touching any files.

> [!NOTE]
> The wizard drives the same `build`, `compile`, and `move` pipeline as the individual subcommands — it is purely a human-friendly front-end, not a separate code path.

---

### 1. `ts-build build`

Clones grammar repositories, checks out pinned commit revisions, installs locked Node dependencies (if a `package-lock.json` is present), and runs `tree-sitter generate` with the configured ABI version.

#### Usage
```bash
ts-build build [flags]
```

#### Flags
| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | string | `""` | Only build the specified language (builds all configured if omitted). |
| `--force` | `-f` | bool | `false` | Re-clone existing repositories by wiping `build/<language>/` before building. |
| `--prune` | | bool | `false` | Prune build cache (`.git`, `node_modules`, test suites, intermediate object files) after building while preserving sources and manifests. |

#### Required System Tools
- `git`
- `tree-sitter` CLI (version must match `tree_sitter_cli_version` in config)
- `npm` (invoked only if repository includes a `package-lock.json`)

#### Outputs
- Grammar sources generated under `build/{language}/`
- Provenance manifest `.source-provenance.json` recording commit revision, generator version, and `SOURCE_DATE_EPOCH`.

---

### 2. `ts-build compile`

Compiles generated grammar sources into platform-specific native shared libraries (`.so`, `.dylib`, or `.dll`).

#### Usage
```bash
ts-build compile [flags]
```

#### Flags
| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | string | `""` | Only compile the specified language. |
| `--os` | `-o` | string | `""` | Target operating system (`linux`, `windows`, `macos`). Defaults to `OS_TARGET` from config. |
| `--arch` | `-a` | string | `""` | Target architecture (`amd64`, `arm64`). Defaults to target arch from config. |
| `--wasm` | | bool | `false` | Build WebAssembly (WASM) parser artifact (`tree-sitter-<lang>.wasm`). |
| `--allow-cross-validation` | | bool | `false` | Allow cross-platform static validation for non-host binaries. |
| `--static-cross-validation`| | bool | `false` | Alias for `--allow-cross-validation`. |

#### Outputs
- Compiled shared library: `build/{language}/{arch}/{language}-{version}-{platform}-{arch}.{ext}`
- WebAssembly artifact (when `--wasm` is passed): `build/{language}/tree-sitter-{language}.wasm`
- Binary provenance file: `*.provenance.json`

---

### 3. `ts-build move`

Stages compiled binaries, AST metadata, and query files; performs dual-mode validation; generates `manifest.json`; and atomically publishes the release catalog.

#### Usage
```bash
ts-build move [flags]
```

#### Flags
| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | string | `""` | Only move the specified language. |
| `--manifest` | | bool | `true` | Generate and validate `manifest.json`. Pass `--manifest=false` (or `--no-manifest`) to skip. |
| `--scm` | | bool | `true` | Publish `.scm` query files. Pass `--scm=false` (or `--no-scm`) to skip copying queries. |
| `--source` | | bool | `false` | Preserve C/C++ parser sources under `data/tree-sitter/grammar/<lang>/src/`. |
| `--c-source` | | bool | `false` | Alias for `--source`. |
| `--js` | | bool | `false` | Preserve `grammar.js` under `data/tree-sitter/grammar/<lang>/grammar.js`. |
| `--wasm` | | bool | `false` | Stage WebAssembly parser artifact under `data/tree-sitter/grammar/<lang>/tree-sitter-<lang>.wasm`. |
| `--compact` | | bool | `false` | Generate, losslessly validate, and stage `compact-node-types.yaml` alongside `node-types.json`, and set `has_compact_node_types: true` in `manifest.json`. |
| `--check-compact` | | bool | `false` | Audit existing `compact-node-types.yaml` against `node-types.json` without regenerating. |
| `--both` | | bool | `false` | Legacy move mode: Move binaries, `node-types.json`, and `queries/`. |
| `--json` | | bool | `false` | Legacy move mode: Move binaries and `node-types.json` only. |
| `--clean` | | bool | `true` | Clean the intermediate build directory for each successfully published language. Pass `--clean=false` to preserve build directories. |
| `--force` | `-f` | bool | `false` | Overwrite existing output files in the catalog. |
| `--allow-cross-validation` | | bool | `false` | Allow cross-platform static validation for non-host binaries. |
| `--static-cross-validation`| | bool | `false` | Alias for `--allow-cross-validation`. |

#### Asset Publishing Flags & Move Modes
By default, `ts-build move` publishes binaries, `node-types.json`, `.scm` queries, and `manifest.json`.
The granular flags allow independent control over published assets:
- `--manifest=false` (or `--no-manifest`): Skip generating and validating `manifest.json`.
- `--scm=false` (or `--no-scm`): Skip copying `queries/` into the destination directory.
- `--source` (or `--c-source`): Preserve compilable C/C++ sources (`parser.c`, `scanner.c`/`scanner.cc`, `tree_sitter/parser.h`, `tree_sitter/alloc.h`) under `src/` and mark `has_c_source: true` in `manifest.json`.
- `--js`: Preserve authoritative grammar specification `grammar.js` and mark `has_js: true` in `manifest.json`.
- `--wasm`: Validate, copy, and checksum `tree-sitter-<lang>.wasm` into the release catalog and mark `has_wasm: true` in `manifest.json`.
- `--compact`: Generate, losslessly validate, and stage token-efficient `compact-node-types.yaml` alongside `node-types.json` and mark `has_compact_node_types: true` in `manifest.json`.
- `--check-compact`: Verify existing `compact-node-types.yaml` against `node-types.json`.

Legacy mode flags (`--both`, `--json`) remain supported for backward compatibility, while explicit `--manifest` or `--scm` flags take precedence.

---

### 4. `ts-build compact`

Generates or validates token-efficient compact AST schema representations (`compact-node-types.yaml`) from `node-types.json` with strict bidirectional lossless validation.

#### Usage
```bash
ts-build compact [flags]
```

#### Flags
| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | string | `""` | Only compact or check the specified language (processes all configured if omitted). |
| `--check` | | bool | `false` | Validate existing `compact-node-types.yaml` without regenerating. Exits with code 1 upon any discrepancy. |

#### Examples
```bash
# Generate compact-node-types.yaml for python:
ts-build compact -l python

# Check / validate existing compact-node-types.yaml without regenerating:
ts-build compact -l python --check

# Audit all published grammars in data/tree-sitter/grammar/ in CI:
ts-build compact --check
```

---

### 5. `ts-build clean`

Cleans or selectively prunes intermediate build directories.

#### Usage
```bash
ts-build clean [flags]
```

#### Flags
| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | string | `""` | Only clean the specified language (cleans all configured if omitted). |
| `--prune` | | bool | `false` | Prune build cache (`.git`, `node_modules`, `test`, `corpus`, `examples`, `target`, `build/Release`, `*.o`, `*.obj`) while strictly preserving sources (`grammar.js`, `src/`), `queries/`, AST schemas (`node-types.json`), and ecosystem manifests (`tree-sitter.json`, `package.json`, `Cargo.toml`, `pyproject.toml`, `setup.py`, `Package.swift`, `go.mod`, `Makefile`, `binding.gyp`, `CMakeLists.txt`, `build.zig`, `LICENSE*`). |

#### Examples
```bash
# Completely remove build output for python:
ts-build clean -l python

# Prune bloat (.git, node_modules, tests) while keeping sources and manifests:
ts-build clean -l python --prune
```

---

### 6. `ts-build version`

Displays the version of `ts-build`.

```bash
ts-build version
# Output: ts-build version 1.0.0
```

---

### 7. `ts-build completion`

Generates auto-completion scripts for your shell.

#### Usage
```bash
ts-build completion [bash|zsh|fish|powershell]
```

#### Enabling Shell Completion
- **Bash:**
  ```bash
  source <(ts-build completion bash)
  ```
- **Zsh:**
  ```bash
  source <(ts-build completion zsh)
  ```
- **Fish:**
  ```bash
  ts-build completion fish | source
  ```
- **PowerShell:**
  ```powershell
  ts-build completion powershell | Out-String | Invoke-Expression
  ```

---

## Environment Variables

| Variable | Description |
| :--- | :--- |
| `SOURCE_DATE_EPOCH` | Overridden automatically by `ts-build` using the git commit timestamp for reproducible binary builds. |
| `CFLAGS` / `CXXFLAGS`| Injected during compilation using flags defined in `build_flags` (e.g. `-O2 -fPIC`). |
| `CC` / `CXX` | Set dynamically by `Compiler` to point to the resolved host compiler, cross-compiler, or Zig wrapper script. |
| `GOOS` / `GOARCH` | Set dynamically to target architecture. |
| `CGO_ENABLED` | Set to `1` during native library builds. |
