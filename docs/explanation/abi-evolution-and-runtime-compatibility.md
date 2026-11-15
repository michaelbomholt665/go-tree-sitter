# Parser ABI and runtime compatibility

The parser ABI is a Tree-sitter interface version. It is not a grammar package version or a source-language version.

A native parser contains one ABI. A runtime accepts a range of ABIs. Compatibility requires the measured parser ABI to fall within that runtime's supported range.

## Generation policy versus binary measurement

The configuration distinguishes a requested generation ABI from a supported range:

```yaml
# tree-sitter-config.yaml (excerpt)
abi_range:
  min: 13
  max: 15
generate_abi: 15
```

The builder passes the requested ABI to `tree-sitter generate`. It checks the generated `LANGUAGE_VERSION` against policy.

If generation fails, runtime policy can permit attempts with lower ABIs down to the configured minimum. An explicit per-language `generate_abi` prevents fallback.

The generated macro and provenance describe source generation. Only loading a native binary and reading the language's ABI supplies the measurement used by a normal manifest.

## Runtime checks

The current Go Tree-sitter runtime supports ABI 13 through 15. Host validation rejects a binary outside the supported runtime and configured bounds.

An incompatible ABI causes language acceptance to fail. Do not assume that an arbitrary language pointer is safe to use merely because a header names an ABI.

ABI acceptance also does not prove grammar correctness. The configured sample and available queries provide additional, bounded checks.

## Manifest schema v2

Schema v2 records `parser_abi` as one measured integer. The compatibility `abi` object uses that same integer for both minimum and maximum.

A consumer compares the artifact's ABI with its own runtime bounds. The consumer must not treat the artifact's `abi` object as a runtime compatibility range.

`tree_sitter_version` identifies the generator. The configured grammar `version` identifies an upstream package. Neither field establishes supported source-language features.

## Foreign artifacts

Foreign static validation does not execute the constructor and therefore does not measure the ABI. An expected ABI in provenance remains an expectation.

Normal manifest generation rejects native artifacts without a measured ABI. Run validation on a compatible target host before generating downstream metadata.

Manifest-only regeneration is intentionally blocked because it cannot establish those measurements.

See [Manifest schema v2](../reference/manifest-schema-v2.md), [Dual-mode validation](dual-mode-validation.md), and [Language support research](../reference/VERIFY_VERSIONS.md).
