#!/bin/sh

set -eu

deployment_root="${1:-/volume1/docker/ledger-two-staging}"
rollback_database="${2:-}"
docker_bin="${DOCKER_BIN:-/usr/local/bin/docker}"
sqlite_bin="${SQLITE_BIN:-sqlite3}"
wget_bin="${WGET_BIN:-wget}"
sleep_bin="${SLEEP_BIN:-sleep}"
max_attempts="${MAX_ATTEMPTS:-18}"
candidate_project="${CANDIDATE_PROJECT:-ledger-two-v12-staging-schema19}"
candidate_image="${CANDIDATE_IMAGE:-ledger-two:1.2.0-rc}"
active_container="${ACTIVE_CONTAINER:-ledger-two-staging}"
health_url="${HEALTH_URL:-http://127.0.0.1:38089/api/healthz}"
timestamp="$(date +%Y%m%d-%H%M%S)"
rollback_container="ledger-two-staging-schema18-rollback-$timestamp"
failed_dir="$deployment_root/backups/predeploy/failed-schema19-$timestamp"

if [ "$(id -u)" -ne 0 ]; then
  echo "run this script with sudo" >&2
  exit 1
fi

if [ -z "$rollback_database" ] || [ ! -f "$rollback_database" ]; then
  echo "schema 18 rollback database is required and must exist" >&2
  exit 1
fi

if [ ! -x "$docker_bin" ]; then
  echo "docker binary not found or not executable: $docker_bin" >&2
  exit 1
fi

if ! command -v "$sqlite_bin" >/dev/null 2>&1; then
  echo "sqlite3 is required for staging promotion checks" >&2
  exit 1
fi

cd "$deployment_root"

if [ ! -f .env ] || [ ! -f docker-compose.yml ] || [ ! -f data/ledger.db ]; then
  echo "staging deployment files are incomplete under $deployment_root" >&2
  exit 1
fi

if ! grep -q '^DEPLOYMENT_CHANNEL=staging$' .env; then
  echo "DEPLOYMENT_CHANNEL must be staging" >&2
  exit 1
fi

if ! grep -q '^IMPORT_XLSX_ENABLED=true$' .env; then
  echo "IMPORT_XLSX_ENABLED must be explicitly true for schema 19 staging" >&2
  exit 1
fi

if ! grep -q '^APP_PORT=38089$' .env; then
  echo "APP_PORT must remain 38089 for the isolated staging instance" >&2
  exit 1
fi

if ! grep -q 'IMPORT_XLSX_ENABLED' docker-compose.yml; then
  echo "docker-compose.yml does not pass IMPORT_XLSX_ENABLED" >&2
  exit 1
fi

database_check="$($sqlite_bin -readonly data/ledger.db 'PRAGMA quick_check;')"
rollback_check="$($sqlite_bin -readonly "$rollback_database" 'PRAGMA quick_check;')"
current_schema="$($sqlite_bin -readonly data/ledger.db "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")"
rollback_schema="$($sqlite_bin -readonly "$rollback_database" "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")"

if [ "$database_check" != "ok" ] || [ "$rollback_check" != "ok" ]; then
  echo "current or rollback database failed quick_check" >&2
  exit 1
fi

if [ "$current_schema" != "18" ] || [ "$rollback_schema" != "18" ]; then
  echo "current and rollback databases must both be schema 18" >&2
  exit 1
fi

before_health="$("$wget_bin" -qO- "$health_url" 2>/dev/null || true)"
if ! echo "$before_health" | grep -q '"schema_version":18' \
  || ! echo "$before_health" | grep -q '"deployment_channel":"staging"' \
  || ! echo "$before_health" | grep -q '"db":"ok"'; then
  echo "active staging health is not the expected schema 18 baseline: $before_health" >&2
  exit 1
fi

if ! "$docker_bin" image inspect "$candidate_image" >/dev/null 2>&1; then
  echo "candidate image not found: $candidate_image" >&2
  exit 1
fi

if ! "$docker_bin" inspect "$active_container" >/dev/null 2>&1; then
  echo "active staging container not found: $active_container" >&2
  exit 1
fi

if "$docker_bin" inspect "$rollback_container" >/dev/null 2>&1; then
  echo "rollback container already exists: $rollback_container" >&2
  exit 1
fi

rollback() {
  echo "schema 19 staging verification failed; restoring schema 18" >&2
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
  if "$docker_bin" inspect "$rollback_container" >/dev/null 2>&1; then
    "$docker_bin" rename "$rollback_container" "$active_container"
  fi
  "$docker_bin" start "$active_container" >/dev/null
  echo "rollback completed; failed database saved in $failed_dir" >&2
}

"$docker_bin" stop "$active_container" >/dev/null
"$docker_bin" rename "$active_container" "$rollback_container"

if ! "$docker_bin" compose -p "$candidate_project" up -d --no-build; then
  rollback
  exit 1
fi

health=""
attempt=1
while [ "$attempt" -le "$max_attempts" ]; do
  health="$("$wget_bin" -qO- "$health_url" 2>/dev/null || true)"
  if echo "$health" | grep -q '"version":"1.2.0-rc"' \
    && echo "$health" | grep -q '"schema_version":19' \
    && echo "$health" | grep -q '"deployment_channel":"staging"' \
    && echo "$health" | grep -q '"import_xlsx_enabled":true' \
    && echo "$health" | grep -q '"db":"ok"'; then
    migrated_check="$($sqlite_bin -readonly data/ledger.db 'PRAGMA quick_check;')"
    if [ "$migrated_check" != "ok" ]; then
      echo "schema 19 database failed quick_check: $migrated_check" >&2
      rollback
      exit 1
    fi
    echo "staging schema 19 promotion succeeded"
    echo "before_health=$before_health"
    echo "health=$health"
    echo "rollback_database=$rollback_database"
    echo "rollback_container=$rollback_container"
    "$docker_bin" compose -p "$candidate_project" ps
    exit 0
  fi
  "$sleep_bin" 5
  attempt=$((attempt + 1))
done

echo "last_health=$health" >&2
rollback
exit 1
