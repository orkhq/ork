#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TMP_DIR="$(mktemp -d)"
ORK_BIN="$TMP_DIR/ork"
ORK_FILE="$TMP_DIR/ork.yml"

ENV_ID="${ORK_S3_STATE_SMOKE_ENV:-s3-state-smoke-one}"
REGION="${ORK_S3_STATE_SMOKE_REGION:-${AWS_REGION:-${AWS_DEFAULT_REGION:-us-east-1}}}"
BUCKET="${ORK_S3_STATE_SMOKE_BUCKET:-}"
PREFIX="${ORK_S3_STATE_SMOKE_PREFIX:-ork-smoke/state}"
COMPONENT="tf"
WORK_DIR="$SCRIPT_DIR/.workdir/ork/$ENV_ID/$COMPONENT"
STATE_KEY="$PREFIX/$ENV_ID/state.json"
ARTIFACT_KEY="$PREFIX/$ENV_ID/artifacts/$COMPONENT/terraform.tfstate"

export AWS_REGION="$REGION"
export AWS_DEFAULT_REGION="$REGION"

cleanup() {
  set +e
  "$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_ID" >/dev/null 2>&1
  rm -rf "$SCRIPT_DIR/.workdir/ork/$ENV_ID"
  rmdir "$SCRIPT_DIR/.workdir/ork" "$SCRIPT_DIR/.workdir" >/dev/null 2>&1
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

if [[ "${ORK_RUN_AWS_SMOKE:-}" != "1" ]]; then
  echo "Skipping S3 state backend smoke test."
  echo "Set ORK_RUN_AWS_SMOKE=1 and ORK_S3_STATE_SMOKE_BUCKET=<bucket> to use a real S3 bucket."
  exit 0
fi

if [[ -z "$BUCKET" ]]; then
  echo "ORK_S3_STATE_SMOKE_BUCKET is required for the S3 state backend smoke test" >&2
  exit 1
fi

if ! command -v aws >/dev/null 2>&1; then
  echo "aws CLI is required for the S3 state backend smoke test" >&2
  exit 1
fi

if ! aws sts get-caller-identity --region "$REGION" >/dev/null; then
  echo "AWS ambient auth is required for the S3 state backend smoke test" >&2
  exit 1
fi

cat > "$ORK_FILE" <<YAML
version: ork/1.0

metadata:
  id: s3-state-smoke
  description: S3 state backend smoke test
  owner:
    name: ork
    email: ork@example.com

state:
  backend: s3
  config:
    bucket: "$BUCKET"
    prefix: "$PREFIX"
    region: "$REGION"
    server_side_encryption: AES256

runners:
  local:
    type: local
    config: {}

components:
  tf:
    type: terraform
    runner: local
    workdir: ./tests/s3-state-smoke/.workdir
    source:
      embedded: |
        terraform {
          required_version = ">= 1.4.0"
        }

        resource "terraform_data" "smoke" {
          input = "ork s3 state smoke"
        }
YAML

cd "$REPO_ROOT"

echo "Building ork CLI..."
go build -o "$ORK_BIN" ./cmd/ork

echo "Starting $ENV_ID with S3 state backend s3://$BUCKET/$PREFIX..."
"$ORK_BIN" up --file "$ORK_FILE" --env-id "$ENV_ID"

if ! aws s3api head-object --bucket "$BUCKET" --key "$STATE_KEY" --region "$REGION" >/dev/null; then
  echo "Expected S3 state object s3://$BUCKET/$STATE_KEY to exist" >&2
  exit 1
fi

if ! aws s3api head-object --bucket "$BUCKET" --key "$ARTIFACT_KEY" --region "$REGION" >/dev/null; then
  echo "Expected S3 artifact object s3://$BUCKET/$ARTIFACT_KEY to exist" >&2
  exit 1
fi

if ! terraform -chdir="$WORK_DIR" state list | grep -q '^terraform_data.smoke$'; then
  echo "Expected terraform_data.smoke in Terraform state" >&2
  exit 1
fi

echo "Removing runner workdir to verify S3 artifact restore..."
rm -rf "$WORK_DIR"

echo "Tearing down $ENV_ID..."
"$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_ID"

if terraform -chdir="$WORK_DIR" state list | grep -q '^terraform_data.smoke$'; then
  echo "Expected terraform_data.smoke to be destroyed" >&2
  exit 1
fi

STATE_JSON="$(aws s3 cp "s3://$BUCKET/$STATE_KEY" - --region "$REGION")"
if ! grep -q '"status": "destroyed"' <<<"$STATE_JSON"; then
  echo "Expected S3 ork component state to be marked destroyed" >&2
  exit 1
fi

echo "S3 state backend smoke test passed"
