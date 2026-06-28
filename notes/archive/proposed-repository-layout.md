# Archived repository layout proposal

This tree predates the current package structure and is retained only as historical design material. See [Repository architecture](../architecture/repository-layout.md) for the current layout.

```text
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
```
