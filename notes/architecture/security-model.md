# Security model

ork is an orchestration tool that executes commands, copies files, persists operational state, and talks to remote runners. Treat its configuration and state with the same care as other infrastructure automation systems.

## Command And Environment Secrecy

ork should not log command invocations or environment values by default.

Runner stdout and stderr are process output. If a hook, script, or provider CLI prints a secret, ork will stream that output because it cannot reliably distinguish secret text from normal process output.

Guidelines:

- pass secrets through environment variables instead of interpolating them directly into shell commands
- prefer `$TOKEN` shell expansion over `${component.outputs.token}` inside command strings
- hook command interpolation does not read OS environment variables; pass them through hook `env` instead
- do not enable future command tracing in environments where commands may contain secrets
- avoid printing tool state, provider output, or environment dumps in scripts and hooks

For SSH runners, ork sends environment exports and the command body through a stdin shell wrapper instead of placing them directly in the remote SSH command string. This keeps environment values out of the remote process arguments. The values still exist in the remote shell environment while the command runs, which is expected for environment-based secret passing.

## SSH Host Keys

SSH runners must verify host keys unless explicitly configured otherwise.

Preferred configuration:

```yaml
runners:
  ionos:
    type: ssh
    config:
      host: example.com
      port: 22
      user: deploy
      auth:
        method: key
        key_path: ~/.ssh/id_ed25519
      host_key:
        known_hosts: ~/.ssh/known_hosts
```

For local development only:

```yaml
host_key:
  insecure: true
```

Rules:

- exactly one host key verification method must be configured
- `known_hosts` uses OpenSSH-style known hosts verification
- `insecure: true` disables host key verification and should not be used for shared, CI, or production-like environments

Pinned fingerprints, direct public-key pinning, and trust-on-first-use are intentionally deferred until there is clearer demand and a stronger persistence policy.

## State And Artifacts

ork state is operational state, not a secret store. Even when sensitive outputs are dropped, state can still contain infrastructure-sensitive information:

- component names and locations
- runner workdirs
- adapter payloads
- non-sensitive outputs
- Terraform state artifacts
- resource identifiers

State backends should be private. Object-store backends should use private buckets, least-privilege access, and encryption. Local `.ork` directories should stay out of source control.

The default `ork state inspect` table intentionally avoids outputs, payloads, and artifact contents.

## Sensitive Outputs

Sensitive outputs are process-local unless a future secrets backend exists.

They are available to downstream components during the same `ork up` process that produced them. They are not persisted in state. If a later `up` or `down` references a sensitive output from an already-applied component, ork should fail clearly instead of inventing or leaking a value.

## Ambient Auth And Teardown

Reliable teardown assumes ambient authority on the runner.

Today, `ork down` still needs the manifest to locate and authenticate to the state backend, reconstruct runner connections, and establish runner provider context. It should not rely on secret material embedded in persisted state. Runner ambient-auth checks surface this boundary by warning when teardown depends on non-ambient credentials supplied through the manifest.

Credential exposure warnings follow the same direction. ork warns when component environment values use keys that look like access mechanisms, such as token, secret, password, private key, credential, access-key, API-key, or known cloud credential environment variables. These warnings name only the keys, not the values.

The planned ork daemon and managed state backend should retain sanitized execution topology and resolve state and runner identities through the control plane, allowing service-initiated teardown without the original manifest checkout.
