---
title: State Backends
description: Configure where ork stores environment state.
---

ork uses one state backend per manifest.

If `state` is omitted, ork uses the local backend with `.ork` as the root.

## Local

```yaml
state:
  backend: local
  config:
    path: .ork
```

Config fields:

| Field | Required | Default | Description |
| --- | --- | --- | --- |
| `path` | No | `.ork` | Local directory where environment state bundles are stored. |

Layout:

```text
<root>/<env-id>/state.json
<root>/<env-id>/artifacts/<component-name>/<artifact-path>
```

After a successful `down`, ork removes `<root>/<env-id>`.

## S3

```yaml
state:
  backend: s3
  auth:
    profile: ork-state
  config:
    bucket: my-ork-state
    prefix: previews
    region: eu-central-1
    server_side_encryption: aws:kms
    kms_key_id: alias/ork
```

Config fields:

| Field | Required | Default | Description |
| --- | --- | --- | --- |
| `bucket` | Yes | None | S3 bucket that stores state objects. |
| `prefix` | No | Empty | Key prefix before `<env-id>/state.json`. Leading and trailing slashes are normalized. |
| `region` | No | AWS SDK default | Region passed to the AWS SDK config loader. |
| `endpoint` | No | AWS S3 | Absolute HTTP or HTTPS endpoint for S3-compatible object storage. Credentials, query parameters, and fragments are not allowed in the URL. |
| `force_path_style` | No | `false` | Use path-style bucket addressing. Commonly required by self-hosted S3-compatible services. |
| `server_side_encryption` | No | Bucket default | Either `AES256` or `aws:kms`. |
| `kms_key_id` | No | Empty | KMS key ID or alias. Requires `server_side_encryption: aws:kms`. |

Auth fields:

| Field | Required | Default | Description |
| --- | --- | --- | --- |
| `profile` | No | Ambient AWS SDK chain | Named profile from the AWS shared credentials and config files. An explicit profile overrides ambient AWS environment credentials for the state backend client only. |
| `access_key_id` | With `secret_access_key` | None | S3-compatible access-key ID. Mutually exclusive with `profile`. |
| `secret_access_key` | With `access_key_id` | None | S3-compatible secret access key. Mutually exclusive with `profile`. |
| `session_token` | No | Empty | Optional session token used with access-key credentials. |

Layout:

```text
s3://<bucket>/<prefix>/<env-id>/state.json
s3://<bucket>/<prefix>/<env-id>/artifacts/<component-name>/<artifact-path>
```

When `auth` is omitted, the S3 backend uses ambient authentication through the AWS SDK default config chain. This includes `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, optional `AWS_SESSION_TOKEN`, shared AWS configuration files, and workload identity when running on AWS. Credentials do not belong in the manifest or endpoint URL.

`auth.profile` selects a profile for the state backend without changing credentials used by local or remote runners. The AWS SDK loads the profile from `~/.aws/credentials` and `~/.aws/config`, or from paths selected by `AWS_SHARED_CREDENTIALS_FILE` and `AWS_CONFIG_FILE`.

Alternatively, provide access-key credentials directly to the state backend client. Interpolate them from the environment instead of committing credential literals:

```yaml
state:
  backend: s3
  auth:
    access_key_id: "${MINIO_ACCESS_KEY_ID}"
    secret_access_key: "${MINIO_SECRET_ACCESS_KEY}"
    # session_token: "${MINIO_SESSION_TOKEN}"
  config:
    bucket: ork-state
    region: us-east-1
    endpoint: "${ORK_STATE_ENDPOINT}"
    force_path_style: true
```

Literal access-key values are accepted, but they make the manifest a secret-bearing document and are not recommended. Explicit access-key credentials override the machine's ambient AWS identity for the state backend client only.

All string values in `state.auth` and `state.config` support `${inputs.name}` and `${ENVIRONMENT_VARIABLE}` interpolation. State interpolation happens before the backend is constructed, so component outputs cannot be referenced. `up`, `down`, and `state inspect` all apply the same resolution rules.

### S3-compatible object storage

Use `endpoint` to store state in services that implement the S3 API, including self-hosted object storage:

```yaml
state:
  backend: s3
  auth:
    profile: ork-minio
  config:
    bucket: ork-state
    prefix: demos
    region: us-east-1
    endpoint: https://objects.demo.tryork.dev
    force_path_style: true
```

For example, the corresponding credentials file can contain:

```ini
[ork-minio]
aws_access_key_id = ork-state
aws_secret_access_key = <secret>
```

The profile name is a user-defined local label. The credential values remain outside the manifest and ork state.

The configured service must support `GetObject`, `PutObject`, `HeadObject`, `ListObjectsV2`, and `DeleteObjects`. `region` is still used for request signing; use the region required by the service, or its documented compatibility value.

`server_side_encryption: aws:kms` and `kms_key_id` are AWS-specific. Omit them unless the selected provider explicitly supports the same request fields. For a self-hosted service, configure encryption and durable storage on the service itself.

After a successful `down`, ork deletes objects under `<prefix>/<env-id>/`.

## Locking

The current backend interface does not implement locking or optimistic revisions. Avoid concurrent writes to the same `env-id`.
