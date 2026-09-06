# Dual-Mode Validation Mechanics

This document explains the technical architecture of **Dual-Mode Validation** in `go-tree-sitter`, implemented in [`internal/tree-sitter/build/validator.go`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/validator.go).

---

## 1. Why Native Validation is Critical

A shared library can compile without syntax errors and still be completely unusable at runtime:
- Missing or misnamed export symbols (e.g. `_tree_sitter_foo` instead of `tree_sitter_foo`).
- Multiple conflicting constructors exported from linked sub-grammars.
- Corrupted parser state tables that crash upon encountering valid syntax.
- Query files (`.scm`) referencing AST node types or field names that do not exist in the grammar.
- Parser ABI versions incompatible with the Go runtime binding.

To prevent any broken artifact from ever being published, [`NativeValidator`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/validator.go#L40) validates all staged libraries before the catalog directory is updated.

---

## 2. The Cross-Platform Dilemma

When building on a Linux host:
- Linux `.so` files **can** be dynamically loaded into the Go test process via `dlopen`.
- Windows `.dll` and macOS `.dylib` files **cannot** be loaded via `dlopen`, because the Linux kernel cannot execute foreign machine code.

Many toolchains solve this by running QEMU or skipping validation for cross-compiled targets entirely. `go-tree-sitter` takes a different approach: **Dual-Mode Validation**.

```
                           Target Binary
                                │
                 Is Host Platform & Architecture?
                                │
                 ┌──────────────┴──────────────┐
                 ▼                             ▼
              [ YES ]                       [ NO ]
         Host Dynamic Mode           Cross Static Mode
                 │                             │
    - validateBinaryFormat()      - validateBinaryFormat()
    - dlopen() / LoadLibrary()      (ELF / Mach-O / PE headers)
    - Invoke constructor symbol   - Parse Export Directory Table
    - Language.AbiVersion()       - Reconcile provenance ABI
    - Parser.SetLanguage()        - Validate node-types schema
    - Parse minimal sample string - Read & inspect .scm files
    - sitter.NewQuery() compile
```

---

## 3. Host Dynamic Validation (Deep Runtime Verification)

When the binary matches the host operating system and architecture, the validator executes deep dynamic verification:

### 1. Dynamic Symbol Inspection
Before loading, the binary is inspected using Go standard library debug packages to ensure it exports **exactly one** constructor matching `tree_sitter_<grammar>`:

```go
// validator.go
func validateConstructorSymbols(path, expected string, symbols []string) error
```

If multiple constructors are detected (e.g. when bundling multiple parsers into one object), the validation fails immediately.

### 2. Live Runtime Loading
The library is loaded into memory using platform-specific dynamic loaders ([`native_loader_unix.go`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/native_loader_unix.go) or [`native_loader_windows.go`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/native_loader_windows.go)).

The validator:
1. Calls the C constructor function to obtain the unsafe language pointer:
   ```go
   library, pointer, err := openNativeLanguage(item.BinaryPath, item.Language.Constructor)
   ```
2. Wraps it into a Go [`*sitter.Language`](file:///home/michael/projects/go/tree-sitter/internal/grammars/registry.go#L28).
3. Reads the measured ABI:
   ```go
   abi := language.AbiVersion()
   ```
4. Asserts that `abi` satisfies the runtime boundaries:
   ```go
   sitter.MIN_COMPATIBLE_LANGUAGE_VERSION <= abi <= sitter.LANGUAGE_VERSION
   ```

### 3. Syntax Tree Smoke Testing
The validator assigns the language to a new parser and parses the language's minimal code snippet (`lang.Sample`):
```go
tree := parser.Parse([]byte(item.Sample), nil)
if tree.RootNode().HasError() {
    return errors.New("parse minimal sample: syntax tree contains errors")
}
```
If the grammar table is corrupt or fails to parse even its canonical smoke test, the build is rejected.

### 4. Query Compilation
Every shipped `.scm` query file under `queries/` is compiled against the live language instance:
```go
query, queryErr := sitter.NewQuery(language, string(querySource))
```
Tree-sitter evaluates every pattern in the query. If a query references a node type or field not defined in the grammar, `sitter.NewQuery` fails and reports the exact line and column number of the error.

---

## 4. Cross-Platform Static Validation (Binary Introspection)

When validating foreign targets (e.g. Windows `.dll` on Linux), the validator performs deep static analysis of container file structures without executing machine code:

### 1. Windows Portable Executable (PE) Inspection
Using `debug/pe`:
- Checks machine type (`IMAGE_FILE_MACHINE_AMD64` or `IMAGE_FILE_MACHINE_ARM64`).
- Reads the **PE Export Directory Table** directly from section headers:
  ```go
  // validator.go
  func readPEExportNames(file *pe.File) ([]string, error)
  ```
- Reads the Relative Virtual Addresses (RVA) and parses null-terminated export strings to confirm that `tree_sitter_<grammar>` is exported.

### 2. macOS Mach-O Inspection
Using `debug/macho`:
- Validates CPU architecture (`CpuAmd64` or `CpuArm64`).
- Confirms file type is `TypeDylib`.
- Scans `file.Symtab.Syms` to verify presence of `_tree_sitter_<grammar>` or `tree_sitter_<grammar>`.

### 3. Linux ELF Inspection
Using `debug/elf`:
- Confirms machine architecture (`EM_X86_64` or `EM_AARCH64`).
- Scans dynamic symbol tables via `file.DynamicSymbols()`.

---

## 5. Summary Table

| Check | Host Dynamic Mode | Cross Static Mode |
| :--- | :--- | :--- |
| File Container Format | Verified (ELF/Mach-O/PE) | Verified (ELF/Mach-O/PE) |
| Architecture Match | Verified | Verified |
| Constructor Symbol Export | Verified via symbol table | Verified via symbol / export table |
| Live `dlopen` Execution | **Yes** | No |
| Measured `AbiVersion()` | **Live runtime call** | Reconciled from provenance chain |
| Smoke Test Parse | **Yes (AST inspected)** | Schema check on `node-types.json` |
| Query Syntax Compilation | **Yes (`sitter.NewQuery`)** | Verified query file readability |
