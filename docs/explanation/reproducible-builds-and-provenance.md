# Provenance and reproducibility

Reproducible builds require stable source identities, recorded toolchains, and a clear connection between metadata and artifact bytes.

The pipeline records these inputs. It does not promise identical bytes across arbitrary compilers, operating systems, or toolchain installations.

## Source identity

Every configured grammar revision must be a full, 40-character hexadecimal commit SHA. Tags, branch names, and abbreviated revisions are rejected.

The grammar's `version` is a package label. The revision selects the actual source. A version update must include the corresponding full revision.

The builder derives `SOURCE_DATE_EPOCH` from the source commit timestamp.

## Source provenance

The generated workspace contains `.source-provenance.json`. It records:

- Source repository and revision.
- Generator version and selected generation ABI.
- Source timestamp.
- SHA-256 of `node-types.json`.
- Hashes of generated source inputs under `src/`.

Before compilation, the compiler verifies the source inputs and relevant configured identities. Editing generated source after generation invalidates its provenance.

Regenerate sources through `build` instead of editing a sidecar to make modified inputs appear valid.

## Binary provenance

Each compiled artifact has a `.provenance.json` sidecar. It records:

- Artifact filename and SHA-256.
- Source repository, revision, timestamp, and AST checksum.
- Generator version and selected generation ABI.
- Target triple, compiler identity, compiler version, and build flags.

The compiler creates fresh temporary output, validates it, and records its checksum before installing it as the final artifact.

The mover checks the artifact checksum against its sidecar and checks configured source identities before staging. An old binary cannot receive current provenance merely because it exists at an expected output path.

## Publication provenance

Manifest generation captures validated binary metadata for downstream consumers. Host ABI measurements come from loaded binaries, not configuration assertions.

Publication holds a catalog lock and completes validation before exchanging the release directory. This prevents a validation error from replacing the published catalog.

See [Architecture](architecture-and-pipeline.md) for filesystem and platform limits.

## What hashes do not prove

A checksum detects changes relative to the recorded value. It does not authenticate the producer, sign a release, or prove that an upstream repository is trustworthy.

A pinned commit also does not establish license permissions or source-language feature support. Those require separate upstream evidence.

For reproducible release jobs, retain the configuration, source provenance, binary sidecars, manifests, tool versions, and license notices.

See [NOTICE](../../NOTICE.md) for the current Go scan and its native-source limits.