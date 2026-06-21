---
title: Security Overview
description: Current security posture and boundaries.
---

ork is an orchestration tool, not a secret manager.

The current security posture is built around a few boundaries:

- Do not persist sensitive outputs in ork state.
- Treat state backends and artifacts as sensitive operational data.
- Prefer ambient authentication for runners and cloud providers.
- Avoid logging command invocations and environment values by default.
- Let shells expand `$ENV_VAR` on the runner instead of interpolating environment variables into command strings.

## Sensitive Outputs

Sensitive outputs are available during the same `ork up` process that produced them. They are not persisted in state.

If a later run needs a sensitive output from a skipped component, ork fails clearly instead of inventing or leaking a value.

## State

State can include operational handles, resource identifiers, adapter payloads, and tool artifacts such as Terraform state.

Store local state outside source control. Keep object-store state private and encrypted.

State backend profiles and interpolated access keys are consumed only by the ork process. Prefer profiles, ambient workload identity, or environment interpolation. Although literal access keys are accepted in `state.auth`, doing so makes the manifest a secret-bearing document and is discouraged.

## Teardown

Today, `ork down` still needs the manifest so it can locate and authenticate to the state backend, reconstruct runner connections, and establish runner provider context. Persisted state contains destroyable operational facts and artifacts, but deliberately does not store those credentials.

The planned ork daemon and managed state backend will keep sanitized execution topology and resolve state and runner identities through the control plane. Teardown through that service will therefore no longer require the original manifest checkout. This removes the manifest dependency without turning state into a secret store.
