# Documentation

The documentation separates learning, task instructions, reference information, and explanations.

The current catalog contains 65 grammar entries. Use `ts-build list` to inspect the selected configuration rather than relying on a fixed documentation count.

## Tutorials

Start here for a guided first result:

- [Getting started](tutorials/getting-started.md): build and publish only Python in an isolated Linux workspace.
- [Windows VM test](../TEST-GUIDE.md): install tools and test only Python through the PowerShell setup script.

## How-to guides

Use these guides for specific tasks:

- [Configure runtime policy and inspect the catalog](how-to/configure-runtime-and-catalog.md)
- [Build and release grammars](how-to/build-and-release-grammars.md)
- [Add a grammar](how-to/add-a-new-grammar.md)
- [Cross-compile parsers](how-to/cross-compile-grammars.md)
- [Compile WebAssembly parsers](how-to/compile-webassembly-parsers.md)
- [Generate and validate compact node types](how-to/generate-and-validate-compact-node-types.md)
- [Synchronize module versions](how-to/sync-versions-with-go-mod.md)
- [Use grammars from Go](how-to/use-compiled-grammars-in-go.md)

## Reference

Look up exact command, configuration, and schema details:

- [CLI](reference/cli.md)
- [Configuration schema](reference/configuration-schema.md)
- [Grammar catalog](reference/grammar-catalog.md)
- [Manifest schema v2](reference/manifest-schema-v2.md)
- [Compact node types schema](reference/compact-node-types-schema.md)
- [Provider coverage metadata](reference/provider-grammar-coverage.md)
- [Provider language-version research](reference/provider-language-version-research.md)
- [Language support verification workflow](reference/VERIFY_VERSIONS.md)
- [Makefile development targets](reference/makefile.md)

## Explanation

Read these pages to understand the design and its limits:

- [Architecture and pipeline](explanation/architecture-and-pipeline.md)
- [Dual-mode validation](explanation/dual-mode-validation.md)
- [ABI evolution and runtime compatibility](explanation/abi-evolution-and-runtime-compatibility.md)
- [Provenance and reproducibility](explanation/reproducible-builds-and-provenance.md)
- [Compact node types design](explanation/compact-node-types-and-token-efficiency.md)

## Configuration and release context

The normal default is a per-user configuration. A current-directory `.env` containing `TS_BUILD_ENV=DEVELOPMENT` selects the repository configuration instead. An explicit `--config` overrides both.

Catalog replacement requires supported atomic exchange operations. Windows replacement is not implemented. Foreign static validation does not supply the measured ABI required by a normal manifest.

See the root [README](../README.md), [CHANGELOG](../CHANGELOG.md), and [NOTICE](../NOTICE.md) for release and license information.
