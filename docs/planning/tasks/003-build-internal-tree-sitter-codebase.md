# 003 - Build the Internal Tree-Sitter Codebase

**Status:** In Progress  
**Design reference:** `docs/planning/design/001-tree-sitter-design.md`  
**Scope:** Implement the CLI support packages under `internal/tree-sitter/` and the workflow package under `internal/tree-sitter/build/`.

## Goal

Build the internal Go code that powers the grammar builder workflow:

1. load and validate YAML config
2. wrap external `tree-sitter` and `git` commands
3. clone/generate grammar source
4. compile shared libraries for target platforms
5. move artifacts into the final grammar directory
6. write manifest metadata
7. clean intermediate build output
8. keep generated parser C/H files as build artifacts instead of source-controlled runtime bindings

## Package Tasks

### `internal/tree-sitter/config.go`

- [x] Define `Config`, `Language`, `ABI`, `BuildTarget`, and `Output` types.
- [x] Parse `tree-sitter-config.yaml` with YAML struct tags.
- [x] Validate required top-level fields.
- [x] Validate language names, versions, repositories, and source subdirectories.
- [x] Validate ABI mappings used by configured grammar versions.
- [x] Validate supported move modes and default move mode.
- [x] Validate configured build targets.
- [x] Return clear errors that include the failing field or language name.

### `internal/tree-sitter/cli.go`

- [x] Implement command execution with `context.Context`.
- [x] Capture command output for useful failure messages.
- [x] Detect missing external commands such as `git` and `tree-sitter`.
- [x] Provide a narrow command runner interface that tests can fake.
- [x] Keep shell-specific behavior out of business logic where possible.

### `internal/tree-sitter/build/builder.go`

- [x] Clone each configured grammar repository into `build/<language>/`.
- [x] Reuse existing checkouts unless force mode is requested.
- [x] Check out configured tags, branches, commits, or pseudo-version commits.
- [x] Support `source_subdir` for repositories that contain more than one grammar.
- [x] Run `tree-sitter generate` or the selected generation command from the correct source directory.
- [x] Produce clear guidance when clone, checkout, or generation fails.

### `internal/tree-sitter/build/compiler.go`

- [x] Validate target OS and architecture.
- [x] Use configured `OS_TARGET` when flags are omitted, falling back to the current OS/architecture if none are configured.
- [x] Compile generated grammar sources into shared libraries.
- [x] Emit platform-appropriate extensions: `.so`, `.dylib`, or `.dll`.
- [x] Use predictable names based on language, version, platform, and architecture.
- [x] Preserve metadata such as `node-types.json` and `queries/` for the move phase.
- [x] Report unsupported platform combinations without corrupting prior output.

### `internal/tree-sitter/build/mover.go`

- [x] Support universal move modes: `--json`, `--scm`, and `--both`.
- [x] Treat configured per-language build modes as deprecated or unsupported if they reappear.
- [x] Copy binaries into `data/tree-sitter/grammar/<language>/`.
- [x] Copy `node-types.json` for JSON mode.
- [x] Copy `queries/` for SCM mode.
- [x] Support force overwrite behavior where requested.
- [x] Skip cleanup when move errors occur.
- [x] Generate manifests unless disabled.

### `internal/tree-sitter/build/manifest.go`

- [x] Generate `manifest.json` for each grammar.
- [x] Record grammar name, version, tree-sitter version, and compile timestamp.
- [x] Record ABI min/max from config.
- [x] List every copied binary with platform, architecture, filename, and SHA-256 checksum.
- [x] Record artifact availability flags for node types, queries, and wasm.
- [x] Use stable JSON formatting for reviewable diffs.

### `internal/tree-sitter/build/cleaner.go`

- [x] Remove `build/<language>/` after successful move when cleanup is enabled.
- [x] Preserve the top-level `build/` directory.
- [x] Provide a safe no-op when a language build directory does not exist.

### Runtime Grammar Registry

- [x] Remove runtime registry from scope.
- [x] Remove generated parser C/H shims from source control.
- [x] Keep grammar binaries as generated artifacts built through `build` and `compile`.

## CLI Entry Point

- [x] Implement `cmd/tree-sitter/main.go`.
- [x] Add `build`, `compile`, and `move` subcommands.
- [x] Add flags from the design doc: `--config`, `--language`, `--force`, `--os`, `--arch`, `--json`, `--scm`, `--both`, `--clean`, and `--no-manifest`.
- [x] Print command usage for unknown commands or invalid flags.
- [x] Return non-zero exit codes on fatal errors.

## Acceptance Criteria

- `go build ./...` succeeds.
- `go run ./cmd/tree-sitter --help` prints usable help.
- `go run ./cmd/tree-sitter build --language json` uses the configured language and build directory.
- `go run ./cmd/tree-sitter compile --language json` produces a platform-specific shared library after build.
- `go run ./cmd/tree-sitter move --language json --both` moves artifacts, writes a manifest, and cleans the language build directory.
- Runtime grammar registry is no longer in scope; generated parser sources are not checked in.

## Dependencies

- `001-go-init-and-repo-setup.md`
- `002-installing-packages.md`
- `004-build-root-config-yaml.md`

## Notes

- Keep command execution behind interfaces so tests do not need network access or the real `tree-sitter` CLI.
- The move mode is universal; do not reintroduce per-language `build_mode` as the source of truth.
- The output is shared library artifacts, not executable binaries.
