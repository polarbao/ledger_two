package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/migrations"
)

const latestMigrationVersion int64 = 22

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
		"metadata_profile_version",
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
		"system_key",
	})
	assertColumnsExist(t, database, "tags", []string{
		"ledger_id",
		"owner_user_id",
		"is_archived",
		"sort_order",
		"system_key",
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
		"classification_status",
		"classification_confidence",
		"classification_source",
		"classification_reason_json",
		"matched_rule_ids_json",
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
		"origin",
		"source_type",
		"apply_mode",
		"confidence",
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
		"idx_categories_ledger_system_key",
		"idx_tags_ledger_system_key",
		"idx_import_rules_ledger_origin_status_priority",
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

func TestTask531MigrationUpgradesSchema21WithCompatibleDefaultsAndPartialUniqueIndexes(t *testing.T) {
	database := openMigrationTestDB(t)
	runMigrationsTo(t, database, 21)

	_, err := database.Exec(`
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at)
		VALUES ('user-53', 'task53', 'Task 53', 'hash', 'user', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z');
		INSERT INTO ledgers (id, name, default_currency, status, version, created_at, updated_at)
		VALUES ('ledger-53', 'Task53 Ledger', 'CNY', 'active', 1, '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z');
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES ('ledger-53', 'user-53', 'owner', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z');
		INSERT INTO categories (id, ledger_id, owner_user_id, name, type, is_system, created_at, updated_at)
		VALUES ('cat-53', 'ledger-53', 'user-53', '历史餐饮', 'expense', 0, '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z');
		INSERT INTO tags (id, ledger_id, owner_user_id, name, created_at, updated_at)
		VALUES ('tag-53', 'ledger-53', 'user-53', '历史标签', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z');
		INSERT INTO import_rules (
			id, ledger_id, keyword, created_by_user_id, created_at, updated_at,
			name, match_type, pattern, priority, result_json, status
		) VALUES (
			'rule-53', 'ledger-53', '咖啡', 'user-53', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z',
			'历史规则', 'merchant_contains', '咖啡', 100, '{"category_id":"cat-53","tag_ids":["tag-53"]}', 'active'
		);
		INSERT INTO import_batches (id, ledger_id, filename, created_by_user_id, status, created_at)
		VALUES ('batch-53', 'ledger-53', 'anonymous.csv', 'user-53', 'ready', '2026-07-17T00:00:00Z');
		INSERT INTO import_items (
			id, batch_id, import_hash, status, created_at, suggested_tag_ids_json, selected_tag_ids_json
		) VALUES (
			'item-53', 'batch-53', 'hash-53', 'pending', '2026-07-17T00:00:00Z', '["tag-53"]', '["tag-53"]'
		);
	`)
	if err != nil {
		t.Fatalf("seed schema 21 task53 fixture: %v", err)
	}

	runMigrations(t, database)
	assertMigrationVersion(t, database, 22)

	var profileVersion int
	if err := database.QueryRow("SELECT metadata_profile_version FROM ledgers WHERE id = 'ledger-53'").Scan(&profileVersion); err != nil {
		t.Fatalf("read metadata profile version: %v", err)
	}
	if profileVersion != 0 {
		t.Fatalf("historical ledger profile version = %d, want 0", profileVersion)
	}

	var origin, applyMode, confidence string
	var sourceType sql.NullString
	if err := database.QueryRow(`
		SELECT origin, source_type, apply_mode, confidence
		FROM import_rules WHERE id = 'rule-53'
	`).Scan(&origin, &sourceType, &applyMode, &confidence); err != nil {
		t.Fatalf("read migrated rule defaults: %v", err)
	}
	if origin != "manual" || sourceType.Valid || applyMode != "suggest" || confidence != "high" {
		t.Fatalf("unexpected migrated rule defaults: origin=%s source=%v mode=%s confidence=%s", origin, sourceType, applyMode, confidence)
	}

	var status, itemConfidence, reasonJSON, ruleIDsJSON string
	var itemSource sql.NullString
	if err := database.QueryRow(`
		SELECT classification_status, classification_confidence, classification_source,
		       classification_reason_json, matched_rule_ids_json
		FROM import_items WHERE id = 'item-53'
	`).Scan(&status, &itemConfidence, &itemSource, &reasonJSON, &ruleIDsJSON); err != nil {
		t.Fatalf("read migrated import item defaults: %v", err)
	}
	if status != "unresolved" || itemConfidence != "none" || itemSource.Valid || reasonJSON != "{}" || ruleIDsJSON != "[]" {
		t.Fatalf("unexpected migrated item defaults: status=%s confidence=%s source=%v reason=%s rules=%s", status, itemConfidence, itemSource, reasonJSON, ruleIDsJSON)
	}

	if _, err := database.Exec("UPDATE categories SET system_key = 'expense_food' WHERE id = 'cat-53'"); err != nil {
		t.Fatalf("set first category system key: %v", err)
	}
	_, err = database.Exec(`
		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, system_key, is_system, created_at, updated_at
		) VALUES (
			'cat-53-duplicate', 'ledger-53', 'user-53', '重复键', 'expense', 'expense_food', 1,
			'2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z'
		)
	`)
	if err == nil {
		t.Fatal("expected partial unique category system key index to reject duplicate key")
	}
}

func TestTask531PreflightConservesAnonymousSchema21Data(t *testing.T) {
	database := openMigrationTestDB(t)
	runMigrationsTo(t, database, 21)
	seedTask50PreflightBase(t, database)
	seedTask506MigrationConservationFixture(t, database)

	snapshot, err := prepareTask53Upgrade(context.Background(), database, 21)
	if err != nil {
		t.Fatalf("prepare task53 schema 21 upgrade: %v", err)
	}
	runMigrations(t, database)
	if err := verifyTask53MigrationSnapshot(context.Background(), database, snapshot); err != nil {
		t.Fatalf("verify task53 migration conservation: %v", err)
	}
	assertMigrationVersion(t, database, 22)
}

func TestTask531PreflightRejectsInvalidJSONAndUnexpectedSourceSchema(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*testing.T, *sql.DB)
		wantMessage string
	}{
		{
			name: "invalid rule result json",
			mutate: func(t *testing.T, database *sql.DB) {
				t.Helper()
				if _, err := database.Exec("UPDATE import_rules SET result_json = '{invalid' WHERE id = 'migration-rule'"); err != nil {
					t.Fatalf("invalidate rule result json: %v", err)
				}
			},
			wantMessage: "migration-rule",
		},
		{
			name: "non object rule result json",
			mutate: func(t *testing.T, database *sql.DB) {
				t.Helper()
				if _, err := database.Exec("UPDATE import_rules SET result_json = 'null' WHERE id = 'migration-rule'"); err != nil {
					t.Fatalf("replace rule result json with null: %v", err)
				}
			},
			wantMessage: "migration-rule",
		},
		{
			name: "invalid selected tag json",
			mutate: func(t *testing.T, database *sql.DB) {
				t.Helper()
				if _, err := database.Exec("UPDATE import_items SET selected_tag_ids_json = '[invalid' WHERE id = 'migration-item'"); err != nil {
					t.Fatalf("invalidate selected tag json: %v", err)
				}
			},
			wantMessage: "migration-item",
		},
		{
			name: "orphan selected tag reference",
			mutate: func(t *testing.T, database *sql.DB) {
				t.Helper()
				if _, err := database.Exec("UPDATE import_items SET selected_tag_ids_json = '[\"missing-tag\"]' WHERE id = 'migration-item'"); err != nil {
					t.Fatalf("orphan selected tag reference: %v", err)
				}
			},
			wantMessage: "migration-item",
		},
		{
			name: "orphan rule category reference",
			mutate: func(t *testing.T, database *sql.DB) {
				t.Helper()
				if _, err := database.Exec("UPDATE import_rules SET result_json = '{\"category_id\":\"missing-category\",\"tag_ids\":[]}' WHERE id = 'migration-rule'"); err != nil {
					t.Fatalf("orphan rule category reference: %v", err)
				}
			},
			wantMessage: "migration-rule",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := openMigrationTestDB(t)
			runMigrationsTo(t, database, 21)
			seedTask50PreflightBase(t, database)
			seedTask506MigrationConservationFixture(t, database)
			test.mutate(t, database)

			_, err := prepareTask53Upgrade(context.Background(), database, 21)
			if err == nil || !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("expected task53 preflight error containing %q, got %v", test.wantMessage, err)
			}
			assertMigrationVersion(t, database, 21)
		})
	}

	database := openMigrationTestDB(t)
	runMigrationsTo(t, database, 19)
	if _, err := prepareTask53Upgrade(context.Background(), database, 19); err == nil || !strings.Contains(err.Error(), "requires schema version 21") {
		t.Fatalf("expected task53 schema 19 rejection, got %v", err)
	}
	assertMigrationVersion(t, database, 19)
}

