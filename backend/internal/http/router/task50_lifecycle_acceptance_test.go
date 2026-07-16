package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTask503ALifecycleRoutesExposeETagRBACAndAtomicArchiveContract(t *testing.T) {
	database := setupRBACRouterDB(t)
	fixture := seedRBACLedger(t, database)
	router := New(database, rbacRouterConfig(t))

	detail := lifecycleRequest(t, router, fixture.UserAID, http.MethodGet, "/api/ledgers/"+fixture.LedgerID, fixture.LedgerID, "", nil)
	if detail.Code != http.StatusOK || detail.Header().Get("ETag") != `"ledger:`+fixture.LedgerID+`:v1"` {
		t.Fatalf("unexpected ledger detail: status=%d etag=%q body=%s", detail.Code, detail.Header().Get("ETag"), detail.Body.String())
	}

	missingVersion := lifecycleRequest(t, router, fixture.UserAID, http.MethodPatch, "/api/ledgers/"+fixture.LedgerID, fixture.LedgerID, "", []byte(`{"name":"Renamed"}`))
	assertRouterError(t, missingVersion, http.StatusBadRequest, "VALIDATION_ERROR")

	renamed := lifecycleRequest(t, router, fixture.UserAID, http.MethodPatch, "/api/ledgers/"+fixture.LedgerID, fixture.LedgerID, `"ledger:`+fixture.LedgerID+`:v1"`, []byte(`{"name":"Renamed"}`))
	if renamed.Code != http.StatusOK || renamed.Header().Get("ETag") != `"ledger:`+fixture.LedgerID+`:v2"` {
		t.Fatalf("unexpected rename: status=%d etag=%q body=%s", renamed.Code, renamed.Header().Get("ETag"), renamed.Body.String())
	}

	preflightDenied := lifecycleRequest(t, router, fixture.UserBID, http.MethodGet, "/api/ledgers/"+fixture.LedgerID+"/archive-preflight", fixture.LedgerID, "", nil)
	assertRouterError(t, preflightDenied, http.StatusForbidden, "LEDGER_ACCESS_DENIED")

	missingAcknowledgement := lifecycleRequest(t, router, fixture.UserAID, http.MethodPost, "/api/ledgers/"+fixture.LedgerID+"/archive", fixture.LedgerID, `"ledger:`+fixture.LedgerID+`:v2"`, []byte(`{}`))
	assertRouterError(t, missingAcknowledgement, http.StatusBadRequest, "VALIDATION_ERROR")

	if _, err := database.Exec(`
		INSERT INTO import_batches (
			id, ledger_id, filename, created_by_user_id, status, created_at, expires_at
		) VALUES ('lifecycle-ready', ?, 'fixture.csv', ?, 'ready', '2026-07-01T00:00:00Z', '2099-01-01T00:00:00Z')
	`, fixture.LedgerID, fixture.UserAID); err != nil {
		t.Fatalf("insert blocking import batch: %v", err)
	}

	blocked := lifecycleRequest(t, router, fixture.UserAID, http.MethodPost, "/api/ledgers/"+fixture.LedgerID+"/archive", fixture.LedgerID, `"ledger:`+fixture.LedgerID+`:v2"`, []byte(`{"acknowledge_unsettled_balance":false}`))
	assertRouterError(t, blocked, http.StatusConflict, "LEDGER_READY_IMPORT_EXISTS")
	var version int64
	if err := database.QueryRow("SELECT version FROM ledgers WHERE id = ?", fixture.LedgerID).Scan(&version); err != nil {
		t.Fatalf("read blocked version: %v", err)
	}
	if version != 2 {
		t.Fatalf("blocked archive persisted version %d", version)
	}

	discarded := lifecycleRequest(t, router, fixture.UserAID, http.MethodPost, "/api/imports/lifecycle-ready/discard", fixture.LedgerID, "", []byte(`{"reason":"user_requested"}`))
	if discarded.Code != http.StatusOK || !bytes.Contains(discarded.Body.Bytes(), []byte(`"discard_reason":"user_requested"`)) {
		t.Fatalf("unexpected discard: status=%d body=%s", discarded.Code, discarded.Body.String())
	}
	archived := lifecycleRequest(t, router, fixture.UserAID, http.MethodPost, "/api/ledgers/"+fixture.LedgerID+"/archive", fixture.LedgerID, `"ledger:`+fixture.LedgerID+`:v2"`, []byte(`{"acknowledge_unsettled_balance":false}`))
	if archived.Code != http.StatusOK || archived.Header().Get("ETag") != `"ledger:`+fixture.LedgerID+`:v3"` {
		t.Fatalf("unexpected archive: status=%d etag=%q body=%s", archived.Code, archived.Header().Get("ETag"), archived.Body.String())
	}

	restored := lifecycleRequest(t, router, fixture.UserAID, http.MethodPost, "/api/ledgers/"+fixture.LedgerID+"/restore", fixture.LedgerID, `"ledger:`+fixture.LedgerID+`:v3"`, nil)
	if restored.Code != http.StatusOK || restored.Header().Get("ETag") != `"ledger:`+fixture.LedgerID+`:v4"` {
		t.Fatalf("unexpected restore: status=%d etag=%q body=%s", restored.Code, restored.Header().Get("ETag"), restored.Body.String())
	}
}

