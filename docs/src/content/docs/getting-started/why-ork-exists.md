---
title: Why ork Exists
description: The environment lifecycle problems ork is designed to solve.
---

Modern teams already have good tools for creating things.

Some teams use Terraform to provision infrastructure. Some use Docker Compose to start services. Some use CloudFormation, Kubernetes, cloud CLIs, internal scripts, or tools specific to their platform.

The hard part is not usually provisioning one resource. The hard part is owning the lifecycle of an environment assembled from many tools.

## Real Environments Span Tools

A preview, test, or development environment often needs several steps:

- provision infrastructure
- start services
- seed or migrate data
- create cloud resources
- run smoke checks
- expose values to later steps

Each tool can do its own job well. ork exists to coordinate the environment as a whole: ordering, dependencies, execution location, state capture, inspection, retry, and teardown.

## Teardown Is First-Class

Many workflows optimize for `up`.

Fewer optimize for `down`.

That is where teams end up with orphaned resources, leaked cloud infrastructure, stale preview environments, and cleanup runbooks that depend on memory.

ork starts from a simple expectation: if an environment can be created, it should also be destroyable.

## State Should Enable Recovery

When an environment spans tools, state is fragmented. Some tools have state files. Some have runtime metadata. Scripts may have no state at all. Cloud APIs often need resource identifiers or names.

ork records the non-sensitive operational facts needed to understand and manage the environment later:

- component status and stage
- runner references and work directories
- adapter payloads
- safe outputs
- tool artifacts required for teardown

State is not just a record of what was created. It is the recovery surface for inspection, retry, and destruction.

## Identities Are Not State

Credentials, SSH passwords, API keys, and cloud tokens are identities. They are not environment state.

ork intentionally separates execution, configuration, state, and identity. State should preserve enough context to manage an environment, but it should not become a secret store just to make teardown convenient.

## Existing Tools Stay First-Class

ork does not replace the tools that create resources, start services, configure systems, or run checks.

It gives them a shared lifecycle.

Adapters own tool-specific behavior, including the non-sensitive state and artifacts required for deterministic teardown. Runners define where work executes. The manifest describes how the environment is assembled.

ork exists because modern environments are assembled from many tools, but owned as one lifecycle.
