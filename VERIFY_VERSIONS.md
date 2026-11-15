# Verify Grammar Language Versions

Use this workflow when asked which source-language versions a grammar supports.
This is **not** grammar package version, Tree-sitter CLI version, or parser ABI verification.

**Default to Rule 2 when the user asks to verify supported languages/versions.**
Rule 3 fixture testing requires an explicit user request; never use it automatically
when documentation is inconclusive.

## Rule 1 — Unknown / unverified

No research or testing is required. Leave `language_support` absent when there is
no assessment; absence means unknown/unverified, not unsupported. Keep tentative
ranges marked `claim: unverified` and explain their uncertainty.

## Rule 2 — Documentation research (default)

1. Read each requested grammar's active full `revision` in `tree-sitter-config.yaml`.
2. Inspect upstream documentation for that exact revision for explicit language-version
   support claims, exclusions, and partial support. Do not substitute current HEAD.
3. Record the assessment's own full 40-character `assessed_revision`, independently
   of the active dependency pin. Record only bounds or limitations justified by the sources.
4. For qualifying evidence, use `kind: documentation`, an HTTPS source URL whose
   path contains the assessed full SHA as a complete segment, and a `description`
   stating the actual upstream claim. Set `claim: verified` only when the documented
   claim supports the recorded scope; this is documentation-backed, **not fixture-tested**.
5. If documentation is missing or inconclusive, report that and leave support
   unverified. Never infer a range from release dates, package tags, commit titles,
   or isolated syntax examples. Do not start Rule 3 without permission.

## Rule 3 — Version-specific fixtures (explicit request only)

1. Agree on the languages, versions, and syntax coverage requested; test the exact
   assessed grammar revision with version-specific fixtures and expected-tree or
   syntax assertions. A parse without errors alone is not full conformance evidence.
2. Archive a report identifying the grammar revision, fixture suite and fixture
   checksums, assertions, results, and exclusions.
3. Record successful evidence as `kind: syntax_fixtures` with `source`, `description`,
   `result: passed`, `tested_versions`, and the report's SHA-256 in `sha256`.
   Verified claims must remain limited to demonstrated coverage; isolated successful
   fixtures do not prove a continuous version range or complete language support.

## Record and report safely

- Follow the field requirements in [Configuration Schema](docs/reference/configuration-schema.md#language-version-assessments-language_support).
  Structural validation does not fetch sources, check report contents/digests, or rerun fixtures.
- Report each grammar's active and assessed revisions, claim, justified bounds,
  evidence kind/source, and limitations. Missing bounds mean unknown, not unlimited support.
- User-owned fixture suites and reports can remain outside this repository. Repository fixture additions are not required.
- Use `ts-build support show --language <name> --json` to inspect the active revision and assessment.
- Use `ts-build support update --language <name> --assessment <file.json> --check-report` to preview an external fixture assessment.
- Add `--apply` to save it in the selected configuration. The command checks report integrity, not test correctness, and never executes fixtures.
- Dependency updates must preserve historical assessment pins and evidence. Reassess
  deliberately; do not simply repin old claims. A revision mismatch makes current support unverified.
- Do not rebuild or republish artifacts merely to research support. Manifests acquire
  metadata through normal binary-validated publication, never manifest-only regeneration.
  Consumers must check the outer manifest `language_support.claim`, not just the historical
  nested assessment. See [Manifest Schema v2](docs/reference/manifest-schema-v2.md#language_support-object-fields).
