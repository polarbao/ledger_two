#!/bin/sh

set -eu
umask 077

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
compose_file="${COMPOSE_FILE:-$script_dir/docker-compose.task53-staging.yml}"
env_file="${ENV_FILE:?ENV_FILE must point to the private Task53 staging env file}"
docker_bin="${DOCKER_BIN:-docker}"
sqlite_bin="${SQLITE_BIN:-sqlite3}"
wget_bin="${WGET_BIN:-wget}"
expected_version="${EXPECTED_APP_VERSION:-1.3.0-rc}"
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
classification_mode="$(read_env IMPORT_CLASSIFICATION_MODE)"
database="$runtime_root/data/ledger.db"
evidence_dir="$runtime_root/evidence/task53-$timestamp"
backup_dir="$runtime_root/backups/preupgrade-task53/$timestamp"
health_url="http://127.0.0.1:$app_port/api/healthz"

case "$runtime_root" in
  /*task53* | /*Task53*) ;;
  *) echo "RUNTIME_ROOT must be an isolated absolute Task53 path: $runtime_root" >&2; exit 1 ;;
esac
case "$candidate_image" in
  ledger-two:1.3.0-rc-task53-*) ;;
  *) echo "CANDIDATE_IMAGE must use a fixed ledger-two:1.3.0-rc-task53-<commit> tag" >&2; exit 1 ;;
esac
case "$classification_mode" in
  off | suggest | graded) ;;
  *) echo "IMPORT_CLASSIFICATION_MODE must be off, suggest, or graded" >&2; exit 1 ;;
esac
if [ "$app_port" != "38092" ]; then
  echo "Task53 staging port must be 38092, got $app_port" >&2
  exit 1
fi
if [ "$container_name" = "ledger-two-v13-staging" ] || [ "$project_name" = "ledger-two-v13-staging" ]; then
  echo "Task53 staging must not reuse the Task50 container or project" >&2
  exit 1
fi
if [ ! -f "$database" ]; then
  echo "isolated Task53 staging database is missing: $database" >&2
  exit 1
fi

mkdir -p "$runtime_root/backups" "$runtime_root/uploads" "$runtime_root/logs" "$evidence_dir" "$backup_dir"
before_schema="$("$sqlite_bin" -readonly "$database" "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")"
if [ "$before_schema" != "21" ] && [ "$before_schema" != "22" ]; then
  echo "Task53 candidate only accepts an isolated schema 21 or 22 database, got $before_schema" >&2
  exit 1
fi
if [ "$("$sqlite_bin" -readonly "$database" "PRAGMA quick_check;")" != "ok" ]; then
  echo "Task53 source database failed quick_check" >&2
  exit 1
fi

source_backup="$backup_dir/ledger-schema${before_schema}.db"
"$sqlite_bin" "$database" ".backup '$source_backup'"
"$sqlite_bin" -readonly "$source_backup" "PRAGMA quick_check;" | grep -qx "ok"
sha256sum "$source_backup" > "$source_backup.sha256"

write_invariants() {
  target_database="$1"
  target_file="$2"
  "$sqlite_bin" -readonly -separator '|' "$target_database" "
    SELECT 'users', COUNT(*), 0 FROM users
    UNION ALL SELECT 'ledgers', COUNT(*), 0 FROM ledgers
    UNION ALL SELECT 'ledger_members', COUNT(*), 0 FROM ledger_members
    UNION ALL SELECT 'transactions', COUNT(*), COALESCE(SUM(amount), 0) FROM transactions
    UNION ALL SELECT 'transaction_splits', COUNT(*), COALESCE(SUM(share_amount), 0) FROM transaction_splits
    UNION ALL SELECT 'settlements', COUNT(*), COALESCE(SUM(amount), 0) FROM settlements
    UNION ALL SELECT 'import_batches', COUNT(*), 0 FROM import_batches
    UNION ALL SELECT 'import_items', COUNT(*), 0 FROM import_items
    UNION ALL SELECT 'transaction_import_refs', COUNT(*), 0 FROM transaction_import_refs
    ORDER BY 1;
  " > "$target_file"
}

write_invariants "$source_backup" "$evidence_dir/before-invariants.txt"
"$sqlite_bin" -readonly -separator '|' "$source_backup" "SELECT id, import_hash FROM import_items ORDER BY id;" > "$evidence_dir/before-import-hashes.txt"

compose() {
  "$docker_bin" compose --env-file "$env_file" -f "$compose_file" -p "$project_name" "$@"
}
if ! "$docker_bin" image inspect "$candidate_image" >/dev/null 2>&1; then
  echo "Task53 candidate image is not available locally: $candidate_image" >&2
  exit 1
fi
compose up -d --no-build --remove-orphans

health=""
attempt=1
while [ "$attempt" -le 24 ]; do
  health="$("$wget_bin" -qO- "$health_url" 2>/dev/null || true)"
  if echo "$health" | grep -q "\"version\":\"$expected_version\"" \
    && echo "$health" | grep -q '"schema_version":22' \
    && echo "$health" | grep -q '"deployment_channel":"staging"' \
    && echo "$health" | grep -q "\"import_classification_mode\":\"$classification_mode\"" \
    && echo "$health" | grep -q '"import_xlsx_enabled":true' \
    && echo "$health" | grep -q '"db":"ok"'; then
    break
  fi
  sleep 5
  attempt=$((attempt + 1))
done
if [ "$attempt" -gt 24 ]; then
  compose logs --no-color > "$evidence_dir/container.log" 2>&1 || true
  echo "Task53 staging health verification timed out: $health" >&2
  exit 1
fi

after_schema="$("$sqlite_bin" -readonly "$database" "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")"
after_quick_check="$("$sqlite_bin" -readonly "$database" "PRAGMA quick_check;")"
foreign_key_failures="$("$sqlite_bin" -readonly "$database" "PRAGMA foreign_key_check;")"
if [ "$after_schema" != "22" ] || [ "$after_quick_check" != "ok" ] || [ -n "$foreign_key_failures" ]; then
  echo "Task53 database verification failed: schema=$after_schema quick_check=$after_quick_check" >&2
  exit 1
fi
write_invariants "$database" "$evidence_dir/after-invariants.txt"
"$sqlite_bin" -readonly -separator '|' "$database" "SELECT id, import_hash FROM import_items ORDER BY id;" > "$evidence_dir/after-import-hashes.txt"
diff -u "$evidence_dir/before-invariants.txt" "$evidence_dir/after-invariants.txt" > "$evidence_dir/invariants.diff"
diff -u "$evidence_dir/before-import-hashes.txt" "$evidence_dir/after-import-hashes.txt" > "$evidence_dir/import-hashes.diff"

printf '%s\n' "$health" > "$evidence_dir/health.json"
compose ps > "$evidence_dir/compose-ps.txt"
{
  echo "candidate_image=$candidate_image"
  echo "runtime_root=$runtime_root"
  echo "before_schema=$before_schema"
  echo "after_schema=$after_schema"
  echo "classification_mode=$classification_mode"
  echo "source_backup=$source_backup"
  echo "source_backup_sha256=$(cut -d' ' -f1 "$source_backup.sha256")"
  echo "quick_check=$after_quick_check"
  echo "foreign_key_check=ok"
} > "$evidence_dir/manifest.txt"

echo "Task53 staging verification passed"
echo "evidence_dir=$evidence_dir"
