# AI Agent Guide for `go-tree-sitter`

Welcome to **go-tree-sitter**. This guide provides autonomous AI coding agents with essential context, architectural rules, command references, and actionable workflows for operating safely and effectively within this repository.

---

## 1. Project Overview & Core Mission

`go-tree-sitter` is a reproducible Tree-sitter grammar builder, compiler, validator, and publication toolchain for Go applications. It clones upstream grammar repositories, generates C parser source files with pinned ABI versions, cross-compiles native shared libraries (`.so`, `.dylib`, `.dll`) and WebAssembly (`.wasm`), validates binaries via **Dual-Mode Validation**, and atomically publishes verified release catalogs to [`data/tree-sitter/grammar/`](file:///home/michael/projects/go/tree-sitter/data/tree-sitter/grammar/).

### Non-Negotiable Invariants for AI Agents

1. **Deterministic Reproducibility**: Every grammar in [`tree-sitter-config.yaml`](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml) **must** pin a full 40-character hexadecimal git commit SHA (`revision`). Short commits, branch names, or tags are strictly rejected.
2. **Cryptographic Provenance**: Every artifact is stamped with `SOURCE_DATE_EPOCH` derived from the source commit timestamp. Generation tool versions, compiler flags, and AST checksums form an unbroken provenance chain.
3. **No Unvalidated Manifests (Downstream Consumer Metadata)**: The `manifest.json` file (see [`data/tree-sitter/grammar/c-sharp/manifest.json`](file:///home/michael/projects/go/tree-sitter/data/tree-sitter/grammar/c-sharp/manifest.json)) was **not built for this app**—it was created specifically for downstream applications where the grammars will be consumed (such as the author's **graph builder** to easily control versions, target platforms, and parser ABI versions). It is generated during `build`/`move` because that is the most reliable time to measure real binary ABIs and checksums. Manifest-only regeneration without validating native binaries is forbidden ([`cmd/regen-manifests/main.go`](file:///home/michael/projects/go/tree-sitter/cmd/regen-manifests/main.go) intentionally exits with an error) because downstream consumers require metadata measured from real compiled binaries.
4. **Zero-Downtime Atomic Swaps**: The production catalog [`data/tree-sitter/grammar/`](file:///home/michael/projects/go/tree-sitter/data/tree-sitter/grammar/) is updated via atomic filesystem directory swaps. If validation fails on any artifact, the entire release is aborted and the production directory remains untouched.

### Why Downstream Artifacts are Generated Here
Both `manifest.json` and `compact-node-types.yaml` exist to serve external downstream consumers rather than `go-tree-sitter` itself:
- **`manifest.json`**: Built for the **graph builder** to inspect and control grammar versions, target platforms, binary checksums, and measured parser ABI versions (`parser_abi: 15`).
- **`compact-node-types.yaml`**: Built for **AI agents and LLMs**. It turns a massive, verbose ~100 KB `node-types.json` into an AI-friendly ~20 KB YAML file (~75–80% token reduction), enabling downstream AI models to read and reason about the full AST schema without causing **context rot**.
- Generating both artifacts during `ts-build move` is simply the easiest and most cohesive place in the pipeline to capture them, as the builder already has the compiled binaries, measured ABIs, checksums, and AST schemas in memory.

---

## 2. Repository Layout

```
.
├── tree-sitter-config.yaml     <-- Central configuration (38 grammars, toolchains, outputs)
├── Makefile                    <-- Convenience automation targets
├── go.mod / go.sum             <-- Go dependencies and runtime bindings
├── cmd/
│   ├── ts-build/               <-- Main CLI toolchain entrypoint
│   ├── sync-config/            <-- Tool to sync go.mod versions into tree-sitter-config.yaml
│   └── regen-manifests/        <-- Deprecated stub (deliberately blocked)
├── internal/
│   ├── grammars/               <-- Go binding registry & CGo constructors (-tags grammars)
│   ├── tests/                  <-- Integration, artifact, and build tests
│   └── tree-sitter/            <-- CLI app and build orchestration
│       ├── cli.go              <-- Cobra CLI commands and flag definitions
│       ├── config.go           <-- Configuration schema parser & validator
│       ├── reporter.go         <-- TextReporter (concise status) & SilentReporter
│       └── build/
│           ├── builder.go      <-- Source checkout & tree-sitter generate
│           ├── compiler.go     <-- Native shared library & WASM compilation
│           ├── mover.go        <-- Staging, validation, manifest writing & atomic publish
│           ├── compact.go      <-- Compact node types generator & lossless validator
│           ├── validator.go    <-- Dual-mode validator (dynamic dlopen & static headers)
│           ├── manifest.go     <-- Manifest schema v2 serializer & invariant checks
│           ├── cleaner.go      <-- Directory cleaner & build cache pruner
│           ├── provenance.go   <-- SourceProvenance & BinaryProvenance structures
│           └── paths.go        <-- Path normalization & template resolvers
├── data/tree-sitter/grammar/   <-- Published release catalog (binaries, ASTs, queries)
└── docs/                       <-- Diátaxis documentation suite (Tutorials, How-To, Reference, Explanation)
```

---

## 3. Command Reference

The primary binary is [`ts-build`](file:///home/michael/projects/go/tree-sitter/cmd/ts-build/main.go), runnable via `go run ./cmd/ts-build <subcommand> [flags]`.

### Global Flags

Available on all subcommands:

| Flag | Shorthand | Default | Description | When to Use |
| :--- | :--- | :--- | :--- | :--- |
| `--config` | `-c` | `tree-sitter-config.yaml` | Path to YAML configuration. | Use when testing alternative configs. |
| `--verbose`| `-v` | `false` | Stream raw child-process stdout/stderr. | Use when diagnosing build, compile, or git errors. |
| `--quiet`  | `-q` | `false` | Silence non-error output. | Use in automated scripts or quiet CI checks. |

---

### Subcommands

#### 0. `wizard` (Interactive Build Wizard — Recommended for Human Use)
Guides the user through pipeline mode, target platform(s), artifact selection, and grammar selection interactively, then executes the chosen pipeline.

```bash
go run ./cmd/ts-build wizard
```

No flags required. The wizard is the recommended entry point for day-to-day human use.

#### 1. `build` (Source Generation)
Clones grammar repositories at pinned commit revisions and runs `tree-sitter generate --abi 15`.

```bash
go run ./cmd/ts-build build [flags]
```

| Flag | Shorthand | Default | Description | When to Use |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | `""` | Target language identifier (builds all if omitted). | Always specify during iteration to save time. |
| `--force` | `-f` | `false` | Wipe intermediate build directory before cloning. | Use if repository clone is corrupt or dirty. |
| `--prune` | | `false` | Prune build cache (`.git`, `node_modules`, tests) after building. | Use to keep disk usage low on large runs. |

#### 2. `compile` (Binary & WASM Compilation)
Compiles generated sources into native shared libraries (`.so`, `.dylib`, `.dll`) or WebAssembly (`.wasm`).

```bash
go run ./cmd/ts-build compile [flags]
```

| Flag | Shorthand | Default | Description | When to Use |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | `""` | Target language identifier. | Specify for targeted builds. |
| `--os` | `-o` | config default | Target OS (`linux`, `windows`, `macos`). | Use when cross-compiling. |
| `--arch` | `-a` | config default | Target CPU arch (`amd64`, `arm64`). | Use when cross-compiling. |
| `--wasm` | | `false` | Build WebAssembly parser (`tree-sitter-<lang>.wasm`). | Use when targeting browser or WASM runtimes. |
| `--allow-cross-validation` | | `false` | Allow static validation for non-host binaries. | Use when compiling foreign platforms. |

#### 3. `move` (Staging, Validation & Atomic Publication)
Stages artifacts into `.grammar-staging-*`, validates binaries, generates `manifest.json`, and atomically swaps the release directory into `data/tree-sitter/grammar/`.

```bash
go run ./cmd/ts-build move [flags]
```

| Flag | Shorthand | Default | Description | When to Use |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | `""` | Target language identifier. | Move a single language. |
| `--force` | `-f` | `false` | Overwrite existing release artifacts. | Recommended when re-releasing. |
| `--clean` | | `true` | Clean intermediate build directory on success. | Set `--clean=false` if inspecting intermediate builds. |
| `--compact` | | `false` | Generate & validate AI-friendly `compact-node-types.yaml`. | Use to publish compact AST schemas (~20 KB vs ~100 KB JSON) to prevent AI context rot. |
| `--check-compact` | | `false` | Audit existing `compact-node-types.yaml` without regenerating. | Use in releases/CI to enforce 100% lossless AI schema integrity. |
| `--wasm` | | `false` | Stage and checksum WebAssembly `.wasm` artifact. | Use when publishing WASM parsers. |
| `--source` / `--c-source` | | `false` | Preserve C/C++ parser sources under `src/`. | Use when distributing raw parser sources. |
| `--js` | | `false` | Preserve `grammar.js`. | Use when distributing grammar specifications. |
| `--scm=false` | | `true` | Skip publishing `.scm` query files. | Use if queries are not desired. |
| `--manifest=false` | | `true` | Skip generating custom `manifest.json`. | Use when publishing raw artifacts without updating the custom manifest metadata used by the graph builder. |

> [!NOTE]
> **Role of Downstream Artifacts (`manifest.json` and `compact-node-types.yaml`):**
> Neither of these files was built for `go-tree-sitter` itself—they were built for the downstream applications that consume these grammars (like the author's **graph builder** and AI coding agents). Generating them here during `ts-build move` is simply the easiest, most reliable point in the pipeline:
> - **`manifest.json`** (see [`data/tree-sitter/grammar/c-sharp/manifest.json`](file:///home/michael/projects/go/tree-sitter/data/tree-sitter/grammar/c-sharp/manifest.json)): A custom manifest schema providing easy programmatic inspection and control over grammar versions, target platforms/architectures, binary checksums, and measured parser ABI versions (`parser_abi: 15`) in the graph builder. Pass `--manifest=false` when raw binaries/schemas are needed without this custom manifest metadata.
> - **`compact-node-types.yaml`** (see [`data/tree-sitter/grammar/python/compact-node-types.yaml`](file:///home/michael/projects/go/tree-sitter/data/tree-sitter/grammar/python/compact-node-types.yaml)): Converts the raw, verbose ~100 KB `node-types.json` into an AI-friendly ~20 KB YAML file (~75–80% reduction). Downstream AI agents need AST schemas to construct queries and traversals, but raw JSON causes severe **context rot** and wastes token budget. Pass `--compact` to generate and losslessly validate it, or `--check-compact` to audit it in CI.

#### 4. `compact` (Compact Node Types Tool)
Generates or validates token-efficient, AI-friendly `compact-node-types.yaml` from `node-types.json` with 100% bidirectional lossless validation (reducing ~100 KB JSON to ~20 KB YAML to prevent AI context rot).

```bash
go run ./cmd/ts-build compact [flags] [path]
```

| Flag | Shorthand | Default | Description | When to Use |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | `""` | Target language identifier or direct file/dir path. | Specify to compact a single language. |
| `--check` | | `false` | Audit existing YAML against JSON without rewriting. | Use in CI pipelines to verify zero drift. |

#### 5. `clean` (Build Cache & Directory Cleaner)
Cleans or prunes intermediate build directories.

```bash
go run ./cmd/ts-build clean [flags]
```

| Flag | Shorthand | Default | Description | When to Use |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | `""` | Clean specific language (cleans all if omitted). | Specify to clean one directory. |
| `--prune` | | `false` | Prune build cache (`.git`, `node_modules`, tests, `*.o`) while preserving sources and manifests. | Use to reclaim disk space without re-cloning. |

---

### Version Synchronization (`sync-config`)

Syncs grammar version strings from [`go.mod`](file:///home/michael/projects/go/tree-sitter/go.mod) into [`tree-sitter-config.yaml`](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml) while preserving comments and structure:

```bash
# Preview changes (dry-run)
make sync-config
# Or: go run ./cmd/sync-config --dry-run

# Apply changes to tree-sitter-config.yaml
make sync-config-apply
# Or: go run ./cmd/sync-config
```

> [!IMPORTANT]
> When `sync-config` updates a `version` string, agents must also check out the corresponding upstream commit and update `revision:` with the full 40-character commit SHA.

---

## 4. Common Agent Workflows

### Workflow 1: Build & Verify a Single Grammar Rapidly

When testing changes to a specific language (e.g. `python`):

```bash
# Option A: Single-step Makefile pipeline (builds, compiles, compacts, publishes, and cleans build cache)
make build-all LANG=python

# Option B: Step-by-step CLI pipeline
# Phase 1: Build source
go run ./cmd/ts-build build -l python

# Phase 2: Compile native library
go run ./cmd/ts-build compile -l python

# Phase 3: Validate, compact, and publish
go run ./cmd/ts-build move -l python --compact --force
```

### Workflow 2: Cross-Compiling for Windows and macOS

To compile foreign binaries on a Linux host (requires MinGW, Clang, or Zig):

```bash
# Build sources once
go run ./cmd/ts-build build -l python

# Compile Windows amd64 (.dll)
go run ./cmd/ts-build compile -l python --os windows --arch amd64

# Compile macOS arm64 and amd64 (.dylib)
go run ./cmd/ts-build compile -l python --os macos --arch arm64
go run ./cmd/ts-build compile -l python --os macos --arch amd64

# Compile Linux host (.so)
go run ./cmd/ts-build compile -l python --os linux --arch amd64

# Publish all targets (dual-mode validation checks host dynamically, foreign statically)
go run ./cmd/ts-build move -l python --force
```

### Workflow 3: Building and Publishing WebAssembly (WASM)

When producing `.wasm` parser binaries (requires `emcc`, `docker`, or `podman`):

```bash
# 1. Build source
go run ./cmd/ts-build build -l python

# 2. Compile WASM binary
go run ./cmd/ts-build compile -l python --wasm

# 3. Publish with WASM flag (sets has_wasm: true in manifest)
go run ./cmd/ts-build move -l python --wasm --force
```

### Workflow 4: Generating & Auditing Compact Node Types

When generating AI-friendly YAML schemas (compressing ~100 KB JSON to ~20 KB YAML, saving 75–80% tokens to eliminate AI context rot in downstream tools):

```bash
# Generate compact-node-types.yaml for python
go run ./cmd/ts-build compact -l python

# Audit lossless capture integrity without modifying files
go run ./cmd/ts-build compact -l python --check

# Publish with release
go run ./cmd/ts-build move -l python --compact --force
```

### Workflow 5: Adding a New Grammar Language

Follow these 5 steps in order:
1. **Edit [`tree-sitter-config.yaml`](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml)**: Add entry under `languages:` with `name`, `grammar`, `constructor`, `version`, `repository`, full 40-char `revision`, and minimal `sample`.
2. **Add Go module (Optional)**: If upstream provides Go bindings, run `go get <module>@<version>`.
3. **Register in [`internal/grammars/registry.go`](file:///home/michael/projects/go/tree-sitter/internal/grammars/registry.go)**: Add binding import and switch case to `GetLanguage()`.
4. **Map in [`cmd/sync-config/main.go`](file:///home/michael/projects/go/tree-sitter/cmd/sync-config/main.go)**: Add mapping to `moduleToLanguage`.
5. **Build and Validate**: Run `go run ./cmd/ts-build build -l <name>`, compile, move, and run `go test ./...`.

### Workflow 6: Running the Test Suite

Always run tests before completing changes:

```bash
# Standard test run
go test ./...

# With race detector and full grammar bindings enabled
go test -v -race -tags grammars ./...
```

---

## 5. Helpful Documentation References

For in-depth explanations and Diátaxis documentation, refer to:
- [docs/README.md](file:///home/michael/projects/go/tree-sitter/docs/README.md) - Main documentation index
- [docs/how-to/build-and-release-grammars.md](file:///home/michael/projects/go/tree-sitter/docs/how-to/build-and-release-grammars.md) - Build & release procedures
- [docs/how-to/compile-webassembly-parsers.md](file:///home/michael/projects/go/tree-sitter/docs/how-to/compile-webassembly-parsers.md) - WebAssembly compilation
- [docs/how-to/generate-and-validate-compact-node-types.md](file:///home/michael/projects/go/tree-sitter/docs/how-to/generate-and-validate-compact-node-types.md) - Compact node types
- [docs/reference/cli.md](file:///home/michael/projects/go/tree-sitter/docs/reference/cli.md) - Complete CLI command reference
- [docs/reference/configuration-schema.md](file:///home/michael/projects/go/tree-sitter/docs/reference/configuration-schema.md) - Full config schema
- [docs/reference/compact-node-types-schema.md](file:///home/michael/projects/go/tree-sitter/docs/reference/compact-node-types-schema.md) - Compact schema specification
- [docs/reference/manifest-schema-v2.md](file:///home/michael/projects/go/tree-sitter/docs/reference/manifest-schema-v2.md) - Manifest schema v2 specification
- [docs/explanation/architecture-and-pipeline.md](file:///home/michael/projects/go/tree-sitter/docs/explanation/architecture-and-pipeline.md) - Architectural deep dive
