# 004 - Build the Root Config YAML

**Status:** In Progress  
**Design reference:** `docs/planning/design/001-tree-sitter-design.md`  
**Scope:** Create and maintain `tree-sitter-config.yaml` in the repository root.

## Goal

Define the grammar build configuration in one root-level YAML file. The config should describe the build directory, OS/architecture build targets, ABI compatibility mapping, grammar repositories, versions, source subdirectories, and output behavior used by the CLI.

## Required Top-Level Shape

- [x] Add `version`.
- [x] Add `build_dir`.
- [x] Add `OS_TARGET`.
- [x] Add `targets`.
- [x] Add `abi_versions`.
- [x] Add `languages`.
- [x] Add `output.grammar_base`.
- [x] Add `output.generate_manifest`.
- [x] Add `output.default_move_mode`.
- [x] Add `output.supported_move_modes`.

## Build Targets

- [x] Add selectable Linux amd64 target.
- [x] Add selectable Windows amd64 target.
- [x] Add selectable macOS amd64 target.
- [x] Add selectable macOS arm64 target.

## ABI Mapping

- [x] Include ABI mapping for every configured `tree_sitter_version`.
- [x] Include `0.20.8` when supporting older community grammars.
- [x] Include `0.23.2` and `0.23.3` for HTML and TypeScript-era grammars.
- [x] Include `0.24.4` for Java.
- [x] Include `0.24.7` and `0.24.8` for SQL/YAML/JSON-era grammars.
- [x] Include `0.25.0` for current core grammar packages.
- [ ] Validate that ABI min/max values match actual generated parser versions before release.

## Language Entries

### Core Languages

- [x] Add Python.
- [x] Add Go.
- [x] Add TypeScript.
- [x] Add JavaScript.
- [x] Add Java.

### Frontend Stack

- [x] Add HTML.
- [x] Add CSS.
- [x] Add TSX.

### Config and Data Formats

- [x] Add JSON.
- [x] Add YAML.
- [x] Add TOML.

### Go Dependency Files

- [ ] Add `go.mod`. Blocked: package not identified yet.
- [ ] Add `go.sum` Go package support. Blocked: the pinned module has no Go package; config still supports building it through clone/generate/compile.

### Query, Database, and RPC

- [x] Add Proto.
- [x] Add SQL.
- [x] Add Cypher.

## Per-Language Fields

Each language entry should include:

- [x] `name`
- [x] `version`
- [x] `tree_sitter_version`
- [x] `repository`

Optional fields:

- [x] `source_subdir` for multi-grammar repositories such as TypeScript/TSX.
- [x] `note` for community-maintained packages, pseudo-versions, and ABI caveats.

## Move Mode Policy

- [x] Keep move mode universal under `output.default_move_mode`.
- [x] Support `--json`, `--scm`, and `--both`.
- [x] Do not use per-language `build_mode` as the source of truth.
- [x] Keep default behavior aligned with the CLI's `move` command.

## Acceptance Criteria

- `tree-sitter-config.yaml` validates through the config loader.
- Every language has a non-empty name, version, tree-sitter version, and repository.
- Every language's tree-sitter version exists in `abi_versions`.
- TypeScript and TSX can resolve different source subdirectories from the same repository.
- `output.grammar_base` points to `data/tree-sitter/grammar`.
- `output.default_move_mode` is one of `output.supported_move_modes`.

## Dependencies

- `001-go-init-and-repo-setup.md`
- `002-installing-packages.md`

## Notes

- The design doc contains an older example config and explicitly notes that some shown versions are wrong. Treat the root config as the current source of truth once it has been validated against installed packages and generated parser output.
- Prefer exact tags or pseudo-versions over floating `latest` values in committed config.
