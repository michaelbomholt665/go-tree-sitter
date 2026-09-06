# Configuration Schema Reference (`tree-sitter-config.yaml`)

This document defines the schema, data types, validation rules, and default behaviors for [tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml) parsed by [`LoadConfig`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/config.go#L68).

---

## Root Structure

```yaml
version: "2.0"
build_dir: "build"
tree_sitter_cli_version: "0.26.8"
abi_range:
  min: 13
  max: 15
generate_abi: 15
build_flags:
  - "-O2"
  - "-fPIC"
OS_TARGET: "linux"
prune: false

targets:
  linux: { ... }
  windows: { ... }
  macos-amd64: { ... }
  macos-arm64: { ... }

languages:
  - { ... }

output:
  grammar_base: "data/tree-sitter/grammar"
  grammar_build: "build/{lang}/{arch}"
  grammar_compile: "build/{lang}/{arch}"
  generate_manifest: true
  default_move_mode: "--both"
  supported_move_modes:
    - "--json"
    - "--scm"
    - "--both"
```

---

## 1. Toolchain & Global Settings

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `version` | string | **Yes** | Schema version. Must be set to `"2.0"`. |
| `build_dir` | string | **Yes** | Path for cloned repositories and intermediate build artifacts. Defaults to `"build"`. |
| `tree_sitter_cli_version` | string | **Yes** (in v2.0) | Exact `tree-sitter` CLI version required on `$PATH` (e.g. `"0.26.8"`). |
| `abi_range` | object | No | Supported ABI range for generated and runtime parsers (`min: 13, max: 15`). Defaults to `13` through `15`. |
| `generate_abi` | integer | No | Target ABI version passed to `tree-sitter generate --abi <n>` (must fall within `abi_range`, e.g. `15`). Defaults to `15`. |
| `build_flags` | list[string] | No | C and C++ compiler flags passed to `CFLAGS` and `CXXFLAGS` during compilation (e.g. `["-O2", "-fPIC"]`). |
| `OS_TARGET` | string | **Yes** | Active target key. Must match one of the keys defined in `targets`. Overridable at runtime with `--os` and `--arch`. |
| `prune` | bool | No | Optional global build pruning flag. When `true`, automatically prunes build cache (`.git`, `node_modules`, test suites, intermediate object files) after building while strictly preserving grammar sources and manifests. |

---

## 2. Target Toolchains (`targets`)

Each entry in `targets` maps a target name (such as `linux`, `windows`, `macos-amd64`, or `macos-arm64`) to its compiler specification.

```yaml
targets:
  linux:
    os: "linux"
    arch: "amd64"
    triple: "x86_64-linux-gnu"
    compiler: "gcc"
    cxx: "g++"
    compiler_version: "14.2.0"
```

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `os` | string | **Yes** | Target operating system: `"linux"`, `"windows"`, or `"macos"`. |
| `arch` | string | **Yes** | Target architecture: `"amd64"` or `"arm64"`. |
| `triple` | string | **Yes** (in v2.0) | Target compilation triple (e.g. `"x86_64-linux-gnu"`). |
| `compiler` | string | **Yes** (in v2.0) | C compiler binary (e.g. `"gcc"`, `"x86_64-w64-mingw32-gcc"`, `"o64-clang"`). |
| `cxx` | string | No | C++ compiler binary (e.g. `"g++"`, `"x86_64-w64-mingw32-g++"`, `"o64-clang++"`). |
| `compiler_version` | string | **Yes** (in v2.0) | Expected compiler version string matched against `--version` output. |

---

## 3. Language Definitions (`languages`)

The `languages` list contains every grammar built and published by the repository.

```yaml
languages:
  - name: "python"
    grammar: "python"
    constructor: "tree_sitter_python"
    version: "v0.25.0"
    repository: "https://github.com/tree-sitter/tree-sitter-python.git"
    revision: "293fdc02038ee2bf0e2e206711b69c90ac0d413f"
    source_subdir: ""
    sample: "x = 1\n"
    note: "Core language"
```

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `name` | string | **Yes** | Language identifier matching `^[A-Za-z0-9._-]+$`. Used in file paths and CLI flags. |
| `grammar` | string | No | C symbol suffix matching `^[A-Za-z0-9_]+$`. Defaults to `name` with `.` and `-` stripped or replaced. |
| `constructor` | string | No | Exported C function name. **Must strictly equal** `"tree_sitter_" + grammar`. |
| `version` | string | **Yes** | Upstream version label (kept in sync with `go.mod`). |
| `generate_abi` | integer | No | Optional per-language target ABI (e.g. `13`, `14`, or `15`) overriding the global `generate_abi`. |
| `repository` | string | **Yes** | Absolute HTTPS git repository URL. |
| `revision` | string | **Yes** (in v2.0) | Pinned 40-character hexadecimal git commit SHA. |
| `source_subdir` | string | No | Subdirectory within repo if multi-grammar repository (e.g. `"tsx"` or `"typescript"`). |
| `sample` | string | **Yes** (in v2.0) | Minimal valid code snippet used to smoke-test parser output during validation. |
| `note` | string | No | Optional human-readable documentation note. |

---

## 4. Output Configuration (`output`)

Defines destinations for compiled binaries, AST metadata, and release catalogs.

```yaml
output:
  grammar_base: "data/tree-sitter/grammar"
  grammar_build: "build/{lang}/{arch}"
  grammar_compile: "build/{lang}/{arch}"
  generate_manifest: true
  default_move_mode: "--both"
  supported_move_modes:
    - "--json"
    - "--scm"
    - "--both"
```

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `grammar_base` | string | `"data/tree-sitter/grammar"` | Base publication directory for final release catalogs. |
| `grammar_build` | string | `"build/{lang}/{arch}"` | Template path for intermediate grammar source files. |
| `grammar_compile` | string | `"build/{lang}/{arch}"` | Template path for compiled binary shared libraries. |
| `generate_manifest` | bool | `true` | Must be `true`. Manifest-only generation without native validation is prohibited. |
| `default_move_mode` | string | `"--both"` | Default move mode used by `ts-build move` if no mode flag is passed. |
| `supported_move_modes` | list[string] | `["--json", "--scm", "--both"]` | List of allowed move modes. Must contain all three. |

### Path Template Variables

The `grammar_build` and `grammar_compile` templates support the following placeholders:

| Placeholder | Substituted Value | Example |
| :--- | :--- | :--- |
| `{lang}` / `{language}` | Language name | `python` |
| `{os}` | Target platform | `linux` |
| `{goos}` | Target GOOS | `linux` |
| `{arch}` | Target architecture | `amd64` |
| `{version}` | Grammar version string | `v0.25.0` |
