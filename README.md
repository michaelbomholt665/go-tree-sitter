# go-tree-sitter

> [!WARNING]
> **DISCLAIMER**
> - Code is 100% written by AI. Use it at your own risk.
> - The examples use the installed `ts-build` command. Add Go's binary directory to your `PATH`.

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Tree-sitter CLI](https://img.shields.io/badge/Tree--sitter_CLI-0.27.0-2B8A3E?style=flat&logo=tree-sitter)](https://github.com/tree-sitter/tree-sitter)
[![Parser ABI](https://img.shields.io/badge/Parser_ABI-15_(runtime_13--15)-blue?style=flat)](docs/explanation/abi-evolution-and-runtime-compatibility.md)
[![Platforms](https://img.shields.io/badge/Targets-Linux_|_macOS_|_Windows_|_WASM-lightgrey?style=flat)](#multi-platform-compilation)
[![Catalog](https://img.shields.io/badge/Catalog-65_Grammar_Entries-success?style=flat)](docs/reference/grammar-catalog.md)

A reproducible Tree-sitter grammar builder, cross-compiler, AST compressor, validator, and publication toolchain for Go applications and downstream AI agents.

`go-tree-sitter` clones pinned upstream grammar repositories, generates C parser sources, compiles native libraries and WebAssembly parsers, validates artifacts, and publishes release catalogs. Version **1.10.0** includes 65 configured grammar entries; see [CHANGELOG.md](CHANGELOG.md) for changes since 1.0.0.

---

## Features

- **Pinned source revisions:** Every configured grammar requires a full, 40-character hexadecimal Git commit SHA. Branch names, tags, and short revisions are rejected.
- **AI-friendly compact node types:** Convert `node-types.json` into `compact-node-types.yaml` with bidirectional semantic validation. The compact schema preserves named and anonymous identities, operator alternatives, field constraints, and root/extra metadata. Size reduction depends on the grammar.
- **Dual-mode binary validation:**
  - Host validation loads the native parser, measures its ABI, parses the configured sample, rejects syntax errors, and compiles queries.
  - Foreign validation inspects binary structure, architecture, and exports without an emulator. It does not measure the parser ABI or qualify an artifact for a measured-ABI manifest.
- **Native and WebAssembly compilation:** Build `.so`, `.dylib`, `.dll`, and optional `.wasm` artifacts with configured compiler toolchains, Emscripten, or container backends.
- **Granular asset preservation:** Select compact schemas, WASM, C/C++ sources, `grammar.js`, queries, and manifests with `--compact`, `--wasm`, `--source`, `--js`, `--scm`, and `--manifest`.
- **Guarded atomic publication:** Validate a staged catalog under an interprocess lock. Replace existing catalogs through atomic directory exchange on supported Linux/macOS filesystems. Unsupported replacement fails without removing the current catalog.
- **Source and binary provenance:** Record source revisions, generator/compiler versions, flags, source hashes, binary checksums, and commit-derived `SOURCE_DATE_EPOCH`. Hashes detect changed inputs and artifacts; they are not digital signatures.
- **Configuration and runtime policy:** Inspect, validate, preview, and edit settings with `config`. Control tool paths, compiler fallback, dependency installation, cleanup, publication, and wizard defaults.
- **Catalog and support evidence:** Inspect grammar identities with `list` and revision-bound source-language assessments with `support`. Grammar package versions and parser ABIs are not source-language version guarantees.
- **Version synchronization and export:** Synchronize package labels from `go.mod` without replacing source pins, and export already published artifacts to another directory.
- **Windows setup and testing:** Use the PowerShell installer and a documented, isolated Python-only VM test. Native Windows/macOS runtime behavior remains unverified.

---

## Requirements

Install the tools required for your selected pipeline:

| Tool | Requirement |
| --- | --- |
| Go | 1.26 or later for installation, development, and Go consumers |
| Git | Source checkout at the configured revision |
| Tree-sitter CLI | The exact configured version; the bundled catalog pins 0.27.0 |
| Node.js/npm | Grammar JavaScript evaluation and dependency installation when required by the grammar |
| C/C++ compiler | GCC, Clang, MinGW, or a configured Zig toolchain; the compiler version must match its target pin |
| Emscripten or containers | For WASM compilation: local `emcc`, or a working Docker/Podman backend |

The compiler pin describes the compiler that actually runs. Inspect `ts-build config get targets` before compiling; do not change a pin merely to suppress an unexplained mismatch.

On Windows, use a native Tree-sitter executable rather than a batch wrapper. Follow [TEST-GUIDE.md](TEST-GUIDE.md); [scripts/install-windows-tools.ps1](scripts/install-windows-tools.ps1) handles tool installation, repository dependencies, and isolated test configuration.

---

## Quick Start

### 1. Install and select a configuration

```bash
go install github.com/michaelbomholt665/go-tree-sitter/cmd/ts-build@v1.10.0
ts-build version
ts-build config path
ts-build config validate
ts-build list
```

Normal use selects the OS user configuration. The first run creates a missing configuration from the catalog embedded in the executable. An upgrade does not overwrite an existing user configuration.

If the current directory's `.env` contains `TS_BUILD_ENV=DEVELOPMENT`, the default becomes `./tree-sitter-config.yaml`. The repository includes this marker. The CLI does not search parent directories for it.

Use an explicit file to make scripts independent of automatic selection:

```bash
ts-build --config ./tree-sitter-config.yaml config validate
```

Paths inside the configuration resolve relative to its directory. Use the [getting-started tutorial](docs/tutorials/getting-started.md) for an isolated Linux configuration and a Python-only first build.

### 2. Interactive Wizard (Recommended for Human Use)

```bash
ts-build wizard
```

The wizard guides you through four choices:

1. **Pipeline:** Select source generation, compilation, or publication stages.
2. **Target platforms:** Select the configured native platforms or WebAssembly.
3. **Artifacts and options:** Select manifests, queries, compact schemas, sources, replacement, and cleanup behavior.
4. **Grammars:** Select a subset of the 65 configured entries, or select all.

Review the summary before confirming execution. Platform selection does not bypass the validation and publication restrictions below.

### 3. Build one grammar directly

With a validated configuration and matching host toolchain:

```bash
ts-build build --language python
ts-build compile --language python
ts-build move --language python --compact --clean=false
ts-build compact --language python --check
```

These commands select only Python. Omitting `--language` selects the configured catalog. The pipeline separates three responsibilities:

| Stage | Result |
| --- | --- |
| `build` | Pinned checkout, generated parser sources, source provenance |
| `compile` | Native or WASM artifact and binary provenance |
| `move` | Validated, staged publication with selected assets and metadata |

If the selected grammar output already exists, replacement requires `move --force`. Read [Publication guarantees and limits](#publication-guarantees-and-limits) before replacing a release.

### 4. Parse code in Go

An upstream Go binding compiles grammar sources into your consumer with CGo. It does not load the shared library published by `ts-build move`.

In a separate consumer module, install the pinned runtime and Python binding:

```bash
go get github.com/tree-sitter/go-tree-sitter@v0.25.0
go get github.com/tree-sitter/tree-sitter-python@v0.25.1-0.20250915213335-26855eabccb1
```

```go
// main.go
package main

import (
    "fmt"

    sitter "github.com/tree-sitter/go-tree-sitter"
    python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

func main() {
    parser := sitter.NewParser()
    defer parser.Close()

    language := sitter.NewLanguage(python.Language())
    if err := parser.SetLanguage(language); err != nil {
        panic(err)
    }

    tree := parser.Parse([]byte("print('hello')\n"), nil)
    if tree == nil {
        panic("parse failed")
    }
    defer tree.Close()

    root := tree.RootNode()
    if root.HasError() {
        panic("sample contains a parse error")
    }
    fmt.Println(root.ToSexp())
}
```

External modules cannot import this repository's `internal/grammars` registry. For module setup and native shared-library integration, see [Use Tree-sitter grammars from Go](docs/how-to/use-compiled-grammars-in-go.md).

---

## Common Workflows

### The step-by-step CLI pipeline

Run `build`, `compile`, and `move` separately to inspect intermediate output or diagnose a failed stage. Use `--config` for an explicit configuration, `--language` for a selected grammar, and `--verbose` for child-process diagnostics.

```bash
ts-build --config ./tree-sitter-config.yaml build --language python
ts-build --config ./tree-sitter-config.yaml compile --language python
ts-build --config ./tree-sitter-config.yaml move --language python --compact --clean=false
```

See [Build and release grammars](docs/how-to/build-and-release-grammars.md) for publication flags and cleanup policy.

### Multi-Platform Compilation

After generating sources, compile configured foreign targets with the required cross-toolchains:

```bash
ts-build compile --language python --os windows --arch amd64 --allow-cross-validation
ts-build compile --language python --os macos --arch arm64 --allow-cross-validation
ts-build compile --language python --os macos --arch amd64 --allow-cross-validation
```

Foreign static validation does not execute the parser or measure its ABI. It cannot produce a normal measured-ABI manifest through cross-validation alone. Build, validate, and publish on the target host when a measured manifest is required.

The [cross-compilation guide](docs/how-to/cross-compile-grammars.md) describes toolchain configuration and the explicit raw-artifact publication path without a custom manifest. Zig and container shell wrappers are not supported on Windows in this implementation.

### WebAssembly (WASM) Compilation

Generate sources and compile a WASM parser with the configured backend:

```bash
ts-build build --language python
ts-build compile --language python --wasm
```

To include WASM alongside a validated host native parser in a fresh release:

```bash
ts-build compile --language python
ts-build move --language python --wasm --compact --clean=false
```

The `.wasm` artifact does not replace the native ABI measurement required by the custom manifest. See [Compile WebAssembly parsers](docs/how-to/compile-webassembly-parsers.md).

### Generating and Auditing Compact Node Types

Generate or audit compact AST schemas without rebuilding grammars:

```bash
ts-build compact --language python
ts-build compact --language python --check
ts-build compact --check
```

`--check` validates the existing YAML against `node-types.json` without rewriting it. The last command audits the published catalog. See [Generate and validate compact node types](docs/how-to/generate-and-validate-compact-node-types.md).

### Configure tools, inspect support, synchronize, and export

```bash
ts-build config show
ts-build config get targets
ts-build config set runtime.compiler.allow_fallback false
ts-build support show --language python
ts-build sync-config --gomod ./go.mod --dry-run
ts-build export --language python --to ./exported-grammars
```

`config set` previews a change unless `--apply` is present. Maps and lists are complete settings; `targets.linux.compiler_version` is not an editable key. Version synchronization changes package labels, not pinned revisions. Export copies artifacts that are already published.

See [Configure runtime policy and inspect the catalog](docs/how-to/configure-runtime-and-catalog.md) and [Synchronize versions with go.mod](docs/how-to/sync-versions-with-go-mod.md).

---

## Publication guarantees and limits

Validation failures leave the published catalog unchanged. An interprocess lock covers staging and publication so concurrent publishers do not silently discard each other's updates.

Existing catalog replacement uses atomic directory exchange on supported Linux/macOS filesystems. If exchange is unavailable, replacement fails rather than creating a missing-catalog interval. Windows catalog replacement is unsupported; use a new output directory for each release. `--force` does not bypass this restriction.

`move --force` replaces the entire selected grammar directory. It does not merge unselected artifacts from the old grammar release. Include every artifact you intend to retain.

---

## Release Catalog Layout

Published output lives under the configured `output.grammar_base`. The repository development configuration uses `data/tree-sitter/grammar/`; installed configurations use an OS user data directory.

An illustrative Python release contains:

```text
# <output.grammar_base>/python/
├── python-<version>-linux-amd64.so   # Host native parser
├── tree-sitter-python.wasm          # Optional WASM parser
├── node-types.json                 # Full AST schema
├── compact-node-types.yaml         # Optional compact AST schema
├── manifest.json                   # Optional custom consumer metadata
├── queries/                        # Published query files
├── src/                            # Optional C/C++ parser sources
└── grammar.js                      # Optional grammar definition
```

### Downstream Manifest Schema (`manifest.json`)

The custom manifest serves downstream consumers such as the graph builder. It records grammar/package identity, target metadata, measured parser ABIs, checksums, artifact flags, and build provenance. The builder itself does not require it.

For example, these fields identify a measured ABI and generator version. This is an excerpt, not a complete manifest:

```json
{
  "schema_version": 2,
  "grammar": "python",
  "tree_sitter_version": "0.27.0",
  "parser_abi": 15
}
```

Manifest-only regeneration without binary validation is blocked. Foreign static validation does not satisfy the measured-ABI requirement. See the [Manifest Schema v2 specification](docs/reference/manifest-schema-v2.md) for the complete structure and invariants.

### AI-friendly schemas (`compact-node-types.yaml`)

Compact schemas serve downstream AI agents and AST consumers. They preserve the source schema's meaning while removing verbose JSON structure. Semantic round-trip validation is required; a fixed percentage of byte or token savings is not guaranteed.

---

## Documentation

The [documentation index](docs/README.md) organizes material by the Diátaxis framework:

- **Tutorials**
  - [Getting started](docs/tutorials/getting-started.md)
- **How-to guides**
  - [Build and release grammars](docs/how-to/build-and-release-grammars.md)
  - [Configure runtime policy and inspect the catalog](docs/how-to/configure-runtime-and-catalog.md)
  - [Compile WebAssembly parsers](docs/how-to/compile-webassembly-parsers.md)
  - [Generate and validate compact node types](docs/how-to/generate-and-validate-compact-node-types.md)
  - [Cross-compile for Windows and macOS](docs/how-to/cross-compile-grammars.md)
  - [Add a new grammar](docs/how-to/add-a-new-grammar.md)
  - [Synchronize versions with go.mod](docs/how-to/sync-versions-with-go-mod.md)
  - [Use grammars from Go](docs/how-to/use-compiled-grammars-in-go.md)
- **Reference**
  - [CLI commands](docs/reference/cli.md)
  - [Configuration schema](docs/reference/configuration-schema.md)
  - [Compact node types schema](docs/reference/compact-node-types-schema.md)
  - [Manifest Schema v2](docs/reference/manifest-schema-v2.md)
  - [Grammar catalog and symbol registry](docs/reference/grammar-catalog.md)
  - [Source-language version verification](docs/reference/VERIFY_VERSIONS.md)
- **Explanation**
  - [Architecture and pipeline](docs/explanation/architecture-and-pipeline.md)
  - [Compact node types and token efficiency](docs/explanation/compact-node-types-and-token-efficiency.md)
  - [Dual-mode validation](docs/explanation/dual-mode-validation.md)
  - [Reproducible builds and provenance](docs/explanation/reproducible-builds-and-provenance.md)
  - [ABI and runtime compatibility](docs/explanation/abi-evolution-and-runtime-compatibility.md)

For release changes, see [CHANGELOG.md](CHANGELOG.md). For Windows setup and the Python-only VM test, see [TEST-GUIDE.md](TEST-GUIDE.md).

## Development and licenses

From a source checkout:

```bash
go mod download
go test ./...
go test -race -tags grammars ./...
```

See [LICENSE](LICENSE) and [NOTICE.md](NOTICE.md). The CLI license check passes, but the optional Cypher Go binding has an unresolved license classification. Go dependency scans do not verify every native grammar license at its pinned revision.
