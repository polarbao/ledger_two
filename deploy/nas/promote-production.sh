#!/bin/sh

set -eu

deployment_root="${1:-/volume1/docker/ledger-two}"
rollback_database="${2:-}"
docker_bin="${DOCKER_BIN:-/usr/local/bin/docker}"
candidate_project="ledger-two-v12-production"
candidate_image="ledger-two:1.2.0-rc"
active_container="ledger-two"
timestamp="$(date +%Y%m%d-%H%M%S)"
rollback_container="ledger-two-v1.1-rollback-$timestamp"
failed_dir="$deployment_root/backups/predeploy/failed-upgrade-$timestamp"

if [ "$(id -u)" -ne 0 ]; then
  echo "run this script with sudo" >&2
  exit 1
fi

if [ -z "$rollback_database" ] || [ ! -f "$rollback_database" ]; then
  echo "rollback database is required and must exist" >&2
  exit 1
fi

cd "$deployment_root"

if ! "$docker_bin" image inspect "$candidate_image" >/dev/null 2>&1; then
  echo "candidate image not found: $candidate_image" >&2
  exit 1
fi

if ! "$docker_bin" inspect "$active_container" >/dev/null 2>&1; then
  echo "active production container not found: $active_container" >&2
  exit 1
fi

if "$docker_bin" inspect "$rollback_container" >/dev/null 2>&1; then
  echo "rollback container already exists: $rollback_container" >&2
  exit 1
fi

rollback() {
  echo "candidate verification failed; restoring schema 12 database and v1.1 container" >&2
  "$docker_bin" compose -p "$candidate_project" down >/dev/null 2>&1 || true
  mkdir -p "$failed_dir"
  if [ -f data/ledger.db ]; then
    cp -p data/ledger.db "$failed_dir/ledger.db.failed"
  fi
  if [ -f data/ledger.db-wal ]; then
    mv data/ledger.db-wal "$failed_dir/ledger.db-wal.failed"
  fi
  if [ -f data/ledger.db-shm ]; then
    mv data/ledger.db-shm "$failed_dir/ledger.db-shm.failed"
  fi
  cp -p "$rollback_database" data/ledger.db
  "$docker_bin" rename "$rollback_container" "$active_container"
  "$docker_bin" start "$active_container" >/dev/null
  echo "rollback completed; failed candidate database saved in $failed_dir" >&2
}

"$docker_bin" stop "$active_container" >/dev/null
"$docker_bin" rename "$active_container" "$rollback_container"

if ! "$docker_bin" compose -p "$candidate_project" up -d --no-build; then
  rollback
  exit 1
fi

health=""
attempt=1
while [ "$attempt" -le 18 ]; do
  health="$(wget -qO- http://127.0.0.1:38088/api/healthz 2>/dev/null || true)"
  if echo "$health" | grep -q '"version":"1.2.0-rc"' \
    && echo "$health" | grep -q '"schema_version":18' \
    && echo "$health" | grep -q '"deployment_channel":"production"' \
    && echo "$health" | grep -q '"db":"ok"'; then
    echo "production promotion succeeded"
    echo "health=$health"
    echo "rollback_container=$rollback_container"
    "$docker_bin" compose -p "$candidate_project" ps
    exit 0
  fi
  sleep 5
  attempt=$((attempt + 1))
done

echo "last_health=$health" >&2
rollback
exit 1
