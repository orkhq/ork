#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ORK_FILE="$SCRIPT_DIR/ork.yml"
TMP_DIR="$(mktemp -d)"
ORK_BIN="$TMP_DIR/ork"

ENV_ID="${ORK_TERRAFORM_SMOKE_ENV:-tf-smoke-one}"
COMPONENT="tf"
WORK_DIR="$SCRIPT_DIR/.workdir/ork/$ENV_ID/$COMPONENT"
STATE_FILE="$REPO_ROOT/.ork/$ENV_ID/state.json"

cleanup() {
  set +e
  "$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_ID" >/dev/null 2>&1
  rm -rf "$REPO_ROOT/.ork/$ENV_ID"
  rm -rf "$SCRIPT_DIR/.workdir/ork/$ENV_ID"
  rmdir "$SCRIPT_DIR/.workdir/ork" "$SCRIPT_DIR/.workdir" >/dev/null 2>&1
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cd "$REPO_ROOT"

echo "Building ork CLI..."
go build -o "$ORK_BIN" ./cmd/ork

echo "Starting $ENV_ID..."
"$ORK_BIN" up --file "$ORK_FILE" --env-id "$ENV_ID"

if [[ ! -f "$STATE_FILE" ]]; then
  echo "Expected state file $STATE_FILE to exist" >&2
  exit 1
fi

if ! terraform -chdir="$WORK_DIR" state list | grep -q '^terraform_data.smoke$'; then
  echo "Expected terraform_data.smoke in Terraform state" >&2
  exit 1
fi

echo "Removing runner workdir to verify state artifact restore..."
rm -rf "$WORK_DIR"

echo "Tearing down $ENV_ID..."
"$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_ID"

if terraform -chdir="$WORK_DIR" state list | grep -q '^terraform_data.smoke$'; then
  echo "Expected terraform_data.smoke to be destroyed" >&2
  exit 1
fi

if ! grep -q '"status": "destroyed"' "$STATE_FILE"; then
  echo "Expected ork component state to be marked destroyed" >&2
  exit 1
fi

echo "Terraform smoke test passed"
