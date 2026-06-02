---
title: CLI
description: Current Ork command reference.
---

## Global

```sh
ork --debug <command>
```

`--debug` enables debug logging.

Most examples use the long flags. Short aliases are available for common options such as `-e` for `--env-id`, `-f` for `--file`, and `-o` for `--output` on `state inspect`.

## init

```sh
ork init [--file ork.yaml] [--id my-project] [--force]
```

Creates a starter manifest. By default, `init` writes `ork.yaml` and refuses to overwrite an existing file.

Flags:

- `--file`, `-f`: manifest path, default `ork.yaml`
- `--id`: manifest metadata ID, defaulting to the current directory name
- `--force`: overwrite an existing manifest

## up

```sh
ork up --env-id <id> [--file ork.yaml] [--param key=value] [--params-file path] [--reapply]
```

Applies the manifest for an environment.

Flags:

- `--env-id`, `-e`: required environment ID
- `--file`, `-f`: manifest path, default `ork.yaml`
- `--param`: repeatable key-value input
- `--params-file`: YAML or env parameter file
- `--reapply`: rerun components already marked applied

When both `--params-file` and `--param` provide the same key, the CLI `--param` value wins.

## down

```sh
ork down --env-id <id> [--file ork.yaml] [--param key=value] [--params-file path]
```

Destroys an environment from persisted state.

`down` still needs the manifest today so Ork can load the state backend and runner topology.

`down` accepts the same `--param` and `--params-file` inputs as `up`, so runner and component environment pointers can be resolved during teardown.

## state inspect

```sh
ork state inspect --env-id <id> [--file ork.yaml] [--output table|json]
```

Inspects persisted state for an environment.

The table view intentionally avoids outputs, payloads, and artifact contents.

## version

```sh
ork version
```
