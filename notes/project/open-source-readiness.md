# Open-source readiness

This document tracks repository-level work needed for early external users and contributors. It complements, rather than duplicates, the public roadmap.

## Foundations in place

- CLI quickstart and public documentation site.
- Versioned manifest parser with current reference documentation.
- Local and SSH execution runners.
- Script, Docker Compose, Terraform, and early CloudFormation adapters.
- State-aware apply, retry, inspection, and reverse teardown.
- Local and S3-compatible state backends.
- Sensitive output filtering and explicit authentication boundaries.
- Focused unit, integration, and opt-in smoke tests.
- Contributor and vulnerability-reporting guidance.

## Before a broader alpha announcement

- Publish repeatable demo repositories that exercise mixed-tool lifecycles.
- Add release notes and a lightweight changelog process.
- Confirm installation and release assets on Linux and macOS.
- Enable branch protection and required CI checks for the default branch.
- Add issue and pull-request templates once external traffic makes them useful.
- Review the public docs against the parser and smoke tests before each tagged release.

## Engineering priorities

### Recovery and concurrency

- Add optimistic revisions or locking for object-store state.
- Define repair operations for exceptional cases that normal retry cannot recover.
- Preserve enough sanitized topology for daemon-initiated teardown without a manifest checkout.

### Adapter maturity

- Keep CloudFormation labeled as early until create, update, output, artifact, and teardown behavior is covered end to end.
- Document tool-version expectations and compatibility for every adapter.
- Add adapter contract tests that can be reused by future component types.

### Runner evolution

- Continue treating the runner as the execution and provider-auth boundary.
- Introduce automatically installed runner-agent CLIs only after more adapters reveal the common typed operations they need.
- Keep the first agent invocation-based; require a long-running daemon only if concrete streaming or scheduling needs justify it.
- Avoid moving provider SDK calls into the local Ork process when that would change execution identity or network location.

### Security

- Keep state credentials separate from runner provider credentials.
- Prefer ambient, profile, workload, or interpolated authentication over literal secrets.
- Audit new logging and events for command, environment, and credential exposure.
- Treat state artifacts as sensitive operational data even when declared component outputs are non-sensitive.

## Documentation policy

- The parser and core schema are authoritative for manifest shape.
- Public docs describe supported user behavior.
- Internal design notes explain invariants and future direction.
- Archived proposals must be clearly labeled and must not appear as valid examples.

## Release posture

ork remains alpha. Compatibility may change, but destructive behavior should never change casually: teardown ordering, state persistence, credential boundaries, and failure recovery require explicit tests and documentation in every release.
