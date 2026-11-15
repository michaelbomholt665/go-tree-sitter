# Synchronize grammar versions with go.mod

Synchronization updates configured grammar package labels from Go module dependencies. It does not select source commits or update revision pins.

## 1. Choose the files

```bash
ts-build config path
ts-build config get runtime.sync_modules
```

The default `go.mod` is beside the selected configuration. If it is elsewhere, pass `--gomod` explicitly.

The `runtime.sync_modules` map connects module paths to catalog identifiers. Add mappings there instead of changing a hard-coded command table.

## 2. Preview the update

```bash
ts-build --config ./tree-sitter-config.yaml sync-config --gomod ./go.mod --dry-run
```

Inspect each proposed package version. A Go module version is not a source-language version.

The parser accepts single-line requirements and indented require blocks. It does not require tab-only indentation.

## 3. Apply package labels

```bash
ts-build --config ./tree-sitter-config.yaml sync-config --gomod ./go.mod
```

The writer locks the configuration, validates the replacement, and checks that the original file has not changed before replacing it.

If another writer holds the lock, wait for it to finish. A persistent lock file alone does not indicate an active lock.

The writer preserves comments where supported by the YAML update. Do not assume byte-for-byte formatting preservation; inspect the diff.

## 4. Update the corresponding source pins

For every changed package label, identify the matching upstream commit. Update `revision` to that full, 40-character SHA.

Do not leave an old source revision with a newly synchronized package label. Synchronization does not verify that those values identify the same source.

If the revision changes, review source-language evidence tied to the previous pin. Existing assessments can become stale.

## 5. Validate and rebuild

For each changed grammar, such as Python:

```bash
ts-build --config ./tree-sitter-config.yaml config validate
ts-build --config ./tree-sitter-config.yaml list --language python --json
ts-build --config ./tree-sitter-config.yaml build --language python
ts-build --config ./tree-sitter-config.yaml compile --language python
```

Inspect the result before publishing. Existing grammar output requires `move --force`, which replaces its entire directory.

The standalone `cmd/sync-config` remains a source-checkout utility. Normal installed CLI use does not require a separate binary or a Makefile shortcut.
