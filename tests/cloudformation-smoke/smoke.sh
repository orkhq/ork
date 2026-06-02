#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ORK_FILE="$SCRIPT_DIR/ork.yml"
TMP_DIR="$(mktemp -d)"
ORK_BIN="$TMP_DIR/ork"

ENV_ID="${ORK_CLOUDFORMATION_SMOKE_ENV:-cf-smoke-one}"
REGION="${ORK_CLOUDFORMATION_SMOKE_REGION:-us-east-1}"
COMPONENT="cf"
STACK_NAME="ork-$ENV_ID-$COMPONENT"
PARAMETER_NAME="/ork/smoke/$STACK_NAME"
STATE_FILE="$REPO_ROOT/.ork/$ENV_ID/state.json"

export AWS_REGION="$REGION"
export AWS_DEFAULT_REGION="$REGION"

cleanup() {
  set +e
  "$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_ID" >/dev/null 2>&1
  aws cloudformation delete-stack --stack-name "$STACK_NAME" --region "$REGION" >/dev/null 2>&1
  aws cloudformation wait stack-delete-complete --stack-name "$STACK_NAME" --region "$REGION" >/dev/null 2>&1
  rm -rf "$REPO_ROOT/.ork/$ENV_ID"
  rm -rf "$SCRIPT_DIR/.workdir/ork/$ENV_ID"
  rmdir "$SCRIPT_DIR/.workdir/ork" "$SCRIPT_DIR/.workdir" >/dev/null 2>&1
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

if [[ "${ORK_RUN_AWS_SMOKE:-}" != "1" ]]; then
  echo "Skipping CloudFormation smoke test."
  echo "Set ORK_RUN_AWS_SMOKE=1 to create and delete a real AWS CloudFormation stack."
  exit 0
fi

if ! command -v aws >/dev/null 2>&1; then
  echo "aws CLI is required for the CloudFormation smoke test" >&2
  exit 1
fi

if ! aws sts get-caller-identity --region "$REGION" >/dev/null; then
  echo "AWS ambient auth is required for the CloudFormation smoke test" >&2
  exit 1
fi

cd "$REPO_ROOT"

echo "Building ork CLI..."
go build -o "$ORK_BIN" ./cmd/ork

echo "Starting $ENV_ID in $REGION..."
"$ORK_BIN" up --file "$ORK_FILE" --env-id "$ENV_ID"

if [[ ! -f "$STATE_FILE" ]]; then
  echo "Expected state file $STATE_FILE to exist" >&2
  exit 1
fi

if ! aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" >/dev/null; then
  echo "Expected CloudFormation stack $STACK_NAME to exist" >&2
  exit 1
fi

if [[ "$(aws ssm get-parameter --name "$PARAMETER_NAME" --region "$REGION" --query 'Parameter.Value' --output text)" != "ork cloudformation smoke" ]]; then
  echo "Expected SSM parameter $PARAMETER_NAME to be created by the stack" >&2
  exit 1
fi

echo "Tearing down $ENV_ID..."
"$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_ID"

if aws cloudformation describe-stacks --stack-name "$STACK_NAME" --region "$REGION" >/dev/null 2>&1; then
  echo "Expected CloudFormation stack $STACK_NAME to be deleted" >&2
  exit 1
fi

if aws ssm get-parameter --name "$PARAMETER_NAME" --region "$REGION" >/dev/null 2>&1; then
  echo "Expected SSM parameter $PARAMETER_NAME to be deleted" >&2
  exit 1
fi

if ! grep -q '"status": "destroyed"' "$STATE_FILE"; then
  echo "Expected Ork component state to be marked destroyed" >&2
  exit 1
fi

echo "CloudFormation smoke test passed"
