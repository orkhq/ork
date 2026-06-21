---
title: State and Recovery
description: How ork persists progress and recovers from failures.
---

ork stores operational state so it can recover from interrupted or failed runs.

State is not a secret store. It records handles needed for recovery and teardown: component names, runners, work directories, adapter payloads, outputs that are safe to persist, lifecycle status, and artifacts.

## Status And Stage

Each component records both a `status` and a `stage`.

Common statuses:

- `applying`
- `applied`
- `destroying`
- `destroyed`
- `failed`

Common stages:

- `config`
- `pre_apply`
- `apply`
- `outputs`
- `artifacts`
- `post_apply`
- `pre_destroy`
- `destroy`
- `post_destroy`

The status says what happened. The stage says where it happened.

## Re-running Up

By default, `ork up` skips components already marked `applied` and rehydrates their non-sensitive outputs for downstream interpolation.

Use `--reapply` to run already-applied components again:

```sh
ork up --env-id demo --reapply
```

## Failed Runs

Apply-side failures can usually be retried with `ork up`.

Destroy-side failures block `up`; run `ork down` again so ork can finish cleanup.

## Successful Down

After every component and post-destroy hook succeeds, `ork down` deletes the environment state bundle.

If teardown fails before completion, state is kept so the next `down` can retry.

## Why Down Still Needs The Manifest

Today, `ork down` needs the manifest for two pieces of live configuration that are intentionally not copied wholesale into state:

- how to locate and authenticate to the configured state backend
- how to reach each referenced runner and establish its provider context

Persisted state supplies the component order, adapter payloads, non-sensitive outputs, destroy hooks, and artifacts needed for teardown. The manifest supplies the current connection and identity configuration. Keeping those responsibilities separate prevents ork state from becoming a credential store.

This is a current CLI architecture constraint, not the intended final recovery experience. The planned ork daemon and managed state backend will retain the environment's sanitized execution topology and resolve state and runner identities from the control plane. Teardown initiated through that service will not require the original manifest checkout. Local, standalone CLI workflows may continue to use the manifest as their configuration source.
