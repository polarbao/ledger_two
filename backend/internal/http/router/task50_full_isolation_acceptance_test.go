package router

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTask506FullModuleReadsAndExportsStayInsideExplicitLedger(t *testing.T) {
	database := setupRBACRouterDB(t)
	cfg := rbacRouterConfig(t)
	httpRouter := New(database, cfg)
	fixture := seedRBACLedger(t, database)
	ledgerBID := insertTask50Ledger(t, database, "task506-ledger-b", fixture.UserAID)
	seedTask506LedgerIsolationFixture(t, database, fixture.LedgerID, ledgerBID, fixture.UserAID)

	writeRBACAttachmentFixture(t, cfg.UploadDir, "leak-a.txt", []byte("LEAK-A-ATTACHMENT"))
	writeRBACAttachmentFixture(t, cfg.UploadDir, "visible-b.txt", []byte("VISIBLE-B-ATTACHMENT"))

	readCases := []struct {
		name       string
		path       string
		visible    string
		forbidden  string
		statusCode int
	}{
		{name: "transactions", path: "/api/transactions?month=2026-07", visible: "VISIBLE-B-TRANSACTION", forbidden: "LEAK-A-TRANSACTION", statusCode: http.StatusOK},
		{name: "metadata categories", path: "/api/metadata/categories/", visible: "VISIBLE-B-CATEGORY", forbidden: "LEAK-A-CATEGORY", statusCode: http.StatusOK},
		{name: "metadata tags", path: "/api/metadata/tags/", visible: "VISIBLE-B-TAG", forbidden: "LEAK-A-TAG", statusCode: http.StatusOK},
		{name: "metadata accounts", path: "/api/metadata/accounts/", visible: "VISIBLE-B-ACCOUNT", forbidden: "LEAK-A-ACCOUNT", statusCode: http.StatusOK},
		{name: "settlements", path: "/api/settlements?month=2026-07", visible: "VISIBLE-B-SETTLEMENT", forbidden: "LEAK-A-SETTLEMENT", statusCode: http.StatusOK},
		{name: "templates", path: "/api/transaction-templates?include_archived=true", visible: "VISIBLE-B-TEMPLATE", forbidden: "LEAK-A-TEMPLATE", statusCode: http.StatusOK},
		{name: "recurring rules", path: "/api/recurring-rules", visible: "VISIBLE-B-RECURRING", forbidden: "LEAK-A-RECURRING", statusCode: http.StatusOK},
		{name: "recurring reminders", path: "/api/recurring-reminders", visible: "VISIBLE-B-RECURRING", forbidden: "LEAK-A-RECURRING", statusCode: http.StatusOK},
		{name: "import rules", path: "/api/import-rules?status=all", visible: "VISIBLE-B-RULE", forbidden: "LEAK-A-RULE", statusCode: http.StatusOK},
		{name: "dashboard", path: "/api/dashboard?month=2026-07", visible: "VISIBLE-B-TRANSACTION", forbidden: "LEAK-A-TRANSACTION", statusCode: http.StatusOK},
		{name: "monthly report", path: "/api/reports/monthly-summary?month=2026-07", visible: "2222", forbidden: "991111", statusCode: http.StatusOK},
		{name: "json export", path: "/api/export/full.json", visible: "VISIBLE-B-TRANSACTION", forbidden: "LEAK-A-", statusCode: http.StatusOK},
		{name: "csv export", path: "/api/export/transactions.csv?month=2026-07", visible: "VISIBLE-B-TRANSACTION", forbidden: "LEAK-A-TRANSACTION", statusCode: http.StatusOK},
	}

	for _, tc := range readCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := task506RouterRequest(
				t,
				httpRouter,
				fixture.UserAID,
				http.MethodGet,
				tc.path,
				ledgerBID,
				nil,
			)
			if recorder.Code != tc.statusCode {
				t.Fatalf("expected %d, got %d body: %s", tc.statusCode, recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), tc.visible) {
				t.Fatalf("response does not contain current-ledger marker %q: %s", tc.visible, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), tc.forbidden) {
				t.Fatalf("response leaked foreign-ledger marker %q: %s", tc.forbidden, recorder.Body.String())
			}
		})
	}

	t.Run("import batch object", func(t *testing.T) {
		foreign := task506RouterRequest(
			t,
			httpRouter,
			fixture.UserAID,
			http.MethodGet,
			"/api/imports/batch-a",
			ledgerBID,
			nil,
		)
		if foreign.Code != http.StatusNotFound || strings.Contains(foreign.Body.String(), "LEAK-A-") {
			t.Fatalf("foreign import batch was discoverable: %d %s", foreign.Code, foreign.Body.String())
		}

		current := task506RouterRequest(
			t,
			httpRouter,
			fixture.UserAID,
			http.MethodGet,
			"/api/imports/batch-b",
			ledgerBID,
			nil,
		)
		if current.Code != http.StatusOK || !strings.Contains(current.Body.String(), "VISIBLE-B-IMPORT") {
			t.Fatalf("current import batch unavailable: %d %s", current.Code, current.Body.String())
		}
	})

	t.Run("attachment object and bare uploads", func(t *testing.T) {
		foreign := task506RouterRequest(
			t,
			httpRouter,
			fixture.UserAID,
			http.MethodGet,
			"/api/attachments/leak-a.txt",
			ledgerBID,
			nil,
		)
		if foreign.Code != http.StatusNotFound {
			t.Fatalf("foreign attachment returned %d: %s", foreign.Code, foreign.Body.String())
		}

		current := task506RouterRequest(
			t,
			httpRouter,
			fixture.UserAID,
			http.MethodGet,
			"/api/attachments/visible-b.txt",
			ledgerBID,
			nil,
		)
		if current.Code != http.StatusOK || current.Body.String() != "VISIBLE-B-ATTACHMENT" {
			t.Fatalf("current attachment unavailable: %d %s", current.Code, current.Body.String())
		}

		bare := task506RouterRequest(
			t,
			httpRouter,
			fixture.UserAID,
			http.MethodGet,
			"/uploads/leak-a.txt",
			ledgerBID,
			nil,
		)
		if bare.Code != http.StatusNotFound {
			t.Fatalf("bare uploads path exposed attachment with status %d", bare.Code)
		}
	})
}

