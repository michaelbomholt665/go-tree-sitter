# Architecture and Pipeline Design

This document explains the architectural principles, component responsibilities, and execution lifecycle of the **go-tree-sitter** grammar build system.

---

## 1. High-Level Architectural Goals

Tree-sitter grammars are polyglot artifacts: they originate as JavaScript DSL specifications in upstream git repositories, get converted to C source files by Node-based CLI tools, compile into native machine code shared libraries via platform C/C++ toolchains, and finally get consumed by Go runtime applications.

Managing this lifecycle reliably across dozens of grammars and multiple operating systems requires solving several challenges:
- **Build isolation**: Individual grammar failures must not corrupt existing release catalogs.
- **Reproducibility**: Builds must be bit-for-bit reproducible and traceable to specific git commits.
- **Cross-platform compilation**: Linux hosts must be able to compile and validate Windows and macOS libraries.
- **Atomic publication**: Consumers reading `data/tree-sitter/grammar/` must never observe a half-built or corrupted state.

To achieve these goals, `go-tree-sitter` divides the build lifecycle into three distinct, decoupled phases.

---

## 2. The 3-Phase Pipeline

```
┌─────────────────┐       ┌─────────────────┐       ┌──────────────────────┐
│  Phase 1: BUILD │ ───►  │Phase 2: COMPILE │ ───►  │    Phase 3: MOVE     │
│                 │       │                 │       │  (Stage, Validate &  │
│                 │       │                 │       │   Publish Atomic)    │
└─────────────────┘       └─────────────────┘       └──────────────────────┘
         │                         │                           │
         ▼                         ▼                           ▼
- Pinned git clone        - Target compiler          - Temp staging dir
- Extract commit epoch      (gcc, clang, zig)        - Dual-mode validation
- npm locked install      - CFLAGS / CXXFLAGS        - Manifest generation
- tree-sitter generate    - Shared lib output        - Atomic directory swap
- .source-provenance.json - .provenance.json         - Cleaner reset
```

### Phase 1: `build` (Source Generation & Pruning)
- **Goal**: Transform upstream grammar repositories into generated C code and schema files.
- **Execution**:
  - Clones the upstream repository at the pinned 40-character commit hash (`revision`).
  - Extracts the exact git commit timestamp to set `SOURCE_DATE_EPOCH`.
  - Installs npm dependencies using `npm ci --ignore-scripts` only if a `package-lock.json` is present; otherwise skips npm to use checked-in sources.
  - Synthesizes `tree-sitter.json` if missing.
  - Invokes `tree-sitter generate --abi 15` (with fallback through `abi_range` if necessary).
  - Captures generation metadata and writes `.source-provenance.json`.
  - Optionally prunes build caches (`.git`, `node_modules`, test suites) if `--prune` or `cfg.Prune` is enabled.

### Phase 2: `compile` (Binary & WASM Compilation)
- **Goal**: Compile generated C/C++ parser code into native dynamic shared libraries (`.so`, `.dylib`, `.dll`) and WebAssembly artifacts.
- **Execution**:
  - Resolves target architecture from configuration or command-line flags (`--os`, `--arch`).
  - Selects the appropriate toolchain (native compiler, dedicated cross-compiler, or synthetic Zig wrapper).
  - When `--wasm` is passed, detects Emscripten (`emcc`) or sets up an ephemeral container engine wrapper (`docker`/`podman`) using `emscripten/emsdk` to emit `tree-sitter-<lang>.wasm`.
  - Injects `SOURCE_DATE_EPOCH` and uniform optimization flags (`-O2`, `-fPIC`).
  - Emits the binary into `build/{lang}/{arch}/` alongside a `.provenance.json` recording compiler identity and compilation parameters.

### Phase 3: `move` (Validation, Compaction, Manifest Generation & Atomic Publication)
- **Goal**: Guarantee correctness before replacing published catalog files.
- **Execution**:
  - Creates an isolated staging directory (`.grammar-staging-*`) seeded from the existing release.
  - Copies compiled binaries, `node-types.json`, and query files (`.scm`) into staging.
  - When `--compact` is requested, generates `compact-node-types.yaml` and executes strict bidirectional lossless AST validation (`ValidateCapture`).
  - Stages WebAssembly artifacts (`--wasm`), C/C++ sources (`--source`), or `grammar.js` (`--js`) if requested.
  - Executes **Dual-Mode Validation** across all staged native libraries.
  - Computes final-byte SHA-256 hashes and generates `manifest.json`.
  - Atomically swaps the staging directory with the production catalog directory (`data/tree-sitter/grammar`).
  - Cleans up intermediate build directories (unless `--clean=false`).

---

## 3. Component Architecture & Dependency Injection

The implementation resides in [`internal/tree-sitter/`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter) and adheres to clean architecture principles:

```
cmd/ts-build/main.go
         │
         ▼
internal/tree-sitter/
├── cli.go              <-- App (Cobra command tree, flag wiring)
├── config.go           <-- Config parser, validator, and path templater
├── reporter.go         <-- TextReporter (progress) & SilentReporter
└── build/
    ├── builder.go      <-- Builder interface & git/generator execution
    ├── compiler.go     <-- Compiler interface, cross-toolchain discovery & WASM
    ├── mover.go        <-- Mover interface, catalog staging & atomic publish
    ├── compact.go      <-- Compactor, compact YAML generation & lossless validation
    ├── validator.go    <-- NativeValidator (dlopen + ELF/Mach-O/PE inspection)
    ├── manifest.go     <-- Manifest generator & schema v2 validator
    ├── cleaner.go      <-- Cleaner (cache pruning & intermediate directory resets)
    ├── provenance.go   <-- SourceProvenance & BinaryProvenance data models
    ├── paths.go        <-- Path resolver functions & template evaluation
    └── native_loader_*.go <-- Platform-specific dynamic loader bindings
```

### Decoupling and Testability
Every major subsystem implements an interface (`CommandRunner`, `PathLookup`, `Builder`, `Compiler`, `Mover`, `ArtifactValidator`, `Reporter`). This enables comprehensive unit and integration testing without invoking external compilers or modifying the host filesystem during tests.

---

## 4. Atomic Staging and Zero-Downtime Rollback

In-place file copying is vulnerable to partial failures: if grammar 18 of 38 fails during compilation or validation, an in-place build leaves the catalog in an inconsistent, broken state.

`go-tree-sitter` guarantees catalog integrity using filesystem-level directory swaps:

1. **Seed Staging**: A temporary hidden directory `.grammar-staging-<random>` is created in the destination filesystem and pre-seeded with current catalog contents.
2. **Stage Updates**: Updated grammars are copied into staging and verified.
3. **Full Catalog Validation**: The entire staged directory is validated in aggregate by [`validateCatalog`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/mover.go#L339). If any grammar fails, the staging directory is deleted, and the production catalog remains untouched.
4. **Atomic Rename**:
   - The current production catalog `data/tree-sitter/grammar` is renamed to `data/tree-sitter/grammar.previous`.
   - The staged directory `.grammar-staging-*` is renamed to `data/tree-sitter/grammar`.
   - On success, `data/tree-sitter/grammar.previous` is removed.
   - If the rename fails, `grammar.previous` is restored immediately.

This design ensures that callers reading `data/tree-sitter/grammar/` always observe a valid, completely validated release.
