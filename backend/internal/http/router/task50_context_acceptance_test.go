package router

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTask50LedgerRoutesRequireExplicitContext(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, rbacRouterConfig(t))
	fixture := seedRBACLedger(t, database)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	req.AddCookie(authCookie(t, fixture.UserAID))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assertRouterError(t, rr, http.StatusBadRequest, "LEDGER_REQUIRED")
}

func TestTask50LedgerPathAndHeaderMustMatch(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, rbacRouterConfig(t))
	fixture := seedRBACLedger(t, database)
	otherLedgerID := insertTask50Ledger(t, database, "other-ledger", fixture.UserAID)

	req := httptest.NewRequest(http.MethodGet, "/api/ledgers/"+fixture.LedgerID+"/members", nil)
	req.Header.Set("X-Ledger-Id", otherLedgerID)
	req.AddCookie(authCookie(t, fixture.UserAID))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assertRouterError(t, rr, http.StatusBadRequest, "LEDGER_CONTEXT_MISMATCH")
}

func TestTask50ArchivedLedgerRejectsBusinessWritesBeforeHandlers(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, rbacRouterConfig(t))
	fixture := seedRBACLedger(t, database)
	archiveTask50Ledger(t, database, fixture.LedgerID, fixture.UserAID)
	if _, err := database.Exec(`
		INSERT INTO recurring_rules (
			id, ledger_id, name, type, title, amount_cents, payer_user_id,
			frequency, next_due_date, created_by_user_id, created_at, updated_at
		) VALUES (
			'archived-due-rule', ?, '归档到期规则', 'expense', '不应生成提醒', 100,
			?, 'monthly', '2026-01-01', ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		)
	`, fixture.LedgerID, fixture.UserAID, fixture.UserAID); err != nil {
		t.Fatalf("insert archived recurring rule: %v", err)
	}

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{name: "transaction", method: http.MethodPost, path: "/api/transactions"},
		{name: "batch tags", method: http.MethodPost, path: "/api/transactions/batch-tag"},
		{name: "metadata", method: http.MethodPost, path: "/api/metadata/categories/"},
		{name: "template", method: http.MethodPost, path: "/api/transaction-templates"},
		{name: "recurring rule", method: http.MethodPost, path: "/api/recurring-rules"},
		{name: "recurring reminder", method: http.MethodPost, path: "/api/recurring-reminders/missing/confirm"},
		{name: "shared expense", method: http.MethodPost, path: "/api/shared-expenses"},
		{name: "settlement", method: http.MethodPost, path: "/api/settlements"},
		{name: "import preview", method: http.MethodPost, path: "/api/imports/preview"},
		{name: "import learn", method: http.MethodPost, path: "/api/imports/missing-batch/rows/missing-row/learn"},
		{name: "import rule", method: http.MethodPost, path: "/api/import-rules"},
		{name: "legacy import", method: http.MethodPost, path: "/api/transactions/import/commit"},
		{name: "attachment upload", method: http.MethodPost, path: "/api/attachments"},
		{name: "member mutation", method: http.MethodPost, path: "/api/ledgers/" + fixture.LedgerID + "/members"},
		{name: "member role", method: http.MethodPatch, path: "/api/ledgers/" + fixture.LedgerID + "/members/" + fixture.UserBID},
		{name: "member remove", method: http.MethodDelete, path: "/api/ledgers/" + fixture.LedgerID + "/members/" + fixture.UserBID},
		{name: "owner transfer", method: http.MethodPost, path: "/api/ledgers/" + fixture.LedgerID + "/members/" + fixture.UserBID + "/transfer-owner"},
		{name: "member leave", method: http.MethodPost, path: "/api/ledgers/" + fixture.LedgerID + "/leave"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(`{}`)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Ledger-Id", fixture.LedgerID)
			req.AddCookie(authCookie(t, fixture.UserAID))
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assertRouterError(t, rr, http.StatusConflict, "LEDGER_ARCHIVED")
		})
	}

	setRBACMemberRole(t, database, fixture.LedgerID, fixture.UserBID, "viewer")
	viewerReq := httptest.NewRequest(http.MethodPost, "/api/imports/preview", bytes.NewReader([]byte(`{}`)))
	viewerReq.Header.Set("Content-Type", "application/json")
	viewerReq.Header.Set("X-Ledger-Id", fixture.LedgerID)
	viewerReq.AddCookie(authCookie(t, fixture.UserBID))
	viewerRecorder := httptest.NewRecorder()
	router.ServeHTTP(viewerRecorder, viewerReq)
	assertRouterError(t, viewerRecorder, http.StatusConflict, "LEDGER_ARCHIVED")

	reqRead := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	reqRead.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqRead.AddCookie(authCookie(t, fixture.UserAID))
	rrRead := httptest.NewRecorder()
	router.ServeHTTP(rrRead, reqRead)
	if rrRead.Code != http.StatusOK {
		t.Fatalf("archived ledger read should remain available, got %d body: %s", rrRead.Code, rrRead.Body.String())
	}

	reminderReq := httptest.NewRequest(http.MethodGet, "/api/recurring-reminders", nil)
	reminderReq.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reminderReq.AddCookie(authCookie(t, fixture.UserAID))
	reminderRecorder := httptest.NewRecorder()
	router.ServeHTTP(reminderRecorder, reminderReq)
	if reminderRecorder.Code != http.StatusOK {
		t.Fatalf("archived recurring reminder read should remain available, got %d body: %s", reminderRecorder.Code, reminderRecorder.Body.String())
	}
	var reminderCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM recurring_reminders WHERE ledger_id = ?", fixture.LedgerID).Scan(&reminderCount); err != nil {
		t.Fatalf("count archived reminders: %v", err)
	}
	if reminderCount != 0 {
		t.Fatalf("archived reminder read generated %d business rows", reminderCount)
	}
	var nextDueDate string
	if err := database.QueryRow("SELECT next_due_date FROM recurring_rules WHERE id = 'archived-due-rule'").Scan(&nextDueDate); err != nil {
		t.Fatalf("read archived recurring rule: %v", err)
	}
	if nextDueDate != "2026-01-01" {
		t.Fatalf("archived reminder read advanced next due date to %s", nextDueDate)
	}
}

