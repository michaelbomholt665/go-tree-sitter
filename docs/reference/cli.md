# CLI Command Reference (`ts-build`)

`ts-build` is the command-line interface for cloning, generating, cross-compiling, validating, and publishing Tree-sitter grammars.

---

## Synopsis

```bash
ts-build [command] [flags]
```

Or via Go:

```bash
go run ./cmd/ts-build [command] [flags]
```

---

## Configuration selection

The binary embeds the repository's complete `tree-sitter-config.yaml` at build time, including grammar pins and historical assessments.
Normal use requires no configuration flag and no `.env` file.

| Context | Selected configuration |
| --- | --- |
| Current directory has `.env` with `TS_BUILD_ENV=DEVELOPMENT` | `./tree-sitter-config.yaml` |
| No development marker, including `TS_BUILD_ENV=PRODUCTION` | OS user configuration below |

| OS | User configuration |
| --- | --- |
| Linux | `$XDG_CONFIG_HOME/ts-build/tree-sitter-config.yaml`, or `~/.config/ts-build/tree-sitter-config.yaml` when unset |
| macOS | `~/Library/Application Support/ts-build/tree-sitter-config.yaml` |
| Windows | `%AppData%/ts-build/tree-sitter-config.yaml` |

If the user configuration is missing, the app creates it from the embedded catalog.
Existing configurations are never overwritten or silently replaced, including after binary upgrades.
Initialized configurations use OS cache paths for builds and OS data paths for published grammars.
Relative paths in a selected configuration resolve against that file's directory, not the current working directory.

The repository includes a tracked `.env` containing `TS_BUILD_ENV=DEVELOPMENT`.
Run `go run ./cmd/ts-build ...` from the repository root to use its local configuration and existing directory layout.
An installed binary invoked from that same marked directory also uses the local configuration.
The app does not search parent directories, import `.env` settings into the process environment, or infer execution mode from the executable path.
If development mode is selected and the local configuration is missing, the command fails rather than selecting the user configuration.
Invalid marker values and invalid existing configurations also produce errors.

Use `ts-build config path` to print the selected configuration file's absolute path.
The command uses the same selection and first-run initialization as other commands.
It does not parse the YAML, so it can locate an existing file with invalid settings.

```bash
ts-build config path
```

The existing `--config`/`-c` flag remains an explicit override for compatibility.
Help, version, completion, and the blocked manifest-regeneration command do not initialize a configuration.

