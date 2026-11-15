# Add a new grammar

Add a source grammar to the selected catalog. A Go binding is optional and separate from artifact compilation.

## 1. Inspect the upstream repository

Identify the grammar root, constructor, sample syntax, queries, and license. If the repository contains multiple grammars, identify the required subdirectory.

Resolve the desired upstream revision to a full, 40-character hexadecimal commit SHA. Do not use a branch, tag, or abbreviated hash as `revision`.

Inspect the source license at that revision before distributing artifacts. See [NOTICE](../../NOTICE.md) for current scan limits.

## 2. Edit the selected configuration

```bash
ts-build config path
```

Open the returned file in a text editor. Add an entry under `languages` using an existing grammar with a similar layout as the model.

Set these fields:

| Field | Meaning |
| --- | --- |
| `name` | Unique catalog identifier |
| `grammar` | Grammar name used by the build |
| `constructor` | Exported C constructor |
| `version` | Upstream package label |
| `repository` | Upstream Git URL |
| `revision` | Full source commit SHA |
| `sample` | Small valid program for host validation |

Configure subdirectory and generation overrides only when the upstream layout requires them. See the [configuration schema](../reference/configuration-schema.md).

`config set` and `config edit` manage non-language settings; they are not grammar-entry editors.

## 3. Record support evidence separately

Do not infer a source-language version from a package version or parser ABI.

Read [VERIFY_VERSIONS.md](../reference/VERIFY_VERSIONS.md). Record documentation evidence for the pinned revision. Leave unsupported claims unknown or unverified.

Version-specific fixtures require an explicit request. Adding a grammar or parsing its smoke sample does not authorize or establish a version-support claim.

## 4. Validate and build only the new grammar

Replace `mylang` below with the new identifier:

```bash
ts-build config validate
ts-build list --language mylang
ts-build build --language mylang
ts-build compile --language mylang
ts-build move --language mylang --compact --clean=false
ts-build compact --language mylang --check
```

Use a separate output root for experimental releases. On Windows, choose a new catalog destination because replacement is unsupported.

## 5. Add a Go binding only if required

The CLI can build grammar repositories without registering Go bindings.

If repository-internal tests require an upstream Go binding, add its module dependency and update the tagged registry in `internal/grammars/`. That registry is not importable by external consumer modules.

If module synchronization must update this grammar's package label, add the module mapping under `runtime.sync_modules` in configuration.

These optional bindings are a subset of the source grammar catalog. Do not describe the full catalog as an equivalent Go binding registry.

## 6. Update release records

Update the grammar catalog reference, changelog, and dependency notices. Run the standard test suite from a source checkout:

```bash
go test ./...
```

See [Build and release](build-and-release-grammars.md) for replacement and validation rules.
