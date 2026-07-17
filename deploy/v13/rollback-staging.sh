#!/bin/sh

set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
compose_file="${COMPOSE_FILE:-$script_dir/docker-compose.staging.yml}"
env_file="${ENV_FILE:?ENV_FILE must point to the private v1.3 staging env file}"
rollback_database="${ROLLBACK_DATABASE:?ROLLBACK_DATABASE must be the verified schema 19 backup}"
rollback_image="${ROLLBACK_IMAGE:?ROLLBACK_IMAGE must be a fixed v1.2 schema 19 image}"
docker_bin="${DOCKER_BIN:-docker}"
sqlite_bin="${SQLITE_BIN:-sqlite3}"
wget_bin="${WGET_BIN:-wget}"
sleep_bin="${SLEEP_BIN:-sleep}"
timestamp="$(date +%Y%m%d-%H%M%S)"

read_env() {
  key="$1"
  grep -E "^${key}=" "$env_file" | tail -n 1 | cut -d= -f2-
}

runtime_root="$(read_env RUNTIME_ROOT)"
project_name="$(read_env COMPOSE_PROJECT_NAME)"
container_name="$(read_env CONTAINER_NAME)"
app_port="$(read_env APP_PORT)"
database="$runtime_root/data/ledger.db"
failed_dir="$runtime_root/backups/failed-schema21/$timestamp"
health_url="http://127.0.0.1:$app_port/api/healthz"

case "$runtime_root" in
  /*v13* | /*v1.3*) ;;
  *)
    echo "RUNTIME_ROOT must be the isolated v13/v1.3 staging directory: $runtime_root" >&2
    exit 1
    ;;
esac
case "$rollback_image" in
  *:latest | latest)
    echo "ROLLBACK_IMAGE must use a fixed tag" >&2
    exit 1
    ;;
esac
if [ ! -f "$rollback_database" ]; then
  echo "rollback database is missing: $rollback_database" >&2
  exit 1
fi
if [ "$("$sqlite_bin" -readonly "$rollback_database" "PRAGMA quick_check;")" != "ok" ]; then
  echo "rollback database failed quick_check" >&2
  exit 1
fi
rollback_schema="$("$sqlite_bin" -readonly "$rollback_database" "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")"
if [ "$rollback_schema" != "19" ]; then
  echo "old v1.2 image is blocked until a schema 19 database is restored; got schema $rollback_schema" >&2
  exit 1
fi
if [ ! -f "$rollback_database.sha256" ]; then
  echo "rollback checksum file is missing: $rollback_database.sha256" >&2
  exit 1
fi
(cd "$(dirname "$rollback_database")" && sha256sum -c "$(basename "$rollback_database").sha256")

compose() {
  "$docker_bin" compose \
    --env-file "$env_file" \
    -f "$compose_file" \
    -p "$project_name" \
    "$@"
}

compose down
mkdir -p "$failed_dir"
for suffix in "" "-wal" "-shm"; do
  if [ -f "$database$suffix" ]; then
    mv "$database$suffix" "$failed_dir/ledger.db$suffix"
  fi
done
cp -p "$rollback_database" "$database"

CANDIDATE_IMAGE="$rollback_image" compose up -d --no-build
health=""
attempt=1
while [ "$attempt" -le 24 ]; do
  health="$("$wget_bin" -qO- "$health_url" 2>/dev/null || true)"
  if echo "$health" | grep -q '"version":"1.2.0-rc"' \
    && echo "$health" | grep -q '"schema_version":19' \
    && echo "$health" | grep -q '"deployment_channel":"staging"' \
    && echo "$health" | grep -q '"db":"ok"'; then
    break
  fi
  "$sleep_bin" 5
  attempt=$((attempt + 1))
done
if [ "$attempt" -gt 24 ]; then
  echo "paired v1.2/schema19 rollback failed: $health" >&2
  exit 1
fi

{
  echo "rollback_image=$rollback_image"
  echo "rollback_database=$rollback_database"
  echo "failed_schema21_dir=$failed_dir"
  echo "container_name=$container_name"
  echo "health=$health"
} > "$failed_dir/rollback-manifest.txt"

echo "paired staging rollback passed"
echo "failed_schema21_dir=$failed_dir"
echo "health=$health"