func TestTask531MigrationRollsBackAllAdditiveChangesWhenIndexCreationFails(t *testing.T) {
	database := openMigrationTestDB(t)
	runMigrationsTo(t, database, 21)
	if _, err := database.Exec("CREATE INDEX idx_tags_ledger_system_key ON tags(id)"); err != nil {
		t.Fatalf("seed conflicting task53 index: %v", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.Up(database, "."); err == nil {
		t.Fatal("expected migration 022 index creation failure")
	}

	assertMigrationVersion(t, database, 21)
	assertColumnDoesNotExist(t, database, "ledgers", "metadata_profile_version")
	assertColumnDoesNotExist(t, database, "categories", "system_key")
	assertColumnDoesNotExist(t, database, "tags", "system_key")
	var categoryIndexCount int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'index' AND name = 'idx_categories_ledger_system_key'
	`).Scan(&categoryIndexCount); err != nil {
		t.Fatalf("inspect rolled back category index: %v", err)
	}
	if categoryIndexCount != 0 {
		t.Fatalf("expected category system key index rollback, got %d", categoryIndexCount)
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

	runMigrationsTo(t, database, 21)

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

func TestInitRejectsDirectSchema19ToTask53Upgrade(t *testing.T) {
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

	opened, err := Init(databasePath)
	if opened != nil {
		_ = opened.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "task53 upgrade requires schema version 21") {
		t.Fatalf("expected direct schema 19 task53 rejection, got %v", err)
	}

	unchanged, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("reopen rejected schema 19 database: %v", err)
	}
	defer unchanged.Close()
	assertMigrationVersion(t, unchanged, 19)
}

func TestInitRunsTask53PreflightBackupAndSchema21Upgrade(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "ledger.db")
	database, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("open schema 21 file: %v", err)
	}
	database.SetMaxOpenConns(1)
	runMigrationsTo(t, database, 21)
	seedTask50PreflightBase(t, database)
	if err := database.Close(); err != nil {
		t.Fatalf("close schema 21 file: %v", err)
	}

	upgraded, err := Init(databasePath)
	if err != nil {
		t.Fatalf("initialize schema 21 database: %v", err)
	}
	defer upgraded.Close()
	assertMigrationVersion(t, upgraded, 22)
	if _, err := os.Stat(databasePath + ".pre_migrate_v21.bak"); err != nil {
		t.Fatalf("expected schema 21 pre-migration backup: %v", err)
	}
}

func TestTask506AnonymousSchema19UpgradeRestoreAndForwardFixRehearsal(t *testing.T) {
	runtimeDir := t.TempDir()
	databasePath := filepath.Join(runtimeDir, "staging", "data", "ledger.db")
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o755); err != nil {
		t.Fatalf("create anonymous staging data directory: %v", err)
	}

	source, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("open anonymous schema 19 database: %v", err)
	}
	source.SetMaxOpenConns(1)
	runMigrationsTo(t, source, 19)
	seedTask50PreflightBase(t, source)
	seedTask506MigrationConservationFixture(t, source)

	before, err := prepareTask50Upgrade(context.Background(), source, 19)
	if err != nil {
		t.Fatalf("capture anonymous schema 19 baseline: %v", err)
	}
	if err := source.Close(); err != nil {
		t.Fatalf("close anonymous schema 19 source: %v", err)
	}

	backupPath := filepath.Join(runtimeDir, "backups", "ledger-schema19.db")
	if err := copyTask506DatabaseFile(databasePath, backupPath); err != nil {
		t.Fatalf("create rehearsal backup: %v", err)
	}
	backupChecksum := task506FileSHA256(t, backupPath)

	upgraded, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("open anonymous schema 19 database for task50 rehearsal: %v", err)
	}
	upgraded.SetMaxOpenConns(1)
	runMigrationsTo(t, upgraded, 21)
	assertMigrationVersion(t, upgraded, 21)
	if err := verifyTask50MigrationSnapshot(context.Background(), upgraded, before); err != nil {
		t.Fatalf("verify anonymous upgrade conservation: %v", err)
	}
	assertTask506QuickCheck(t, upgraded)
	if err := upgraded.Close(); err != nil {
		t.Fatalf("close upgraded rehearsal database: %v", err)
	}

	if err := os.Remove(databasePath); err != nil {
		t.Fatalf("remove upgraded rehearsal database before restore: %v", err)
	}
	if err := copyTask506DatabaseFile(backupPath, databasePath); err != nil {
		t.Fatalf("restore schema 19 rehearsal backup: %v", err)
	}
	if restoredChecksum := task506FileSHA256(t, databasePath); restoredChecksum != backupChecksum {
		t.Fatalf("restored backup checksum mismatch: got %s want %s", restoredChecksum, backupChecksum)
	}

	restored, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("open restored schema 19 database: %v", err)
	}
	restored.SetMaxOpenConns(1)
	assertMigrationVersion(t, restored, 19)
	assertTask506QuickCheck(t, restored)
	restoredSnapshot, err := prepareTask50Upgrade(context.Background(), restored, 19)
	if err != nil {
		t.Fatalf("verify restored schema 19 preflight: %v", err)
	}
	if err := restored.Close(); err != nil {
		t.Fatalf("close restored schema 19 database: %v", err)
	}

	forwardFixed, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatalf("open restored database for schema 21 forward-fix: %v", err)
	}
	forwardFixed.SetMaxOpenConns(1)
	runMigrationsTo(t, forwardFixed, 21)
	defer forwardFixed.Close()
	assertMigrationVersion(t, forwardFixed, 21)
	assertTask506QuickCheck(t, forwardFixed)
	if err := verifyTask50MigrationSnapshot(context.Background(), forwardFixed, restoredSnapshot); err != nil {
		t.Fatalf("verify forward-fix conservation: %v", err)
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

func seedTask506MigrationConservationFixture(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, err := database.Exec(`
		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, color, is_archived, created_at, updated_at
		) VALUES (
			'migration-category', 'ledger-a', 'user-a', 'Anonymous Category',
			'expense', '#16a34a', 0, '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
		);
		INSERT INTO tags (
			id, ledger_id, name, owner_user_id, color, is_archived, created_at, updated_at
		) VALUES (
			'migration-tag', 'ledger-a', 'Anonymous Tag', 'user-a',
			'#16a34a', 0, '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
		);
		INSERT INTO accounts (
			id, ledger_id, owner_user_id, name, type, currency, initial_balance,
			is_archived, created_at, updated_at
		) VALUES (
			'migration-account', 'ledger-a', 'user-a', 'Anonymous Account', 'cash',
			'CNY', 123, 0, '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
		);
		INSERT INTO transaction_defaults (
			ledger_id, user_id, type, category_id, account_id, payer_user_id,
			visibility, split_method, tag_names, updated_at
		) VALUES (
			'ledger-a', 'user-a', 'expense', 'migration-category', 'migration-account',
			'user-a', 'partner_readable', 'equal', 'Anonymous Tag', '2026-07-17T08:00:00Z'
		);
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id,
			visibility, split_method, attachment_paths, status, created_at, updated_at
		) VALUES (
			'migration-transaction', 'ledger-a', 'shared_expense', 'Anonymous Expense',
			12345, 'CNY', '2026-07-17T08:00:00Z', 'user-a', 'user-a', 'user-a',
			'migration-account', 'migration-category', 'shared', 'equal',
			'["/uploads/anonymous.png"]', 'normal',
			'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
		);
		INSERT INTO transaction_tags (transaction_id, tag_id)
		VALUES ('migration-transaction', 'migration-tag');
		INSERT INTO transaction_splits (
			id, transaction_id, user_id, share_amount, share_ratio, created_at, updated_at
		) VALUES (
			'migration-split', 'migration-transaction', 'user-a', 12345, 100,
			'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
		);
		INSERT INTO settlements (
			id, ledger_id, from_user_id, to_user_id, amount, currency, occurred_at,
			note, created_by_user_id, created_at
		) VALUES (
			'migration-settlement', 'ledger-a', 'user-a', 'user-b', 678, 'CNY',
			'2026-07-17T08:00:00Z', 'Anonymous Settlement', 'user-a',
			'2026-07-17T08:00:00Z'
		);
		INSERT INTO transaction_templates (
			id, ledger_id, name, type, title, amount_cents, category_id, account_id,
			payer_user_id, created_by_user_id, created_at, updated_at
		) VALUES (
			'migration-template', 'ledger-a', 'Anonymous Template', 'expense',
			'Anonymous Template', 100, 'migration-category', 'migration-account',
			'user-a', 'user-a', '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
		);
		INSERT INTO recurring_rules (
			id, ledger_id, name, type, title, amount_cents, category_id,
			payer_user_id, frequency, next_due_date, created_by_user_id,
			created_at, updated_at
		) VALUES (
			'migration-recurring', 'ledger-a', 'Anonymous Recurring', 'expense',
			'Anonymous Recurring', 100, 'migration-category', 'user-a',
			'monthly', '2026-08-01', 'user-a',
			'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
		);
		INSERT INTO recurring_reminders (
			id, ledger_id, rule_id, due_date, status, created_at, updated_at
		) VALUES (
			'migration-reminder', 'ledger-a', 'migration-recurring', '2026-07-17',
			'pending', '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
		);
		INSERT INTO import_batches (
			id, ledger_id, filename, created_by_user_id, status, source_type,
			file_sha256, total_rows, new_rows, imported_rows, file_format,
			parser_metadata_json, created_at, updated_at
		) VALUES (
			'migration-batch', 'ledger-a', 'anonymous.csv', 'user-a', 'completed',
			'alipay', 'anonymous-file-hash', 1, 1, 1, 'csv', '{}',
			'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
		);
		INSERT INTO import_items (
			id, batch_id, transaction_id, import_hash, status, row_number, source_type,
			title, merchant, amount_cents, direction, target_transaction_type,
			duplicate_status, row_status, normalized_json, visibility, created_at
		) VALUES (
			'migration-item', 'migration-batch', 'migration-transaction',
			'anonymous-item-hash', 'imported', 1, 'alipay', 'Anonymous Item',
			'Anonymous Merchant', 12345, 'out', 'expense', 'new', 'imported',
			'{}', 'shared', '2026-07-17T08:00:00Z'
		);
		INSERT INTO transaction_import_refs (
			id, ledger_id, transaction_id, import_batch_id, import_row_id,
			import_hash, source_type, created_at
		) VALUES (
			'migration-ref', 'ledger-a', 'migration-transaction', 'migration-batch',
			'migration-item', 'anonymous-item-hash', 'alipay', '2026-07-17T08:00:00Z'
		);
		INSERT INTO import_rules (
			id, ledger_id, keyword, category_id, tag_names, account_id,
			created_by_user_id, name, match_type, pattern, priority, result_json,
			status, created_at, updated_at
		) VALUES (
			'migration-rule', 'ledger-a', 'Anonymous', 'migration-category',
			'Anonymous Tag', 'migration-account', 'user-a', 'Anonymous Rule',
			'merchant_contains', 'Anonymous', 100, '{}', 'active',
			'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
		);
		INSERT INTO audit_logs (
			id, ledger_id, actor_user_id, action, entity_type, entity_id,
			after_json, created_at
		) VALUES (
			'migration-audit', 'ledger-a', 'user-a', 'transaction_create',
			'transaction', 'migration-transaction', '{}', '2026-07-17T08:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed Task50.6 migration fixture: %v", err)
	}
}

func copyTask506DatabaseFile(sourcePath string, destinationPath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destinationPath, data, 0o600)
}

func task506FileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read checksum file %s: %v", path, err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func assertTask506QuickCheck(t *testing.T, database *sql.DB) {
	t.Helper()
	results, err := readQuickCheckResults(context.Background(), database)
	if err != nil {
		t.Fatalf("read quick_check results: %v", err)
	}
	if err := validateQuickCheckResults(results); err != nil {
		t.Fatalf("quick_check failed: %v", err)
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
