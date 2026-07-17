package safety

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	ledgerctx "ledger_two/internal/ledger"
	"ledger_two/migrations"
)

func TestTask506LedgerJSONExportIsCompleteVisibleAndIsolated(t *testing.T) {
	database := setupTask506ExportDB(t)
	seedTask506ExportFixture(t, database)

	ctx := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID:     "user-owner",
		LedgerID:   "ledger-a",
		Role:       ledgerctx.RoleOwner,
		Status:     ledgerctx.LedgerStatusActive,
		Version:    4,
		IsExplicit: true,
	})
	payload, err := NewService(database, nil).ExportJSON(ctx, "user-owner")
	if err != nil {
		t.Fatalf("export ledger JSON: %v", err)
	}

	var exported map[string]any
	if err := json.Unmarshal(payload, &exported); err != nil {
		t.Fatalf("decode ledger JSON export: %v", err)
	}

	manifest := requireExportObject(t, exported, "manifest")
	if manifest["format"] != "ledger_two_ledger_export" ||
		manifest["purpose"] != "portable_read_only_snapshot" ||
		manifest["restorable"] != false ||
		manifest["schema_version"] != float64(21) {
		t.Fatalf("unexpected export manifest: %+v", manifest)
	}
	ledger := requireExportObject(t, manifest, "ledger")
	if ledger["id"] != "ledger-a" || ledger["status"] != "active" || ledger["version"] != float64(4) {
		t.Fatalf("unexpected manifest ledger: %+v", ledger)
	}

	expectedIDs := map[string][]string{
		"users":                   {"user-owner", "user-editor", "user-former"},
		"categories":              {"category-a"},
		"tags":                    {"tag-a"},
		"accounts":                {"account-a"},
		"transaction_defaults":    {"user-owner"},
		"transactions":            {"transaction-a"},
		"transaction_tags":        {"transaction-a"},
		"transaction_splits":      {"split-a"},
		"settlements":             {"settlement-a"},
		"transaction_templates":   {"template-a"},
		"recurring_rules":         {"recurring-a"},
		"recurring_reminders":     {"reminder-a"},
		"import_batches":          {"batch-a"},
		"import_items":            {"item-a"},
		"transaction_import_refs": {"import-ref-a"},
		"import_rules":            {"rule-a"},
		"audit_logs":              {"audit-a", "audit-import-a"},
	}
	for section, ids := range expectedIDs {
		for _, id := range ids {
			if !exportSectionContainsID(exported, section, id) {
				t.Fatalf("export section %s does not contain %s: %+v", section, id, exported[section])
			}
		}
	}

	for _, forbidden := range []string{
		"ledger-b",
		"category-b",
		"tag-b",
		"account-b",
		"transaction-b",
		"template-b",
		"recurring-b",
		"batch-b",
		"item-b",
		"rule-b",
		"audit-b",
		"private-hidden",
		"user-private",
		"user-outsider",
		"batch-private",
		"item-private",
	} {
		if exportContainsString(exported, forbidden) {
			t.Fatalf("ledger export leaked forbidden marker %q", forbidden)
		}
	}

	transactions := requireExportArray(t, exported, "transactions")
	transaction := transactions[0].(map[string]any)
	attachments, ok := transaction["attachment_paths"].([]any)
	if !ok || len(attachments) != 1 || attachments[0] != "receipt-a.png" {
		t.Fatalf("visible attachment references missing from transaction export: %+v", transaction)
	}
	if _, exists := exported["app_settings"]; exists {
		t.Fatal("ledger export must not contain instance app_settings")
	}
	if _, exists := exported["instance_admins"]; exists {
		t.Fatal("ledger export must not contain instance administration data")
	}

	csvPayload, err := NewService(database, nil).ExportCSV(ctx, "user-owner", "2026-07")
	if err != nil {
		t.Fatalf("export ledger CSV: %v", err)
	}
	csvText := string(csvPayload)
	if !strings.Contains(csvText, "Visible A") || !strings.Contains(csvText, "Former Member") {
		t.Fatalf("CSV export lost visible historical actor data: %s", csvText)
	}
	for _, forbidden := range []string{"Private Hidden", "Ledger B Secret", "Global Outsider"} {
		if strings.Contains(csvText, forbidden) {
			t.Fatalf("CSV export leaked %q: %s", forbidden, csvText)
		}
	}

	editorContext := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID:     "user-editor",
		LedgerID:   "ledger-a",
		Role:       ledgerctx.RoleEditor,
		Status:     ledgerctx.LedgerStatusActive,
		Version:    4,
		IsExplicit: true,
	})
	editorPayload, err := NewService(database, nil).ExportJSON(editorContext, "user-editor")
	if err != nil {
		t.Fatalf("export editor ledger JSON: %v", err)
	}
	var editorExport map[string]any
	if err := json.Unmarshal(editorPayload, &editorExport); err != nil {
		t.Fatalf("decode editor ledger JSON export: %v", err)
	}
	for _, section := range []string{"import_batches", "import_items", "import_rules"} {
		if records := requireExportArray(t, editorExport, section); len(records) != 0 {
			t.Fatalf("editor export included owner-only %s: %+v", section, records)
		}
	}
	if exportSectionContainsID(editorExport, "audit_logs", "audit-import-a") {
		t.Fatalf("editor export included owner-only import audit: %+v", editorExport["audit_logs"])
	}
	if !exportSectionContainsID(editorExport, "transaction_import_refs", "import-ref-a") {
		t.Fatalf("editor export lost visible transaction import reference: %+v", editorExport["transaction_import_refs"])
	}
	for _, forbidden := range []string{"private-hidden", "batch-private", "item-private", "user-private"} {
		if exportContainsString(editorExport, forbidden) {
			t.Fatalf("editor export leaked private import marker %q", forbidden)
		}
	}
}

