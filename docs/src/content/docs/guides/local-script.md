---
title: Local Script Environment
description: Run a local script component and capture outputs.
---

Use the script adapter when you need custom setup, checks, glue code, or simple automation.

```yaml
version: ork/1.0

metadata:
  id: script-example
  description: Local script environment
  owner:
    name: ork
    email: ork@example.com

runners:
  local:
    type: local
    config: {}

components:
  setup:
    type: script
    runner: local
    source:
      embedded: |
        echo "message=hello from ork" >> "$ORK_OUTPUT_ENV"
    outputs:
      - name: message
```

Apply it:

```sh
ork up --env-id script-demo
```

Inspect state:

```sh
ork state inspect --env-id script-demo
```

Destroy it:

```sh
ork down --env-id script-demo
```
