#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ORK_FILE="$SCRIPT_DIR/ork.yml"
TMP_DIR="$(mktemp -d)"
ORK_BIN="$TMP_DIR/ork"

export PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"

ENV_ONE="${ORK_SMOKE_ENV_ONE:-smoke-one}"
ENV_TWO="${ORK_SMOKE_ENV_TWO:-smoke-two}"
COMPONENT="web"
SERVICE="web"
SERVICE_PORT="80"

cleanup() {
  set +e
  "$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_ONE" >/dev/null 2>&1
  "$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_TWO" >/dev/null 2>&1
  rm -rf "$REPO_ROOT/.ork/$ENV_ONE" "$REPO_ROOT/.ork/$ENV_TWO"
  rm -rf "$SCRIPT_DIR/.workdir/ork/$ENV_ONE" "$SCRIPT_DIR/.workdir/ork/$ENV_TWO"
  rmdir "$SCRIPT_DIR/.workdir/ork" "$SCRIPT_DIR/.workdir" >/dev/null 2>&1
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

project_name() {
  local env_id="$1"
  printf "ork_%s_%s" "$env_id" "$COMPONENT"
}

service_url() {
  local env_id="$1"
  local project
  local mapped
  project="$(project_name "$env_id")"
  mapped="$(docker compose -p "$project" port "$SERVICE" "$SERVICE_PORT")"
  printf "http://%s" "${mapped/0.0.0.0/127.0.0.1}"
}

wait_until_reachable() {
  local url="$1"
  local attempts=30

  for _ in $(seq 1 "$attempts"); do
    if curl -fsS "$url" >/dev/null; then
      return 0
    fi
    sleep 1
  done

  echo "Timed out waiting for $url" >&2
  return 1
}

assert_project_removed() {
  local env_id="$1"
  local project
  local remaining
  project="$(project_name "$env_id")"
  remaining="$(docker ps -a --filter "label=com.docker.compose.project=$project" --format '{{.ID}}')"
  if [[ -n "$remaining" ]]; then
    echo "Expected project $project to be removed, found containers:" >&2
    docker ps -a --filter "label=com.docker.compose.project=$project" >&2
    return 1
  fi
}

cd "$REPO_ROOT"

echo "Building ork CLI..."
go build -o "$ORK_BIN" ./cmd/ork

echo "Starting $ENV_ONE..."
"$ORK_BIN" up --file "$ORK_FILE" --env-id "$ENV_ONE"

echo "Starting $ENV_TWO..."
"$ORK_BIN" up --file "$ORK_FILE" --env-id "$ENV_TWO"

URL_ONE="$(service_url "$ENV_ONE")"
URL_TWO="$(service_url "$ENV_TWO")"

if [[ "$URL_ONE" == "$URL_TWO" ]]; then
  echo "Expected different published ports, got $URL_ONE for both envs" >&2
  exit 1
fi

echo "Checking $ENV_ONE at $URL_ONE..."
wait_until_reachable "$URL_ONE"

echo "Checking $ENV_TWO at $URL_TWO..."
wait_until_reachable "$URL_TWO"

echo "Tearing down $ENV_ONE..."
"$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_ONE"

echo "Tearing down $ENV_TWO..."
"$ORK_BIN" down --file "$ORK_FILE" --env-id "$ENV_TWO"

assert_project_removed "$ENV_ONE"
assert_project_removed "$ENV_TWO"

echo "Docker Compose smoke test passed:"
echo "  $ENV_ONE -> $URL_ONE"
echo "  $ENV_TWO -> $URL_TWO"
