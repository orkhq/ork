# Internal design notes

This directory records implementation contracts, architectural decisions, and future direction for contributors. Public user documentation lives in [`docs/src/content/docs/`](../docs/src/content/docs/).

The implementation remains the final authority when a note drifts. In particular, use [`pkg/manifest/core/schema.go`](../pkg/manifest/core/schema.go) and [`pkg/manifest/parsers/`](../pkg/manifest/parsers/) to determine the accepted manifest shape.

## Architecture

- [Repository layout](architecture/repository-layout.md) — package ownership and dependency direction.
- [Artifact persistence](architecture/artifact-persistence.md) — how tool-state files move between runners and backends.
- [Runner execution](architecture/runner-execution.md) — why component work executes inside its selected runner context.
- [Security model](architecture/security-model.md) — credentials, command output, SSH, state, and teardown boundaries.

## Current design contracts

- [Apply policy](design/apply-policy.md) — repeated `up`, partial failures, and reapply behavior.
- [Lifecycle hooks](design/lifecycle-hooks.md) — hook phases, execution, interpolation, and persistence.
- [Script adapter](design/script-adapter.md) — source handling, outputs, and teardown behavior.
- [State backends](design/state-backends.md) — backend interface, storage layout, authentication, and deletion.
- [State recovery](design/state-recovery.md) — statuses, stages, retry rules, and manifest dependency.

## Project direction

- [Future direction](project/future-direction.md) — adapters, the minimal control plane, and runner agents.
- [Open-source readiness](project/open-source-readiness.md) — completed foundations and remaining release work.

## Archive

[`archive/`](archive/) contains superseded proposals and manifest sketches retained for historical context. They are not implementation documentation and should not be copied into examples.

## Updating these notes

Update the relevant note when a change alters a lifecycle invariant, persistence contract, authentication boundary, or accepted manifest shape. User-visible behavior must also be reflected in the public documentation.
