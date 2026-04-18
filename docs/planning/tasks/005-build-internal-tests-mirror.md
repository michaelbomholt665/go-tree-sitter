# 005 - Build Tests in `internal/tests/{mirror of internal}/`

**Status:** In Progress  
**Design reference:** `docs/planning/design/001-tree-sitter-design.md`  
**Scope:** Add focused tests under `internal/tests/` that mirror the implementation packages under `internal/tree-sitter/`.

## Goal

Create test coverage for config parsing, command wrapping, build orchestration, compilation, artifact movement, cleanup, and manifest generation. Tests should mirror the implementation tree while using fakes for network and external CLI execution.

## Directory Layout

- [x] Add `internal/tests/tree-sitter/cli_test.go`.
- [x] Add `internal/tests/tree-sitter/config_test.go`.
- [x] Remove `internal/tests/tree-sitter/grammars_test.go`; runtime registry is no longer in scope.
- [x] Add `internal/tests/build/builder_test.go`.
- [x] Add `internal/tests/build/compiler_test.go`.
- [x] Add `internal/tests/build/mover_test.go`.
- [x] Add `internal/tests/build/cleaner_test.go`.
- [x] Add `internal/tests/build/manifest_test.go`.
- [x] Add `internal/tests/testutil/` for shared fakes and filesystem helpers.

## Config Tests

- [x] Load a valid config.
- [ ] Reject missing config files.
- [ ] Reject missing required fields.
- [x] Reject invalid language definitions.
- [ ] Reject languages whose tree-sitter version is missing from `abi_versions`.
- [x] Validate universal move mode configuration.
- [x] Validate configured `OS_TARGET` handling.
- [ ] Validate `source_subdir` handling for TypeScript and TSX.

## CLI Wrapper Tests

- [x] Verify commands are invoked with expected executable names and arguments.
- [ ] Verify command environment handling.
- [ ] Verify context cancellation is propagated.
- [ ] Verify failures include useful output.
- [x] Verify missing command behavior can be tested without requiring real external tools.

## Builder Tests

- [x] Clone into `build/<language>/`.
- [x] Skip existing checkouts unless force mode is enabled.
- [x] Checkout configured version/tag/commit.
- [x] Run generation from the correct source directory.
- [ ] Surface clone, checkout, and generation failures.

## Compiler Tests

- [x] Validate target OS and architecture combinations.
- [x] Use configured `OS_TARGET` when no target flags are supplied.
- [x] Generate expected shared library filenames.
- [x] Use `.so`, `.dylib`, or `.dll` as appropriate.
- [x] Preserve artifacts needed by move tests.
- [x] Fail cleanly when source output is missing.

## Mover Tests

- [x] Move binaries into `data/tree-sitter/grammar/<language>/`.
- [x] Copy `node-types.json` in JSON mode.
- [x] Copy `queries/` in SCM mode.
- [x] Copy both artifact sets in BOTH mode.
- [x] Generate manifest by default.
- [ ] Skip manifest with `--no-manifest`.
- [ ] Preserve existing files unless force mode is enabled.
- [ ] Skip cleanup when move errors occur.

## Cleaner Tests

- [x] Remove only the selected language build directory.
- [x] Preserve the top-level build directory.
- [x] Treat missing language build directories as safe no-ops.

## Manifest Tests

- [x] Generate stable JSON with expected fields.
- [x] Calculate SHA-256 checksums.
- [x] Include all copied binaries.
- [x] Include ABI min/max from config.
- [x] Record artifact availability flags.
- [ ] Return errors for missing ABI mappings and unreadable binaries.

## Runtime Grammar Registry Tests

- [x] Removed from scope; the repo no longer checks in generated C/H parser shims or exposes a Go runtime registry.

## Acceptance Criteria

- `go test ./...` passes.
- `go test -race ./...` passes or has documented exclusions.
- Tests do not require network access.
- Tests do not require the real `tree-sitter` CLI except for explicitly marked integration tests.
- Temporary test files are written under test temp directories.
- The mirrored test layout makes it obvious which implementation area each test file covers.

## Dependencies

- `001-go-init-and-repo-setup.md`
- `002-installing-packages.md`
- `003-build-internal-tree-sitter-codebase.md`
- `004-build-root-config-yaml.md`

## Notes

- Keep unit tests deterministic by faking command execution and filesystem fixtures.
- Add integration tests separately if they need real grammar repositories or the external `tree-sitter` CLI.
