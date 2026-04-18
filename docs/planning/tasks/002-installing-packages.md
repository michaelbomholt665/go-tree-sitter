# 002 - Installing Packages

**Status:** In Progress  
**Design reference:** `docs/planning/design/001-tree-sitter-design.md`  
**Scope:** Pin YAML configuration and grammar package versions used to align the root build config.

## Goal

Add the Go package dependencies used by the CLI implementation and as pinned references for configured grammar versions. The external `tree-sitter` CLI is still a separate system dependency.

## Package Groups

### Core Runtime and Config

- [x] Add `github.com/tree-sitter/go-tree-sitter`.
- [x] Add `gopkg.in/yaml.v3`.

### Core and Web Grammars

- [x] Add `github.com/tree-sitter/tree-sitter-python`.
- [x] Add `github.com/tree-sitter/tree-sitter-go`.
- [x] Add `github.com/tree-sitter/tree-sitter-javascript`.
- [x] Add `github.com/tree-sitter/tree-sitter-typescript`.
- [x] Add `github.com/tree-sitter/tree-sitter-html`.
- [x] Add `github.com/tree-sitter/tree-sitter-css`.
- [x] Add `github.com/tree-sitter/tree-sitter-java`.

### Config, Data, and Dependency Grammars

- [x] Add `github.com/tree-sitter/tree-sitter-json`.
- [x] Add `github.com/tree-sitter-grammars/tree-sitter-yaml`.
- [x] Add TOML grammar support.
- [ ] Add `go.mod` grammar support. Blocked: package not identified yet.
- [ ] Add `go.sum` Go package support. Blocked: `github.com/tree-sitter-grammars/tree-sitter-go-sum v1.0.0` does not expose a Go package, so `go mod tidy` cannot retain it as an import.

### Query and RPC Grammars

- [x] Add Proto grammar support.
- [x] Add SQL grammar support.
- [x] Add Cypher grammar support.

## Version Alignment Checklist

- [x] Align package versions with `tree-sitter-config.yaml`.
- [x] Prefer published packages when they are available.
- [x] Remove local compatibility glue/generated shims from source control.
- [x] Ensure `go.sum` is updated after module changes.
- [x] Verify ABI expectations from known parser/package metadata and add matching entries to config.
- [x] Confirm TypeScript and TSX use the correct source subdirectories.

## External Tools

- [x] Document that `github.com/tree-sitter/go-tree-sitter` is not the `tree-sitter` CLI.
- [x] Document that the `tree-sitter` binary must be installed separately and available on `PATH`.
- [x] Document compiler requirements for local and cross-platform shared library builds.
- [x] Decide whether Zig or platform-specific cross compilers are the supported cross-compilation path.

## Acceptance Criteria

- `go.mod` includes all required runtime/config dependencies.
- `go mod tidy` succeeds. It retains grammar modules that expose usable Go packages. `go.sum` grammar building remains config-driven because its module does not expose a Go package.
- `go test ./...` can resolve all imports without missing module errors.
- The README or another developer-facing doc clearly separates Go modules from the external `tree-sitter` CLI.
- Every grammar listed in `tree-sitter-config.yaml` has an installed package except `go.mod`, which is intentionally left unresolved for now.

## Dependencies

- `001-go-init-and-repo-setup.md`

## Notes

- Generated parser sources are not vendored. Package requirements are retained as version pins/reference metadata for the configured grammar set.
- Avoid `latest` in committed package requirements where a stable version or pseudo-version is already known.
