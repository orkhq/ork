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
| `config` | No | Backend defaults | Backend-specific configuration. |

See [State Backends](/reference/state-backends/) for backend details.