func TestTask50TransactionObjectIsScopedToExplicitLedger(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, rbacRouterConfig(t))
	fixture := seedRBACLedger(t, database)
	otherLedgerID := insertTask50Ledger(t, database, "other-ledger", fixture.UserAID)
	insertTask50Transaction(t, database, fixture.LedgerID, fixture.UserAID, "ledger-a-transaction")

	req := httptest.NewRequest(http.MethodGet, "/api/transactions/ledger-a-transaction", nil)
	req.Header.Set("X-Ledger-Id", otherLedgerID)
	req.AddCookie(authCookie(t, fixture.UserAID))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assertRouterError(t, rr, http.StatusNotFound, "LEDGER_OBJECT_NOT_FOUND")

	insertTask50Template(t, database, fixture.LedgerID, fixture.UserAID, "ledger-a-template")
	templateReq := httptest.NewRequest(http.MethodGet, "/api/transaction-templates/ledger-a-template", nil)
	templateReq.Header.Set("X-Ledger-Id", otherLedgerID)
	templateReq.AddCookie(authCookie(t, fixture.UserAID))
	templateRecorder := httptest.NewRecorder()
	router.ServeHTTP(templateRecorder, templateReq)
	assertRouterError(t, templateRecorder, http.StatusNotFound, "LEDGER_OBJECT_NOT_FOUND")
}