func setupTask506ExportDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open export database: %v", err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set migration dialect: %v", err)
	}
	if err := goose.Up(database, "."); err != nil {
		t.Fatalf("run export migrations: %v", err)
	}
	return database
}

func seedTask506ExportFixture(t *testing.T, database *sql.DB) {
	t.Helper()
	const now = "2026-07-17T08:00:00Z"
	statement := `
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at) VALUES
			('user-owner', 'owner', 'Current Owner', 'hash', 'user', ?, ?),
			('user-editor', 'editor', 'Current Editor', 'hash', 'user', ?, ?),
			('user-former', 'former', 'Former Member', 'hash', 'user', ?, ?),
			('user-private', 'private-user', 'Private Historical User', 'hash', 'user', ?, ?),
			('user-outsider', 'outsider', 'Global Outsider', 'hash', 'user', ?, ?);

		INSERT INTO ledgers (id, name, default_currency, status, version, created_at, updated_at) VALUES
			('ledger-a', 'Ledger A', 'CNY', 'active', 4, ?, ?),
			('ledger-b', 'Ledger B', 'CNY', 'active', 2, ?, ?);

		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at) VALUES
			('ledger-a', 'user-owner', 'owner', ?, ?),
			('ledger-a', 'user-editor', 'editor', ?, ?),
			('ledger-b', 'user-owner', 'owner', ?, ?);

		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, color, sort_order, is_system,
			is_archived, created_at, updated_at
		) VALUES
			('category-a', 'ledger-a', 'user-owner', 'Category A', 'expense', '#16a34a', 0, 0, 0, ?, ?),
			('category-b', 'ledger-b', 'user-owner', 'Category B', 'expense', '#dc2626', 0, 0, 0, ?, ?);

		INSERT INTO tags (
			id, ledger_id, name, owner_user_id, color, sort_order, is_archived, created_at, updated_at
		) VALUES
			('tag-a', 'ledger-a', 'Tag A', 'user-owner', '#16a34a', 0, 0, ?, ?),
			('tag-b', 'ledger-b', 'Tag B', 'user-owner', '#dc2626', 0, 0, ?, ?);

		INSERT INTO accounts (
			id, ledger_id, owner_user_id, name, type, currency, initial_balance,
			sort_order, is_archived, created_at, updated_at
		) VALUES
			('account-a', 'ledger-a', 'user-owner', 'Account A', 'cash', 'CNY', 0, 0, 0, ?, ?),
			('account-b', 'ledger-b', 'user-owner', 'Account B', 'cash', 'CNY', 0, 0, 0, ?, ?);

		INSERT INTO transaction_defaults (
			ledger_id, user_id, type, category_id, account_id, payer_user_id,
			visibility, split_method, tag_names, updated_at
		) VALUES (
			'ledger-a', 'user-owner', 'expense', 'category-a', 'account-a', 'user-owner',
			'partner_readable', 'equal', 'Tag A', ?
		);

		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id,
			visibility, split_method, note, attachment_paths, status, created_at, updated_at
		) VALUES
			(
				'transaction-a', 'ledger-a', 'shared_expense', 'Visible A', 12345, 'CNY', ?,
				'user-former', 'user-former', 'user-former', 'account-a', 'category-a',
				'shared', 'equal', 'visible note', '["receipt-a.png"]', 'normal', ?, ?
			),
			(
				'private-hidden', 'ledger-a', 'expense', 'Private Hidden', 999, 'CNY', ?,
				'user-private', 'user-private', 'user-private', NULL, 'category-a',
				'private', NULL, NULL, NULL, 'normal', ?, ?
			),
			(
				'transaction-b', 'ledger-b', 'expense', 'Ledger B Secret', 888, 'CNY', ?,
				'user-owner', 'user-owner', 'user-owner', 'account-b', 'category-b',
				'partner_readable', NULL, NULL, NULL, 'normal', ?, ?
			);

		INSERT INTO transaction_tags (transaction_id, tag_id) VALUES
			('transaction-a', 'tag-a'),
			('transaction-b', 'tag-b');

		INSERT INTO transaction_splits (
			id, transaction_id, user_id, share_amount, share_ratio, created_at, updated_at
		) VALUES ('split-a', 'transaction-a', 'user-former', 12345, 100, ?, ?);

		INSERT INTO settlements (
			id, ledger_id, from_user_id, to_user_id, amount, currency, occurred_at,
			note, created_by_user_id, created_at
		) VALUES (
			'settlement-a', 'ledger-a', 'user-former', 'user-owner', 500, 'CNY', ?,
			'Historical settlement', 'user-owner', ?
		);

		INSERT INTO transaction_templates (
			id, ledger_id, name, type, title, amount_cents, category_id, account_id,
			payer_user_id, split_method, tag_names, note, created_by_user_id,
			is_archived, created_at, updated_at
		) VALUES
			(
				'template-a', 'ledger-a', 'Template A', 'expense', 'Template A', 1200,
				'category-a', 'account-a', 'user-owner', 'equal', 'Tag A', NULL,
				'user-owner', 0, ?, ?
			),
			(
				'template-b', 'ledger-b', 'Template B', 'expense', 'Template B', 1200,
				'category-b', 'account-b', 'user-owner', 'equal', 'Tag B', NULL,
				'user-owner', 0, ?, ?
			);

		INSERT INTO recurring_rules (
			id, ledger_id, name, type, title, amount_cents, category_id, payer_user_id,
			split_method, tag_names, note, frequency, next_due_date,
			created_by_user_id, created_at, updated_at
		) VALUES
			(
				'recurring-a', 'ledger-a', 'Recurring A', 'expense', 'Recurring A', 600,
				'category-a', 'user-owner', 'equal', 'Tag A', NULL, 'monthly',
				'2026-08-01', 'user-owner', ?, ?
			),
			(
				'recurring-b', 'ledger-b', 'Recurring B', 'expense', 'Recurring B', 600,
				'category-b', 'user-owner', 'equal', 'Tag B', NULL, 'monthly',
				'2026-08-01', 'user-owner', ?, ?
			);

		INSERT INTO recurring_reminders (
			id, ledger_id, rule_id, due_date, status, created_at, updated_at
		) VALUES ('reminder-a', 'ledger-a', 'recurring-a', '2026-07-17', 'pending', ?, ?);

		INSERT INTO import_batches (
			id, ledger_id, filename, created_by_user_id, status, source_type, file_sha256,
			total_rows, new_rows, imported_rows, file_format, parser_metadata_json,
			created_at, updated_at
		) VALUES
			(
				'batch-a', 'ledger-a', 'anonymous-a.csv', 'user-owner', 'completed', 'alipay',
				'hash-a', 1, 1, 1, 'csv', '{}', ?, ?
			),
			(
				'batch-b', 'ledger-b', 'anonymous-b.csv', 'user-owner', 'completed', 'alipay',
				'hash-b', 1, 1, 1, 'csv', '{}', ?, ?
			),
			(
				'batch-private', 'ledger-a', 'private.csv', 'user-private', 'completed', 'alipay',
				'hash-private', 1, 1, 1, 'csv', '{}', ?, ?
			);

		INSERT INTO import_items (
			id, batch_id, transaction_id, import_hash, status, row_number, source_type,
			occurred_at, title, amount_cents, direction, target_transaction_type,
			duplicate_status, row_status, normalized_json, visibility, created_at
		) VALUES
			(
				'item-a', 'batch-a', 'transaction-a', 'item-hash-a', 'imported', 1, 'alipay',
				?, 'Imported A', 12345, 'out', 'expense', 'new', 'imported', '{}', 'shared', ?
			),
			(
				'item-b', 'batch-b', 'transaction-b', 'item-hash-b', 'imported', 1, 'alipay',
				?, 'Imported B', 888, 'out', 'expense', 'new', 'imported', '{}', 'private', ?
			),
			(
				'item-private', 'batch-private', 'private-hidden', 'item-hash-private', 'imported', 1, 'alipay',
				?, 'Imported Private', 999, 'out', 'expense', 'new', 'imported', '{}', 'private', ?
			);

		INSERT INTO transaction_import_refs (
			id, ledger_id, transaction_id, import_batch_id, import_row_id, import_hash,
			source_type, created_at
		) VALUES (
			'import-ref-a', 'ledger-a', 'transaction-a', 'batch-a', 'item-a',
			'item-hash-a', 'alipay', ?
		);

		INSERT INTO import_rules (
			id, ledger_id, keyword, category_id, tag_names, account_id, created_by_user_id,
			name, match_type, pattern, priority, result_json, status, created_at, updated_at
		) VALUES
			(
				'rule-a', 'ledger-a', 'A', 'category-a', 'Tag A', 'account-a', 'user-owner',
				'Rule A', 'merchant_contains', 'A', 100, '{}', 'active', ?, ?
			),
			(
				'rule-b', 'ledger-b', 'B', 'category-b', 'Tag B', 'account-b', 'user-owner',
				'Rule B', 'merchant_contains', 'B', 100, '{}', 'active', ?, ?
			);

		INSERT INTO audit_logs (
			id, ledger_id, actor_user_id, actor_role, action, entity_type, entity_id,
			after_json, created_at
		) VALUES
			('audit-a', 'ledger-a', 'user-former', 'editor', 'transaction_create', 'transaction', 'transaction-a', '{}', ?),
			('audit-import-a', 'ledger-a', 'user-owner', 'owner', 'import_commit', 'import_batch', 'batch-a', '{"marker":"visible-import"}', ?),
			('audit-import-private', 'ledger-a', 'user-private', 'owner', 'import_commit', 'import_batch', 'batch-private', '{"marker":"private-import"}', ?),
			('audit-b', 'ledger-b', 'user-owner', 'owner', 'transaction_create', 'transaction', 'transaction-b', '{}', ?);

		INSERT INTO app_settings (key, value, updated_at)
		VALUES ('instance-secret-marker', 'must-not-export', ?);
	`
	statement = strings.ReplaceAll(statement, "?", "'"+now+"'")
	if _, err := database.Exec(statement); err != nil {
		t.Fatalf("seed Task50.6 export fixture: %v", err)
	}
}

func requireExportObject(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("export key %s is not an object: %#v", key, parent[key])
	}
	return value
}

func requireExportArray(t *testing.T, exported map[string]any, key string) []any {
	t.Helper()
	value, ok := exported[key].([]any)
	if !ok {
		t.Fatalf("export key %s is not an array: %#v", key, exported[key])
	}
	return value
}

func exportSectionContainsID(exported map[string]any, section string, expectedID string) bool {
	items, ok := exported[section].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if record["id"] == expectedID || record["user_id"] == expectedID || record["transaction_id"] == expectedID {
			return true
		}
	}
	return false
}

func exportContainsString(value any, forbidden string) bool {
	switch typed := value.(type) {
	case string:
		return typed == forbidden
	case []any:
		for _, item := range typed {
			if exportContainsString(item, forbidden) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if exportContainsString(item, forbidden) {
				return true
			}
		}
	}
	return false
}