func TestTask503ACreateAndStatusFilterUseFrozenHTTPContract(t *testing.T) {
	database := setupRBACRouterDB(t)
	fixture := seedRBACLedger(t, database)
	router := New(database, rbacRouterConfig(t))

	unknownField := lifecycleRequest(t, router, fixture.UserAID, http.MethodPost, "/api/ledgers", "", "", []byte(`{"name":"Invalid","unexpected":true}`))
	assertRouterError(t, unknownField, http.StatusBadRequest, "BAD_REQUEST")

	created := lifecycleRequest(t, router, fixture.UserAID, http.MethodPost, "/api/ledgers", "", "", []byte(`{"name":"  新账本  "}`))
	if created.Code != http.StatusCreated || created.Header().Get("ETag") == "" {
		t.Fatalf("unexpected create: status=%d etag=%q body=%s", created.Code, created.Header().Get("ETag"), created.Body.String())
	}
	var payload struct {
		Data struct {
			Name   string `json:"name"`
			Role   string `json:"role"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if payload.Data.Name != "新账本" || payload.Data.Role != "owner" || payload.Data.Status != "active" {
		t.Fatalf("unexpected create DTO: %+v", payload.Data)
	}

	invalidStatus := lifecycleRequest(t, router, fixture.UserAID, http.MethodGet, "/api/ledgers?status=invalid", "", "", nil)
	assertRouterError(t, invalidStatus, http.StatusBadRequest, "VALIDATION_ERROR")

	active := lifecycleRequest(t, router, fixture.UserAID, http.MethodGet, "/api/ledgers", "", "", nil)
	if active.Code != http.StatusOK {
		t.Fatalf("list default active ledgers: %d %s", active.Code, active.Body.String())
	}
}

func TestTask503AArchiveUsesTrustedBalanceWithoutCreatingSettlement(t *testing.T) {
	database := setupRBACRouterDB(t)
	fixture := seedRBACLedger(t, database)
	router := New(database, rbacRouterConfig(t))

	var categoryID string
	if err := database.QueryRow("SELECT id FROM categories WHERE ledger_id = ? LIMIT 1", fixture.LedgerID).Scan(&categoryID); err != nil {
		t.Fatalf("load category: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, occurred_at, owner_user_id,
			created_by_user_id, payer_user_id, category_id, visibility,
			split_method, status, created_at, updated_at
		) VALUES (
			'lifecycle-shared', ?, 'shared_expense', 'Lifecycle expense', 2000,
			'2026-07-01T00:00:00Z', ?, ?, ?, ?, 'shared', 'equal', 'normal',
			'2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z'
		);
		INSERT INTO transaction_splits (
			id, transaction_id, user_id, share_amount, created_at, updated_at
		) VALUES
			('lifecycle-split-owner', 'lifecycle-shared', ?, 1000, '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z'),
			('lifecycle-split-partner', 'lifecycle-shared', ?, 1000, '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z');
	`, fixture.LedgerID, fixture.UserAID, fixture.UserAID, fixture.UserAID, categoryID, fixture.UserAID, fixture.UserBID); err != nil {
		t.Fatalf("seed unsettled expense: %v", err)
	}

	preflight := lifecycleRequest(t, router, fixture.UserAID, http.MethodGet, "/api/ledgers/"+fixture.LedgerID+"/archive-preflight", fixture.LedgerID, "", nil)
	if preflight.Code != http.StatusOK || preflight.Header().Get("ETag") != `"ledger:`+fixture.LedgerID+`:v1"` || !bytes.Contains(preflight.Body.Bytes(), []byte(`"amount_cents":1000`)) {
		t.Fatalf("unexpected trusted preflight: status=%d etag=%q body=%s", preflight.Code, preflight.Header().Get("ETag"), preflight.Body.String())
	}

	withoutAck := lifecycleRequest(t, router, fixture.UserAID, http.MethodPost, "/api/ledgers/"+fixture.LedgerID+"/archive", fixture.LedgerID, `"ledger:`+fixture.LedgerID+`:v1"`, []byte(`{"acknowledge_unsettled_balance":false}`))
	assertRouterError(t, withoutAck, http.StatusBadRequest, "VALIDATION_ERROR")

	archived := lifecycleRequest(t, router, fixture.UserAID, http.MethodPost, "/api/ledgers/"+fixture.LedgerID+"/archive", fixture.LedgerID, `"ledger:`+fixture.LedgerID+`:v1"`, []byte(`{"acknowledge_unsettled_balance":true}`))
	if archived.Code != http.StatusOK || archived.Header().Get("ETag") != `"ledger:`+fixture.LedgerID+`:v2"` {
		t.Fatalf("archive unsettled ledger: status=%d etag=%q body=%s", archived.Code, archived.Header().Get("ETag"), archived.Body.String())
	}

	var version int64
	var settlementCount, transactionCount int
	if err := database.QueryRow("SELECT version FROM ledgers WHERE id = ?", fixture.LedgerID).Scan(&version); err != nil {
		t.Fatalf("read archived version: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM settlements WHERE ledger_id = ?", fixture.LedgerID).Scan(&settlementCount); err != nil {
		t.Fatalf("count settlements: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM transactions WHERE ledger_id = ?", fixture.LedgerID).Scan(&transactionCount); err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	if version != 2 || settlementCount != 0 || transactionCount != 1 {
		t.Fatalf("archive mutated accounting history: version=%d settlements=%d transactions=%d", version, settlementCount, transactionCount)
	}
}

func lifecycleRequest(t *testing.T, router http.Handler, userID, method, path, ledgerID, ifMatch string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if ledgerID != "" {
		req.Header.Set("X-Ledger-Id", ledgerID)
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	req.AddCookie(authCookie(t, userID))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}
