# Security policy

ork is alpha software that executes commands, copies files, connects to remote runners, and persists operational state. Do not use it as a secret manager, and review manifests and scripts with the same care as other infrastructure automation.

## Supported versions

Security fixes currently target the latest commit on the default branch. There are no supported release branches while the project remains alpha.

## Reporting a vulnerability

Please use GitHub's private vulnerability reporting for `orkhq/ork` when it is available. Do not open a public issue for a suspected vulnerability that could expose credentials, state, remote execution, or infrastructure resources.

Include:

- affected version or commit;
- reproduction steps;
- expected and observed behavior;
- potential impact;
- any suggested mitigation.

If private reporting is unavailable, contact the repository maintainers privately before publishing details.

## Security boundaries

- State contains operational data and may contain tool artifacts such as Terraform state. Protect the backend accordingly.
- Sensitive component outputs are intentionally excluded from persisted state.
- State and runner identities are separate authentication contexts.
- Remote runner host keys should be verified; insecure mode is for disposable development environments only.
- Script and hook output is user-controlled and may disclose secrets if the invoked process prints them.

See the [security model](notes/architecture/security-model.md) and the public [security documentation](docs/src/content/docs/security/overview.md) for details.
