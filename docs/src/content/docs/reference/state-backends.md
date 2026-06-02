---
title: State Backends
description: Configure where Ork stores environment state.
---

Ork uses one state backend per manifest.

If `state` is omitted, Ork uses the local backend with `.ork` as the root.

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

After a successful `down`, Ork removes `<root>/<env-id>`.

## S3

```yaml
state:
  backend: s3
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
| `server_side_encryption` | No | Bucket default | Either `AES256` or `aws:kms`. |
| `kms_key_id` | No | Empty | KMS key ID or alias. Requires `server_side_encryption: aws:kms`. |

Layout:

```text
s3://<bucket>/<prefix>/<env-id>/state.json
s3://<bucket>/<prefix>/<env-id>/artifacts/<component-name>/<artifact-path>
```

The S3 backend uses ambient AWS authentication through the AWS SDK default config chain.

After a successful `down`, Ork deletes objects under `<prefix>/<env-id>/`.

## Locking

The current backend interface does not implement locking or optimistic revisions. Avoid concurrent writes to the same `env-id`.