func TestTask506ReplacementMemberVisibilityKeepsHistoryWithoutPrivateLeak(t *testing.T) {
	database := setupRBACRouterDB(t)
	httpRouter := New(database, rbacRouterConfig(t))
	fixture := seedRBACLedger(t, database)
	replacementID := insertRBACUser(t, database, "task506-replacement", "Replacement Member")
	seedTask506ReplacementHistory(t, database, fixture)

	if _, err := database.Exec(
		"DELETE FROM ledger_members WHERE ledger_id = ? AND user_id = ?",
		fixture.LedgerID,
		fixture.UserBID,
	); err != nil {
		t.Fatalf("remove former fixture member: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES (?, ?, 'viewer', '2026-07-17T08:30:00Z', '2026-07-17T08:30:00Z')
	`, fixture.LedgerID, replacementID); err != nil {
		t.Fatalf("insert replacement fixture member: %v", err)
	}

	replacementList := task506RouterRequest(
		t,
		httpRouter,
		replacementID,
		http.MethodGet,
		"/api/transactions?month=2026-07",
		fixture.LedgerID,
		nil,
	)
	if replacementList.Code != http.StatusOK {
		t.Fatalf("replacement history list failed: %d %s", replacementList.Code, replacementList.Body.String())
	}
	if strings.Contains(replacementList.Body.String(), "FORMER-PRIVATE") ||
		!strings.Contains(replacementList.Body.String(), "FORMER-PARTNER") ||
		!strings.Contains(replacementList.Body.String(), "FORMER-SHARED") {
		t.Fatalf("replacement visibility contract failed: %s", replacementList.Body.String())
	}

	privateDetail := task506RouterRequest(
		t,
		httpRouter,
		replacementID,
		http.MethodGet,
		"/api/transactions/former-private",
		fixture.LedgerID,
		nil,
	)
	assertRouterError(t, privateDetail, http.StatusNotFound, "LEDGER_OBJECT_NOT_FOUND")

	for _, transactionID := range []string{"former-partner", "former-shared"} {
		detail := task506RouterRequest(
			t,
			httpRouter,
			replacementID,
			http.MethodGet,
			"/api/transactions/"+transactionID,
			fixture.LedgerID,
			nil,
		)
		if detail.Code != http.StatusOK {
			t.Fatalf("replacement cannot read %s: %d %s", transactionID, detail.Code, detail.Body.String())
		}
	}

	settlements := task506RouterRequest(
		t,
		httpRouter,
		replacementID,
		http.MethodGet,
		"/api/settlements?month=2026-07",
		fixture.LedgerID,
		nil,
	)
	if settlements.Code != http.StatusOK || !strings.Contains(settlements.Body.String(), "FORMER-SETTLEMENT") {
		t.Fatalf("replacement cannot read historical settlement: %d %s", settlements.Code, settlements.Body.String())
	}

	former := task506RouterRequest(
		t,
		httpRouter,
		fixture.UserBID,
		http.MethodGet,
		"/api/transactions",
		fixture.LedgerID,
		nil,
	)
	assertRouterError(t, former, http.StatusForbidden, "LEDGER_ACCESS_DENIED")
}

func seedTask506LedgerIsolationFixture(
	t *testing.T,
	database *sql.DB,
	ledgerAID string,
	ledgerBID string,
	userID string,
) {
	t.Helper()
	insertMetadata := func(ledgerID string, suffix string, marker string) {
		t.Helper()
		if _, err := database.Exec(`
			INSERT INTO categories (
				id, ledger_id, owner_user_id, name, type, color, is_archived, created_at, updated_at
			) VALUES (?, ?, ?, ?, 'expense', '#16a34a', 0, '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z');
			INSERT INTO tags (
				id, ledger_id, name, owner_user_id, color, is_archived, created_at, updated_at
			) VALUES (?, ?, ?, ?, '#16a34a', 0, '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z');
			INSERT INTO accounts (
				id, ledger_id, owner_user_id, name, type, currency, is_archived, created_at, updated_at
			) VALUES (?, ?, ?, ?, 'cash', 'CNY', 0, '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z')
		`,
			"category-"+suffix, ledgerID, userID, marker+"CATEGORY",
			"tag-"+suffix, ledgerID, marker+"TAG", userID,
			"account-"+suffix, ledgerID, userID, marker+"ACCOUNT",
		); err != nil {
			t.Fatalf("insert %s metadata: %v", ledgerID, err)
		}
	}
	insertMetadata(ledgerAID, "a", "LEAK-A-")
	insertMetadata(ledgerBID, "b", "VISIBLE-B-")

	insertLedgerObjects := func(
		ledgerID string,
		suffix string,
		marker string,
		amount int64,
		attachment string,
	) {
		t.Helper()
		statement := fmt.Sprintf(`
			INSERT INTO transactions (
				id, ledger_id, type, title, amount, currency, occurred_at,
				owner_user_id, created_by_user_id, payer_user_id, account_id, category_id,
				visibility, attachment_paths, status, created_at, updated_at
			) VALUES (
				'transaction-%[1]s', ?, 'expense', ?, %[2]d, 'CNY', '2026-07-17T08:00:00Z',
				?, ?, ?, 'account-%[1]s', 'category-%[1]s',
				'partner_readable', ?, 'normal', '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
			);
			INSERT INTO transaction_tags (transaction_id, tag_id)
			VALUES ('transaction-%[1]s', 'tag-%[1]s');
			INSERT INTO settlements (
				id, ledger_id, from_user_id, to_user_id, amount, currency, occurred_at,
				note, created_by_user_id, created_at
			) VALUES (
				'settlement-%[1]s', ?, ?, ?, 100, 'CNY', '2026-07-17T08:00:00Z',
				?, ?, '2026-07-17T08:00:00Z'
			);
			INSERT INTO transaction_templates (
				id, ledger_id, name, type, title, amount_cents, category_id, account_id,
				payer_user_id, created_by_user_id, created_at, updated_at
			) VALUES (
				'template-%[1]s', ?, ?, 'expense', ?, 100, 'category-%[1]s', 'account-%[1]s',
				?, ?, '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
			);
			INSERT INTO recurring_rules (
				id, ledger_id, name, type, title, amount_cents, category_id,
				payer_user_id, frequency, next_due_date, created_by_user_id,
				created_at, updated_at
			) VALUES (
				'recurring-%[1]s', ?, ?, 'expense', ?, 100, 'category-%[1]s',
				?, 'monthly', '2026-09-01', ?,
				'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
			);
			INSERT INTO recurring_reminders (
				id, ledger_id, rule_id, due_date, status, created_at, updated_at
			) VALUES (
				'reminder-%[1]s', ?, 'recurring-%[1]s', '2026-07-17', 'pending',
				'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
			);
			INSERT INTO import_batches (
				id, ledger_id, filename, created_by_user_id, status, source_type,
				file_sha256, total_rows, new_rows, file_format, parser_metadata_json,
				created_at, updated_at
			) VALUES (
				'batch-%[1]s', ?, ?, ?, 'completed', 'alipay', ?,
				1, 1, 'csv', '{}', '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
			);
			INSERT INTO import_items (
				id, batch_id, import_hash, status, row_number, source_type, title, merchant,
				amount_cents, direction, target_transaction_type, duplicate_status,
				row_status, normalized_json, visibility, created_at
			) VALUES (
				'item-%[1]s', 'batch-%[1]s', 'hash-item-%[1]s', 'skipped', 1, 'alipay', ?, ?,
				%[2]d, 'out', 'expense', 'new', 'pending', '{}', 'private',
				'2026-07-17T08:00:00Z'
			);
			INSERT INTO import_rules (
				id, ledger_id, keyword, created_by_user_id, name, match_type, pattern,
				priority, result_json, status, created_at, updated_at
			) VALUES (
				'rule-%[1]s', ?, ?, ?, ?, 'merchant_contains', ?, 100, '{}', 'active',
				'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
			);
			INSERT INTO audit_logs (
				id, ledger_id, actor_user_id, actor_role, action, entity_type,
				entity_id, after_json, created_at
			) VALUES (
				'audit-%[1]s', ?, ?, 'owner', 'transaction_create', 'transaction',
				'transaction-%[1]s', ?, '2026-07-17T08:00:00Z'
			)
		`, suffix, amount)
		if _, err := database.Exec(
			statement,
			ledgerID, marker+"TRANSACTION", userID, userID, userID, `["`+attachment+`"]`,
			ledgerID, userID, userID, marker+"SETTLEMENT", userID,
			ledgerID, marker+"TEMPLATE", marker+"TEMPLATE", userID, userID,
			ledgerID, marker+"RECURRING", marker+"RECURRING", userID, userID,
			ledgerID,
			ledgerID, marker+"IMPORT.csv", userID, "hash-batch-"+suffix,
			marker+"IMPORT", marker+"IMPORT-MERCHANT",
			ledgerID, marker+"RULE", userID, marker+"RULE", marker+"RULE",
			ledgerID, userID, `{"marker":"`+marker+`AUDIT"}`,
		); err != nil {
			t.Fatalf("insert %s business objects: %v", ledgerID, err)
		}
	}
	insertLedgerObjects(ledgerAID, "a", "LEAK-A-", 991111, "/uploads/leak-a.txt")
	insertLedgerObjects(ledgerBID, "b", "VISIBLE-B-", 2222, "/uploads/visible-b.txt")
}

func seedTask506ReplacementHistory(t *testing.T, database *sql.DB, fixture rbacFixture) {
	t.Helper()
	if _, err := database.Exec(`
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, visibility,
			split_method, status, created_at, updated_at
		) VALUES
			(
				'former-private', ?, 'expense', 'FORMER-PRIVATE', 100, 'CNY',
				'2026-07-17T08:00:00Z', ?, ?, ?, 'private', NULL, 'normal',
				'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
			),
			(
				'former-partner', ?, 'expense', 'FORMER-PARTNER', 200, 'CNY',
				'2026-07-17T08:00:00Z', ?, ?, ?, 'partner_readable', NULL, 'normal',
				'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
			),
			(
				'former-shared', ?, 'shared_expense', 'FORMER-SHARED', 300, 'CNY',
				'2026-07-17T08:00:00Z', ?, ?, ?, 'shared', 'equal', 'normal',
				'2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'
			);
		INSERT INTO transaction_splits (
			id, transaction_id, user_id, share_amount, created_at, updated_at
		) VALUES
			('former-split-owner', 'former-shared', ?, 150, '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z'),
			('former-split-member', 'former-shared', ?, 150, '2026-07-17T08:00:00Z', '2026-07-17T08:00:00Z');
		INSERT INTO settlements (
			id, ledger_id, from_user_id, to_user_id, amount, currency, occurred_at,
			note, created_by_user_id, created_at
		) VALUES (
			'former-settlement', ?, ?, ?, 50, 'CNY', '2026-07-17T08:00:00Z',
			'FORMER-SETTLEMENT', ?, '2026-07-17T08:00:00Z'
		)
	`,
		fixture.LedgerID, fixture.UserBID, fixture.UserBID, fixture.UserBID,
		fixture.LedgerID, fixture.UserBID, fixture.UserBID, fixture.UserBID,
		fixture.LedgerID, fixture.UserBID, fixture.UserBID, fixture.UserBID,
		fixture.UserAID, fixture.UserBID,
		fixture.LedgerID, fixture.UserBID, fixture.UserAID, fixture.UserAID,
	); err != nil {
		t.Fatalf("insert replacement history: %v", err)
	}
}

func task506RouterRequest(
	t *testing.T,
	httpRouter http.Handler,
	userID string,
	method string,
	path string,
	ledgerID string,
	body []byte,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("X-Ledger-Id", ledgerID)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.AddCookie(authCookie(t, userID))
	recorder := httptest.NewRecorder()
	httpRouter.ServeHTTP(recorder, request)
	return recorder
}

func TestTask506IsolationFixtureDoesNotWriteOutsideTempDirectories(t *testing.T) {
	cfg := rbacRouterConfig(t)
	runtimeRoot := filepath.Dir(filepath.Dir(cfg.DSN))
	for _, directory := range []string{cfg.UploadDir, cfg.BackupDir, cfg.LogDir} {
		if !strings.HasPrefix(filepath.Clean(directory), filepath.Clean(runtimeRoot)) {
			t.Fatalf("test runtime directory escaped temporary root: %s", directory)
		}
	}
	if _, err := os.Stat(runtimeRoot); err != nil && !os.IsNotExist(err) {
		t.Fatalf("inspect temporary runtime root: %v", err)
	}
}
