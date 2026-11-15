# Changelog

## 1.10.0

This release compares the `v1.0.0` Git tag with the current application. The notes use the old and new README, configuration, and implementation changes.

### Added

- Expanded the configured grammar catalog from 38 entries to 65. The old README's count of 42 did not match its tagged configuration.
- Added Blade, Clojure, CUE, Dart, Fish, F#, F# signatures, Gleam, Groovy, HCL, JSON5, Kotlin, Lean, Less, MDX, Nix, Objective-C, PowerShell, Prisma, Sass, SCSS, Solidity, Svelte, Swift, Typst, Vue, and Zsh entries.
- Added `list` and `support show` for inspecting grammar identities and revision-specific source-language assessments.
- Added guarded `support update` previews and writes, with revision guards and optional local report checksum checks.
- Added `config path`, `show`, `get`, `set`, `edit`, and `validate` for configuration inspection and non-language settings.
- Added configurable runtime policy for tools, compiler fallback, Node installation, generation, cleanup, publication, queries, WASM, and wizard defaults.
- Added per-user configuration initialization from the embedded catalog and current-directory development selection through `.env`.
- Added the installed `sync-config` command and configurable module-to-grammar mappings.
- Added `export` for copying selected published grammar directories to external consumers.
- Added the Windows tool installer, repository setup, isolated Python-only configuration, and one-command-at-a-time VM test guide.
- Added the `skills/ts-grammar-build` agent skill with configuration, build, publication, and grammar catalog references.

### Changed

- Renamed the C# catalog identifier from `c-sharp` to `csharp`.
- Updated grammar package labels and pinned source revisions, including Lua's upstream source selection.
- Strengthened compact AST schemas to preserve named/anonymous identities, operator alternatives, requiredness, multiplicity, child constraints, subtypes, root metadata, and extra metadata.
- Made unsupported schema properties fail instead of silently losing information.
- Documented installed `ts-build` commands as the normal interface instead of source-only commands and Makefile runtime shortcuts.
- Reorganized tutorials, task guides, and explanations around configuration selection, isolated builds, actual publication guarantees, and validation limits.

### Fixed

- Serialized catalog staging and publication so concurrent publishers do not lose each other's updates.
- Replaced the two-rename publication gap with supported atomic directory exchange. Unsupported replacement preserves the current catalog.
- Preserved the last release during recovery and preserved existing workspaces after failed checkout unless `build --force` authorizes removal.
- Required `move --force` before replacing existing selected grammar output.
- Added cancellation checks before publication and cleanup, descendant-process cancellation, bounded command capture, and bounded process wait behavior.
- Prevented stale compilation output from receiving fresh provenance. Compilation uses temporary output, verifies generated source hashes, and records binary checksums.
- Verified artifact checksums and source identities before publication.
- Stopped treating configured or provenance ABIs as foreign binary measurements. Normal manifests reject native artifacts without measured ABIs.
- Hardened Windows npm execution and case-insensitive environment merging. Unsupported batch and compiler shell-wrapper paths fail explicitly.
- Rejected unexpected positional arguments, including `clean python`, rather than silently selecting the whole catalog.
- Rejected unsupported configuration versions and unpinned source revisions.
- Made configuration writes lock-protected, conflict-checked replacements. Module parsing now accepts space-indented require blocks.
- Added noninteractive winget source-agreement handling to prevent a hidden first-run prompt during installer preflight.
- Corrected documentation that used Go's plugin loader for plain C parser libraries or exposed the repository's internal binding registry to external consumers.

### Migration and known limits

- Update scripts and catalog paths that use `c-sharp` to use `csharp`. Existing published directories are not automatically renamed.
- Existing per-user configurations remain unchanged during upgrades. Merge the new catalog and runtime settings deliberately.
- Use full, 40-character source revisions and configuration schema `2.0`.
- Match the installed generator and compiler versions to the selected pins. The repository currently pins Tree-sitter CLI `0.27.0`.
- Rebuild old intermediate artifacts if they lack the required source hashes or binary checksum provenance.
- `move --force` replaces the complete selected grammar directory. Include every artifact you intend to retain.
- Existing catalog replacement requires a supported Linux or macOS atomic exchange. Windows replacement is not implemented; use a fresh release destination.
- Foreign static validation cannot supply the measured ABI required by a normal manifest. Validate on a compatible host, or deliberately publish without the custom manifest.
- Native Windows and macOS runtime behavior remains unverified. Cross-builds and mocked installer tests do not prove native execution.
- The optional Cypher Go binding has an unresolved `go-licenses` classification. Go reports do not verify all native grammar licenses at their configured revisions; see [NOTICE.md](NOTICE.md).

## 1.0.0

The tagged baseline already provided the build/compile/move pipeline, interactive wizard, native and WASM compilation, manifests, compact schemas, provenance, and Makefile helpers. These are not new features of 1.10.0.
