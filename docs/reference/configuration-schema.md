# Configuration Schema Reference (`tree-sitter-config.yaml`)

This document defines the schema, data types, validation rules, and default behaviors for [tree-sitter-config.yaml](../../tree-sitter-config.yaml) parsed by [`LoadConfig`](../../internal/tree-sitter/config.go#L68).

---

## Root Structure

```yaml
version: "2.0"
build_dir: "build"
tree_sitter_cli_version: "0.27.0"
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

output:
  grammar_base: "data/tree-sitter/grammar"
  grammar_build: "{build_dir}/{lang}/{arch}"
  grammar_compile: "{build_dir}/{lang}/{arch}"
  generate_manifest: true
  default_move_mode: "--both"
  supported_move_modes:
    - "--json"
    - "--scm"
    - "--both"

runtime:
  cleanup:
    after_move: "remove" # keep, prune, or remove, after successful publication
  # See Runtime Policies below for the complete settings.

languages:
  - { ... }

```

---

## 1. Toolchain & Global Settings

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `version` | string | **Yes** | Schema version. Must be set to `"2.0"`. |
| `build_dir` | string | **Yes** | Path for cloned repositories and intermediate build artifacts. Defaults to `"build"`. |
| `tree_sitter_cli_version` | string | **Yes** (in v2.0) | Exact `tree-sitter` CLI version required from `runtime.tools.tree_sitter` (e.g. `"0.27.0"`). |
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
    version: "v0.25.1-0.20250915213335-26855eabccb1"
    repository: "https://github.com/tree-sitter/tree-sitter-python.git"
    revision: "26855eabccb19c6abf499fbc5b8dc7cc9ab8bc64"
    source_subdir: ""
    sample: "x = 1\n"
    note: "Core language"
```

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `name` | string | **Yes** | Language identifier matching `^[A-Za-z0-9._-]+$`. Used in publication directories, binary prefixes, manifest `grammar`, and CLI flags. |
| `grammar` | string | No | C symbol suffix matching `^[A-Za-z0-9_]+$`. Defaults to `name` with `.` and `-` stripped or replaced. |
| `constructor` | string | No | Exported C function name. **Must strictly equal** `"tree_sitter_" + grammar`. |
| `version` | string | **Yes** | Upstream version label (kept in sync with `go.mod`). |
| `generate_abi` | integer | No | Optional per-language target ABI (e.g. `13`, `14`, or `15`) overriding the global `generate_abi`. |
| `repository` | string | **Yes** | Absolute HTTPS git repository URL. |
| `revision` | string | **Yes** (in v2.0) | Pinned 40-character hexadecimal git commit SHA. |
| `source_subdir` | string | No | Subdirectory within repo if multi-grammar repository (e.g. `"tsx"` or `"typescript"`). |
| `sample` | string | **Yes** (in v2.0) | Minimal valid code snippet used to smoke-test parser output during validation. |
| `note` | string | No | Optional human-readable documentation note. |
| `language_support` | object | No | Revision-pinned language syntax assessment; absent means unknown/unverified. See below. |

### Language-version assessments (`language_support`)

An assessment describes **language syntax versions**, not the grammar package version,
Tree-sitter CLI version, or parser ABI. Omit it until research is performed; omission
requires no work and never implies compatibility.

Illustrative metadata only (not an assessment of any published grammar):

```yaml
language_support:
  assessed_revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  claim: unverified
  min_version: "2020"
  max_version: "2024"
  limitations:
    - "Declared range has not yet been verified."
```

| Field | Required | Meaning |
| :--- | :--- | :--- |
| `assessed_revision` | Yes, when the object exists | Full 40-character hexadecimal grammar commit SHA for this assessment, independent of active `revision`. |
| `claim` | No | `unverified` (default) or `verified`. Verification is limited to the recorded evidence and scope. |
| `min_version`, `max_version` | No | Inclusive declared version bounds, expressed as language-specific labels (not necessarily SemVer). Missing bounds are unknown, **not** unlimited support. |
| `limitations` | No | Nonempty strings recording exclusions, partial support, or uncertainty. |
| `evidence` | Required for `verified` | List of documentation or successful version-specific syntax-fixture records. |

A verified assessment also needs at least one version bound or explicit limitation.
Do not infer a continuous compatibility range or full conformance from isolated
feature tests. `description` must state the explicit documentation claim or the
actual tested scope, including any gaps; record exclusions in `limitations`.

Evidence records use these fields:

| Field | Meaning |
| :--- | :--- |
| `kind` | `documentation` or `syntax_fixtures`; other evidence types are rejected. |
| `source` | Required source reference. Documentation requires an HTTPS URL without credentials. For a verified documentation claim, its URL path must contain the exact `assessed_revision` as a complete segment (for example, `https://github.com/OWNER/REPO/blob/<full-SHA>/README.md`). A branch URL is not pinned evidence. |
| `description` | Required text stating the explicit upstream claim or precise tested coverage. A commit title or grammar release number is not a language-support claim. |
| `result` | Syntax fixtures only; must be `passed`. |
| `tested_versions` | Syntax fixtures only; nonempty list of version labels actually covered. |
| `sha256` | Syntax fixtures only; 64-character hexadecimal digest of the archived test report referenced by `source`. |

Fixture reports must identify the assessed grammar revision, fixture suite and
fixture checksums, expected-tree or syntax assertions, actual results, and exclusions.
They establish only the recorded syntax coverage, not complete language conformance.

The loader rejects unknown assessment/evidence fields and checks record structure and pin format. It does **not** fetch documents,
interpret their claims, check report contents/digests, or rerun fixture tests. The
assessment author must review the evidence before setting `claim: verified`;
`verified` is an evidence-backed recorded assessment, not an automatic certification.

**Assessment workflow:**

1. No assessment: leave absent and unverified; no research or tests are needed.
2. Documentation research: perform as a separate, explicitly authorized follow-up.
   Record only supported conclusions; ranges without qualifying evidence stay unverified.
3. Verification: perform version-specific fixture work only when explicitly requested.
   An explicit support claim found during step 2 can instead qualify directly through
   pinned documentation. Do not start additional verification merely because a dependency changes.

Changing `go.mod`, synchronizing versions, or updating active `revision` must not
change `assessed_revision`, bounds, or evidence. Create a new assessment deliberately
when reassessing an updated grammar. A stale assessment is retained in the manifest,
but the **current binaries** are marked unverified with `revision_mismatch`.
See [Manifest Schema v2](manifest-schema-v2.md) for the emitted representation.

---

## 4. Output Configuration (`output`)

Defines destinations for compiled binaries, AST metadata, and release catalogs.

```yaml
output:
  grammar_base: "data/tree-sitter/grammar"
  grammar_build: "{build_dir}/{lang}/{arch}"
  grammar_compile: "{build_dir}/{lang}/{arch}"
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
| `grammar_build` | string | `"{build_dir}/{lang}/{arch}"` | Template path for intermediate grammar source files. |
| `grammar_compile` | string | `"{build_dir}/{lang}/{arch}"` | Template path for compiled binary shared libraries. |
| `generate_manifest` | bool | `true` | Must be `true`. Manifest-only generation without native validation is prohibited. |
| `default_move_mode` | string | `"--both"` | Default move mode used by `ts-build move` if no mode flag is passed. |
| `supported_move_modes` | list[string] | `["--json", "--scm", "--both"]` | List of allowed move modes. Must contain all three. |

### Path Template Variables

The `grammar_build` and `grammar_compile` templates support the following placeholders:

| Placeholder | Substituted Value | Example |
| :--- | :--- | :--- |
| `{build_dir}` | Intermediate workspace directory | `/tmp/ts-build/work` |
| `{lang}` / `{language}` | Language name | `python` |
| `{os}` | Target platform | `linux` |
| `{goos}` | Target GOOS | `linux` |
| `{arch}` | Target architecture | `amd64` |
| `{version}` | Grammar version string | `v0.25.0` |

## 5. Runtime Policies (`runtime`)

All runtime settings precede `languages` in the shipped YAML. The language catalog is the final root section.
The Makefile remains a development shortcut, not a runtime setting.

Omitted settings retain the previous execution defaults. Explicit `false`, empty lists, and empty maps replace their defaults.
Lists and maps replace the entire configured collection; they do not merge with defaults.

### Workspace and cleanup

`build_dir` controls intermediate files. `output.grammar_base` controls the final published catalog.
Use `{build_dir}` in both intermediate path templates so changing the workspace also changes source and binary locations.

```yaml
# tree-sitter-config.yaml (place before languages)
build_dir: "/tmp/ts-build/work"
output:
  grammar_base: "/srv/tree-sitter/grammar"
  grammar_build: "{build_dir}/sources/{lang}/{arch}"
  grammar_compile: "{build_dir}/compiled/{lang}/{os}/{arch}"
runtime:
  cleanup:
    after_move: "prune"
```

| `runtime.cleanup.after_move` | Action after successful publication |
| --- | --- |
| `keep` | Retain intermediate files. |
| `prune` | Remove configured cache files and directories; retain sources and binaries. |
| `remove` | Remove the selected language's intermediate source and binary directories. This is the default. |

Cleanup follows the actual path templates, including all configured target directories for the selected language.
A failed validation or publication retains the workspace. Cleanup never precedes the atomic catalog swap.
If cleanup fails after publication, the command reports an error; the validated release remains published.

WARNING: `remove` deletes the selected language's intermediate files for all configured targets.
Use `keep` to retain these files after publication.

Intermediate templates must contain `{lang}` or `{language}` and resolve strictly inside `build_dir`.
The workspace and publication directory must not overlap. Symlink targets are checked before cleanup.

`move --clean=false` overrides the YAML with `keep`.
`move --clean=true` overrides the YAML with `remove`.
Without an explicit flag, the command uses `after_move`.
`clean --prune` and post-build pruning use the same prune rules.
`clean` without `--prune` removes intermediate files regardless of `after_move`.

The `runtime.cleanup.prune` object has these lists:

| Field | Matching and precedence |
| --- | --- |
| `directories` | Directory basenames or relative directory patterns to remove. |
| `files` | File patterns to remove outside preserved directories. |
| `preserve_directories` | Directory patterns to retain, including their contents except `protected_files`. Defaults include `src` and `queries`. |
| `preserve_files` | File patterns retained outside preserved directories, before applying `files`. |
| `protected_files` | File patterns removable inside preserved directories. Defaults are `*.o`, `*.obj`, `*.a`, and `*.lib`. |

Patterns use filepath-style globs without absolute paths or parent traversal.
The shipped YAML contains the complete default lists.

### Tools, generation, and compilation

| Field | Default | Behavior |
| --- | --- | --- |
| `tools.tree_sitter`, `tools.git`, `tools.npm` | `tree-sitter`, `git`, `npm` | Executables used for source generation and dependency installation. |
| `tools.emcc`, `tools.docker`, `tools.podman`, `tools.zig` | Corresponding executable names | Executables used for WASM and compiler fallback. |
| `wasm.backend` | `auto` | `auto` searches emcc, Docker, then Podman. An explicit `emcc`, `docker`, or `podman` forbids selecting another backend. |
| `wasm.image` | `emscripten/emsdk` | Container image for Docker or Podman. Set a tag or digest to pin it; the compatibility default is unpinned. |
| `compiler.allow_fallback` | `true` | Allow host compiler, candidate compiler, and Zig fallback if the selected toolchain is unavailable. |
| `compiler.host_cc` | `cc` | Host compiler fallback executable. |
| `compiler.candidates` | Shipped per-target map | Ordered lists of `{cc, cxx}` pairs keyed by `linux/amd64`, `windows/amd64`, `macos/amd64`, or `macos/arm64`. |
| `generation.abi_fallback` | `true` | Retry generation at lower ABIs inside `abi_range`. Set `false` to require the selected ABI. |
| `supported_targets` | Linux amd64, Windows amd64, macOS amd64/arm64 | Map from OS to architecture lists. Supported OS names are `linux`, `windows`, and `macos`; architecture names are `amd64` and `arm64`. |

Tool values are executable names or paths, not shell command strings.
The CLI resolves relative paths containing separators against the selected configuration directory.
Bare executable names use PATH. Command arguments remain separate from executable names.
The existing `targets` section still defines explicit compilers, triples, and compiler versions.

### Node dependencies

| Field | Default | Behavior |
| --- | --- | --- |
| `node.install` | `auto` | `auto` installs only with a lockfile and no existing `node_modules`; `never` skips; `always` reinstalls, including without a lockfile. |
| `node.ci_args` | `[ci, --ignore-scripts]` | npm arguments when a lockfile exists. |
| `node.install_args` | `[install, --ignore-scripts, --no-audit, --no-fund]` | npm arguments without a lockfile, or after a failed CI install. |
| `node.retry_install` | `true` | Retry a failed CI install with `install_args`. |

Both argument lists must retain `--ignore-scripts`. Script-enabling overrides are rejected.

### Publication and wizard defaults

`runtime.publication` sets defaults for `manifest`, `scm`, `compact`, `check_compact`, `source`, `js`, `wasm`, `force`,
and `allow_cross_validation`. Manifest generation defaults to `true`; the other boolean defaults are `false`.
Omitted `scm` follows `output.default_move_mode`.
Explicit CLI flags take precedence, including `--manifest=false`, `--scm=false`, and `--wasm=false`.
The `wasm` and `allow_cross_validation` defaults also apply to `compile`.
Disabling manifest output does not disable binary validation.

The wizard is an interactive CLI. Its choices use these settings:

| Field | Default | Allowed entries |
| --- | --- | --- |
| `wizard.platforms` | `[host]` | `host`, `wasm`, or configured OS/architecture pairs such as `linux/amd64`. |
| `wizard.artifacts` | `[manifest, scm]` | `manifest`, `scm`, `compact`, `source`, `js`, `wasm`. |

`host` uses the current OS and architecture. An unavailable host target is not preselected.
Wizard selections override its initial choices; its cleanup uses `cleanup.after_move`.

### Query files and module synchronization

`runtime.queries.paths` maps a language name to repository-relative `.scm` filenames.
A configured language entry replaces automatic query discovery; an empty list disables copying its queries.
Paths must remain inside the checked-out repository, including symlink targets.

`runtime.queries.rewrites` is a list of `{language, file, find, replace}` records.
Each record replaces every literal `find` occurrence in the named staged query file before validation.
Missing files are ignored. The compatibility default removes ` (shebang)` from OCaml-interface `highlights.scm`.

`runtime.sync_modules` maps Go module paths to lists of language names.
`sync-config` uses this map when it exists; an empty map disables synchronization.
If the map is absent, the command uses the built-in compatibility mapping.
Synchronization updates version labels only, not source revisions or assessment pins.
