#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ORK_FILE="$SCRIPT_DIR/ork.yml"
TMP_DIR="$(mktemp -d)"
ORK_BIN="$TMP_DIR/ork"
ENV_ID="${ORK_SCRIPT_SMOKE_ENV_ID:-script-smoke}"

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

echo "Running script smoke..."
"$ORK_BIN" up --file "$ORK_FILE" --env-id "$ENV_ID"

STATE_FILE="$REPO_ROOT/.ork/$ENV_ID/state.json"
grep -q '"token": "abc"' "$STATE_FILE"
grep -q '"url": "http://localhost:8080"' "$STATE_FILE"
grep -q '"seen_token": "abc"' "$STATE_FILE"

echo "Tearing down script smoke..."
"$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_ID"

echo "Script smoke test passed"
