---
title: State
description: Manifest state backend configuration.
---

`state` configures where ork stores component state and artifacts.

```yaml
state:
  backend: local
  config:
    path: .ork
```

If `state` is omitted, ork uses the local backend.

| Field | Required | Default | Description |
| --- | --- | --- | --- |
| `backend` | No | `local` | State backend name. |
| `auth` | No | Backend ambient authentication | Backend-specific authentication selection. Values support input and environment interpolation. |
| `config` | No | Backend defaults | Backend-specific configuration. |

String values inside `state.auth` and `state.config` support manifest inputs and OS environment interpolation. They are resolved before ork constructs the backend. Component outputs are not available at this stage.

See [State Backends](/reference/state-backends/) for backend details.
