# How to Synchronize Versions with go.mod

This guide explains how to use the version synchronization tool (`sync-config`) to keep grammar versions declared in [tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml) in sync with the Go module versions listed in [go.mod](file:///home/michael/projects/go/tree-sitter/go.mod).

---

## Why Synchronize Versions?

The `go-tree-sitter` repository maintains dual representations of grammar versions:
1. **[go.mod](file:///home/michael/projects/go/tree-sitter/go.mod)**: Controls the Go dependencies used by Go bindings in [`internal/grammars/registry.go`](file:///home/michael/projects/go/tree-sitter/internal/grammars/registry.go).
2. **[tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml)**: Governs which git repositories and tags are cloned by `ts-build`.

The `sync-config` tool parses [go.mod](file:///home/michael/projects/go/tree-sitter/go.mod) and updates [tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml) in-place while preserving all YAML formatting, structure, and comments.

---

## Step 1: Preview Version Differences (Dry-Run)

Before modifying the configuration, preview the changes:

```bash
# Using Makefile
make sync-config

# Or using the tool directly
go run ./cmd/sync-config --dry-run
```

If differences are detected, you will see output detailing each changed version:

```text
  python               v0.24.0  →  v0.25.0
  typescript           v0.23.2  →  v0.23.3-0.20250130221139-75b3874edb2d
  tsx                  v0.23.2  →  v0.23.3-0.20250130221139-75b3874edb2d

(dry-run: no files written)
```

If the configuration is already up to date, it reports:

```text
tree-sitter-config.yaml is already up to date.
```

---

## Step 2: Apply Version Updates

To write the updated versions directly into [tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml):

```bash
# Using Makefile
make sync-config-apply

# Or using the tool directly
go run ./cmd/sync-config
```

Output:

```text
  python               v0.24.0  →  v0.25.0
  typescript           v0.23.2  →  v0.23.3-0.20250130221139-75b3874edb2d
  tsx                  v0.23.2  →  v0.23.3-0.20250130221139-75b3874edb2d

Updated tree-sitter-config.yaml
```

---

## Step 3: Update Commit Revisions

`sync-config` updates the semantic `version:` field. In configuration version `2.0`, an explicit 40-character git commit hash (`revision:`) is also required for reproducible builds.

If a module version changed:
1. Verify the corresponding upstream commit hash for that version.
2. Update the `revision:` field in [tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml).
3. Run `make test` or `go run ./cmd/ts-build build -l <name>` to ensure the commit checks out cleanly.

---

## Supported Custom Flags

The `sync-config` utility accepts the following command-line flags:

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--config <path>` | `tree-sitter-config.yaml` | Path to the YAML configuration file |
| `--gomod <path>` | `go.mod` | Path to the `go.mod` file to read |
| `--dry-run` | `false` | Print changes without modifying files |

Example specifying custom file locations:
```bash
go run ./cmd/sync-config --config path/to/config.yaml --gomod path/to/go.mod --dry-run
```
