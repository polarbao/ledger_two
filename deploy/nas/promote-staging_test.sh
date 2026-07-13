#!/bin/sh

set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
promotion_script="$script_dir/promote-staging.sh"
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

create_fixture() {
  fixture_root="$1"
  mkdir -p "$fixture_root/data" "$fixture_root/backups/predeploy/schema18" "$fixture_root/state"
  cat > "$fixture_root/.env" <<'EOF'
APP_ENV=production
DEPLOYMENT_CHANNEL=staging
IMPORT_XLSX_ENABLED=true
APP_PORT=38089
EOF
  cat > "$fixture_root/docker-compose.yml" <<'EOF'
services:
  ledger-two:
    environment:
      IMPORT_XLSX_ENABLED: "${IMPORT_XLSX_ENABLED:-}"
EOF
  sqlite3 "$fixture_root/data/ledger.db" <<'EOF'
CREATE TABLE goose_db_version (id INTEGER PRIMARY KEY, version_id INTEGER NOT NULL, is_applied INTEGER NOT NULL);
INSERT INTO goose_db_version(version_id, is_applied) VALUES (18, 1);
EOF
  cp "$fixture_root/data/ledger.db" "$fixture_root/backups/predeploy/schema18/ledger.db"
  : > "$fixture_root/state/ledger-two-staging"
}

cat > "$tmp_root/fake-docker" <<'EOF'
#!/bin/sh
set -eu
state="$FAKE_DOCKER_STATE"
deployment="$FAKE_DEPLOYMENT_ROOT"
command_name="$1"
shift
case "$command_name" in
  image)
    exit 0
    ;;
  inspect)
    test -f "$state/$1"
    ;;
  stop)
    test -f "$state/$1"
    ;;
  rename)
    mv "$state/$1" "$state/$2"
    ;;
  start)
    : > "$state/$1"
    ;;
  compose)
    action=""
    for arg in "$@"; do
      case "$arg" in
        up|down|ps) action="$arg"; break ;;
      esac
    done
    case "$action" in
      up)
        sqlite3 "$deployment/data/ledger.db" "UPDATE goose_db_version SET version_id=19 WHERE is_applied=1;"
        : > "$state/ledger-two-staging"
        ;;
      down)
        rm -f "$state/ledger-two-staging"
        ;;
      ps)
        echo "ledger-two-staging ledger-two:1.2.0-rc healthy"
        ;;
      *)
        echo "unexpected compose action: $*" >&2
        exit 1
        ;;
    esac
    ;;
  *)
    echo "unexpected docker command: $command_name $*" >&2
    exit 1
    ;;
esac
EOF
chmod +x "$tmp_root/fake-docker"

cat > "$tmp_root/fake-wget" <<'EOF'
#!/bin/sh
set -eu
count=0
if [ -f "$FAKE_HEALTH_COUNT" ]; then
  count="$(cat "$FAKE_HEALTH_COUNT")"
fi
count=$((count + 1))
printf '%s' "$count" > "$FAKE_HEALTH_COUNT"
if [ "$count" -eq 1 ]; then
  printf '%s\n' '{"success":true,"data":{"db":"ok","deployment_channel":"staging","schema_version":18,"status":"ok","version":"1.2.0-rc"}}'
elif [ "$FAKE_HEALTH_MODE" = "success" ]; then
  printf '%s\n' '{"success":true,"data":{"db":"ok","deployment_channel":"staging","import_xlsx_enabled":true,"schema_version":19,"status":"ok","version":"1.2.0-rc"}}'
else
  printf '%s\n' '{"success":true,"data":{"db":"error","deployment_channel":"staging","schema_version":19,"status":"ok","version":"1.2.0-rc"}}'
fi
EOF
chmod +x "$tmp_root/fake-wget"

run_promotion() {
  fixture_root="$1"
  mode="$2"
  rm -f "$fixture_root/health-count"
  FAKE_DOCKER_STATE="$fixture_root/state" \
  FAKE_DEPLOYMENT_ROOT="$fixture_root" \
  FAKE_HEALTH_COUNT="$fixture_root/health-count" \
  FAKE_HEALTH_MODE="$mode" \
  DOCKER_BIN="$tmp_root/fake-docker" \
  WGET_BIN="$tmp_root/fake-wget" \
  SLEEP_BIN=true \
  MAX_ATTEMPTS=1 \
  sh "$promotion_script" "$fixture_root" "$fixture_root/backups/predeploy/schema18/ledger.db"
}

success_root="$tmp_root/success"
create_fixture "$success_root"
run_promotion "$success_root" success >/dev/null
test "$(sqlite3 -readonly "$success_root/data/ledger.db" 'SELECT version_id FROM goose_db_version WHERE is_applied=1;')" = "19"
test -f "$success_root/state/ledger-two-staging"
find "$success_root/state" -name 'ledger-two-staging-schema18-rollback-*' | grep -q .

failure_root="$tmp_root/failure"
create_fixture "$failure_root"
if run_promotion "$failure_root" failure >/dev/null 2>&1; then
  echo "expected failed health verification to return non-zero" >&2
  exit 1
fi
test "$(sqlite3 -readonly "$failure_root/data/ledger.db" 'SELECT version_id FROM goose_db_version WHERE is_applied=1;')" = "18"
test -f "$failure_root/state/ledger-two-staging"
find "$failure_root/backups/predeploy" -name 'ledger.db.failed' | grep -q .

echo "promote-staging tests passed"
