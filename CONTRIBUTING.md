# Contributing to ork

ork is an alpha-stage project. Contributions are welcome, but lifecycle and manifest changes deserve extra care because they affect whether environments can be recovered and destroyed safely.

## Start here

1. Read the [project README](README.md) and [internal design index](notes/README.md).
2. Check the public [documentation](docs/src/content/docs/) for current user-facing behavior.
3. For manifest questions, treat the parser and core schema as the source of truth.
4. Open an issue before a large feature or architectural change so the direction can be agreed first.

## Development setup

Requirements:

- Go as declared in `go.mod`
- Node.js 20 or newer for the documentation site
- Adapter-specific CLIs only when running their smoke tests

Run the Go test suite:

```sh
go test ./...
```

Build the CLI:

```sh
go build -o bin/ork ./cmd/ork
```

Build the documentation:

```sh
cd docs
npm install
npm run build
```

Smoke tests live under `tests/`. They may require Docker, Terraform, AWS credentials, or other external tools and should be run only when relevant to the change.

## Change guidelines

- Keep changes focused; avoid unrelated cleanup in feature commits.
- Follow patterns already present in the package being changed.
- Add focused tests near parsing, orchestration, adapter, runner, and state behavior.
- Update public docs when CLI behavior, manifest shape, authentication, or lifecycle semantics change.
- Preserve existing worktree changes that are unrelated to your contribution.
- Never commit credentials, generated state, Terraform state, or environment parameter files.

## Lifecycle changes

For changes to `up`, `down`, state, hooks, adapters, or runners, verify all of the following:

- failure state is persisted before returning when recovery may be needed;
- teardown uses recorded state rather than assuming apply completed;
- component destruction remains in reverse recorded order;
- sensitive outputs are not persisted;
- runner execution still happens in the selected runner context;
- an interrupted operation has an understandable retry path.

The relevant contracts are documented under [`notes/design/`](notes/design/) and [`notes/architecture/`](notes/architecture/).

## Pull requests

Describe:

- the problem and intended behavior;
- important design decisions or rejected alternatives;
- tests and documentation updated;
- any compatibility or teardown risk.

Small, reviewable pull requests are strongly preferred.
