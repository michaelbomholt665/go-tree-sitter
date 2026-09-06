# How to Cross-Compile for Windows and macOS

This guide explains how to compile and validate Windows (`.dll`) and macOS (`.dylib`) Tree-sitter binaries from a Linux host environment.

---

## 1. Toolchain Prerequisites

Cross-compilation requires either platform-specific cross-compilers or [Zig](https://ziglang.org/) installed on your `$PATH`.

### Option A: Zig (Recommended, Zero Configuration)

If `zig` is installed on `$PATH`, the compiler automatically detects it and creates temporary `cc-wrapper` and `cxx-wrapper` scripts targeting the requested foreign triple:
- Linux amd64: `x86_64-linux-gnu`
- Windows amd64: `x86_64-windows-gnu`
- macOS amd64: `x86_64-macos`
- macOS arm64: `aarch64-macos`

To install Zig:
```bash
# Example via package manager or direct download:
curl -LO https://ziglang.org/download/0.13.0/zig-linux-x86_64-0.13.0.tar.xz
tar -xf zig-linux-x86_64-0.13.0.tar.xz
export PATH="$PWD/zig-linux-x86_64-0.13.0:$PATH"
zig version
```

### Option B: Dedicated Cross-Compilers

If you prefer native toolchains:
- **Windows amd64**: `x86_64-w64-mingw32-gcc` and `x86_64-w64-mingw32-g++` (Ubuntu: `apt install mingw-w64`).
- **macOS amd64 & arm64**: osxcross toolchain with `o64-clang` and `oa64-clang`.

---

## 2. Compiling Foreign Targets

### Windows (amd64)

Compile grammar sources into Windows `.dll` shared libraries:

```bash
# Using Makefile
make compile-windows

# Or using the ts-build CLI directly
go run ./cmd/ts-build compile --os windows --arch amd64
```

To compile a single language:
```bash
go run ./cmd/ts-build compile --os windows --arch amd64 --language python
```

Output:
```text
[✓] compile   python [windows/amd64]         (1.1s)
```

Generated binary:
- `build/python/amd64/python-v0.25.0-windows-amd64.dll`
- `build/python/amd64/python-v0.25.0-windows-amd64.dll.provenance.json`

---

### macOS (Apple Silicon arm64 & Intel amd64)

Compile for both macOS architectures:

```bash
# Using Makefile (compiles both amd64 and arm64)
make compile-mac

# Or using ts-build CLI
go run ./cmd/ts-build compile --os macos --arch arm64
go run ./cmd/ts-build compile --os macos --arch amd64
```

Output:
```text
[✓] compile   python [macos/arm64]           (0.9s)
[✓] compile   python [macos/amd64]           (0.9s)
```

Generated binaries:
- `build/python/arm64/python-v0.25.0-macos-arm64.dylib`
- `build/python/amd64/python-v0.25.0-macos-amd64.dylib`

---

## 3. Full Multi-Platform Release Pipeline

To build sources once, compile for all platforms, validate every native binary, and publish the unified catalog:

```bash
make release-all
```

This target runs:
```bash
# 1. Generate grammar sources
go run ./cmd/ts-build build --config tree-sitter-config.yaml

# 2. Compile all targets
go run ./cmd/ts-build compile --config tree-sitter-config.yaml --os linux --arch amd64
go run ./cmd/ts-build compile --config tree-sitter-config.yaml --os windows --arch amd64
go run ./cmd/ts-build compile --config tree-sitter-config.yaml --os macos --arch amd64
go run ./cmd/ts-build compile --config tree-sitter-config.yaml --os macos --arch arm64

# 3. Validate binaries (dynamic for host, static for foreign) and publish
go run ./cmd/ts-build move --config tree-sitter-config.yaml --force
```

---

## 4. How Cross-Validation Works

When running `move` on a Linux host with Windows `.dll` and macOS `.dylib` binaries present:

1. **Host Binaries (Linux ELF)**:
   - Dynamically loaded via `dlopen`.
   - Executes `tree_sitter_<lang>()` constructor.
   - Parses the language's minimal sample string with `go-tree-sitter`.
2. **Foreign Binaries (Windows PE & macOS Mach-O)**:
   - Evaluated without `dlopen` using Go's standard library `debug/pe` and `debug/macho` packages.
   - Inspects file headers, target CPU architecture (`amd64`, `arm64`), section tables, and dynamic symbol export tables.
   - Asserts that the sole exported constructor matches `tree_sitter_<grammar>`.
   - Reconciles measured provenance ABI against the generator manifest.

---

## 5. Troubleshooting Cross-Compilation

### Missing Compiler Toolchain Error
```text
compile python for windows/amd64: no compiler toolchain available for windows/amd64; install x86_64-w64-mingw32-gcc, zig, or a target-specific cross-compiler
```
**Resolution:** Install `mingw-w64` or add `zig` to your `$PATH`.

### `dlopen` Warning during Cross-Compilation
During `tree-sitter build`, the upstream CLI may print:
```text
Error opening dynamic library: ... cannot open shared object file
```
The compiler automatically detects that this error comes from foreign binary execution on the host and suppresses it as long as a valid non-empty shared library was created.
