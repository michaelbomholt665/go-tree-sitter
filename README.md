# go-tree-sitter

A Go-based Tree-Sitter grammar builder CLI that clones grammar repositories, generates parser sources, compiles shared libraries, and organizes runtime artifacts.

## What it does

The CLI implements the full workflow described in `docs/planning/design/001-tree-sitter-design.md`:

1. `build` clones configured grammar repositories and runs `tree-sitter generate`
2. `compile` builds platform-specific shared libraries with predictable filenames
3. `move` copies binaries and selected artifacts into `data/tree-sitter/grammar/<language>/`, writes `manifest.json`, and cleans `build/`

Supported grammar coverage in the default config:

- Core languages: Python, Go, TypeScript, JavaScript
- JVM languages: Java
- Frontend stack: HTML, CSS, TSX
- Config/data: JSON, YAML, TOML
- Platform/dependency files: `go.mod`, `go.sum`
- Data/query layers: SQL, Cypher
- RPC/protocols: Proto

## Requirements

- Go 1.26+
- `tree-sitter` CLI available on `PATH`
- A C/C++ toolchain for the target platform
- For cross-compilation: Zig or target-specific cross compilers

## Quick start

```bash
# show available workflow commands
make help

# generate grammar sources
make grammar-build

# compile for OS_TARGET
make compile

# move binaries + node-types + queries and generate manifests
make move
```

The default configuration lives in [`tree-sitter-config.yaml`](./tree-sitter-config.yaml). `OS_TARGET` selects one entry from `targets`; change it to decide which platform the default `build`, `compile`, and `move` workflow uses.

```yaml
OS_TARGET: "linux"

targets:
  linux:
    os: "linux"
    arch: "amd64"
  windows:
    os: "windows"
    arch: "amd64"
  macos-amd64:
    os: "macos"
    arch: "amd64"
  macos-arm64:
    os: "macos"
    arch: "arm64"
```

The output paths are also configurable:

```yaml
output:
  grammar_base: "data/tree-sitter/grammar"
  grammar_build: "build/{lang}/{arch}"
  grammar_compile: "build/{lang}/{arch}"
```

## Output layout

The move step writes organized artifacts here:

```text
data/tree-sitter/grammar/
└── <language>/
    ├── manifest.json
    ├── <language>-<version>-<platform>-<arch>.<so|dylib|dll>
    ├── node-types.json
    └── queries/
```

`manifest.json` includes:

- grammar name and version
- tree-sitter parser version
- ABI min/max compatibility
- compiled timestamp
- per-binary SHA-256 checksums
- artifact availability flags

## Commands

### Makefile Workflow

```bash
make build          # build Go packages and CLI
make grammar-build  # clone grammar repos and run tree-sitter generate
make compile        # compile generated grammars for OS_TARGET
make move           # move artifacts, overwriting existing output files
make move-safe      # move artifacts without overwriting existing output files
```

`make grammar-build` and `make compile` require the external `tree-sitter` CLI. Install it with:

```bash
make install-tree-sitter-cli
```

### CLI Build

```bash
go run ./cmd/tree-sitter build [--config path] [--language name] [--force]
```

### CLI Compile

```bash
go run ./cmd/tree-sitter compile [--config path] [--language name] [--os linux|windows|macos] [--arch amd64|arm64]
```

When `--os` or `--arch` is provided, the command compiles one target and uses the flag values as an override. When both are omitted, `OS_TARGET` selects the configured target. If `OS_TARGET` is omitted, the current platform is used.

### CLI Move

```bash
go run ./cmd/tree-sitter move [--config path] [--language name] [--json|--scm|--both] [--clean] [--no-manifest] [--force]
```

## Project structure

```text
cmd/tree-sitter/                 CLI entry point
internal/tree-sitter/           config loading, command dispatch, command runner
internal/tree-sitter/build/     build / compile / move / clean / manifest logic
internal/tests/                 package-level unit tests
tree-sitter-config.yaml         default 15-grammar configuration
data/tree-sitter/grammar/       final artifact root
build/                          intermediate checkout and compilation root
```

## Development

```bash
make help
make build
make grammar-build
make compile
make move
make test
make fmt
make vet
```

## Notes

- `github.com/tree-sitter/go-tree-sitter` is the runtime parser library, **not** the CLI. Installing the Go module does not install the `tree-sitter` binary.
- The builder commands still depend on the external `tree-sitter` CLI being available on `PATH`.
- The YAML config now carries the universal move modes and default explicitly under `output.default_move_mode` / `output.supported_move_modes`.
- Generated parser C files and headers are build artifacts. They are produced under `build/` by `tree-sitter generate` and are not checked in.
- The compiler emits real shared library extensions (`.so`, `.dylib`, `.dll`) instead of executable names because `tree-sitter build` produces shared libraries.
- TSX and TypeScript are both sourced from the `tree-sitter-typescript` repository via per-language `source_subdir` configuration.
- The repository keeps generated grammar outputs out of source control via `.gitignore`.

## License

MIT. See [LICENSE](./LICENSE).
