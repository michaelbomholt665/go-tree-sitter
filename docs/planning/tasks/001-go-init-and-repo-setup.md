# 001 - Go Init and Repository Setup

**Status:** In Progress  
**Design reference:** `docs/planning/design/001-tree-sitter-design.md`  
**Scope:** Establish the Go module, baseline repository structure, ignored generated artifacts, and developer entry points.

## Goal

Create the repository foundation for the Tree-Sitter Grammar Builder CLI. This task should leave the repo buildable as a Go module with predictable directories for source code, generated grammar outputs, intermediate build artifacts, tests, and planning docs.

## Context

The design expects a Go 1.26+ project with:

- CLI entry point under `cmd/tree-sitter/`
- implementation packages under `internal/tree-sitter/`
- mirrored tests under `internal/tests/`
- generated grammar artifacts under `data/tree-sitter/grammar/`
- temporary clone/build output under `build/`
- root-level config at `tree-sitter-config.yaml`
- developer commands in `Makefile`

Generated grammar outputs and intermediate build artifacts should not be committed, but their directories should exist with `.gitkeep` files so the workflow has stable default paths.

## Task List

- [x] Initialize the Go module with the intended module path.
- [x] Set the Go version to `1.26.0` or newer in `go.mod`.
- [x] Create the CLI package directory at `cmd/tree-sitter/`.
- [x] Create the implementation package directory at `internal/tree-sitter/`.
- [x] Create the build subpackage directory at `internal/tree-sitter/build/`.
- [x] Remove the runtime grammar registry from scope; generated parser C/H files stay build artifacts.
- [x] Create the mirrored test root at `internal/tests/`.
- [x] Create `build/.gitkeep`.
- [x] Create `data/tree-sitter/grammar/.gitkeep`.
- [x] Add `.gitignore` rules for generated build output, grammar output, coverage files, and local editor/system files.
- [x] Add `README.md` with the project purpose, requirements, commands, and output layout.
- [x] Add `LICENSE`.
- [x] Add `NOTICE.md` placeholder or generated third-party notice content.
- [x] Add `Makefile` targets for build, test, fmt, vet, lint, clean, and CLI help.
- [x] Confirm `go build ./...` can discover the module without path errors.

## Acceptance Criteria

- `go.mod` exists and declares the correct module path and Go version.
- `cmd/tree-sitter/`, `internal/tree-sitter/`, `internal/tree-sitter/build/`, and `internal/tests/` exist.
- `build/` and `data/tree-sitter/grammar/` are present but ready to stay mostly empty in git.
- Generated outputs are ignored while `.gitkeep` files keep required directories.
- `make help` lists the available developer commands.
- `go build ./...` reaches dependency or implementation errors only; it must not fail because of missing repository layout.

## Dependencies

None. This is the first setup task.

## Notes

- Keep generated parser artifacts out of source control; the repo builds copyable shared library artifacts.
- The repository name and module path should stay aligned with import examples in `README.md`.
- If the module path changes later, update imports, README examples, and tests in the same change.
