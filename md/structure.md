ork/
├── cmd/
│ ├── ork/ # CLI binary (agent)
│ │ ├── main.go
│ │ └── commands/
│ │ ├── up.go
│ │ ├── down.go
│ │ ├── validate.go
│ │ └── status.go
│ │
│ └── orchd/ # Control plane binary (daemon)
│ ├── main.go
│ └── handlers/
│ ├── sandbox_handler.go
│ ├── state_handler.go
│ └── agent_handler.go
│
├── pkg/
│ ├── manifest/ # Manifest parsing & validation
│ │ ├── schema.go
│ │ ├── parser.go
│ │ └── validator.go
│ │
│ ├── inputs/ # Input templating & resolution
│ │ ├── resolver.go
│ │ └── interpolation.go
│ │
│ ├── adapters/ # Cloud/service adapters
│ │ ├── terraform/
│ │ │ ├── executor.go
│ │ │ └── state.go
│ │ ├── dockercompose/
│ │ │ ├── executor.go
│ │ │ └── utils.go
│ │ ├── cloudformation/
│ │ │ ├── executor.go
│ │ │ └── state.go
│ │ ├── helm/
│ │ │ └── executor.go
│ │ └── interface.go # Common adapter interface
│ │
│ ├── hooks/ # Lifecycle hooks (preRun, postRun, etc.)
│ │ ├── runner.go
│ │ └── context.go
│ │
│ ├── state/ # State storage (local JSON or remote)
│ │ ├── manager.go
│ │ ├── local_store.go
│ │ └── remote_store.go
│ │
│ ├── cloud/ # Cloud credential management
│ │ ├── aws.go
│ │ ├── gcp.go
│ │ ├── azure.go
│ │ └── onprem.go
│ │
│ ├── api/ # API server (for orchd)
│ │ ├── routes.go
│ │ ├── middleware.go
│ │ ├── models.go
│ │ └── client.go # ork client SDK
│ │
│ ├── utils/ # Shared helpers
│ │ ├── logging.go
│ │ ├── errors.go
│ │ ├── exec.go
│ │ └── file.go
│ │
│ └── config/ # Config file & environment parsing
│ ├── loader.go
│ └── defaults.go
│
├── internal/
│ ├── orchestration/ # High-level orchestration engine
│ │ ├── engine.go
│ │ ├── planner.go
│ │ ├── executor.go
│ │ └── teardown.go
│ │
│ ├── controlplane/ # Control plane logic for orchd
│ │ ├── registry.go
│ │ ├── scheduler.go
│ │ ├── api_server.go
│ │ └── agent_client.go
│
├── tests/
│ ├── manifests/
│ │ ├── basic_terraform.yaml
│ │ ├── docker_inline.yaml
│ │ └── full_integration.yaml
│ │
│ └── integration/
│ ├── ork_up_down_test.go
│ ├── terraform_adapter_test.go
│ └── docker_compose_test.go
│
├── go.mod
├── go.sum
└── README.md

## State persistence decisions

Ork owns one environment state document per environment ID. The default local backend stores it at:

```text
.ork/<env-id>/state.json
```

State storage is pluggable through `pkg/state.Backend`. The interface stays in `pkg/state` because it depends on core state types such as `OrkState` and `Artifact`. Backend implementations live under `pkg/state/backends` with package name `statebackends`. This avoids a Go import cycle: backend implementations can import `pkg/state`, while `pkg/state` does not import its implementations.

Current wiring uses the local backend explicitly:

```go
state.NewManager(envID, statebackends.NewLocal(".ork"))
```

Adapters return `ComponentStateData` after apply. This contains the component workdir, adapter payload, and any tool-state artifacts that must be preserved for teardown. The rule for artifacts is intentionally narrow:

- artifact paths are relative to the component workdir
- artifact paths are captured from and restored to the same path
- absolute paths and `../` escapes are rejected
- artifacts are files, not directories
- local backend writes artifact files with owner-only permissions (`0600`)

Terraform is the first adapter that needs tool-state artifacts. Docker Compose state lives in the Docker daemon under the Compose project, and CloudFormation state lives in AWS. Terraform with the default local backend needs `terraform.tfstate` to destroy resources from a stateless runner, so the Terraform adapter declares:

```text
terraform.tfstate
terraform.tfstate.backup
.terraform.lock.hcl
```

During `up`, Ork saves the Ork state document and then captures declared artifacts from the runner into the configured state backend. During `down`, Ork restores artifacts to the runner before calling the adapter destroy path.

Artifact capture currently uses temporary files as a bridge. The runner API copies files to and from filesystem paths, and the backend API also saves/restores from filesystem paths. We cannot assume direct runner-to-backend streaming, especially for future remote/object-store backends, so the flow is:

```text
capture: runner path -> temp local file -> state backend
restore: state backend -> temp local file -> runner path
```

The temp files are created in the OS temp directory using Go's `os.CreateTemp`; the `*` in the pattern is only Go's random suffix placeholder. They are removed after each artifact operation.

Terraform destroy also rehydrates the Terraform module source before running `terraform init` and `terraform destroy`. Source staging excludes `.terraform/`, `terraform.tfstate`, and `terraform.tfstate.backup` so a stale source-local state file cannot overwrite the restored artifact state.
