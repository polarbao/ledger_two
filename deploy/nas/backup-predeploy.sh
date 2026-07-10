#!/bin/sh

set -eu

deployment_root="${1:-/volume1/docker/ledger-two}"
release_label="${2:-v1.2.0-rc}"
timestamp="$(date +%Y%m%d-%H%M%S)"
backup_dir="$deployment_root/backups/predeploy/$release_label-$timestamp"
database_path="$deployment_root/data/ledger.db"
backup_database="$backup_dir/ledger.db"

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "sqlite3 is required for a consistent online backup" >&2
  exit 1
fi

if [ ! -f "$database_path" ]; then
  echo "database not found: $database_path" >&2
  exit 1
fi

mkdir -p "$backup_dir"
sqlite3 "$database_path" ".backup '$backup_database'"

quick_check="$(sqlite3 -readonly "$backup_database" "PRAGMA quick_check;")"
if [ "$quick_check" != "ok" ]; then
  echo "backup database integrity check failed: $quick_check" >&2
  exit 1
fi

schema_version="$(sqlite3 -readonly "$backup_database" "SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1;")"
row_counts="$(sqlite3 -readonly "$backup_database" "SELECT (SELECT COUNT(*) FROM users) || '|' || (SELECT COUNT(*) FROM ledgers) || '|' || (SELECT COUNT(*) FROM transactions) || '|' || (SELECT COUNT(*) FROM settlements);")"

cp -p "$deployment_root/docker-compose.yml" "$backup_dir/docker-compose.yml"
cp -p "$deployment_root/.env" "$backup_dir/env.backup"
chmod 600 "$backup_dir/env.backup"

if [ -d "$deployment_root/uploads" ]; then
  tar -C "$deployment_root" -czf "$backup_dir/uploads.tgz" uploads
else
  tar -czf "$backup_dir/uploads.tgz" --files-from /dev/null
fi

cat > "$backup_dir/manifest.txt" <<EOF
release_label=$release_label
created_at=$(date '+%Y-%m-%dT%H:%M:%S%z')
source_database=$database_path
schema_version=$schema_version
row_counts=users|ledgers|transactions|settlements
row_count_values=$row_counts
quick_check=$quick_check
EOF

sha256sum "$backup_database" "$backup_dir/uploads.tgz" > "$backup_dir/SHA256SUMS"

echo "backup_dir=$backup_dir"
echo "schema_version=$schema_version"
echo "row_counts=$row_counts"
echo "quick_check=$quick_check"
