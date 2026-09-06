# How to Build and Release Grammars

This guide demonstrates how to build, compile, validate, and publish Tree-sitter grammars using both the [`Makefile`](file:///home/michael/projects/go/tree-sitter/Makefile) and the [`ts-build`](file:///home/michael/projects/go/tree-sitter/cmd/ts-build/main.go) command-line interface.

---

## 1. Running the Full Default Release

To run the complete host release pipeline (build sources, compile native libraries, losslessly generate `compact-node-types.yaml`, publish to catalog, and automatically delete intermediate build caches):

```bash
# Build, compile, compact, publish, and clean all 38 grammars
make build-all

# Or run the legacy release target:
make release
```

This target automatically executes four phases:
1. `check-tree-sitter`: Verifies that `tree-sitter` CLI is installed on `$PATH`.
2. `grammar-build`: Clones repositories and generates C sources for all configured languages.
3. `compile`: Compiles shared libraries (`.so`, `.dylib`, or `.dll`) for the target.
4. `move`: Validates artifacts, losslessly generates `compact-node-types.yaml`, writes `manifest.json`, atomically updates `data/tree-sitter/grammar/`, and wipes intermediate build caches.

---

## 2. Building a Specific Language

To build and release a single grammar rapidly, choose the approach that suits your workflow:

### Option A: Interactive Wizard (Recommended)

```bash
go run ./cmd/ts-build wizard
```

The wizard prompts you to select pipeline mode, target platforms, artifacts, and grammars interactively. A summary is displayed before anything runs, and you confirm with a single keypress. This is the recommended approach for day-to-day local builds.

### Option B: One-Step Makefile Pipeline

```bash
make build-all LANG=python
```

This single command clones python, compiles the native library, generates and validates `compact-node-types.yaml`, updates `manifest.json`, publishes to `data/tree-sitter/grammar/python/`, and completely deletes the intermediate `build/python/` directory.

### Option C: Step-by-Step via CLI

To control each phase individually using [`ts-build`](file:///home/michael/projects/go/tree-sitter/cmd/ts-build/main.go):

#### Step 1: Generate grammar source
```bash
go run ./cmd/ts-build build --language python
```

Use `--force` (`-f`) if you need to wipe existing clones and re-fetch from git:
```bash
go run ./cmd/ts-build build -l python -f
```

#### Step 2: Compile the shared library
```bash
go run ./cmd/ts-build compile --language python
```

#### Step 3: Validate, compact, and publish
```bash
go run ./cmd/ts-build move --language python --compact --force
```

---

## 3. Controlling Progress Output

`ts-build` defaults to clean, concise status updates:

```text
[✓] build     python                         (1.4s)
[✓] compile   python [linux/amd64]           (0.8s)
[✓] move      data/tree-sitter/grammar       [1 grammars, 1 binaries] (0.2s)
```

### Streaming Verbose Logs
If a build failure occurs or you need to inspect raw git/npm/compiler output, pass `--verbose` (`-v`):

```bash
go run ./cmd/ts-build build -v -l python
```

### Quiet Mode (CI / Scripting)
To suppress all non-error output:

```bash
go run ./cmd/ts-build build -q
```

---

## 4. Selecting Artifact Move Modes & Preservation Flags

When moving artifacts, you can choose which files accompany the compiled binaries via CLI flags or the `output.default_move_mode` configuration setting:

| Flag | Included Artifacts | Use Case |
| :--- | :--- | :--- |
| `--both` (default) | Binaries + `node-types.json` + `queries/` | Full feature release |
| `--json` | Binaries + `node-types.json` | Syntax trees and AST node inspection only |
| `--scm` | Binaries + `queries/` | Syntax highlighting and query matching only |

### Granular Asset Preservation Flags

In addition to the base move modes, `ts-build move` provides granular flags to preserve additional source and binary artifacts:

| Flag | Artifact Published | Manifest Flag Updated |
| :--- | :--- | :--- |
| `--compact` | Generates and validates `compact-node-types.yaml` alongside `node-types.json` | `has_compact_node_types: true` |
| `--check-compact` | Audits existing `compact-node-types.yaml` against `node-types.json` | `has_compact_node_types: true` |
| `--wasm` | Copies and checksums WebAssembly parser `tree-sitter-<lang>.wasm` | `has_wasm: true` |
| `--source` / `--c-source` | Preserves C/C++ parser sources under `src/` (`parser.c`, `scanner.c`, headers) | `has_c_source: true` |
| `--js` | Preserves grammar specification `grammar.js` | `has_js: true` |
| `--scm=false` (or `--no-scm`) | Skips copying query files (`queries/`) | `has_queries: false` |
| `--manifest=false` (or `--no-manifest`) | Skips generating `manifest.json` | (manifest omitted) |

Example publishing complete sources, WASM, and compact node types:
```bash
go run ./cmd/ts-build move -l python --source --js --wasm --compact --force
```

---

## 5. Cleaning and Pruning Build Directories

### Build Cache Pruning (`--prune`)
Cloning upstream repositories can introduce significant disk bloat from `.git` histories, `node_modules`, test corpus directories, and intermediate object files (`*.o`). 

You can prune this cache while strictly preserving grammar sources (`grammar.js`, `src/`), queries, and ecosystem manifests (`tree-sitter.json`, `package.json`, `Cargo.toml`, etc.):

```bash
# Prune immediately during build
go run ./cmd/ts-build build -l python --prune

# Or prune existing build directories after the fact
go run ./cmd/ts-build clean -l python --prune
```

### Full Clean
By default, `ts-build move` cleans up intermediate build directories under `build/<language>/` upon successful publication.

To retain intermediate build artifacts for debugging or inspection:
```bash
go run ./cmd/ts-build move --clean=false
```

To wipe intermediate build directories for a language completely:
```bash
go run ./cmd/ts-build clean --language python
```

To clean intermediate test output and coverage profiles manually:
```bash
make clean
```

---

## 6. Verifying the Published Catalog

Once published, inspect the catalog under `data/tree-sitter/grammar/<language>/`:

```bash
ls -l data/tree-sitter/grammar/python/
```

Verify that `manifest.json` is present and that its checksums match the binary files:
```bash
sha256sum data/tree-sitter/grammar/python/python-*.so
grep checksum_sha256 data/tree-sitter/grammar/python/manifest.json
```
