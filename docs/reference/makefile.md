# Makefile Target Reference

This document provides a reference for all targets declared in [Makefile](file:///home/michael/projects/go/tree-sitter/Makefile).

---

## Configuration Variables

The following variables can be overridden on the command line:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `TREE_SITTER` | `tree-sitter` | Path to the Tree-sitter CLI executable. |
| `TREE_SITTER_VERSION` | `0.26.8` | Pinned Tree-sitter CLI version installed by `install-tree-sitter-cli`. |
| `NPM` | `npm` | Node package manager binary used to install `tree-sitter-cli`. |
| `CONFIG` | `tree-sitter-config.yaml` | Path to configuration YAML file passed to CLI invocations. |
| `LANG` / `LANGUAGE` | `""` | Optional grammar name filter (e.g., `LANG=python`). If omitted, targets all configured grammars. |

Example overrides:
```bash
# Build, compile, compact, publish, and clean a single language
make build-all LANG=python

# Run with alternative configuration
make release CONFIG=custom-config.yaml
```

---

## Release Targets

| Target | Dependencies | Description |
| :--- | :--- | :--- |
| `build-all` | `check-tree-sitter`, `grammar-build`, `compile`, `move` | Complete host pipeline (`build` + `compile` + `move` with `--compact`), losslessly generating compact node types and automatically deleting intermediate build cache. Supports `LANG=<name>`. |
| `release` | `check-tree-sitter`, `grammar-build`, `compile`, `move` | Complete release pipeline for the active target configured in `OS_TARGET`. |
| `release-linux` | `check-tree-sitter`, `grammar-build`, `compile-linux`, `move` | Full pipeline targeting Linux (`amd64`). |
| `release-windows`| `check-tree-sitter`, `grammar-build`, `compile-windows`, `move` | Full pipeline targeting Windows (`amd64`). |
| `release-mac` | `check-tree-sitter`, `grammar-build`, `compile-mac`, `move` | Full pipeline targeting macOS (`amd64` and `arm64`). |
| `release-all` | `check-tree-sitter`, `grammar-build`, `compile-all`, `move` | Full pipeline building and validating all supported platforms. |

---

## Grammar Build & Compilation Targets

| Target | Description |
| :--- | :--- |
| `grammar-build` | Clones grammar repositories and runs `tree-sitter generate --abi 15`. Supports `LANG=<name>`. |
| `compile` | Compiles shared libraries for the default `OS_TARGET`. Supports `LANG=<name>`. |
| `compile-wasm` | Compiles WebAssembly parser (`tree-sitter-<lang>.wasm`). Supports `LANG=<name>`. |
| `compile-linux` | Compiles shared libraries for Linux `amd64`. Supports `LANG=<name>`. |
| `compile-windows` | Compiles shared libraries for Windows `amd64` using MinGW or Zig. Supports `LANG=<name>`. |
| `compile-mac` | Compiles shared libraries for macOS `amd64` and `arm64` using Clang or Zig. Supports `LANG=<name>`. |
| `compile-all` | Compiles for all platform targets in sequence. |
| `move` | Validates compiled native binaries, losslessly generates `compact-node-types.yaml`, and publishes to `data/tree-sitter/grammar/`, automatically cleaning the intermediate build directory. Supports `LANG=<name>`. |

---

## Toolchain & Dependency Targets

| Target | Description |
| :--- | :--- |
| `check-tree-sitter` | Checks if `tree-sitter` is available in `$PATH` and prints installation instructions if missing. |
| `install-tree-sitter-cli` | Installs the pinned Tree-sitter CLI globally via npm (`npm install -g tree-sitter-cli@0.26.8`). |
| `sync-config` | Dry-run preview of grammar version updates from `go.mod` to `tree-sitter-config.yaml`. |
| `sync-config-apply` | Writes grammar version updates from `go.mod` directly to `tree-sitter-config.yaml`. |

---

## Quality, Testing & Hygiene Targets

| Target | Description |
| :--- | :--- |
| `build` | Compiles all Go packages and CLI binaries (`go build ./...`). |
| `test` | Runs all Go unit and integration tests with data race detection, generating `coverage.out` and `coverage.html`. |
| `fmt` | Formats all Go code using `go fmt ./...`. |
| `vet` | Runs standard Go static analysis using `go vet ./...`. |
| `lint` | Runs `golangci-lint run ./...`. |
| `licenses` | Checks and reports open-source licenses for all third-party Go dependencies via `go-licenses`. |
| `tool-help` | Runs `go run ./cmd/ts-build --help`. |
| `clean` | Removes coverage reports, temporary test files, and runs `go clean`. |
| `help` | (Default) Prints summary of available Makefile commands. |
