# Runner execution model

ork currently treats a runner as the execution boundary for a component. If a component is assigned to an SSH runner, the component should be applied and destroyed from that remote machine, using the files, tools, network access, and ambient credentials available there.

Today this is mostly handled through shell execution:

- Docker Compose runs `docker compose` on the runner.
- Terraform runs `terraform` on the runner.
- CloudFormation can be driven through `aws cloudformation` on the runner.

This is simple and matches runner semantics, but it means every runner must have the required CLIs installed.

## SDK vs Runner Semantics

For cloud APIs, it is tempting to call provider SDKs directly from the main `ork` process. That removes the need for tools like the AWS CLI on the runner, but it changes where the action happens:

- SDK in `ork`: credentials, region, network, and permissions come from the machine running `ork`.
- Runner execution: credentials, region, network, and permissions come from the assigned runner.

For remote runners, these are not equivalent. A component assigned to a runner should not silently execute from the local control machine.

## Planned agent model

After the adapter contract has been exercised by more integrations and the minimal control plane exists, ork should introduce runner agents.

A runner agent is a versioned CLI application installed automatically on the target runner. It can be invoked over the existing runner transport; it does not need to start as a permanently running daemon. Instead of every adapter assembling native shell commands itself, adapters can call typed agent commands that perform operations inside the runner environment.

Examples:

- Report capabilities and protocol compatibility.
- Upload, materialize, or validate staged component files.
- Execute process commands with inherited ambient environment.
- Perform provider SDK calls using runner-local credentials.
- Stream structured events and command output back to `ork`.
- Return outputs, payload metadata, and artifact declarations.

This keeps the important invariant:

> Component actions happen from the assigned runner.

while reducing the long-term dependency on manually installed CLIs and avoiding provider SDK execution in the wrong identity context.

Automatic installation must be versioned and verifiable: Ork should select a compatible binary, verify its checksum or provenance, install it into an Ork-owned location without assuming root access, and perform a capability handshake before use. Restricted runners should support preinstallation or a configured artifact mirror.

See [Future direction](../project/future-direction.md) for sequencing, control-plane scope, and open protocol questions.

## MVP Position

For now, adapters may continue to use runner shell execution and document required tools. This keeps the system understandable and compatible with local and SSH runners.

Runner agents are a future architecture milestone, not a prerequisite for the current CLI. More adapter experience should shape the agent protocol before it is made stable.
