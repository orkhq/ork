#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ORCH_FILE="$SCRIPT_DIR/orch.yml"
TMP_DIR="$(mktemp -d)"
ORCH_BIN="$TMP_DIR/orch"
ENV_ID="${ORCH_SCRIPT_SMOKE_ENV_ID:-script-smoke}"

cleanup() {
  set +e
  "$ORCH_BIN" down --file "$ORCH_FILE" --env-id "$ENV_ID" >/dev/null 2>&1
  rm -rf "$REPO_ROOT/.orch/$ENV_ID"
  rm -rf "$SCRIPT_DIR/.workdir/orch/$ENV_ID"
  rmdir "$SCRIPT_DIR/.workdir/orch" "$SCRIPT_DIR/.workdir" >/dev/null 2>&1
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cd "$REPO_ROOT"

echo "Building orch CLI..."
go build -o "$ORCH_BIN" ./cmd/orch

echo "Running script smoke..."
"$ORCH_BIN" up --file "$ORCH_FILE" --env-id "$ENV_ID"

STATE_FILE="$REPO_ROOT/.orch/$ENV_ID/state.json"
grep -q '"token": "abc"' "$STATE_FILE"
grep -q '"url": "http://localhost:8080"' "$STATE_FILE"
grep -q '"seen_token": "abc"' "$STATE_FILE"

echo "Tearing down script smoke..."
"$ORCH_BIN" down --file "$ORCH_FILE" --env-id "$ENV_ID"

echo "Script smoke test passed"
