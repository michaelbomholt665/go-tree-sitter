# go-tree-sitter Documentation

Welcome to the documentation for **go-tree-sitter**, a reproducible Tree-sitter grammar builder, compiler, and release toolchain for Go.

This documentation is structured according to the [Diátaxis framework](https://diataxis.fr/), separating content into four distinct modes of documentation based on user needs:

```
                  PRACTICAL STEPS
                        ▲
                        │
       Tutorials        │       How-To Guides
   (Learning-oriented)  │     (Problem-oriented)
                        │
◄───────────────────────┼───────────────────────►
  MOST USEFUL TO BEGINNERS       MOST USEFUL TO EXPERTS
                        │
      Explanation       │         Reference
 (Understanding-oriented)│   (Information-oriented)
                        │
                        ▼
                 THEORETICAL KNOWLEDGE
```

---

## 🟢 Tutorials (Learning-Oriented)

*Step-by-step lessons designed to take newcomers through a complete, hands-on experience.*

- [Getting Started with go-tree-sitter](file:///home/michael/projects/go/tree-sitter/docs/tutorials/getting-started.md): Set up required tools, build a grammar, compile a native library, and parse your first code snippet in Go.

---

## 🟡 How-To Guides (Problem-Oriented)

*Practical recipes and step-by-step procedures to solve specific tasks.*

- [How to Build and Release Grammars](file:///home/michael/projects/go/tree-sitter/docs/how-to/build-and-release-grammars.md): Build individual or all grammars, select asset preservation modes, prune build caches, and publish release catalogs.
- [How to Cross-Compile for Windows and macOS](file:///home/michael/projects/go/tree-sitter/docs/how-to/cross-compile-grammars.md): Set up toolchains (MinGW, Apple Clang, Zig fallback) to compile and validate foreign binaries from Linux.
- [How to Compile WebAssembly (WASM) Parsers](file:///home/michael/projects/go/tree-sitter/docs/how-to/compile-webassembly-parsers.md): Compile WebAssembly parser artifacts using native Emscripten (`emcc`) or automated Docker/Podman wrappers.
- [How to Generate and Validate Compact Node Types](file:///home/michael/projects/go/tree-sitter/docs/how-to/generate-and-validate-compact-node-types.md): Generate token-efficient YAML schemas from `node-types.json`, audit for lossless integrity, and publish with releases.
- [How to Add a New Grammar Language](file:///home/michael/projects/go/tree-sitter/docs/how-to/add-a-new-grammar.md): Configure, wire, and validate a new grammar repository in the build system.
- [How to Synchronize Versions with go.mod](file:///home/michael/projects/go/tree-sitter/docs/how-to/sync-versions-with-go-mod.md): Keep grammar versions in sync with Go module dependencies using `sync-config`.
- [How to Consume and Parse in Go Applications](file:///home/michael/projects/go/tree-sitter/docs/how-to/use-compiled-grammars-in-go.md): Load static and dynamic grammars, parse source files, run Tree-sitter query expressions, and consume AST schemas.

---

## 🔵 Reference (Information-Oriented)

*Authoritative technical descriptions, command flags, configuration schemas, and data specifications.*

- [CLI Command Reference (`ts-build`)](file:///home/michael/projects/go/tree-sitter/docs/reference/cli.md): Command flags, subcommands (`wizard`, `build`, `compile`, `move`, `clean`, `compact`), and environment variables.
- [Configuration Schema Reference (`tree-sitter-config.yaml`)](file:///home/michael/projects/go/tree-sitter/docs/reference/configuration-schema.md): Complete specification of keys, validation rules, build pruning, and templates in `tree-sitter-config.yaml`.
- [Compact Node Types Schema Reference](file:///home/michael/projects/go/tree-sitter/docs/reference/compact-node-types-schema.md): Field specifications, typing syntax (`?`, `[...]`, `|`), supertypes, and lossless validation contracts.
- [Manifest Schema v2 Specification](file:///home/michael/projects/go/tree-sitter/docs/reference/manifest-schema-v2.md): Field contract, artifact flags (`has_compact_node_types`, `has_wasm`, `has_c_source`, etc.), and JSON schema.
- [Supported Grammar Catalog & Symbol Registry](file:///home/michael/projects/go/tree-sitter/docs/reference/grammar-catalog.md): Complete list of all 38 configured grammars, repos, revisions, constructors, and C symbols.
- [Makefile Target Reference](file:///home/michael/projects/go/tree-sitter/docs/reference/makefile.md): Reference guide for all build, test, and release targets in `Makefile`.

---

## 🟣 Explanation (Understanding-Oriented)

*Discussions, architectural rationales, and deep dives into core design decisions.*

- [Architecture and Pipeline Design](file:///home/michael/projects/go/tree-sitter/docs/explanation/architecture-and-pipeline.md): The 3-phase pipeline (`build`, `compile`, `move`), atomic directory swaps, rollback mechanisms, and internal subsystem wiring.
- [Reproducible Builds, Provenance, and Determinism](file:///home/michael/projects/go/tree-sitter/docs/explanation/reproducible-builds-and-provenance.md): Commit pinning, `SOURCE_DATE_EPOCH`, provenance chains, and why manifest-only generation is prohibited.
- [Compact Node Types and Token Efficiency](file:///home/michael/projects/go/tree-sitter/docs/explanation/compact-node-types-and-token-efficiency.md): Solving the LLM context window problem, 60%–80% compression theory, and bidirectional lossless reconciliation.
- [Dual-Mode Validation Mechanics](file:///home/michael/projects/go/tree-sitter/docs/explanation/dual-mode-validation.md): Dynamic runtime loading (`dlopen`) vs. static header inspection (ELF, Mach-O, PE).
- [Tree-sitter ABI & Runtime Compatibility](file:///home/michael/projects/go/tree-sitter/docs/explanation/abi-evolution-and-runtime-compatibility.md): Understanding ABI versions, generator flags, and Go runtime compatibility boundaries.
