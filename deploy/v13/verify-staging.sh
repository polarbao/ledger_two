#!/bin/sh

set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
compose_file="${COMPOSE_FILE:-$script_dir/docker-compose.staging.yml}"
env_file="${ENV_FILE:?ENV_FILE must point to the private v1.3 staging env file}"
docker_bin="${DOCKER_BIN:-docker}"
sqlite_bin="${SQLITE_BIN:-sqlite3}"
wget_bin="${WGET_BIN:-wget}"
sleep_bin="${SLEEP_BIN:-sleep}"
expected_version="${EXPECTED_APP_VERSION:-1.3.0-rc}"
expected_schema="${EXPECTED_SCHEMA_VERSION:-21}"
expected_port="${EXPECTED_APP_PORT:-38091}"
max_attempts="${MAX_ATTEMPTS:-24}"
timestamp="$(date +%Y%m%d-%H%M%S)"

read_env() {
  key="$1"
  value="$(grep -E "^${key}=" "$env_file" | tail -n 1 | cut -d= -f2- || true)"
  if [ -z "$value" ]; then
    echo "$key is required in $env_file" >&2
    exit 1
  fi
  printf '%s' "$value"
}

runtime_root="$(read_env RUNTIME_ROOT)"
candidate_image="$(read_env CANDIDATE_IMAGE)"
app_port="$(read_env APP_PORT)"
container_name="$(read_env CONTAINER_NAME)"
project_name="$(read_env COMPOSE_PROJECT_NAME)"
database="$runtime_root/data/ledger.db"
evidence_dir="$runtime_root/evidence/$timestamp"
preupgrade_dir="$runtime_root/backups/preupgrade/$timestamp"
preupgrade_database="$preupgrade_dir/ledger-schema19.db"
health_url="http://127.0.0.1:$app_port/api/healthz"

case "$runtime_root" in
  /*) ;;
  *)
    echo "RUNTIME_ROOT must be an absolute path: $runtime_root" >&2
    exit 1
    ;;
esac
case "$runtime_root" in
  *v13* | *v1.3*) ;;
  *)
    echo "RUNTIME_ROOT must be an explicitly named v13/v1.3 directory: $runtime_root" >&2
    exit 1
    ;;
esac
case "$candidate_image" in
  *:latest | latest)
    echo "CANDIDATE_IMAGE must use a fixed tag, not latest" >&2
    exit 1
    ;;
  ledger-two:1.3.0-rc-task50.6-*) ;;
  *)
    echo "unexpected v1.3 candidate tag: $candidate_image" >&2
    exit 1
    ;;
esac
if [ "$app_port" != "$expected_port" ]; then
  echo "isolated v1.3 staging port must be $expected_port, got $app_port" >&2
  exit 1
fi
if [ "$container_name" = "ledger-two" ] || [ "$project_name" = "ledger-two-local" ]; then
  echo "v1.3 staging must not reuse the v1.2 local container or project" >&2
  exit 1
fi
if [ ! -f "$database" ]; then
  echo "isolated staging database is missing: $database" >&2
  exit 1
fi

mkdir -p "$runtime_root/backups" "$runtime_root/uploads" "$runtime_root/logs" "$evidence_dir" "$preupgrade_dir"
before_schema="$("$sqlite_bin" -readonly "$database" "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")"
before_quick_check="$("$sqlite_bin" -readonly "$database" "PRAGMA quick_check;")"
if [ "$before_quick_check" != "ok" ]; then
  echo "staging source database failed quick_check: $before_quick_check" >&2
  exit 1
fi
if [ "$before_schema" != "19" ] && [ "$before_schema" != "$expected_schema" ]; then
  echo "v1.3 candidate only accepts schema 19 or $expected_schema, got $before_schema" >&2
  exit 1
fi

if [ "$before_schema" = "19" ]; then
  "$sqlite_bin" "$database" ".backup '$preupgrade_database'"
  "$sqlite_bin" -readonly "$preupgrade_database" "PRAGMA quick_check;" | grep -qx "ok"
  sha256sum "$preupgrade_database" > "$preupgrade_database.sha256"
fi

compose() {
  "$docker_bin" compose \
    --env-file "$env_file" \
    -f "$compose_file" \
    -p "$project_name" \
    "$@"
}

if ! "$docker_bin" image inspect "$candidate_image" >/dev/null 2>&1; then
  echo "candidate image is not available locally: $candidate_image" >&2
  exit 1
fi
compose up -d --no-build

health=""
attempt=1
while [ "$attempt" -le "$max_attempts" ]; do
  health="$("$wget_bin" -qO- "$health_url" 2>/dev/null || true)"
  if echo "$health" | grep -q "\"version\":\"$expected_version\"" \
    && echo "$health" | grep -q "\"schema_version\":$expected_schema" \
    && echo "$health" | grep -q '"deployment_channel":"staging"' \
    && echo "$health" | grep -q '"import_xlsx_enabled":true' \
    && echo "$health" | grep -q '"db":"ok"'; then
    break
  fi
  "$sleep_bin" 5
  attempt=$((attempt + 1))
done
if [ "$attempt" -gt "$max_attempts" ]; then
  echo "v1.3 staging health verification timed out: $health" >&2
  compose logs --no-color > "$evidence_dir/container.log" 2>&1 || true
  exit 1
fi

after_schema="$("$sqlite_bin" -readonly "$database" "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")"
after_quick_check="$("$sqlite_bin" -readonly "$database" "PRAGMA quick_check;")"
if [ "$after_schema" != "$expected_schema" ] || [ "$after_quick_check" != "ok" ]; then
  echo "v1.3 staging database verification failed: schema=$after_schema quick_check=$after_quick_check" >&2
  exit 1
fi

printf '%s\n' "$health" > "$evidence_dir/health.json"
"$docker_bin" image inspect "$candidate_image" > "$evidence_dir/image-inspect.json"
compose ps > "$evidence_dir/compose-ps.txt"
{
  echo "candidate_image=$candidate_image"
  echo "container_name=$container_name"
  echo "runtime_root=$runtime_root"
  echo "database=$database"
  echo "before_schema=$before_schema"
  echo "after_schema=$after_schema"
  echo "quick_check=$after_quick_check"
  echo "health_url=$health_url"
  if [ -f "$preupgrade_database.sha256" ]; then
    echo "preupgrade_database=$preupgrade_database"
    echo "preupgrade_sha256=$(cut -d' ' -f1 "$preupgrade_database.sha256")"
  fi
} > "$evidence_dir/manifest.txt"

echo "v1.3 staging verification passed"
echo "evidence_dir=$evidence_dir"
echo "health=$health"
