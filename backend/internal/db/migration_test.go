package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/migrations"
)

const latestMigrationVersion int64 = 12

func TestEmbeddedMigrationsUpgradeEmptyDatabase(t *testing.T) {
	database := openMigrationTestDB(t)
	runMigrations(t, database)

	version, err := goose.GetDBVersion(database)
	if err != nil {
		t.Fatalf("get migration version: %v", err)
	}
	if version != latestMigrationVersion {
		t.Fatalf("expected migration version %d, got %d", latestMigrationVersion, version)
	}

	assertTablesExist(t, database, []string{
		"users",
		"ledgers",
		"ledger_members",
		"accounts",
		"categories",
		"tags",
		"transactions",
		"transaction_splits",
		"transaction_tags",
		"settlements",
		"audit_logs",
		"app_settings",
		"transaction_templates",
		"recurring_rules",
		"recurring_reminders",
		"import_batches",
		"import_items",
		"import_rules",
		"transaction_defaults",
	})

	assertColumnsExist(t, database, "transactions", []string{
		"ledger_id",
		"owner_user_id",
		"payer_user_id",
		"visibility",
		"split_method",
		"deleted_at",
		"attachment_paths",
	})
	assertColumnsExist(t, database, "ledger_members", []string{
		"ledger_id",
		"user_id",
		"role",
	})
	assertColumnsExist(t, database, "categories", []string{
		"ledger_id",
		"owner_user_id",
		"type",
		"is_archived",
	})
	assertColumnsExist(t, database, "tags", []string{
		"ledger_id",
		"owner_user_id",
		"is_archived",
		"sort_order",
	})
	assertColumnsExist(t, database, "accounts", []string{
		"ledger_id",
		"owner_user_id",
		"is_archived",
		"sort_order",
	})
	assertColumnsExist(t, database, "transaction_defaults", []string{
		"ledger_id",
		"user_id",
		"type",
		"category_id",
		"account_id",
		"payer_user_id",
		"visibility",
		"split_method",
		"tag_names",
	})
	assertColumnsExist(t, database, "transaction_templates", []string{
		"is_archived",
		"archived_at",
	})

	assertIndexesExist(t, database, []string{
		"idx_transactions_ledger_month",
		"idx_transactions_owner",
		"idx_splits_transaction",
		"idx_settlements_users",
		"idx_templates_ledger",
		"idx_recurring_rules_ledger",
		"idx_import_items_hash",
		"idx_import_rules_ledger",
		"idx_categories_ledger_type_name",
		"idx_tags_ledger_name",
		"idx_accounts_ledger_name",
		"idx_tags_ledger_sort",
		"idx_accounts_ledger_sort",
		"idx_transaction_defaults_ledger_user",
		"idx_templates_ledger_archived",
	})
}

func TestMigrationPromotesOwnerForLegacyLedger(t *testing.T) {
	database := openMigrationTestDB(t)
	runMigrationsTo(t, database, 8)

	_, err := database.Exec(`
		INSERT INTO ledgers (id, name, default_currency, created_at, updated_at)
		VALUES ('legacy-ledger', 'Legacy Ledger', 'CNY', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at)
		VALUES
			('user-a', 'alice', 'Alice', 'hash-a', 'user', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-b', 'bob', 'Bob', 'hash-b', 'user', '2026-01-01T00:00:01Z', '2026-01-01T00:00:01Z');
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES
			('legacy-ledger', 'user-a', 'editor', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('legacy-ledger', 'user-b', 'editor', '2026-01-01T00:00:01Z', '2026-01-01T00:00:01Z');
	`)
	if err != nil {
		t.Fatalf("seed legacy ledger: %v", err)
	}

	runMigrations(t, database)

	var ownerCount int
	err = database.QueryRow("SELECT COUNT(*) FROM ledger_members WHERE ledger_id = 'legacy-ledger' AND role = 'owner'").Scan(&ownerCount)
	if err != nil {
		t.Fatalf("query owner count: %v", err)
	}
	if ownerCount != 1 {
		t.Fatalf("expected exactly one owner after migration, got %d", ownerCount)
	}

	var firstUserRole string
	err = database.QueryRow("SELECT role FROM ledger_members WHERE ledger_id = 'legacy-ledger' AND user_id = 'user-a'").Scan(&firstUserRole)
	if err != nil {
		t.Fatalf("query first user role: %v", err)
	}
	if firstUserRole != "owner" {
		t.Fatalf("expected first legacy member to become owner, got %s", firstUserRole)
	}
}

func openMigrationTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open memory database: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	database.SetMaxOpenConns(1)
	return database
}

func runMigrations(t *testing.T, database *sql.DB) {
	t.Helper()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.Up(database, "."); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
}

func runMigrationsTo(t *testing.T, database *sql.DB, version int64) {
	t.Helper()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.UpTo(database, ".", version); err != nil {
		t.Fatalf("run migrations to %d: %v", version, err)
	}
}

func assertTablesExist(t *testing.T, database *sql.DB, names []string) {
	t.Helper()

	for _, name := range names {
		var found string
		err := database.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
			name,
		).Scan(&found)
		if err != nil {
			t.Fatalf("expected table %q to exist: %v", name, err)
		}
	}
}

func assertColumnsExist(t *testing.T, database *sql.DB, table string, names []string) {
	t.Helper()

	rows, err := database.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("read columns for %s: %v", table, err)
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan column for %s: %v", table, err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns for %s: %v", table, err)
	}

	for _, name := range names {
		if !columns[name] {
			t.Fatalf("expected column %s.%s to exist", table, name)
		}
	}
}

func assertIndexesExist(t *testing.T, database *sql.DB, names []string) {
	t.Helper()

	for _, name := range names {
		var found string
		err := database.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = 'index' AND name = ?",
			name,
		).Scan(&found)
		if err != nil {
			t.Fatalf("expected index %q to exist: %v", name, err)
		}
	}
}
