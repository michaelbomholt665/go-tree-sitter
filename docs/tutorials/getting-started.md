# Tutorial: publish your first Python parser

This tutorial builds only Python on Linux amd64. It uses a new workspace and an explicit configuration so existing releases remain separate.

For Windows, use the script-based [VM test guide](../../TEST-GUIDE.md). For other targets, read [Cross-compile parsers](../how-to/cross-compile-grammars.md).

## Prerequisites

Install `ts-build` on `PATH`, Git, Node.js/npm, GCC/G++, and the Tree-sitter CLI version required by your configuration.

Check the tools:

```bash
ts-build version
git --version
node --version
npm --version
gcc --version
g++ --version
tree-sitter --version
```

The configured Tree-sitter CLI version is a reproducibility pin, not a minimum version.

## 1. Create an isolated configuration

Create a new directory:

```bash
mkdir ts-build-tutorial
cd ts-build-tutorial
```

Resolve and copy the selected default configuration:

```bash
ts-build config path
cp "$(ts-build config path)" ./tutorial.yaml
```

Inspect Python before executing the pipeline:

```bash
ts-build --config ./tutorial.yaml list --language python
ts-build --config ./tutorial.yaml config get tree_sitter_cli_version
```

If the CLI versions differ, install the configured Tree-sitter CLI version before continuing.

Give this tutorial separate intermediate and output directories:

```bash
ts-build --config ./tutorial.yaml config set build_dir ./tutorial-build --apply
ts-build --config ./tutorial.yaml config set output.grammar_base ./tutorial-output --apply
ts-build --config ./tutorial.yaml config set OS_TARGET linux --apply
ts-build --config ./tutorial.yaml config set targets "{\"linux\":{\"os\":\"linux\",\"arch\":\"amd64\",\"triple\":\"x86_64-linux-gnu\",\"compiler\":\"gcc\",\"cxx\":\"g++\",\"compiler_version\":\"$(gcc -dumpfullversion -dumpversion)\"}}" --apply
ts-build --config ./tutorial.yaml config set runtime.compiler.allow_fallback false --apply
ts-build --config ./tutorial.yaml config validate
```

Each `config set` previews changes unless `--apply` is present. Paths above resolve relative to `tutorial.yaml`.

`targets` is one map setting. The command replaces it with a Linux-only map in the tutorial copy.

## 2. Generate Python sources

```bash
ts-build --config ./tutorial.yaml build --language python
```

The builder checks out the configured full revision, installs grammar dependencies according to runtime policy, and generates parser sources. Source provenance records the revision, generator, timestamp, and input hashes.

If checkout fails, inspect the error. The builder preserves an existing workspace unless you explicitly request `build --force`.

## 3. Compile the host library

```bash
ts-build --config ./tutorial.yaml compile --language python --os linux --arch amd64
```

Compilation checks source provenance and produces a fresh artifact with a checksum sidecar. A failed compilation does not turn a stale binary into a successful result.

## 4. Validate and publish

```bash
ts-build --config ./tutorial.yaml move --language python --compact --clean=false
```

The mover loads the host binary, measures its ABI, parses the sample, and validates available queries. It stages a manifest and compact AST schema before publication.

The output is under `tutorial-output/python/`. The intermediate checkout remains because `--clean=false` is explicit.

```bash
ls ./tutorial-output/python
cat ./tutorial-output/python/manifest.json
ts-build --config ./tutorial.yaml compact --language python --check
```

A successful compact check confirms that the YAML preserves the JSON schema semantics.

## 5. Understand repeat runs

A second publication of Python requires explicit replacement:

```bash
ts-build --config ./tutorial.yaml move --language python --compact --clean=false --force
```

If the filesystem does not support atomic exchange, replacement fails and the existing catalog remains intact.

WARNING: `clean` removes intermediate build files. Do not run it until you finish inspecting the checkout.

```bash
ts-build --config ./tutorial.yaml clean --language python
```

The published catalog is separate from the intermediate workspace. Continue with [Build and release grammars](../how-to/build-and-release-grammars.md).