func TestTask50RecurringRuleRejectsCrossLedgerReferences(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, rbacRouterConfig(t))
	fixture := seedRBACLedger(t, database)
	otherLedgerID := insertTask50Ledger(t, database, "other-ledger", fixture.UserAID)
	if _, err := database.Exec(`
		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, color, is_archived, created_at, updated_at
		) VALUES (
			'other-ledger-category', ?, ?, '其他账本分类', 'expense', '#22c55e', 0,
			'2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		)
	`, otherLedgerID, fixture.UserAID); err != nil {
		t.Fatalf("insert other-ledger category: %v", err)
	}

	body := bytes.NewBufferString(`{
		"name":"跨账本周期规则",
		"type":"expense",
		"amount_cents":100,
		"category_id":"other-ledger-category",
		"payer_user_id":"` + fixture.UserAID + `",
		"frequency":"monthly",
		"next_due_date":"2026-08-01"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/recurring-rules", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Ledger-Id", fixture.LedgerID)
	req.AddCookie(authCookie(t, fixture.UserAID))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected cross-ledger recurring metadata to return 400, got %d body: %s", rr.Code, rr.Body.String())
	}
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM recurring_rules WHERE name = '跨账本周期规则'").Scan(&count); err != nil {
		t.Fatalf("count recurring rules: %v", err)
	}
	if count != 0 {
		t.Fatalf("cross-ledger recurring rule was persisted")
	}
}

func TestTask50DashboardAndReportsOnlyExposeLedgerMembers(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, rbacRouterConfig(t))
	fixture := seedRBACLedger(t, database)
	outsiderID := insertRBACUser(t, database, "outside-dashboard-user", "Outside Dashboard User")

	for _, path := range []string{
		"/api/dashboard?month=2026-07",
		"/api/reports/member-summary?month=2026-07",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Ledger-Id", fixture.LedgerID)
		req.AddCookie(authCookie(t, fixture.UserAID))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %s failed: %d %s", path, rr.Code, rr.Body.String())
		}
		if bytes.Contains(rr.Body.Bytes(), []byte(outsiderID)) || bytes.Contains(rr.Body.Bytes(), []byte("Outside Dashboard User")) {
			t.Fatalf("request %s exposed a non-member: %s", path, rr.Body.String())
		}
	}
}

func TestTask50GlobalRoutesDoNotConsumeLedgerContext(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, rbacRouterConfig(t))
	fixture := seedRBACLedger(t, database)

	reqMe := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	reqMe.Header.Set("X-Ledger-Id", "unknown-ledger")
	reqMe.AddCookie(authCookie(t, fixture.UserAID))
	rrMe := httptest.NewRecorder()
	router.ServeHTTP(rrMe, reqMe)
	if rrMe.Code != http.StatusOK {
		t.Fatalf("global me route should ignore ledger header, got %d body: %s", rrMe.Code, rrMe.Body.String())
	}
	var mePayload struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rrMe.Body.Bytes(), &mePayload); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if mePayload.Data["instance_admin"] != true {
		t.Fatalf("expected instance_admin=true, got %+v", mePayload.Data)
	}
	if _, exists := mePayload.Data["ledger_id"]; exists {
		t.Fatalf("/auth/me must not return a current ledger: %+v", mePayload.Data)
	}

	reqDiagnostics := httptest.NewRequest(http.MethodGet, "/api/admin/diagnostics", nil)
	reqDiagnostics.Header.Set("X-Ledger-Id", "unknown-ledger")
	reqDiagnostics.AddCookie(authCookie(t, fixture.UserAID))
	rrDiagnostics := httptest.NewRecorder()
	router.ServeHTTP(rrDiagnostics, reqDiagnostics)
	if rrDiagnostics.Code != http.StatusOK {
		t.Fatalf("instance admin route should ignore ledger header, got %d body: %s", rrDiagnostics.Code, rrDiagnostics.Body.String())
	}

	reqDenied := httptest.NewRequest(http.MethodGet, "/api/admin/diagnostics", nil)
	reqDenied.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqDenied.AddCookie(authCookie(t, fixture.UserBID))
	rrDenied := httptest.NewRecorder()
	router.ServeHTTP(rrDenied, reqDenied)
	assertRouterError(t, rrDenied, http.StatusForbidden, "INSTANCE_ADMIN_REQUIRED")
}

func assertRouterError(t *testing.T, rr *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("expected status %d, got %d body: %s", status, rr.Code, rr.Body.String())
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if payload.Error.Code != code {
		t.Fatalf("expected error %s, got %s body: %s", code, payload.Error.Code, rr.Body.String())
	}
}

func insertTask50Ledger(t *testing.T, database *sql.DB, ledgerID string, ownerUserID string) string {
	t.Helper()
	now := time.Now().Format(time.RFC3339)
	if _, err := database.Exec(`
		INSERT INTO ledgers (id, name, default_currency, status, version, created_at, updated_at)
		VALUES (?, ?, 'CNY', 'active', 1, ?, ?)
	`, ledgerID, ledgerID, now, now); err != nil {
		t.Fatalf("insert task50 ledger: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES (?, ?, 'owner', ?, ?)
	`, ledgerID, ownerUserID, now, now); err != nil {
		t.Fatalf("insert task50 owner: %v", err)
	}
	return ledgerID
}

func archiveTask50Ledger(t *testing.T, database *sql.DB, ledgerID string, actorUserID string) {
	t.Helper()
	now := time.Now().Format(time.RFC3339)
	if _, err := database.Exec(`
		UPDATE ledgers
		SET status = 'archived', archived_at = ?, archived_by_user_id = ?, version = version + 1
		WHERE id = ?
	`, now, actorUserID, ledgerID); err != nil {
		t.Fatalf("archive task50 ledger fixture: %v", err)
	}
}

func insertTask50Transaction(t *testing.T, database *sql.DB, ledgerID string, userID string, transactionID string) {
	t.Helper()
	now := time.Now().Format(time.RFC3339)
	if _, err := database.Exec(`
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, visibility,
			status, created_at, updated_at
		) VALUES (?, ?, 'expense', 'Scoped expense', 100, 'CNY', ?, ?, ?, ?, 'partner_readable', 'normal', ?, ?)
	`, transactionID, ledgerID, now, userID, userID, userID, now, now); err != nil {
		t.Fatalf("insert task50 transaction: %v", err)
	}
}

func insertTask50Template(t *testing.T, database *sql.DB, ledgerID string, userID string, templateID string) {
	t.Helper()
	now := time.Now().Format(time.RFC3339)
	if _, err := database.Exec(`
		INSERT INTO transaction_templates (
			id, ledger_id, name, type, created_by_user_id, created_at, updated_at
		) VALUES (?, ?, 'Scoped template', 'expense', ?, ?, ?)
	`, templateID, ledgerID, userID, now, now); err != nil {
		t.Fatalf("insert task50 template: %v", err)
	}
}
