# Cross-compile grammars

Cross-compilation produces artifacts for another operating system or architecture. It does not prove that those artifacts execute correctly on that target.

## 1. Choose the release contract

| Goal | Required validation |
| --- | --- |
| Publish a normal manifest with measured parser ABIs | Execute validation on a compatible target host |
| Produce foreign artifacts for later target-host validation | Compile and retain artifacts plus provenance |
| Publish raw foreign artifacts without the custom manifest | Explicit static validation and `--manifest=false` |

Do not combine foreign artifacts into a normal manifest by copying the configured generation ABI into metadata.

## 2. Configure the cross toolchain

Inspect target settings:

```bash
ts-build config get targets
ts-build config get runtime.compiler.allow_fallback
```

Set each target's OS, architecture, triple, compiler, and compiler version. If the grammar has a C++ scanner, configure a compatible C++ compiler.

The target compiler must actually support the selected triple. A configured name or a Zig installation alone does not establish a usable cross toolchain.

Use explicit compiler paths and matching version pins for repeatable builds. Windows hosts do not support the current Zig/Emscripten shell-wrapper paths.

## 3. Generate once and compile a foreign target

For example, from Linux with a configured Windows amd64 cross compiler:

```bash
ts-build build --language python
ts-build compile --language python --os windows --arch amd64 --allow-cross-validation
```

Other configured targets use the same selection pattern:

```bash
ts-build compile --language python --os macos --arch arm64 --allow-cross-validation
```

These commands require working cross compilers. They are not toolchain installation commands.

Keep generated sources and provenance until target validation completes.

## 4. Validate releases on their target hosts

For normal downstream manifests, use a runner for each supported target. On each runner, select matching configuration and perform the host pipeline:

```bash
ts-build build --language python
ts-build compile --language python
ts-build move --language python --compact --clean=false
```

Ensure `OS_TARGET` and the target architecture match that runner. Choose a fresh release destination if catalog replacement is unsupported.

A target runner must load the binary and measure its ABI. A cross-build or mocked test does not provide native execution evidence.

## 5. Publish without a manifest only when intentional

WARNING: `--manifest=false` removes measured-ABI downstream metadata. Use it only if the consumer accepts that contract.

For a foreign-only workspace with static validation explicitly allowed:

```bash
ts-build move --language python --allow-cross-validation --manifest=false --clean=false
```

The resulting output is raw distribution material, not a measured-ABI release. Static validation checks binary structure and expected exports; it does not parse samples on the foreign target.

Existing selected grammar output still requires `--force`. Replacement still requires a supported atomic directory exchange.

## Windows setup and testing

Use [TEST-GUIDE.md](../../TEST-GUIDE.md) and the root [PowerShell installer](../../install-windows-tools.ps1) for native Windows development setup.

The optional `-RunGrammarTest` path builds only Python for Windows amd64. It does not run the full Go suite, other grammars, or WASM tests. The script refuses an existing test catalog because Windows replacement is unsupported.

WARNING: The installer's uninstall action removes its listed tools, including tools installed before the script. Inspect the uninstall scope before confirming.

Native Windows and macOS runtime validation remains outstanding for this release.

See [Dual-mode validation](../explanation/dual-mode-validation.md).
