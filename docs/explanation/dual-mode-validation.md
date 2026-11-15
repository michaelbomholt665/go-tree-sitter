# Dual-mode validation

A host binary can execute on the current machine. A foreign binary cannot. These conditions require different checks and produce different evidence.

## What each mode proves

| Check | Host dynamic validation | Foreign static validation |
| --- | --- | --- |
| Binary format and expected constructor | Checked during validation/loading | Checked without execution |
| Parser ABI measured from binary | Yes | No |
| Runtime accepts the language | Yes | Not tested |
| Configured sample parses without errors | Yes | Not tested |
| Queries compile against the language | Yes, for available queries | Not tested by loading |
| Normal measured-ABI manifest eligibility | Yes, after all required checks pass | No |

Static validation is not a substitute for executing a foreign parser. Provenance can describe the requested generation ABI, but that number is not a measurement of the resulting library.

## Host validation

The validator loads the shared library and resolves the configured constructor. It reads the returned language's ABI and compares it with the configured and runtime bounds.

The validator then parses the configured sample and validates available query files. A successful sample proves only that sample. It does not establish complete support for a source-language release.

Library loading also depends on the host loader, architecture, and shared dependencies.

## Foreign validation

The validator inspects ELF, PE, or Mach-O structure and exported symbols. Cross-validation requires explicit permission through the CLI or runtime publication policy.

The validator does not add a foreign artifact to the measured ABI map. A configured ABI, generated header, or provenance sidecar cannot supply the missing measurement.

If normal manifest generation encounters an unmeasured native binary, publication fails. `--allow-cross-validation` does not bypass this requirement.

## Release implications

For downstream manifests, build and validate each target on a compatible host. A Windows runner validates Windows artifacts; a macOS runner validates macOS artifacts.

If a consumer deliberately needs raw artifacts without measured manifests, the CLI supports `--manifest=false`. This removes downstream metadata; it does not convert static checks into execution evidence.

Do not copy foreign artifacts into a manifest and label their ABI as measured.

## Validation limits

Checksums detect artifact changes, not malicious provenance replacement. Parsing samples and compiling queries do not prove correctness for every program.

Native Windows and macOS runtime behavior has not yet been verified for this release. Helper cross-builds and mocked installer tests do not provide that evidence.

See [Cross-compile parsers](../how-to/cross-compile-grammars.md) and [ABI compatibility](abi-evolution-and-runtime-compatibility.md).
