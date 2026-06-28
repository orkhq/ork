# Artifact persistence

ork owns one environment state document per environment ID. Adapters can additionally declare tool-state artifacts that must survive between apply and destroy.

## Why artifacts are separate

Operational metadata fits in `state.json`, but native tool state is often an opaque file. Terraform using its local backend, for example, needs `terraform.tfstate` to destroy resources from a later process or stateless runner.

Adapters return `ComponentStateData` containing:

- the component work directory;
- an adapter-specific, non-sensitive payload;
- files that must be captured and restored for teardown.

Artifact paths follow strict rules:

- paths are relative to the component work directory;
- capture and restore use the same relative path;
- absolute paths and `../` traversal are rejected;
- artifacts are files, not directory trees;
- the local backend writes files with owner-only permissions (`0600`).

## Transfer flow

The runner and state backend APIs both operate on filesystem paths. Ork uses a temporary local file as the bridge so the same contract works for local, SSH, and object-store combinations:

```text
capture: runner path -> temporary local file -> state backend
restore: state backend -> temporary local file -> runner path
```

Temporary files are created in the operating-system temp directory and removed after each operation.

## Terraform behavior

The Terraform adapter may preserve:

```text
terraform.tfstate
terraform.tfstate.backup
.terraform.lock.hcl
```

During teardown, Ork stages the Terraform source and then restores captured state artifacts before running `terraform init` and `terraform destroy`. Source staging excludes `.terraform/` and local state files so stale source content cannot overwrite the restored artifacts.

Artifact contents should be treated as sensitive operational data even if the manifest declares no sensitive outputs.
