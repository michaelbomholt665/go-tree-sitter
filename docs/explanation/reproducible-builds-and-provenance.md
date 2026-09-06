# Reproducible Builds, Provenance, and Determinism

This document explains the reproducibility strategy, provenance data structures, and deterministic guarantees built into **go-tree-sitter**.

---

## 1. The Challenge of Non-Deterministic Parser Builds

In standard build workflows, compiling identical source code at different times or across different machines often produces differing binaries:
- Embedded compile timestamps (`__DATE__`, `__TIME__`).
- Differing compiler versions or default flag sets.
- Upstream branch head drift or mutable git tags.
- Generation toolchain variation (`tree-sitter` CLI versions).

When building software that relies on native code AST parsers, non-deterministic binaries make supply chain audits impossible and cause cache thrashing in downstream packaging systems.

`go-tree-sitter` implements strict cryptographic provenance tracking and deterministic builds from source checkout to final publication.

---

## 2. Commit Pinning and Source Immutability

Git branches and tags can be modified, deleted, or force-pushed. To eliminate upstream drift:

1. **Explicit 40-Character Commit SHAs**:
   In [tree-sitter-config.yaml](file:///home/michael/projects/go/tree-sitter/tree-sitter-config.yaml), every grammar definition requires a pinned 40-character hexadecimal revision:
   ```yaml
   revision: "b780e47fc780ddc8da13afa35a3f4ed5c157823d"
   ```
   [`validateLanguage`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/config.go#L274) rejects short hashes or branch names.

2. **Detached Head Verification**:
   During `build`, the [`Builder`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/builder.go#L22) checks out the commit and runs `git rev-parse HEAD`. If the checked-out hash does not match `revision:`, the build aborts immediately.

---

## 3. Timestamp Clamping with `SOURCE_DATE_EPOCH`

To prevent compilers and code generators from embedding the current system time:

1. The builder queries git for the commit timestamp of the grammar repository:
   ```bash
   git -C <repo> show -s --format=%ct HEAD
   ```
2. This Unix epoch integer is exported as the standard environment variable `SOURCE_DATE_EPOCH` during both `tree-sitter generate` and C/C++ compilation.
3. GCC, Clang, and modern linkers honor `SOURCE_DATE_EPOCH`, replacing timestamps in PE headers, ELF headers, and object files with the fixed commit timestamp.
4. The `compiled_at` timestamp in `manifest.json` is formatted directly from this epoch:
   ```go
   CompiledAt: time.Unix(compiledEpoch, 0).UTC().Format(time.RFC3339)
   ```

---

## 4. Cryptographic Provenance Chains

Every build step emits cryptographic hashes and environmental metadata that flow into subsequent stages:

```
[Git Repo @ revision]
       │
       ▼
   [build] ───► Emits .source-provenance.json
       │         - source_repository
       │         - source_revision
       │         - generator_version
       │         - generate_abi
       │         - source_date_epoch
       │         - node_types_sha256
       │
       ▼
  [compile] ──► Emits <binary>.provenance.json
       │         - Inherits source provenance
       │         - target_triple
       │         - compiler name & version
       │         - build_flags
       │
       ▼
    [move] ───► Emits manifest.json
                 - Verifies node-types checksum vs provenance
                 - Verifies measured parser ABI vs binary
                 - Computes final binary checksum_sha256
                 - Bundles build_provenance array
```

### Why Node-Types Hash Checking Matters
During `move`, [`stageLanguage`](file:///home/michael/projects/go/tree-sitter/internal/tree-sitter/build/mover.go#L172) hashes the staged `node-types.json` and compares it against `NodeTypesSHA256` recorded in the binary provenance. If the grammar source files were modified or regenerated between compilation and movement, the mismatch is caught and publication is rejected.

---

## 5. Why Manifest-Only Regeneration is Prohibited

Earlier versions of grammar release workflows allowed a command called `regen-manifests` that parsed configuration files and rewrote `manifest.json` on disk without touching native binaries.

In `go-tree-sitter`, this is explicitly disabled:

```go
// cmd/regen-manifests/main.go
func main() {
    fmt.Fprintln(os.Stderr, "manifest-only regeneration is unsafe and no longer supported; run tree-sitter build, compile, and move so the final native binaries are validated")
    os.Exit(2)
}
```

### The Rationale
A manifest is a cryptographic claim about the properties of binary files. If a tool rewrites `manifest.json` based on assumptions in a YAML file without:
1. Loading the native binary into memory,
2. Checking its dynamic symbols,
3. Invoking `Language.AbiVersion()`, and
4. Hashing the exact final bytes on disk,

the manifest risks certifying a corrupted or incompatible parser. In `go-tree-sitter`, metadata is never decoupled from the verified native binary.
