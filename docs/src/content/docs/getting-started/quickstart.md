---
title: Quickstart
description: Run a local script component with Ork.
---

This quickstart runs a local script component, captures an output, inspects state, and tears the environment down.

## Install Ork

```sh
curl -fsSL https://tryork.dev/install | sh
```

## Create A Manifest

Generate a starter manifest:

```sh
ork init --id hello
```

Or create `ork.yaml` manually:


```yaml
version: ork/1.0

metadata:
  id: hello
  description: Local script example
  owner:
    name: Ork
    email: ork@example.com

runners:
  local:
    type: local
    config: {}

components:
  hello:
    type: script
    runner: local
    source:
      embedded: |
        echo "message=hello from ork" >> "$ORK_OUTPUT_ENV"
    outputs:
      - name: message
```

## Apply

```sh
ork up --env-id demo
```

Ork applies the component and writes state under `.ork/demo` by default.

## Inspect

```sh
ork state inspect --env-id demo
```

The default table output shows component status, stage, type, runner, and timestamps. It intentionally does not print outputs, payloads, or artifact contents.

## Destroy

```sh
ork down --env-id demo
```

After a successful destroy, Ork deletes the whole environment state bundle, including artifacts.
