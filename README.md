# go-tree-sitter

> [!WARNING]
> **DISCLAIMER**
> - code is 100% written by AI, use at your own risk
> - running it uncompiled, use "go run ./cmd/...", if its in your PATH, use ts-build

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Tree-sitter CLI](https://img.shields.io/badge/Tree--sitter_CLI-0.26.8-2B8A3E?style=flat&logo=tree-sitter)](https://github.com/tree-sitter/tree-sitter)
[![Parser ABI](https://img.shields.io/badge/Parser_ABI-15_(backwards_13--15)-blue?style=flat)](docs/explanation/abi-evolution-and-runtime-compatibility.md)
[![Platforms](https://img.shields.io/badge/Platforms-Linux_|_macOS_|_Windows_|_WASM-lightgrey?style=flat)](#multi-platform-compilation)
[![Catalog](https://img.shields.io/badge/Supported_Grammars-38_Languages-success?style=flat)](docs/reference/grammar-catalog.md)

A reproducible Tree-sitter grammar builder, cross-compiler, AST compressor, validator, and publication toolchain for Go applications and downstream AI agents.

`go-tree-sitter` clones upstream grammar repositories, generates C parser source files with pinned ABI versions, cross-compiles native shared libraries (`.so`, `.dylib`, `.dll`) and WebAssembly (`.wasm`), performs **Dual-Mode Validation** against runtime and binary invariants, compresses AST schemas into token-efficient YAML to prevent LLM context rot, and atomically publishes verified release catalogs to [`data/tree-sitter/grammar/`](data/tree-sitter/grammar/).

---

## Features

- **Deterministic Reproducibility**: Every grammar in [`tree-sitter-config.yaml`](tree-sitter-config.yaml) is pinned to a full 40-character hexadecimal git commit SHA (`revision`). Short commits, branch names, and tags are strictly rejected.
- **AI-Friendly Compact Node Types (`compact-node-types.yaml`)**: Built for LLMs and autonomous coding agents. Losslessly converts verbose ~100 KB `node-types.json` files into compact ~20 KB YAML files (**75%–83% token reduction**) with 100% bidirectional schema validation to prevent LLM context rot.
- **Dual-Mode Binary Validation**:
  - **Host Target (e.g. Linux on Linux)**: Dynamic runtime validation via `dlopen` executing grammar constructors, ABI measurements, sample parsing, syntax error rejection, and query compilation.
  - **Foreign Target (e.g. Windows `.dll`, macOS `.dylib` on Linux)**: Static binary header inspection validating Windows PE (`debug/pe`) and macOS Mach-O (`debug/macho`) export tables, architecture signatures, and symbols without foreign emulators.
- **WebAssembly (WASM) Support**: Compiles `.wasm` parser binaries via Emscripten (`emcc`) or automated Docker/Podman fallbacks for sandboxed or pure-Go runtime environments.
- **Granular Asset Preservation**: Choose what to stage and publish using modular flags: `--compact`, `--wasm`, `--source` (C/C++ sources for CGo embedding), `--js` (`grammar.js`), `--scm` (query files), and `--manifest`.
- **Zero-Downtime Atomic Swaps**: Production catalogs are updated via filesystem directory renames (`.grammar-staging-*` → `data/tree-sitter/grammar/`). Releases abort cleanly and leave existing catalogs untouched if any artifact fails validation.
- **Cryptographic Provenance**: Every artifact is stamped with `SOURCE_DATE_EPOCH`, generator version, compiler flags, and AST checksums forming an unbroken provenance chain in `manifest.json`.

---

## Requirements

Ensure the following tools are installed on your host machine:

- **Go 1.26 or higher**
- **Tree-sitter CLI 0.26.8** (install via `make install-tree-sitter-cli` or `npm install -g tree-sitter-cli@0.26.8`)
- **C/C++ Compiler**: GCC, Clang, MinGW (for Windows cross-compilation), or [Zig](https://ziglang.org/) (for universal cross-compilation)
- **Git**

> [!NOTE]
> WebAssembly compilation requires local Emscripten (`emcc`) or a running container engine (`docker` or `podman`).

---

## Quick Start

### 1. Interactive Wizard (Recommended for Human Use)

The easiest way to run the build pipeline is the interactive wizard. It guides you through every configuration choice step by step — no flags to memorise:

```bash
go run ./cmd/ts-build wizard
```

The wizard walks you through four steps:

1. **Pipeline** — choose how far to run (build only, build+compile, full end-to-end, etc.)
2. **Target Platforms** — select one or more of Linux, macOS arm64/amd64, Windows, or WebAssembly
3. **Artifacts & Options** — toggle `manifest.json`, `.scm` query files, compact node types, C sources, `grammar.js`, WASM staging, force-overwrite, and build-cache pruning
4. **Grammars** — pick any subset of the 38 configured languages (or select all)

A compact summary is shown before anything runs, and you confirm or abort with a single keypress.

### 2. One-liner Makefile Pipeline (Scripting / CI)

Run the complete pipeline for a single language (e.g., `python`) non-interactively:

```bash
make build-all LANG=python
```

This single command:
1. Clones the pinned repository revision into `build/python/amd64/`.
2. Generates C parser sources with `tree-sitter generate --abi 15`.
3. Compiles the native shared library (`.so`, `.dylib`, or `.dll`).
4. Generates and losslessly validates `compact-node-types.yaml`.
5. Validates binary ABIs, query syntax, and AST schemas.
6. Atomically updates [`data/tree-sitter/grammar/python/`](data/tree-sitter/grammar/python/) and wipes intermediate build caches.

### 3. Parse Code in Go

Once published, you can consume compiled grammars either via dynamic runtime loading or static CGo bindings:

```go
package main

import (
	"fmt"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/michaelbomholt665/go-tree-sitter/internal/grammars"
)

func main() {
	parser := tree_sitter.NewParser()
	defer parser.Close()

	// Load language via constructor registry
	lang := grammars.GetLanguage("python")
	if lang == nil {
		panic("python grammar not registered")
	}
	parser.SetLanguage(lang)

	sourceCode := []byte("def hello_world():\n    print('Hello, Tree-sitter!')\n")
	tree := parser.Parse(sourceCode, nil)
	defer tree.Close()

	root := tree.RootNode()
	fmt.Printf("Root node: %s [%d-%d]\n", root.Kind(), root.StartByte(), root.EndByte())
	fmt.Printf("S-expression:\n%s\n", root.ToSexp())
}
```

---

## Common Workflows

### Interactive Wizard (Recommended for Human Use)

```bash
go run ./cmd/ts-build wizard
```

Guides you through pipeline mode, target platforms, artifact selection, and grammar choice interactively. Ideal for day-to-day local builds.

### The Step-by-Step CLI Pipeline (`ts-build`)

You can execute each phase of the build toolchain individually:

```bash
# Phase 1: Source Generation
ts-build build -l python

# Phase 2: Compilation
ts-build compile -l python

# Phase 3: Validation, Compaction, and Atomic Publishing
ts-build move -l python --compact --force
```

### Multi-Platform Compilation

Compile native shared libraries for multiple target operating systems from a single Linux host:

```bash
# Compile for Windows amd64 (.dll)
ts-build compile -l python --os windows --arch amd64

# Compile for macOS arm64 & amd64 (.dylib)
ts-build compile -l python --os macos --arch arm64
ts-build compile -l python --os macos --arch amd64

# Compile for Linux host (.so)
ts-build compile -l python --os linux --arch amd64

# Publish all binaries with dual-mode validation
ts-build move -l python --compact --force
```

### WebAssembly (WASM) Compilation

Generate WebAssembly parser binaries for in-browser execution or pure-Go WASM runtimes:

```bash
# Compile WASM artifact (tree-sitter-<lang>.wasm)
ts-build compile -l python --wasm

# Stage, checksum, and publish WASM parser
ts-build move -l python --wasm --force
```

### Generating & Auditing Compact Node Types

Manage token-efficient AST schemas independently using the `compact` subcommand:

```bash
# Generate compact-node-types.yaml from node-types.json
ts-build compact -l python

# Audit lossless schema capture without modifying files (returns code 1 on discrepancy)
ts-build compact -l python --check

# Audit all 38 published grammars in CI
ts-build compact --check
```

---

## Makefile Automation

| Command | Description |
| :--- | :--- |
| `make build-all [LANG=...]` | Full pipeline for host target: build, compile, compact, publish, and clean build cache. |
| `make grammar-build [LANG=...]` | Clones upstream repos and generates C parser sources. |
| `make compile [LANG=...]` | Compiles native shared libraries for the default `OS_TARGET`. |
| `make compile-wasm [LANG=...]` | Compiles WebAssembly parser binaries (`tree-sitter-<lang>.wasm`). |
| `make compile-all` | Cross-compiles for Linux, Windows, and macOS. |
| `make move [LANG=...]` | Validates binaries, generates compact node types, and atomically publishes. |
| `make release` | Complete release pipeline for the active target configured in `tree-sitter-config.yaml`. |
| `make release-all` | Full release pipeline across all supported platforms. |
| `make sync-config` | Dry-run preview of grammar version synchronizations from `go.mod`. |
| `make sync-config-apply` | Synchronizes grammar versions from `go.mod` to `tree-sitter-config.yaml`. |
| `make test` | Runs the test suite with race detection and coverage reporting. |
| `make clean` | Cleans temporary test files and coverage profiles. |

---

## Release Catalog Layout

Each language catalog directory in `data/tree-sitter/grammar/<language>/` contains:

```
data/tree-sitter/grammar/python/
├── python-v0.25.0-linux-amd64.so       # Compiled native shared library
├── tree-sitter-python.wasm              # WebAssembly parser (optional with --wasm)
├── node-types.json                      # Full Tree-sitter AST schema
├── compact-node-types.yaml              # AI-friendly compressed AST schema (~80% token reduction)
├── manifest.json                        # Metadata, measured ABI, checksums & provenance
├── queries/                             # Syntax highlighting and tagging queries (*.scm)
├── src/                                 # C/C++ parser sources (optional with --source)
└── grammar.js                           # Upstream grammar definition (optional with --js)
```

### Downstream Manifest Schema (`manifest.json`)

The custom `manifest.json` provides programmatic version control, binary checksums, and measured parser ABIs:

```json
{
  "schema_version": 2,
  "grammar": "python",
  "version": "v0.25.0",
  "compiled_at": "2026-09-06T12:04:17Z",
  "tree_sitter_version": "0.26.8",
  "parser_abi": 15,
  "binaries": [
    {
      "platform": "linux",
      "arch": "amd64",
      "filename": "python-v0.25.0-linux-amd64.so",
      "checksum_sha256": "73652f6601772fdcceb74232f2f3ec2e197cab1c5be88e8322f9babd7afeceb4"
    }
  ],
  "artifacts": {
    "has_node_types": true,
    "has_compact_node_types": true,
    "has_queries": true,
    "has_wasm": false,
    "has_c_source": false,
    "has_js": false
  }
}
```

---

## Documentation

Comprehensive documentation organized according to the [Diátaxis framework](https://diataxis.fr/) is available in [`docs/`](docs/README.md):

- **Tutorials**:
  - [Getting Started with go-tree-sitter](docs/tutorials/getting-started.md)
- **How-To Guides**:
  - [How to Build and Release Grammars](docs/how-to/build-and-release-grammars.md)
  - [How to Compile WebAssembly (WASM) Parsers](docs/how-to/compile-webassembly-parsers.md)
  - [How to Generate and Validate Compact Node Types](docs/how-to/generate-and-validate-compact-node-types.md)
  - [How to Cross-Compile for Windows and macOS](docs/how-to/cross-compile-grammars.md)
  - [How to Add a New Grammar Language](docs/how-to/add-a-new-grammar.md)
  - [How to Synchronize Versions with go.mod](docs/how-to/sync-versions-with-go-mod.md)
  - [How to Consume and Parse in Go Applications](docs/how-to/use-compiled-grammars-in-go.md)
- **Reference**:
  - [CLI Command Reference (`ts-build`)](docs/reference/cli.md)
  - [Configuration Schema Reference (`tree-sitter-config.yaml`)](docs/reference/configuration-schema.md)
  - [Compact Node Types Schema Reference](docs/reference/compact-node-types-schema.md)
  - [Manifest Schema v2 Specification](docs/reference/manifest-schema-v2.md)
  - [Supported Grammar Catalog & Symbol Registry](docs/reference/grammar-catalog.md)
  - [Makefile Target Reference](docs/reference/makefile.md)
- **Explanation**:
  - [Architecture and Pipeline Design](docs/explanation/architecture-and-pipeline.md)
  - [Compact Node Types and Token Efficiency](docs/explanation/compact-node-types-and-token-efficiency.md)
  - [Dual-Mode Validation Mechanics](docs/explanation/dual-mode-validation.md)
  - [Reproducible Builds, Provenance, and Determinism](docs/explanation/reproducible-builds-and-provenance.md)
  - [Tree-sitter ABI & Runtime Compatibility](docs/explanation/abi-evolution-and-runtime-compatibility.md)
