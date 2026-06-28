# Future direction

The near-term plan is to broaden Ork carefully, then introduce the smallest useful control plane, and only then add runner agents. The sequence matters: additional adapters should teach us which remote operations are genuinely common before we freeze an agent protocol around guesses.

## Phase 1: Broaden adapter coverage

Add a small number of adapters that represent meaningfully different lifecycle shapes. Candidates should be chosen for what they teach the orchestration model, not simply for ecosystem coverage.

Useful categories include:

- another infrastructure tool with external or local state;
- a service deployment tool with update and health semantics;
- a cloud API workflow with long-running asynchronous operations;
- a data/bootstrap tool whose teardown is partial or intentionally absent.

Each adapter should define:

- supported source modes;
- required runner capabilities and native tools;
- apply, reapply, and destroy semantics;
- declared outputs and sensitive-output behavior;
- operational payload and artifacts required for recovery;
- minimum compatible tool versions;
- failure checkpoints and retry expectations.

The goal is not to accumulate integrations rapidly. It is to make the adapter contract strong enough that future adapters fit without weakening teardown guarantees.

## Phase 2: Minimal control plane

The first control plane should remain a thin remote lifecycle coordinator around the existing engine. It should not become a second orchestration implementation.

Its minimum responsibilities are:

- accept and identify environment lifecycle requests;
- store environment metadata, sanitized execution topology, and state-backend references;
- invoke the same planning and lifecycle logic used by the CLI;
- expose environment status and structured events;
- initiate teardown without requiring the original manifest checkout;
- resolve control-plane and runner identity references without copying credentials into state;
- serialize or reject conflicting operations for the same environment.

TTL cleanup, pull-request webhooks, scheduling, organization policy, and a web interface can follow after the basic remote `up`, inspect, and `down` loop is trustworthy.

The control plane should preserve a useful local mode. Running `ork` directly must remain a supported path rather than becoming a thin client that requires a hosted service.

## Phase 3: Runner agents

A runner agent is a versioned CLI application installed automatically on a target runner. It does not need to begin as a permanently running daemon.

Instead of every adapter assembling native shell commands itself, an adapter can invoke a typed agent command through the runner transport. The agent then performs the operation inside the runner's network, filesystem, and provider-auth context.

Conceptually:

```text
control plane or local ork
  -> lifecycle engine
    -> adapter
      -> runner transport
        -> agent CLI
          -> native tool or provider API
```

Examples of agent responsibilities:

- inspect runner capabilities and native tool versions;
- stage or validate component source;
- execute typed Terraform, Compose, cloud, or script operations;
- use provider SDKs without moving credentials to the control plane;
- stream structured progress and diagnostics;
- return outputs, payload metadata, and artifact declarations;
- support cancellation and operation timeouts;
- report a stable machine-readable protocol version.

### Automatic installation

Agent bootstrap should be explicit and auditable even when automatic:

1. Detect runner operating system and architecture.
2. Select a version compatible with the invoking Ork release.
3. Download from a trusted release source or use an existing cached binary.
4. Verify checksum and, when available, signature or provenance.
5. Install into an Ork-owned directory without modifying unrelated system tools.
6. Run a capability and protocol handshake before lifecycle work begins.

Offline and restricted runners should be able to use a preinstalled agent or a configured artifact mirror. Automatic installation must not silently require root access.

### Why agents come later

The current shell-based runner model is operationally simple and exposes real adapter requirements. Building agents after several adapters exist lets the protocol reflect observed common operations instead of mirroring one adapter's assumptions.

Agents should reduce runner setup and improve typed operations, but they must not erase the runner boundary. Provider SDK calls performed by an agent still use runner-local identity and network access.

## Invariants across all phases

- Existing tools remain first-class; Ork coordinates rather than replaces them.
- Components execute in their selected runner context.
- State-backend authentication belongs to the Ork control process.
- Runner and provider authentication remain scoped to the runner.
- Persisted state stores operational facts, not reusable credentials.
- Every successful apply path must define a credible teardown or explicitly document why it cannot.
- The control plane, CLI, and agents share lifecycle contracts rather than implementing subtly different behavior.
- Protocol and persisted-state changes require versioning and migration rules.

## Open design questions

- One general agent CLI versus a small family of adapter-specific agent CLIs.
- Whether native tools remain external dependencies or selected agents bundle libraries/SDKs.
- How agent and adapter versions negotiate compatibility.
- How bootstrap works for hosts without outbound internet access.
- How signed releases, checksums, mirrors, and rollback are managed.
- Which operations need streaming RPC semantics rather than one-shot CLI execution.
- How the control plane delegates runner identity without storing long-lived credentials.

These questions should be answered through prototypes and adapter experience before the protocol is declared stable.
