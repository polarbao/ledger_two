#!/bin/sh

set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
compose_file="${COMPOSE_FILE:-$script_dir/docker-compose.task53-staging.yml}"
rollback_compose_file="${ROLLBACK_COMPOSE_FILE:-$script_dir/docker-compose.staging.yml}"
env_file="${ENV_FILE:?ENV_FILE must point to the private Task53 staging env file}"
rollback_database="${ROLLBACK_DATABASE:?ROLLBACK_DATABASE must be the verified schema 21 backup}"
rollback_image="${ROLLBACK_IMAGE:?ROLLBACK_IMAGE must be a fixed Task50 schema 21 image}"
docker_bin="${DOCKER_BIN:-docker}"
sqlite_bin="${SQLITE_BIN:-sqlite3}"
wget_bin="${WGET_BIN:-wget}"
timestamp="$(date +%Y%m%d-%H%M%S)"

read_env() {
  grep -E "^$1=" "$env_file" | tail -n 1 | cut -d= -f2-
}

runtime_root="$(read_env RUNTIME_ROOT)"
project_name="$(read_env COMPOSE_PROJECT_NAME)"
app_port="$(read_env APP_PORT)"
database="$runtime_root/data/ledger.db"
failed_dir="$runtime_root/backups/failed-schema22/$timestamp"
health_url="http://127.0.0.1:$app_port/api/healthz"

case "$runtime_root" in
  /*task53* | /*Task53*) ;;
  *) echo "RUNTIME_ROOT must be the isolated Task53 staging directory" >&2; exit 1 ;;
esac
case "$rollback_image" in
  ledger-two:1.3.0-rc-task50.6-*) ;;
  *) echo "ROLLBACK_IMAGE must be a fixed Task50.6 schema 21 image" >&2; exit 1 ;;
esac
if [ ! -f "$rollback_database" ] || [ ! -f "$rollback_database.sha256" ]; then
  echo "verified rollback database and checksum are required" >&2
  exit 1
fi
(cd "$(dirname "$rollback_database")" && sha256sum -c "$(basename "$rollback_database").sha256")
if [ "$("$sqlite_bin" -readonly "$rollback_database" "PRAGMA quick_check;")" != "ok" ] || \
  [ "$("$sqlite_bin" -readonly "$rollback_database" "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")" != "21" ]; then
  echo "Task53 rollback requires a healthy schema 21 backup" >&2
  exit 1
fi

task53_compose() {
  "$docker_bin" compose --env-file "$env_file" -f "$compose_file" -p "$project_name" "$@"
}
rollback_compose() {
  CANDIDATE_IMAGE="$rollback_image" IMPORT_CLASSIFICATION_MODE=off "$docker_bin" compose \
    --env-file "$env_file" -f "$rollback_compose_file" -p "$project_name" "$@"
}
task53_compose down
mkdir -p "$failed_dir"
for suffix in "" "-wal" "-shm"; do
  if [ -f "$database$suffix" ]; then
    mv "$database$suffix" "$failed_dir/ledger.db$suffix"
  fi
done
cp -p "$rollback_database" "$database"
rollback_compose up -d --no-build --remove-orphans

health=""
attempt=1
while [ "$attempt" -le 24 ]; do
  health="$("$wget_bin" -qO- "$health_url" 2>/dev/null || true)"
  if echo "$health" | grep -q '"version":"1.3.0-rc"' \
    && echo "$health" | grep -q '"schema_version":21' \
    && echo "$health" | grep -q '"deployment_channel":"staging"' \
    && echo "$health" | grep -q '"db":"ok"'; then
    break
  fi
  sleep 5
  attempt=$((attempt + 1))
done
if [ "$attempt" -gt 24 ]; then
  echo "paired Task50/schema21 rollback failed: $health" >&2
  exit 1
fi
printf '%s\n' "$health" > "$failed_dir/rollback-health.json"
echo "Task53 paired staging rollback passed"
echo "failed_schema22_dir=$failed_dir"
