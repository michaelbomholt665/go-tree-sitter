# Provider grammar language-version research

Reference for the 23 grammar identifiers added for the provider catalog.
Research date: 2026-11-15.
Policy: [VERIFY_VERSIONS.md](VERIFY_VERSIONS.md), Rule 2.

## Result

No new supported-version bounds were established.
The existing limited Lean 4 assessment remains verified through documentation.
The other 22 identifiers retain `claim: unverified`.
Missing minimum and maximum versions mean unknown bounds, not unlimited support.

Kotlin documents a **2.4.21 fixture target**, but that is not complete language-version support.
Its README and archived fixture report disagree on coverage counts.
PowerShell's 7.3 reference and F#'s 4.1 specification basis are not compatibility guarantees.

No version fixtures were executed.
No `syntax_fixtures` evidence was added.
No build/compile/move pipeline was run for this research; configured grammar pins and published artifacts remain unchanged.

## Assessed revisions and findings

The full SHA in each row is both the active `revision` and `language_support.assessed_revision`.
Each README link points to that SHA, not current HEAD.
Every existing minimum and maximum bound remains absent.

| Grammar | Active and assessed revision | Claim | Documentation and limitation |
| --- | --- | --- | --- |
| `powershell` | `65ea6919e816a7ac142acec7ee72757f89db8439` | unverified | [README](https://github.com/airbus-cert/tree-sitter-powershell/blob/65ea6919e816a7ac142acec7ee72757f89db8439/README.md): Reference to a PowerShell 7.3 specification page, not a compatibility claim. |
| `fsharp` | `aefd0c8741bdf3aeb827a228aa4a996a6536697e` | unverified | [README](https://github.com/ionide/tree-sitter-fsharp/blob/aefd0c8741bdf3aeb827a228aa4a996a6536697e/README.md): Based on F# specification 4.1, mostly Appendix A; no supported-version range. |
| `fsharp-signature` | `aefd0c8741bdf3aeb827a228aa4a996a6536697e` | unverified | [README](https://github.com/ionide/tree-sitter-fsharp/blob/aefd0c8741bdf3aeb827a228aa4a996a6536697e/README.md): Shares the F# specification 4.1 reference; no separate signature-version guarantee. |
| `kotlin` | `d6af96d2e23ef0a8ff5ab512864378cf08fba311` | unverified | [README](https://github.com/fwcd/tree-sitter-kotlin/blob/d6af96d2e23ef0a8ff5ab512864378cf08fba311/README.md): Kotlin 2.4.21 fixture target is explicit; README and archived report disagree. See the detailed finding. |
| `groovy` | `2a6ddd558b6aa39c5b77d8db9fe9baf817486b2c` | unverified | [README](https://github.com/murtaza64/tree-sitter-groovy/blob/2a6ddd558b6aa39c5b77d8db9fe9baf817486b2c/README.md): README claims most Groovy features and lists features, but gives no language-version bounds. |
| `lean` | `0d28e5d6e8c4755f5df7ef9bde3b6ec6f59fae94` | verified, limited Lean 4 scope | [README](https://github.com/Julian/tree-sitter-lean/blob/0d28e5d6e8c4755f5df7ef9bde3b6ec6f59fae94/README.md): Explicit Lean 4 static-syntax target with dynamic notation, macro, tactic, and structure-field limitations. |
| `swift` | `82bb3a533e0801fd2bbaa11dc49676e10bf41948` | unverified | [README](https://github.com/alex-pinkus/tree-sitter-swift/blob/82bb3a533e0801fd2bbaa11dc49676e10bf41948/README.md): README describes Swift parsing and package/runtime setup, not supported Swift versions. Live badges are not pinned evidence. |
| `dart` | `0038280b774b65c1b89ba7fec50624d8baed9fe1` | unverified | [README](https://github.com/UserNobody14/tree-sitter-dart/blob/0038280b774b65c1b89ba7fec50624d8baed9fe1/README.md): README gives no supported Dart version. The nested changelog concerns runtime bindings, not Dart syntax compatibility. |
| `gleam` | `fa6d0d94804f1b342e175865a3dbeb316ee5115e` | unverified | [README](https://github.com/gleam-lang/tree-sitter-gleam/blob/fa6d0d94804f1b342e175865a3dbeb316ee5115e/README.md): README gives no supported Gleam version. |
| `typst` | `46cf4ded12ee974a70bf8457263b67ad7ee0379d` | unverified | [README](https://github.com/uben0/tree-sitter-typst/blob/46cf4ded12ee974a70bf8457263b67ad7ee0379d/README.md): README and DOC.md give no supported Typst versions; DOC.md documents parsing limitations. |
| `clojure` | `e43eff80d17cf34852dcd92ca5e6986d23a7040f` | unverified | [README](https://github.com/sogaiu/tree-sitter-clojure/blob/e43eff80d17cf34852dcd92ca5e6986d23a7040f/README.md): Targets Clojure and ClojureScript. Scope and limits documents do not establish supported versions. |
| `cue` | `eefee1c1ee76ccf0d7880814155b67c1738d5f89` | unverified | [README](https://github.com/eonpatapon/tree-sitter-cue/blob/eefee1c1ee76ccf0d7880814155b67c1738d5f89/README.md): README gives no supported CUE version; development tool versions are not CUE compatibility evidence. |
| `hcl` | `64ad62785d442eb4d45df3a1764962dafd5bc98b` | unverified | [README](https://github.com/tree-sitter-grammars/tree-sitter-hcl/blob/64ad62785d442eb4d45df3a1764962dafd5bc98b/README.md): README links the HCL specification and reports corpus parses, but gives no supported HCL or Terraform version. |
| `nix` | `64d4f7827dc1a8c7c88108ffdb47b961a729df05` | unverified | [README](https://github.com/cstrahan/tree-sitter-nix/blob/64d4f7827dc1a8c7c88108ffdb47b961a729df05/README.md): README identifies Nix without supported-version bounds. |
| `zsh` | `7a593401efb5418ffdedbe3c0e4c61c6d240166d` | unverified | [README](https://github.com/georgeharker/tree-sitter-zsh/blob/7a593401efb5418ffdedbe3c0e4c61c6d240166d/README.md): README identifies Zsh and distinguishes the fork; no supported Zsh version is specified. |
| `fish` | `b7f1d682941e0c62dfcd6bf9ef481638351bab16` | unverified | [README](https://github.com/ram02z/tree-sitter-fish/blob/b7f1d682941e0c62dfcd6bf9ef481638351bab16/README.md): README gives no supported fish language version. Grammar package tags are not language-version claims. |
| `prisma` | `3556b2c1f20ec9ac91e92d32c43d9d2a0ca3cc49` | unverified | [README](https://github.com/victorhqc/tree-sitter-prisma/blob/3556b2c1f20ec9ac91e92d32c43d9d2a0ca3cc49/README.md): README describes an unofficial Prisma parser and links specifications, but gives no supported Prisma version. |
| `vue` | `ce8011a414fdf8091f4e4071752efc376f4afb08` | unverified | [README](https://github.com/tree-sitter-grammars/tree-sitter-vue/blob/ce8011a414fdf8091f4e4071752efc376f4afb08/README.md): README identifies Vue files without a Vue 2/3 compatibility claim. |
| `svelte` | `ae5199db47757f785e43a14b332118a5474de1a2` | unverified | [README](https://github.com/tree-sitter-grammars/tree-sitter-svelte/blob/ae5199db47757f785e43a14b332118a5474de1a2/README.md): README gives no supported Svelte language version. |
| `objc` | `181a81b8f23a2d593e7ab4259981f50122909fda` | unverified | [README](https://github.com/tree-sitter-grammars/tree-sitter-objc/blob/181a81b8f23a2d593e7ab4259981f50122909fda/README.md): README identifies Objective-C, not a supported language-version range or complete Objective-C++ coverage. |
| `json5` | `248b8564567087d7866be76569b182f6dd7e14e9` | unverified | [Repository inventory](https://github.com/Joakker/tree-sitter-json5/tree/248b8564567087d7866be76569b182f6dd7e14e9): No upstream README was present at this revision. The repository name does not prove JSON5 specification-version conformance. |
| `less` | `02988c765d30adb0476657b5d220e8dfde1c07d3` | unverified | [README](https://github.com/mdovale/tree-sitter-less/blob/02988c765d30adb0476657b5d220e8dfde1c07d3/README.md): README gives no supported Less language version. |
| `mdx` | `3aa29e8de1bf0213948a04fe953039b6ab73777b` | unverified | [README](https://github.com/srazzak/tree-sitter-mdx/blob/3aa29e8de1bf0213948a04fe953039b6ab73777b/README.md): README describes adapted JavaScript/JSX and Markdown grammars, but gives no supported MDX major version. |

## Kotlin: useful upstream evidence, incomplete Rule 3 provenance

The [pinned README](https://github.com/fwcd/tree-sitter-kotlin/blob/d6af96d2e23ef0a8ff5ab512864378cf08fba311/README.md) names Kotlin 2.4.21 at JetBrains commit
`9180ce6a7a7d3627f88b424717583f28ddca052d`.
It documents structural CST expectations, expected parser failures, and explicit exclusions.
The [tooling documentation](https://github.com/fwcd/tree-sitter-kotlin/blob/d6af96d2e23ef0a8ff5ab512864378cf08fba311/tools/README.md) describes hashed source/golden pairs and import policies.

The same revision contains an [archived fixture report](https://github.com/fwcd/tree-sitter-kotlin/blob/d6af96d2e23ef0a8ff5ab512864378cf08fba311/test/jetbrains-fixtures.json).
Its counts disagree with the README:

| Measure | README | Archived report |
| --- | ---: | ---: |
| PSI-clean original pairs | 447 | 447 |
| Eligible/compared inputs | 438 | 436 |
| Structural expectations/tests | 399 | 417 |
| Expected parser failures | 39 | 19 |
| Exclusions | 9 | 11 |

The report names the JetBrains fixture commit and hashes parser/scanner files.
It also records fixture-content SHA-256
`946c62787d7d902c2881055f669ad81602fcee998bd5844f25c194bc329e4354`.

The report file's SHA-256 is
`9978add5d2b11eda49202c56703300a1a159064421b89e91a2d573be0942c119`.
A checksum identifies an artifact; it does not prove that its assertions passed.

This research does not resolve the conflicting inventories or establish a passing result for the assessed grammar.
The report is therefore documentary context, not recorded Rule 3 evidence.
Rule 3 requires agreed version-specific coverage and an archived, attributable fixture result.
A future fixture assessment must reconcile these inventories and restrict its claim to demonstrated syntax.
Expected failures and experimental syntax exclusions prohibit a complete Kotlin 2.4.21 conformance claim.

## References that do not establish supported versions

- [PowerShell README](https://github.com/airbus-cert/tree-sitter-powershell/blob/65ea6919e816a7ac142acec7ee72757f89db8439/README.md): a References entry labels a specification page “Powershell 7.3”.
  A specification link does not assert that the grammar implements that version.
- [F# README](https://github.com/ionide/tree-sitter-fsharp/blob/aefd0c8741bdf3aeb827a228aa4a996a6536697e/README.md): “Based on the 4.1 F# language specification” identifies a basis.
  It does not establish complete 4.1 support, minimum support, maximum support, or compatibility with later F# releases.
- [HCL changelog](https://github.com/tree-sitter-grammars/tree-sitter-hcl/blob/64ad62785d442eb4d45df3a1764962dafd5bc98b/CHANGELOG.md): grammar releases and feature fixes do not establish HCL/Terraform version bounds.
- [Clojure scope](https://github.com/sogaiu/tree-sitter-clojure/blob/e43eff80d17cf34852dcd92ca5e6986d23a7040f/doc/scope.md) and
  [limits](https://github.com/sogaiu/tree-sitter-clojure/blob/e43eff80d17cf34852dcd92ca5e6986d23a7040f/doc/limits.md): these describe scope and parser limitations, not a language-version range.
- [Typst parser notes](https://github.com/uben0/tree-sitter-typst/blob/46cf4ded12ee974a70bf8457263b67ad7ee0379d/DOC.md): parser limitations do not establish supported Typst versions.

## Web-search audit and limits

Searches used public repository names only.
General Google, Bing, DuckDuckGo, Brave, and Mojeek searches were attempted.
Consent pages, access failures, rate limits, and irrelevant results limited their usefulness.
GitHub issue searches supplied additional discovery leads.
The grouped queries below cover all 22 unique upstream repositories:

- [Repository group 1](https://api.github.com/search/issues?q=repo%3Aairbus-cert%2Ftree-sitter-powershell%20repo%3Aionide%2Ftree-sitter-fsharp%20repo%3Afwcd%2Ftree-sitter-kotlin%20repo%3Amurtaza64%2Ftree-sitter-groovy%20version&per_page=100): `airbus-cert/tree-sitter-powershell`, `ionide/tree-sitter-fsharp`, `fwcd/tree-sitter-kotlin`, `murtaza64/tree-sitter-groovy`.
- [Repository group 2](https://api.github.com/search/issues?q=repo%3AJulian%2Ftree-sitter-lean%20repo%3Aalex-pinkus%2Ftree-sitter-swift%20repo%3AUserNobody14%2Ftree-sitter-dart%20repo%3Agleam-lang%2Ftree-sitter-gleam%20version&per_page=100): `Julian/tree-sitter-lean`, `alex-pinkus/tree-sitter-swift`, `UserNobody14/tree-sitter-dart`, `gleam-lang/tree-sitter-gleam`.
- [Repository group 3](https://api.github.com/search/issues?q=repo%3Auben0%2Ftree-sitter-typst%20repo%3Asogaiu%2Ftree-sitter-clojure%20repo%3Aeonpatapon%2Ftree-sitter-cue%20repo%3Atree-sitter-grammars%2Ftree-sitter-hcl%20version&per_page=100): `uben0/tree-sitter-typst`, `sogaiu/tree-sitter-clojure`, `eonpatapon/tree-sitter-cue`, `tree-sitter-grammars/tree-sitter-hcl`.
- [Repository group 4](https://api.github.com/search/issues?q=repo%3Acstrahan%2Ftree-sitter-nix%20repo%3Ageorgeharker%2Ftree-sitter-zsh%20repo%3Aram02z%2Ftree-sitter-fish%20repo%3Avictorhqc%2Ftree-sitter-prisma%20version&per_page=100): `cstrahan/tree-sitter-nix`, `georgeharker/tree-sitter-zsh`, `ram02z/tree-sitter-fish`, `victorhqc/tree-sitter-prisma`.
- [Repository group 5](https://api.github.com/search/issues?q=repo%3Atree-sitter-grammars%2Ftree-sitter-vue%20repo%3Atree-sitter-grammars%2Ftree-sitter-svelte%20repo%3Atree-sitter-grammars%2Ftree-sitter-objc%20repo%3AJoakker%2Ftree-sitter-json5%20version&per_page=100): `tree-sitter-grammars/tree-sitter-vue`, `tree-sitter-grammars/tree-sitter-svelte`, `tree-sitter-grammars/tree-sitter-objc`, `Joakker/tree-sitter-json5`.
- [Repository group 6](https://api.github.com/search/issues?q=repo%3Amdovale%2Ftree-sitter-less%20repo%3Asrazzak%2Ftree-sitter-mdx%20version&per_page=100): `mdovale/tree-sitter-less`, `srazzak/tree-sitter-mdx`.

Additional repository-specific searches used `is:issue (support OR version OR specification)`.
GitHub later returned search-rate-limit errors.
These searches are discovery work, not exhaustive proof that no compatibility statement exists.

Two illustrative leads were inspected:

- [Typst issue #35](https://github.com/uben0/tree-sitter-typst/issues/35) requests support for Typst 0.11.0+ and describes a `context` problem.
- [Dart issue #82](https://github.com/UserNobody14/tree-sitter-dart/issues/82) reports incomplete extension-type declarations and links the Dart 3.3 feature specification.

Neither issue establishes the state of the exact configured revision.
Their URLs also do not contain the assessed full SHA.
Neither was promoted to qualifying documentation evidence or used to infer version bounds.

## Files retained as evidence

The configuration retains pinned README evidence for the additions with READMEs.
Kotlin also records the pinned tooling documentation and archived report as `kind: documentation`.
Clojure also records the pinned scope document.
JSON5 remains unverified without a README evidence claim.
The [provider coverage map](provider-grammar-coverage.md) records routing, aliases, and syntax limitations separately from version compatibility.
