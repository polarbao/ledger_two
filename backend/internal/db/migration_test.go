package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/migrations"
)

const latestMigrationVersion int64 = 21

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
		"transaction_import_refs",
		"transaction_defaults",
		"instance_admins",
		"instance_audit_logs",
	})
	assertColumnsExist(t, database, "ledgers", []string{
		"status",
		"archived_at",
		"archived_by_user_id",
		"version",
	})
	assertColumnsExist(t, database, "audit_logs", []string{
		"actor_role",
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
	assertColumnsExist(t, database, "import_batches", []string{
		"source_type",
		"file_sha256",
		"total_rows",
		"new_rows",
		"duplicate_rows",
		"suspicious_rows",
		"invalid_rows",
		"imported_rows",
		"skipped_rows",
		"failed_rows",
		"updated_at",
		"committed_at",
		"expires_at",
		"file_format",
		"parser_metadata_json",
	})
	assertColumnsExist(t, database, "import_items", []string{
		"row_number",
		"source_type",
		"external_order_id",
		"occurred_at",
		"title",
		"merchant",
		"description",
		"amount_cents",
		"direction",
		"target_transaction_type",
		"duplicate_status",
		"row_status",
		"normalized_json",
		"user_adjustment_json",
		"error_code",
		"error_message",
		"generated_transaction_id",
		"suggested_category_id",
		"suggested_account_id",
		"suggested_tag_ids_json",
		"suggested_rule_id",
		"suggestion_reason",
		"selected_category_id",
		"selected_account_id",
		"selected_tag_ids_json",
		"visibility",
	})
	assertColumnsExist(t, database, "import_rules", []string{
		"name",
		"match_type",
		"pattern",
		"amount_min_cents",
		"amount_max_cents",
		"priority",
		"result_json",
		"status",
		"archived_at",
	})
	assertColumnsExist(t, database, "transaction_import_refs", []string{
		"ledger_id",
		"transaction_id",
		"import_batch_id",
		"import_row_id",
		"import_hash",
		"external_order_id",
		"source_type",
		"created_at",
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
		"idx_import_batches_ledger_status",
		"idx_import_items_batch_row_number",
		"idx_import_items_batch_duplicate_status",
		"idx_import_items_batch_row_status",
		"idx_import_items_selected_category",
		"idx_import_items_selected_account",
		"idx_transaction_import_refs_hash",
		"idx_transaction_import_refs_tx",
		"idx_transaction_import_refs_batch",
		"idx_import_rules_ledger_status_priority",
		"idx_import_items_suggested_rule",
		"idx_ledgers_status_created",
		"idx_ledger_members_user_ledger",
		"idx_ledger_members_one_owner",
		"idx_audit_logs_ledger_created",
		"idx_instance_audit_created",
	})
	assertTriggersExist(t, database, []string{
		"trg_ledger_members_max_two_before_insert",
	})

	var instanceAdminCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM instance_admins").Scan(&instanceAdminCount); err != nil {
		t.Fatalf("count instance administrators: %v", err)
	}
	if instanceAdminCount != 0 {
		t.Fatalf("expected empty migration to create no instance administrator, got %d", instanceAdminCount)
	}
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

func TestTask50UpgradePreflightAcceptsValidSchema19AndPreservesData(t *testing.T) {
	database := openMigrationTestDB(t)
	runMigrationsTo(t, database, 19)
	seedTask50PreflightBase(t, database)

	_, err := database.Exec(`
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, visibility, status, created_at, updated_at
		) VALUES (
			'transaction-a', 'ledger-a', 'expense', 'Dinner', 12345, 'CNY',
			'2026-07-01T00:00:00Z', 'user-a', 'user-a', 'private', 'normal',
			'2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z'
		);
	`)
	if err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	snapshot, err := prepareTask50Upgrade(context.Background(), database, 19)
	if err != nil {
		t.Fatalf("preflight valid schema 19: %v", err)
	}

	runMigrations(t, database)

	if err := verifyTask50MigrationSnapshot(context.Background(), database, snapshot); err != nil {
		t.Fatalf("verify migration conservation: %v", err)
	}
	assertMigrationVersion(t, database, 21)

	var status string
	var version int64
	if err := database.QueryRow("SELECT status, version FROM ledgers WHERE id = 'ledger-a'").Scan(&status, &version); err != nil {
		t.Fatalf("query migrated ledger: %v", err)
	}
	if status != "active" || version != 1 {
		t.Fatalf("expected active/v1 ledger, got %s/v%d", status, version)
	}

	var adminUserID string
	if err := database.QueryRow("SELECT user_id FROM instance_admins").Scan(&adminUserID); err != nil {
		t.Fatalf("query initial instance admin: %v", err)
	}
	if adminUserID != "user-a" {
		t.Fatalf("expected earliest ledger owner user-a, got %s", adminUserID)
	}
}

func TestMigrationSelectsEarliestLedgerOwnerAsInitialInstanceAdmin(t *testing.T) {
	database := openMigrationTestDB(t)
	runMigrationsTo(t, database, 19)

	_, err := database.Exec(`
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at) VALUES
			('user-a', 'alice', 'Alice', 'hash-a', 'user', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-b', 'bob', 'Bob', 'hash-b', 'user', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z');
		INSERT INTO ledgers (id, name, default_currency, created_at, updated_at) VALUES
			('ledger-later', 'Later', 'CNY', '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z'),
			('ledger-earlier', 'Earlier', 'CNY', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at) VALUES
			('ledger-later', 'user-a', 'owner', '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z'),
			('ledger-earlier', 'user-b', 'owner', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("seed deterministic instance administrator fixture: %v", err)
	}

	if _, err := prepareTask50Upgrade(context.Background(), database, 19); err != nil {
		t.Fatalf("preflight deterministic fixture: %v", err)
	}
	runMigrations(t, database)

	var adminUserID string
	if err := database.QueryRow("SELECT user_id FROM instance_admins").Scan(&adminUserID); err != nil {
		t.Fatalf("query deterministic instance administrator: %v", err)
	}
	if adminUserID != "user-b" {
		t.Fatalf("expected earliest ledger owner user-b, got %s", adminUserID)
	}
}

func TestTask50UpgradePreflightRejectsSchema19InvariantViolations(t *testing.T) {
	tests := []struct {
		name        string
		seed        string
		wantMessage string
	}{
		{
			name: "three members",
			seed: `
				INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at) VALUES
					('ledger-a', 'user-b', 'editor', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z'),
					('ledger-a', 'user-c', 'viewer', '2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z');
			`,
			wantMessage: "ledger-a has 3 members",
		},
		{
			name:        "missing owner",
			seed:        "UPDATE ledger_members SET role = 'editor' WHERE ledger_id = 'ledger-a';",
			wantMessage: "ledger-a has 0 owners",
		},
		{
			name: "multiple owners",
			seed: `
				INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
				VALUES ('ledger-a', 'user-b', 'owner', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z');
			`,
			wantMessage: "ledger-a has 2 owners",
		},
		{
			name: "orphan transaction ledger",
			seed: `
				INSERT INTO transactions (
					id, ledger_id, type, title, amount, currency, occurred_at,
					owner_user_id, created_by_user_id, visibility, status, created_at, updated_at
				) VALUES (
					'orphan-transaction', 'missing-ledger', 'expense', 'Orphan', 100, 'CNY',
					'2026-07-01T00:00:00Z', 'user-a', 'user-a', 'private', 'normal',
					'2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z'
				);
			`,
			wantMessage: "foreign key violation in transactions",
		},
		{
			name: "orphan member user",
			seed: `
				INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
				VALUES ('ledger-a', 'missing-user', 'editor', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z');
			`,
			wantMessage: "foreign key violation in ledger_members",
		},
		{
			name: "invalid member role",
			seed: `
				INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
				VALUES ('ledger-a', 'user-b', 'administrator', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z');
			`,
			wantMessage: "invalid member role administrator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database := openMigrationTestDB(t)
			runMigrationsTo(t, database, 19)
			seedTask50PreflightBase(t, database)

			if _, err := database.Exec(tt.seed); err != nil {
				t.Fatalf("seed invariant violation: %v", err)
			}

			_, err := prepareTask50Upgrade(context.Background(), database, 19)
			if err == nil || !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("expected error containing %q, got %v", tt.wantMessage, err)
			}

			assertMigrationVersion(t, database, 19)
			assertColumnDoesNotExist(t, database, "ledgers", "status")
		})
	}
}

func TestTask50UpgradePreflightRejectsUnexpectedSchemaVersion(t *testing.T) {
	database := openMigrationTestDB(t)
	runMigrationsTo(t, database, 18)

	_, err := prepareTask50Upgrade(context.Background(), database, 18)
	if err == nil || !strings.Contains(err.Error(), "requires schema version 19") {
		t.Fatalf("expected schema version rejection, got %v", err)
	}
	assertMigrationVersion(t, database, 18)
}

func TestTask50UpgradePreflightRejectsUnversionedApplicationDatabase(t *testing.T) {
	database := openMigrationTestDB(t)
	if _, err := database.Exec("CREATE TABLE legacy_data (id TEXT PRIMARY KEY)"); err != nil {
		t.Fatalf("seed unversioned application table: %v", err)
	}

	_, err := prepareTask50Upgrade(context.Background(), database, 0)
	if err == nil || !strings.Contains(err.Error(), "unversioned non-empty database") {
		t.Fatalf("expected unversioned database rejection, got %v", err)
	}
}

func TestTask50UpgradePreflightRejectsFailedQuickCheck(t *testing.T) {
	err := validateQuickCheckResults([]string{"*** in database main ***", "Page 3 is never used"})
	if err == nil || !strings.Contains(err.Error(), "quick_check failed") {
		t.Fatalf("expected quick_check rejection, got %v", err)
	}
}

func TestTask50PostMigrationVerificationRejectsForeignKeyDrift(t *testing.T) {
	database := openMigrationTestDB(t)
	runMigrationsTo(t, database, 19)
	seedTask50PreflightBase(t, database)

	snapshot, err := prepareTask50Upgrade(context.Background(), database, 19)
	if err != nil {
		t.Fatalf("preflight valid schema 19: %v", err)
	}
	runMigrations(t, database)

	if _, err := database.Exec(`
		UPDATE ledger_members
		SET user_id = 'missing-user'
		WHERE ledger_id = 'ledger-a' AND user_id = 'user-a'
	`); err != nil {
		t.Fatalf("seed post-migration foreign key drift: %v", err)
	}

	err = verifyTask50MigrationSnapshot(context.Background(), database, snapshot)
	if err == nil || !strings.Contains(err.Error(), "foreign key violation in ledger_members") {
		t.Fatalf("expected post-migration foreign key rejection, got %v", err)
	}
}

func TestInitRunsTask50PreflightBackupAndSchema19Upgrade(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "ledger.db")
	database, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("open schema 19 file: %v", err)
	}
	database.SetMaxOpenConns(1)
	runMigrationsTo(t, database, 19)
	seedTask50PreflightBase(t, database)
	if err := database.Close(); err != nil {
		t.Fatalf("close schema 19 file: %v", err)
	}

	upgraded, err := Init(databasePath)
	if err != nil {
		t.Fatalf("initialize schema 19 database: %v", err)
	}
	defer upgraded.Close()

	assertMigrationVersion(t, upgraded, 21)
	if _, err := os.Stat(databasePath + ".pre_migrate_v19.bak"); err != nil {
		t.Fatalf("expected pre-migration backup: %v", err)
	}
}

func TestInitRejectsSchema18WithoutApplyingTask50Migration(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "ledger.db")
	database, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("open schema 18 file: %v", err)
	}
	database.SetMaxOpenConns(1)
	runMigrationsTo(t, database, 18)
	if err := database.Close(); err != nil {
		t.Fatalf("close schema 18 file: %v", err)
	}

	opened, err := Init(databasePath)
	if opened != nil {
		_ = opened.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "requires schema version 19") {
		t.Fatalf("expected schema 18 rejection, got %v", err)
	}

	unchanged, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("reopen rejected database: %v", err)
	}
	defer unchanged.Close()
	assertMigrationVersion(t, unchanged, 18)
	assertColumnDoesNotExist(t, unchanged, "ledgers", "status")
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

func seedTask50PreflightBase(t *testing.T, database *sql.DB) {
	t.Helper()

	_, err := database.Exec(`
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at) VALUES
			('user-a', 'alice', 'Alice', 'hash-a', 'user', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-b', 'bob', 'Bob', 'hash-b', 'user', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z'),
			('user-c', 'cara', 'Cara', 'hash-c', 'user', '2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z');
		INSERT INTO ledgers (id, name, default_currency, created_at, updated_at)
		VALUES ('ledger-a', 'Ledger A', 'CNY', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES ('ledger-a', 'user-a', 'owner', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("seed task50 preflight base: %v", err)
	}
}

func assertMigrationVersion(t *testing.T, database *sql.DB, want int64) {
	t.Helper()

	version, err := goose.GetDBVersion(database)
	if err != nil {
		t.Fatalf("get migration version: %v", err)
	}
	if version != want {
		t.Fatalf("expected migration version %d, got %d", want, version)
	}
}

func assertColumnDoesNotExist(t *testing.T, database *sql.DB, table string, column string) {
	t.Helper()

	rows, err := database.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("read columns for %s: %v", table, err)
	}
	defer rows.Close()

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
		if name == column {
			t.Fatalf("expected column %s.%s not to exist", table, column)
		}
	}
}

func assertTriggersExist(t *testing.T, database *sql.DB, names []string) {
	t.Helper()

	for _, name := range names {
		var found string
		err := database.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = 'trigger' AND name = ?",
			name,
		).Scan(&found)
		if err != nil {
			t.Fatalf("expected trigger %q to exist: %v", name, err)
		}
	}
}
