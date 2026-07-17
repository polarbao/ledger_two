#!/bin/sh

set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
compose_file="${COMPOSE_FILE:-$script_dir/docker-compose.task53-staging.yml}"
env_file="${ENV_FILE:?ENV_FILE must point to the private Task53 staging env file}"
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
evidence_dir="$runtime_root/evidence/task53-mode-cycle-$timestamp"
health_url="http://127.0.0.1:$app_port/api/healthz"

case "$runtime_root" in
  /*task53* | /*Task53*) ;;
  *) echo "RUNTIME_ROOT must be an isolated absolute Task53 path" >&2; exit 1 ;;
esac
if [ "$app_port" != "38092" ] || [ ! -f "$database" ]; then
  echo "Task53 mode cycle requires the isolated 38092 database" >&2
  exit 1
fi
if [ "$("$sqlite_bin" -readonly "$database" "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")" != "22" ]; then
  echo "Task53 mode cycle requires schema 22" >&2
  exit 1
fi
mkdir -p "$evidence_dir"
"$sqlite_bin" -readonly -separator '|' "$database" "
  SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM transactions;
  SELECT COUNT(*), COALESCE(SUM(share_amount), 0) FROM transaction_splits;
  SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM settlements;
" > "$evidence_dir/before-business-invariants.txt"
"$sqlite_bin" -readonly -separator '|' "$database" "SELECT id, import_hash FROM import_items ORDER BY id;" > "$evidence_dir/before-import-hashes.txt"

for mode in off suggest graded suggest off; do
  IMPORT_CLASSIFICATION_MODE="$mode" "$docker_bin" compose \
    --env-file "$env_file" -f "$compose_file" -p "$project_name" \
    up -d --force-recreate --no-build --remove-orphans
  health=""
  attempt=1
  while [ "$attempt" -le 24 ]; do
    health="$("$wget_bin" -qO- "$health_url" 2>/dev/null || true)"
    if echo "$health" | grep -q '"schema_version":22' \
      && echo "$health" | grep -q '"deployment_channel":"staging"' \
      && echo "$health" | grep -q "\"import_classification_mode\":\"$mode\"" \
      && echo "$health" | grep -q '"db":"ok"'; then
      break
    fi
    sleep 5
    attempt=$((attempt + 1))
  done
  if [ "$attempt" -gt 24 ]; then
    echo "Task53 mode $mode failed health verification: $health" >&2
    exit 1
  fi
  printf '%s\n' "$health" >> "$evidence_dir/health-cycle.jsonl"
done

"$sqlite_bin" -readonly -separator '|' "$database" "
  SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM transactions;
  SELECT COUNT(*), COALESCE(SUM(share_amount), 0) FROM transaction_splits;
  SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM settlements;
" > "$evidence_dir/after-business-invariants.txt"
"$sqlite_bin" -readonly -separator '|' "$database" "SELECT id, import_hash FROM import_items ORDER BY id;" > "$evidence_dir/after-import-hashes.txt"
diff -u "$evidence_dir/before-business-invariants.txt" "$evidence_dir/after-business-invariants.txt" > "$evidence_dir/business-invariants.diff"
diff -u "$evidence_dir/before-import-hashes.txt" "$evidence_dir/after-import-hashes.txt" > "$evidence_dir/import-hashes.diff"

echo "Task53 off -> suggest -> graded -> suggest -> off cycle passed"
echo "evidence_dir=$evidence_dir"
