# Configure runtime policy and inspect the catalog

Use configuration commands to control execution without changing grammar source pins.

## Select the file explicitly

```bash
ts-build config path
ts-build config validate
ts-build --config ./tree-sitter-config.yaml config show
```

An explicit `--config` overrides automatic selection. Without it, a current-directory development marker selects the repository config; normal use selects the OS user config.

Relative paths resolve from the configuration directory. Existing user configurations are not replaced automatically during an upgrade.

## Preview and save a setting

```bash
ts-build config get runtime.compiler.allow_fallback
ts-build config set runtime.compiler.allow_fallback false
ts-build config set runtime.compiler.allow_fallback false --apply
ts-build config validate
```

`config set` previews by default. Only `--apply` saves the validated change.

Use `config edit` for an editor-based update. Setting commands reject unknown keys and validate the complete configuration before saving.

Maps and lists are complete settings. For example, update `targets` as one map; `targets.linux.arch` is not an editable key.

Configuration updates use a shared OS lock, a temporary replacement, and an original-content check. If another writer changes the file, the update fails instead of overwriting that change.

A lock file can remain after the lock is released. Do not delete it to override a running writer.

## Configure tools and policy

Inspect the effective runtime section before editing:

```bash
ts-build config show
```

| Section | Controls |
| --- | --- |
| `runtime.tools` | Paths for Tree-sitter, Git, npm, Emscripten, containers, and Zig |
| `runtime.compiler` | Host compiler preference, candidates, and fallback permission |
| `runtime.node` | Dependency installation mode, arguments, and retry permission |
| `runtime.generation` | Generation ABI fallback permission |
| `runtime.cleanup` | Post-move cleanup and pruning rules |
| `runtime.publication` | Defaults for manifests, queries, compact schemas, sources, and replacement |
| `runtime.wasm` | WASM backend and container image |
| `runtime.wizard` | Initial platform and artifact choices |
| `runtime.queries` | Query search paths and configured rewrites |
| `runtime.sync_modules` | Module-to-grammar mappings for version synchronization |

A compiler version pin must describe the compiler that actually runs. Do not change it merely to suppress an unexplained mismatch.

On Windows, use a native Tree-sitter executable. Batch wrappers are not accepted as executable tool paths. npm package operations resolve the npm JavaScript CLI through Node.

See the [configuration reference](../reference/configuration-schema.md) for exact keys.

## Inspect recorded source-language support

```bash
ts-build list
ts-build list --language python --json
ts-build support show --language python
```

These commands display configured grammar identities and recorded assessments. They do not execute language-version fixtures.

A grammar package version and parser ABI are not source-language version bounds. Evidence is tied to a pinned revision; changing that revision can make an assessment stale.

Read [VERIFY_VERSIONS.md](../reference/VERIFY_VERSIONS.md) before researching language support. Use documentation evidence by default. Run version-specific fixtures only after an explicit request.

For a complete assessment saved in a JSON file, preview before applying:

```bash
ts-build support update --language python --assessment ./assessment.json
ts-build support update --language python --assessment ./assessment.json --apply
```

The updater validates the assessment structure. It does not prove the claim or execute tests. Use `--expect-revision` for guarded updates and `--check-report` to verify local report checksums.

The [provider coverage reference](../reference/provider-grammar-coverage.md) records research metadata. Provider dispatch is not implemented.

## Export published output

```bash
ts-build export --language python --to ./consumer-grammars
```

The destination contains a `python/` directory copied from the selected published catalog. Export does not change grammar pins or generate a new manifest.

See [CLI reference](../reference/cli.md) for flags and [Synchronize versions](sync-versions-with-go-mod.md) for module updates.
