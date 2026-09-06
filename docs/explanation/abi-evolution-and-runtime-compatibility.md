# Tree-sitter ABI & Runtime Compatibility

This document explains the Tree-sitter ABI (Application Binary Interface), how parser versioning works across generations, and how `go-tree-sitter` enforces compatibility.

---

## 1. What is the Tree-sitter ABI?

When `tree-sitter generate` converts a grammar's `grammar.js` into C source code (`src/parser.c`), it defines an internal numerical macro:

```c
// parser.c
#define LANGUAGE_VERSION 15
```

When compiled, the grammar constructor `tree_sitter_<grammar>()` returns a pointer to a `TSLanguage` struct:

```c
const TSLanguage *tree_sitter_c(void);
```

The memory layout of `TSLanguage` (its fields, state table offsets, lexical scanner callbacks, and node index widths) depends directly on `LANGUAGE_VERSION`.

If a parser binary generated with ABI 15 is loaded by an older Tree-sitter runtime that only understands ABI 13 or 14, the runtime will read memory at incorrect struct offsets, causing invalid parse trees or segmentation faults.

---

## 2. Runtime Compatibility Policy

The Go binding used by `go-tree-sitter` is [`github.com/tree-sitter/go-tree-sitter v0.25.0`](file:///home/michael/projects/go/tree-sitter/go.mod#L11).

In this binding:
- `sitter.LANGUAGE_VERSION` = `15` (the current highest supported ABI).
- `sitter.MIN_COMPATIBLE_LANGUAGE_VERSION` = `13` (the lowest supported legacy ABI).

When [`validateRuntimeABI`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/validator.go#L164) tests a binary:

```go
func validateRuntimeABI(abi, minimum, maximum uint32) error {
    if abi < minimum || abi > maximum {
        return fmt.Errorf("parser ABI %d is outside runtime range %d-%d", abi, minimum, maximum)
    }
    return nil
}
```

It ensures that any compiled parser will be fully usable by the Go runtime without undefined memory behavior.

---

## 3. ABI Range Support & Flexible Code Generation

In older toolchains, `tree-sitter generate` chose an ABI automatically based on whatever Tree-sitter CLI version happened to be installed, or hardcoded a single ABI version that broke older grammars.

In `go-tree-sitter` (configuration version 2.0):
- The supported runtime range is explicitly declared in [tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml):
  ```yaml
  abi_range:
    min: 13
    max: 15
  generate_abi: 15
  ```
- During code generation, the builder passes the configured `generate_abi` (default: 15, or language override).
- If a grammar's upstream syntax was designed for an earlier ABI (e.g. 14 or 13), the builder automatically falls back through the supported range (`14` then `13`).
- The builder then scans `src/parser.c` via [`detectGeneratedABI`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/builder.go#L416) to confirm the emitted `#define LANGUAGE_VERSION <n>` falls within `[13, 15]`.
- Because ABI 15 is backward-compatible with ABI 14 and 13, any grammar generated in this range runs correctly under the Go runtime.

---

## 4. The Shift from Schema v1 to Schema v2

### The Flaw in Schema v1
Schema v1 attempted to describe compatibility by assigning a `min_version` and `max_version` to each binary in `manifest.json`:

```json
// Flawed Schema v1 approach
"abi": {
  "min_version": 13,
  "max_version": 14,
  "parser_version": "0.20.8"
}
```

This model was conceptually flawed:
1. **A binary has only ONE ABI**: A compiled native library has a single, fixed struct layout determined when it was compiled (e.g., ABI 15). It does not have a "range" of ABIs.
2. **Compatibility is a property of the consumer, not the artifact**: Whether ABI 15 can be executed is determined by the Tree-sitter C runtime linked into the consuming application, not by the binary on disk.
3. **`parser_version` ambiguity**: In v1, this field was often populated with the upstream language package version rather than the code generator tool version.

### The Schema v2 Solution
Schema v2 redesigns metadata around measured reality:

1. **`parser_abi` (Scalar uint32)**:
   Records the exact ABI integer returned when `Language.AbiVersion()` is executed on the native binary.
2. **`tree_sitter_version`**:
   Records the exact version output by the `tree-sitter --version` CLI executable.
3. **Decoupled Consumer Policy**:
   Consumers reading `manifest.json` inspect `parser_abi` and compare it against their own runtime's supported bounds:
   ```go
   if manifest.ParserABI > myRuntimeMax || manifest.ParserABI < myRuntimeMin {
       // Incompatible runtime
   }
   ```
4. **Strict Compatibility Fallback**:
   For backward compatibility with tools expecting the `abi` object, Schema v2 sets both bounds strictly equal to the measured scalar:
   ```json
   "abi": {
     "min_version": 15,
     "max_version": 15,
     "parser_version": "0.26.8"
   }
   ```