Runtime defaults come from `runtime` in the selected YAML. Explicit flags override them, including explicit `false` values.
The flag tables below show compatibility defaults; YAML can change publication, WASM, and cross-validation defaults.
See [Runtime Policies](configuration-schema.md#5-runtime-policies-runtime) for the full schema.

`build_dir` is the intermediate workspace; `output.grammar_base` is the published destination.
Use `{build_dir}` in the intermediate output templates to relocate the complete workspace.
After successful publication, `runtime.cleanup.after_move` selects `keep`, `prune`, or `remove` (default).
`move --clean=false` forces `keep`; `move --clean=true` forces `remove`.
A failed validation or publication retains intermediate files.

The wizard is interactive. `runtime.wizard` defines its initial platform and artifact selections.
The Makefile remains a development-only command shortcut and is not part of runtime configuration.

---

## Settings editor

`ts-build config edit` opens a categorized editor for all non-language settings.
The main wizard also provides **Settings**, **Build grammars**, and **Exit** sections.
After saving settings, the wizard returns to its main menu. A subsequent build loads the saved configuration.

The editor shows current values with defaults applied. It provides enum choices, boolean prompts, and text or integer inputs.
Lists and mappings provide add, edit, and remove actions. Structured entries provide field menus rather than requiring JSON input.
Changes remain in memory until **Review and save** and an explicit confirmation. Discard or interruption does not save pending changes.
Each change must preserve a valid configuration. Adjust dependent settings in a valid order.
For example, change wizard platform defaults before removing a platform from `runtime.supported_targets`.

The same settings are available through direct commands:

```bash
ts-build config show --json
ts-build config get runtime.cleanup.after_move
ts-build config set runtime.cleanup.after_move keep
ts-build config set runtime.cleanup.after_move keep --apply
ts-build config set runtime.sync_modules '{"example.com/grammar":["python"]}' --apply
ts-build config validate
ts-build config edit
```

| Command | Behavior |
| --- | --- |
| `config path` | Print the selected configuration path. |
| `config show [--json]` | Show all non-language settings with defaults applied. |
| `config get <key> [--json]` | Read one editable setting. |
| `config set <key> <value>` | Preview one complete setting replacement; default does not save. |
| `config set <key> <value> --apply` | Validate and save one replacement. |
| `config edit` | Edit settings interactively, review the complete change list, then confirm saving. |
| `config validate [--json]` | Validate the schema and check workspace/publication directory separation. |

Keys use YAML names separated by dots, such as `output.grammar_base` and `runtime.tools.git`.
Root names are case-sensitive; use `OS_TARGET` for the default operating system.
Maps and lists are complete values in `config set`; individual module names or list indexes are not dotted keys.
Use ordinary text or a quoted JSON string for strings. Use typed JSON for booleans, integers, arrays, and objects.
Nullable settings accept `null` to restore their fallback behavior. Empty arrays and objects retain their schema-defined meaning.
If a value starts with a dash, place `--` before positional arguments or provide it as a quoted JSON string.
`config set` supports `--dry-run`, `--apply`, and `--json`. The two mode flags are mutually exclusive.
Its JSON result includes `key`, `before`, `after`, `changed`, `applied`, and `warnings`.
Defaulted values are displayed without rebasing relative paths; relative paths still resolve from the selected configuration directory during execution.

Both editors preserve unrelated values, comments, grammar pins, and assessments. YAML formatting can change.
They share atomic file replacement, concurrency checks, and the update lock with `support update`.
Symlink configurations are rejected when writing. A saved setting takes effect in subsequent commands, not a running build.
Editing settings never executes tools, deletes workspaces, or republishes artifacts.
Tool and cleanup settings affect later build/clean/publication commands; review their warnings before saving.
The editor cannot change `languages`; use `support update` for assessments and edit grammar pins separately.
Deprecated fields remain editable for compatibility but retain their documented legacy behavior.

## Persistent Global Flags

These flags are available on all subcommands:

| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--config` | `-c` | string | `tree-sitter-config.yaml` | Explicit configuration override; otherwise select the development or OS user configuration automatically. |
| `--verbose` | `-v` | bool | `false` | Stream raw child-process stdout/stderr in real time. |
| `--quiet` | `-q` | bool | `false` | Silence all non-error output. |
| `--dry-run` | | bool | `false` | Validate inputs and preview actions without executing tools or changing files. |
| `--help` | `-h` | bool | | Display help information for any command. |

---

## Dry runs

Add `--dry-run` before or after a subcommand to preview its actions.
The flag does not run external tools, clone repositories, write artifacts, publish releases, or clean workspaces.
Missing OS user configuration is prepared in memory, not created on disk.
Explicit configuration paths and development markers retain their normal validation rules.

```bash
ts-build build --language python --force --dry-run
ts-build compile --language python --os linux --arch amd64 --dry-run
ts-build move --language python --js --manifest=false --dry-run
ts-build clean --language python --prune --dry-run
ts-build export --language python --to /tmp/parsers --dry-run
ts-build wizard --dry-run
```

| Command group | Dry-run behavior |
| --- | --- |
| `build`, `compile`, `move`, `clean`, `compact` | Print JSON plans with selected grammars, resolved request options, paths, tools, and planned actions. |
| `export` | Check source and destination conflicts, reject unsafe file types, and show the copy plan without creating staging directories. |
| `wizard` | Keep interactive selection; print plans instead of executing the selected pipeline. |
| `config edit`, wizard Settings | Keep interactive editing and review; confirmation previews saving without writing. |
| `config set`, `support update` | Validate and preview changes; reject combining `--dry-run` with `--apply`. |
| `sync-config` | Compute and report version changes without saving them. |
| Read-only commands, help, completion | Retain their normal read-only output. |
| `regen-manifests` | Retain the manifest-only regeneration refusal. |

Dry runs validate flags, configuration, language selection, and requested target compatibility.
They do not prove that a build succeeds. Tool availability, tool versions, network access, binary validity, and runtime permissions remain unchecked.
Compact generation and lossless checks are planned, not performed.
Report checksums requested with `support update --check-report` are still checked locally; no fixture code executes.
A nonzero exit status identifies a detected input or preflight error, not a simulated build failure.

---

## Subcommands

### `ts-build list`

Lists configured grammars, not only published grammars.
Shows recorded source-language bounds, the effective claim, evidence rule, and assessment status.
Missing bounds display as `unknown`.

```bash
ts-build list
ts-build list --language kotlin
ts-build list --json
```

| Rule | Meaning |
| --- | --- |
| 1 | No recorded assessment. |
| 2 | Recorded documentation assessment, including inconclusive research and estimated bounds. Documentation-backed claims can be verified. |
| 3 | Recorded successful version-specific fixture evidence. Verification remains limited to its tested versions and syntax coverage. |

Rule numbers describe recorded evidence; this command does not fetch sources, verify report digests, or execute fixtures.
`--json` includes active revisions, historical assessment revisions, limitations, and full evidence records.
If the revisions differ, the effective claim is `unverified`, bounds display as unknown, and the assessment is `stale`.
The JSON retains historical bounds inside `assessment`; they do not describe the active revision.
Package versions and parser ABIs are not source-language version bounds.

### `ts-build support show` / `ts-build support update`

Inspect or replace one grammar's `language_support` assessment in the selected configuration.
Users can supply their own external fixture suites and reports; repository fixtures are not required.
The command does not execute tests, fetch sources, build parsers, or publish artifacts.

```bash
ts-build support show --language python --json
ts-build support update --language python --assessment ./assessment.json --check-report --dry-run --json
ts-build support update --language python --assessment ./assessment.json --check-report --apply --json
```

| Update flag | Default | Meaning |
| --- | --- | --- |
| `--language`, `-l` | Required | Exact configured grammar identifier. |
| `--assessment` | Required | Complete assessment JSON file; `-` reads stdin. |
| `--expect-revision` | Unset | Require this full SHA to match the current grammar pin. |
| `--dry-run` | Preview | Validate and show before/after values without saving. Preview also applies when neither mode flag is supplied. |
| `--apply` | `false` | Save the validated assessment. Mutually exclusive with `--dry-run`. |
| `--check-report` | `false` | Read every syntax-fixture report and compare its SHA-256. Require at least one fixture record. |
| `--json` | `false` | Emit before/after values, `changed`, `applied`, and `reports_checked`; assessment errors return an `error` field. |

Use the existing assessment fields: `assessed_revision`, `claim`, `min_version`, `max_version`, `limitations`, and `evidence`.
The claim must be explicit: `unverified` or `verified`.
The assessed full SHA must match the active grammar revision. Unknown JSON fields are rejected.
The update replaces the complete assessment; it does not append evidence to an older assessment.

For example, create `assessment.json` with this structure. Replace the placeholders before use:

```jsonc
// assessment.json (save as strict JSON without this comment)
{
  "assessed_revision": "FULL_40_CHARACTER_ACTIVE_GRAMMAR_SHA",
  "claim": "verified",
  "min_version": "3.12",
  "max_version": "3.12",
  "limitations": ["Only the syntax assertions in the supplied suite were tested; not complete Python conformance."],
  "evidence": [{
    "kind": "syntax_fixtures",
    "source": "./report.json",
    "description": "User-owned version-specific fixtures, syntax assertions, fixture checksums, results, and exclusions.",
    "sha256": "SHA256_OF_REPORT_FILE",
    "result": "passed",
    "tested_versions": ["3.12"]
  }]
}
```

Record the exact grammar revision, suite identity, fixture checksums, assertions, results, and exclusions in the report.
Calculate the report's SHA-256 after completing it. The command accepts any regular report file; it does not certify its contents.
Fixture evidence is Rule 3. Documentation-backed verification remains Rule 2, not Rule 3.
Successful isolated fixtures do not prove full conformance or a continuous version range.
See [VERIFY_VERSIONS.md](../../VERIFY_VERSIONS.md) for the evidence policy.

Relative fixture report paths resolve against the assessment file's directory, or the current directory for stdin.
The command saves local report references as absolute paths. Reports can remain outside this repository.
Without `--check-report`, remote report references remain references only.
With `--check-report`, reports must use local paths or local `file://` URLs; HTTP downloads are not supported.
The checksum check verifies report integrity, not the truth of the test results.

The command preserves unrelated values, YAML comments, key order, and file permissions. YAML formatting can change.
It stages the replacement beside the selected configuration and checks for content changes before replacing the file.
Cooperating update commands use `<config>.support.lock`; symlink configurations are rejected when writing.
If an interrupted process leaves a lock file, confirm no update is running before removing that lock.
Other editors do not honor this lock; avoid simultaneous edits from those programs.
The command does not change grammar pins, other runtime settings, or published manifests.

### `ts-build export`

Copies published grammar directories to a destination without deleting the published originals.
Copies every regular file, including binaries, `node-types.json`, optional `grammar.js`, queries, and optional manifests.
A manifest is not required.

```bash
ts-build --config /path/to/tree-sitter-config.yaml export --to /path/to/consumer/grammars
ts-build --config /path/to/tree-sitter-config.yaml export --language python --to /path/to/consumer/grammars
```

The destination receives `<destination>/<grammar-name>/...`.
The source is `output.grammar_base`, resolved relative to the configuration file when it is not absolute.
An explicit configuration path makes export usable from another working directory through an installed `ts-build`.
The default selection includes configured grammars with published directories; it skips grammars that are not yet published.
If an explicitly selected grammar is not published, the command fails.

WARNING: `--force` replaces each selected destination grammar directory, including files absent from the source. Preserve consumer-specific files outside those directories.

Without `--force`, an existing selected directory causes failure before export changes grammar files.
Unselected destination directories remain untouched.
Export stages all selected grammars before replacing destination directories and attempts rollback if installation fails.
Source and destination paths must not overlap, including through symbolic links.
Grammar contents must be regular files or directories; symbolic links and special files are rejected.
Do not modify source or destination directories concurrently with export.

### `ts-build sync-config`

Synchronizes grammar package versions from `go.mod` into the YAML configuration.
It preserves grammar revision pins and historical language-support assessments.
The default `go.mod` path is beside the configuration file; `--gomod` overrides it.

```bash
ts-build --config /path/to/tree-sitter-config.yaml sync-config --dry-run
ts-build --config /path/to/tree-sitter-config.yaml sync-config --gomod /path/to/go.mod
```

After a version changes, resolve its corresponding upstream commit and update the full revision pin deliberately.
Do not repin historical assessments without new evidence.
The compatibility entry point `go run ./cmd/sync-config` uses the same implementation.

### `ts-build regen-manifests`

Retains the standalone tool's safety refusal as an installed CLI subcommand.
It returns an error without writing artifacts.
Manifest-only regeneration remains disabled because downstream ABI and checksum metadata require validation of real binaries.
Use the binary-validated `build` → `compile` → `move --manifest=true` pipeline instead.
The compatibility entry point `go run ./cmd/regen-manifests` also remains disabled.

### Publish raw artifacts with `grammar.js`

The existing `move --js` flag includes `grammar.js`; `--manifest=false` omits custom manifest generation.
These options also appear in the wizard.
Binary validation still applies, and successful publication cleans intermediate build directories by default.

```bash
ts-build build --language python
ts-build compile --language python
ts-build move --language python --json --js --manifest=false
```

---

### `ts-build wizard` *(recommended for human use)*

Launches a four-step interactive wizard that guides you through every configuration choice before running the build pipeline. No flags are required.

```bash
ts-build wizard
```

The wizard walks through the following steps in order:

| Step | Prompt | Description |
| :--- | :--- | :--- |
| 1 | **Pipeline** | Choose which phases to run: build only, build+compile, full end-to-end, compile only, or publish only. |
| 2 | **Target Platforms** | Multi-select from Linux amd64, macOS arm64, macOS amd64, Windows amd64, and WebAssembly. |
| 3 | **Artifacts & Options** | Toggle `manifest.json`, `.scm` query files, `compact-node-types.yaml`, C/C++ sources, `grammar.js`, WebAssembly staging, force-overwrite, and build-cache pruning. |
| 4 | **Grammars** | Select any subset of the 65 configured languages, or choose **[ ALL ]** to process everything. |

Before executing, the wizard displays a compact summary and prompts for final confirmation. Pressing `N` aborts cleanly without touching any files.

> [!NOTE]
> The wizard drives the same `build`, `compile`, and `move` pipeline as the individual subcommands — it is purely a human-friendly front-end, not a separate code path.

---

### 1. `ts-build build`

Clones grammar repositories, checks out pinned commit revisions, installs locked Node dependencies (if a `package-lock.json` is present), and runs `tree-sitter generate` with the configured ABI version.

#### Usage
```bash
ts-build build [flags]
```

#### Flags
| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | string | `""` | Only build the specified language (builds all configured if omitted). |
| `--force` | `-f` | bool | `false` | Re-clone existing repositories by wiping `build/<language>/` before building. |
| `--prune` | | bool | `false` | Prune build cache (`.git`, `node_modules`, test suites, intermediate object files) after building while preserving sources and manifests. |

#### Required System Tools
- `git`
- `tree-sitter` CLI (version must match `tree_sitter_cli_version` in config)
- `npm` (invoked only if repository includes a `package-lock.json`)

#### Outputs
- Grammar sources generated under `build/{language}/`
- Provenance manifest `.source-provenance.json` recording commit revision, generator version, and `SOURCE_DATE_EPOCH`.

---

### 2. `ts-build compile`

Compiles generated grammar sources into platform-specific native shared libraries (`.so`, `.dylib`, or `.dll`).

#### Usage
```bash
ts-build compile [flags]
```

#### Flags
| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | string | `""` | Only compile the specified language. |
| `--os` | `-o` | string | `""` | Target operating system (`linux`, `windows`, `macos`). Defaults to `OS_TARGET` from config. |
| `--arch` | `-a` | string | `""` | Target architecture (`amd64`, `arm64`). Defaults to target arch from config. |
| `--wasm` | | bool | `false` | Build WebAssembly (WASM) parser artifact (`tree-sitter-<lang>.wasm`). |
| `--allow-cross-validation` | | bool | `false` | Allow cross-platform static validation for non-host binaries. |
| `--static-cross-validation`| | bool | `false` | Alias for `--allow-cross-validation`. |

#### Outputs
- Compiled shared library: `build/{language}/{arch}/{language}-{version}-{platform}-{arch}.{ext}`
- WebAssembly artifact (when `--wasm` is passed): `build/{language}/tree-sitter-{language}.wasm`
- Binary provenance file: `*.provenance.json`

---

### 3. `ts-build move`

Stages compiled binaries, AST metadata, and query files; performs dual-mode validation; generates `manifest.json`; and atomically publishes the release catalog.

#### Usage
```bash
ts-build move [flags]
```

#### Flags
| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | string | `""` | Only move the specified language. |
| `--manifest` | | bool | `true` | Generate and validate `manifest.json`. Pass `--manifest=false` (or `--no-manifest`) to skip. |
| `--scm` | | bool | `true` | Publish `.scm` query files. Pass `--scm=false` (or `--no-scm`) to skip copying queries. |
| `--source` | | bool | `false` | Preserve C/C++ parser sources under `data/tree-sitter/grammar/<lang>/src/`. |
| `--c-source` | | bool | `false` | Alias for `--source`. |
| `--js` | | bool | `false` | Preserve `grammar.js` under `data/tree-sitter/grammar/<lang>/grammar.js`. |
| `--wasm` | | bool | `false` | Stage WebAssembly parser artifact under `data/tree-sitter/grammar/<lang>/tree-sitter-<lang>.wasm`. |
| `--compact` | | bool | `false` | Generate, losslessly validate, and stage `compact-node-types.yaml` alongside `node-types.json`, and set `has_compact_node_types: true` in `manifest.json`. |
| `--check-compact` | | bool | `false` | Audit existing `compact-node-types.yaml` against `node-types.json` without regenerating. |
| `--both` | | bool | `false` | Legacy move mode: Move binaries, `node-types.json`, and `queries/`. |
| `--json` | | bool | `false` | Legacy move mode: Move binaries and `node-types.json` only. |
| `--clean` | | bool | `true` | Clean the intermediate build directory for each successfully published language. Pass `--clean=false` to preserve build directories. |
| `--force` | `-f` | bool | `false` | Overwrite existing output files in the catalog. |
| `--allow-cross-validation` | | bool | `false` | Allow cross-platform static validation for non-host binaries. |
| `--static-cross-validation`| | bool | `false` | Alias for `--allow-cross-validation`. |

#### Asset Publishing Flags & Move Modes
By default, `ts-build move` publishes binaries, `node-types.json`, `.scm` queries, and `manifest.json`.
The granular flags allow independent control over published assets:
- `--manifest=false` (or `--no-manifest`): Skip generating and validating `manifest.json`.
- `--scm=false` (or `--no-scm`): Skip copying `queries/` into the destination directory.
- `--source` (or `--c-source`): Preserve compilable C/C++ sources (`parser.c`, `scanner.c`/`scanner.cc`, `tree_sitter/parser.h`, `tree_sitter/alloc.h`) under `src/` and mark `has_c_source: true` in `manifest.json`.
- `--js`: Preserve authoritative grammar specification `grammar.js` and mark `has_js: true` in `manifest.json`.
- `--wasm`: Validate, copy, and checksum `tree-sitter-<lang>.wasm` into the release catalog and mark `has_wasm: true` in `manifest.json`.
- `--compact`: Generate, losslessly validate, and stage token-efficient `compact-node-types.yaml` alongside `node-types.json` and mark `has_compact_node_types: true` in `manifest.json`.
- `--check-compact`: Verify existing `compact-node-types.yaml` against `node-types.json`.

Legacy mode flags (`--both`, `--json`) remain supported for backward compatibility, while explicit `--manifest` or `--scm` flags take precedence.

---

### 4. `ts-build compact`

Generates or validates token-efficient compact AST schema representations (`compact-node-types.yaml`) from `node-types.json` with strict bidirectional lossless validation.

#### Usage
```bash
ts-build compact [flags]
```

#### Flags
| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | string | `""` | Only compact or check the specified language (processes all configured if omitted). |
| `--check` | | bool | `false` | Validate existing `compact-node-types.yaml` without regenerating. Exits with code 1 upon any discrepancy. |

#### Examples
```bash
# Generate compact-node-types.yaml for python:
ts-build compact -l python

# Check / validate existing compact-node-types.yaml without regenerating:
ts-build compact -l python --check

# Audit all published grammars in data/tree-sitter/grammar/ in CI:
ts-build compact --check
```

---

### 5. `ts-build clean`

Cleans or selectively prunes intermediate build directories.

#### Usage
```bash
ts-build clean [flags]
```

#### Flags
| Flag | Shorthand | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--language` | `-l` | string | `""` | Only clean the specified language (cleans all configured if omitted). |
| `--prune` | | bool | `false` | Prune build cache (`.git`, `node_modules`, `test`, `corpus`, `examples`, `target`, `build/Release`, `*.o`, `*.obj`) while strictly preserving sources (`grammar.js`, `src/`), `queries/`, AST schemas (`node-types.json`), and ecosystem manifests (`tree-sitter.json`, `package.json`, `Cargo.toml`, `pyproject.toml`, `setup.py`, `Package.swift`, `go.mod`, `Makefile`, `binding.gyp`, `CMakeLists.txt`, `build.zig`, `LICENSE*`). |

#### Examples
```bash
# Completely remove build output for python:
ts-build clean -l python

# Prune bloat (.git, node_modules, tests) while keeping sources and manifests:
ts-build clean -l python --prune
```

---

### 6. `ts-build version`

Displays the version of `ts-build`.

```bash
ts-build version
# Output: ts-build version 1.0.0
```

---

### 7. `ts-build completion`

Generates auto-completion scripts for your shell.

#### Usage
```bash
ts-build completion [bash|zsh|fish|powershell]
```

#### Enabling Shell Completion
- **Bash:**
  ```bash
  source <(ts-build completion bash)
  ```
- **Zsh:**
  ```bash
  source <(ts-build completion zsh)
  ```
- **Fish:**
  ```bash
  ts-build completion fish | source
  ```
- **PowerShell:**
  ```powershell
  ts-build completion powershell | Out-String | Invoke-Expression
  ```

---

## Environment Variables

| Variable | Description |
| :--- | :--- |
| `SOURCE_DATE_EPOCH` | Overridden automatically by `ts-build` using the git commit timestamp for reproducible binary builds. |
| `CFLAGS` / `CXXFLAGS`| Injected during compilation using flags defined in `build_flags` (e.g. `-O2 -fPIC`). |
| `CC` / `CXX` | Set dynamically by `Compiler` to point to the resolved host compiler, cross-compiler, or Zig wrapper script. |
| `GOOS` / `GOARCH` | Set dynamically to target architecture. |
| `CGO_ENABLED` | Set to `1` during native library builds. |
