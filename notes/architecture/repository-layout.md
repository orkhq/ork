# Repository architecture

ork is a Go CLI that coordinates environment lifecycles while delegating resource-specific work to existing tools. Package boundaries follow that division of responsibility.

## Entry points

- `cmd/ork` wires Cobra commands, flags, parameter loading, and logging.
- `cmd/orkd` is reserved for the future daemon/control-plane process and is not feature-complete.

Command packages should remain thin. Lifecycle decisions belong in orchestration, and provider/tool behavior belongs in adapters or runners.

## Core lifecycle

- `internal/orchestration` plans component order and coordinates apply, recovery, hooks, state writes, and reverse teardown.
- `internal/adapters` implements component behavior for scripts, Docker Compose, Terraform, and the early CloudFormation adapter.
- `internal/scaffold` creates starter manifests.

These packages are internal because their APIs are implementation details of the CLI.

## Reusable packages

- `pkg/manifest` loads manifests and dispatches to versioned parsers.
- `pkg/manifest/core` defines the canonical in-memory manifest model.
- `pkg/runners` owns execution and file-copy boundaries for local and SSH targets.
- `pkg/state` defines persisted lifecycle state and the backend contract.
- `pkg/state/backends` implements local and S3-compatible persistence.
- `pkg/varresolvers` resolves inputs, environment variables, and component outputs.
- `pkg/events` renders structured lifecycle events.
- `pkg/logging` provides structured diagnostic logging that is separate from lifecycle events.
- `pkg/utils` contains small filesystem, shell, validation, and output helpers.

## Dependency direction

```text
cmd/ork
  └── internal/orchestration
    ├── internal/adapters
    ├── pkg/manifest/core
    ├── pkg/runners
    ├── pkg/state
    ├── pkg/events
    └── pkg/varresolvers
```

Adapters operate through the runner interface. A component assigned to an SSH runner must not silently perform its provider work from the local Ork process.

State backends are different: they are control-plane dependencies and always execute from the machine running Ork. State authentication therefore does not come from runner provider configuration.

## Sources of truth

- Accepted manifest shape: parser and core schema.
- Lifecycle behavior: orchestration code and tests.
- Adapter capabilities: adapter contract and implementation.
- Persisted format: `pkg/state` models and backend layout tests.
- User-facing behavior: public docs, kept in sync with the implementation.
