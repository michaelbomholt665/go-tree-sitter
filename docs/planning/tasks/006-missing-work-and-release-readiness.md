# 006 - Missing Work and Release Readiness

**Status:** In Progress  
**Design reference:** `docs/planning/design/001-tree-sitter-design.md`  
**Scope:** Track validation, documentation, integration gaps, and release criteria not fully covered by tasks 001 through 005.

## Goal

Close the gaps between the design, implementation, test coverage, and a usable developer workflow. This task is intentionally broad and should be revisited after the main implementation tasks are complete.

## Design Consistency

- [ ] Reconcile the design doc's older config example with the root `tree-sitter-config.yaml`.
- [ ] Ensure the design consistently describes shared libraries rather than executables where applicable.
- [x] Document that `build_mode` is no longer per-language and move mode is universal.
- [x] Confirm the listed grammar count matches the actual config and README.
- [x] Confirm every documented command flag exists in the CLI.

## Developer Workflow

- [ ] Verify `make build`.
- [ ] Verify `make test`.
- [ ] Verify `make fmt`.
- [ ] Verify `make vet`.
- [ ] Decide whether `make lint` is required for release or optional because `golangci-lint` may not be installed.
- [x] Add or update troubleshooting notes for missing `tree-sitter`, missing compilers, and cross-compilation failures.

## Integration Workflow

- [ ] Run one end-to-end grammar flow against a small grammar such as JSON.
- [ ] Verify `build` clones and generates source.
- [ ] Verify `compile` emits a loadable shared library for the current platform.
- [ ] Verify `move --both` copies binaries, node types, queries when present, and manifest metadata.
- [ ] Verify cleanup leaves `build/` present but removes completed language directories.
- [ ] Verify repeated runs handle existing output according to force mode.

## Artifact and Manifest Validation

- [x] Validate manifest timestamps are ISO 8601.
- [x] Validate binary checksums match copied files.
- [x] Validate platform/arch fields match filenames.
- [x] Validate missing optional artifacts produce warnings instead of fatal errors.
- [x] Validate manifest generation is fatal when integrity metadata cannot be produced.

## Runtime Registry Validation

- [x] Removed from scope; the project now builds copyable grammar binaries and does not check in generated parser C/H files.

## Documentation

- [x] Update README quick start after implementation behavior is final.
- [x] Add example config snippets that match the current root config.
- [x] Add command examples for single-language build, compile, and move.
- [x] Add output layout examples using real shared library extensions.
- [x] Add notes about generated artifacts being ignored by git.
- [x] Add notes about network access required by build but not unit tests.

## Release Criteria

- [ ] `go mod tidy` produces no unexpected changes. Deferred: grammar packages are intentionally pinned in `go.mod` even when not imported.
- [x] `go fmt ./...` produces no unexpected changes.
- [x] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `go build ./...` passes.
- [ ] At least one end-to-end grammar build has been manually verified.
- [ ] README and planning docs no longer conflict on command behavior or grammar versions.
- [ ] Known external prerequisites are documented.

## Dependencies

- `001-go-init-and-repo-setup.md`
- `002-installing-packages.md`
- `003-build-internal-tree-sitter-codebase.md`
- `004-build-root-config-yaml.md`
- `005-build-internal-tests-mirror.md`

## Notes

- This task should be the place for cross-cutting cleanup instead of mixing release polish into implementation tasks.
- Keep future generated parser artifacts out of source control.
