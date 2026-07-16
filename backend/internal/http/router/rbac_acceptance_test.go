package router

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/internal/config"
	"ledger_two/internal/db/repo"
	"ledger_two/migrations"
)

const rbacTestSecret = "rbac-test-secret"

func TestRBACAcceptanceViewerCannotCreateTransaction(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, rbacRouterConfig(t))

	fixture := seedRBACLedger(t, database)
	setRBACMemberRole(t, database, fixture.LedgerID, fixture.UserBID, "viewer")

	beforeCount := countTransactions(t, database)
	payload := map[string]interface{}{
		"type":          "expense",
		"title":         "viewer should not write",
		"amount_cents":  int64(1234),
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": fixture.UserBID,
		"visibility":    "private",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("X-Ledger-Id", fixture.LedgerID)
	req.AddCookie(authCookie(t, fixture.UserBID))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected viewer write to return 403, got %d body: %s", rr.Code, rr.Body.String())
	}
	if afterCount := countTransactions(t, database); afterCount != beforeCount {
		t.Fatalf("viewer write should not create transactions, before=%d after=%d", beforeCount, afterCount)
	}

	readReq := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	readReq.Header.Set("X-Ledger-Id", fixture.LedgerID)
	readReq.AddCookie(authCookie(t, fixture.UserBID))
	readRecorder := httptest.NewRecorder()
	router.ServeHTTP(readRecorder, readReq)
	if readRecorder.Code != http.StatusOK {
		t.Fatalf("viewer should retain read-only transaction history, got %d body: %s", readRecorder.Code, readRecorder.Body.String())
	}
}

func TestRBACAcceptanceImportManagementOwnerOnly(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, rbacRouterConfig(t))

	fixture := seedRBACLedger(t, database)
	payload, _ := json.Marshal(map[string]any{
		"name":       "forbidden rule",
		"match_type": "merchant_contains",
		"pattern":    "coffee",
		"result": map[string]any{
			"category_id": "missing-category",
		},
	})
	endpoints := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/api/import-rules?status=all"},
		{method: http.MethodPost, path: "/api/import-rules", body: payload},
		{method: http.MethodPatch, path: "/api/import-rules/missing-rule", body: payload},
		{method: http.MethodPost, path: "/api/import-rules/missing-rule/archive"},
		{method: http.MethodPost, path: "/api/import-rules/missing-rule/restore"},
		{method: http.MethodDelete, path: "/api/import-rules/missing-rule"},
		{method: http.MethodGet, path: "/api/imports/missing-batch"},
	}

	for _, role := range []string{"editor", "viewer"} {
		setRBACMemberRole(t, database, fixture.LedgerID, fixture.UserBID, role)
		for _, endpoint := range endpoints {
			t.Run(role+" "+endpoint.method+" "+endpoint.path, func(t *testing.T) {
				req := httptest.NewRequest(endpoint.method, endpoint.path, bytes.NewReader(endpoint.body))
				req.Header.Set("X-Ledger-Id", fixture.LedgerID)
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(authCookie(t, fixture.UserBID))
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)

				if rr.Code != http.StatusForbidden {
					t.Fatalf("expected forbidden import management request, got %d body: %s", rr.Code, rr.Body.String())
				}
			})
		}
	}

	var ruleCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM import_rules").Scan(&ruleCount); err != nil {
		t.Fatalf("count import rules: %v", err)
	}
	if ruleCount != 0 {
		t.Fatalf("forbidden import management requests must not create rules, got %d", ruleCount)
	}
}

func TestRBACAcceptanceNonMemberCannotReadLedgerTransactions(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, rbacRouterConfig(t))

	fixture := seedRBACLedger(t, database)
	outsiderID := insertRBACUser(t, database, "outsider", "Outsider")

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	req.Header.Set("X-Ledger-Id", fixture.LedgerID)
	req.AddCookie(authCookie(t, outsiderID))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
		t.Fatalf("expected non-member read to return 403 or 404, got %d body: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), fixture.LedgerName) {
		t.Fatalf("non-member response should not leak ledger name, body: %s", rr.Body.String())
	}
}

func TestRBACAcceptancePrivateAttachmentCannotBypassVisibility(t *testing.T) {
	database := setupRBACRouterDB(t)
	cfg := rbacRouterConfig(t)
	router := New(database, cfg)

	fixture := seedRBACLedger(t, database)
	const filename = "r03-private.png"
	const fileBody = "private attachment content"
	writeRBACAttachmentFixture(t, cfg.UploadDir, filename, []byte(fileBody))
	insertRBACAttachmentTransaction(t, database, fixture, "/uploads/"+filename)

	reqOwner := httptest.NewRequest(http.MethodGet, "/api/attachments/"+filename, nil)
	reqOwner.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqOwner.AddCookie(authCookie(t, fixture.UserAID))
	rrOwner := httptest.NewRecorder()
	router.ServeHTTP(rrOwner, reqOwner)
	if rrOwner.Code != http.StatusOK {
		t.Fatalf("expected owner attachment access to return 200, got %d body: %s", rrOwner.Code, rrOwner.Body.String())
	}
	if rrOwner.Body.String() != fileBody {
		t.Fatalf("expected owner to receive attachment body")
	}

	reqPartner := httptest.NewRequest(http.MethodGet, "/api/attachments/"+filename, nil)
	reqPartner.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqPartner.AddCookie(authCookie(t, fixture.UserBID))
	rrPartner := httptest.NewRecorder()
	router.ServeHTTP(rrPartner, reqPartner)
	if rrPartner.Code != http.StatusForbidden && rrPartner.Code != http.StatusNotFound {
		t.Fatalf("expected partner private attachment access to return 403 or 404, got %d body: %s", rrPartner.Code, rrPartner.Body.String())
	}

	reqBare := httptest.NewRequest(http.MethodGet, "/uploads/"+filename, nil)
	rrBare := httptest.NewRecorder()
	router.ServeHTTP(rrBare, reqBare)
	if rrBare.Code == http.StatusOK || strings.Contains(rrBare.Body.String(), fileBody) {
		t.Fatalf("bare uploads path must not expose private attachment, status=%d body=%s", rrBare.Code, rrBare.Body.String())
	}
}

func TestRBACAcceptanceDiagnosticsOwnerOnlyAndSanitized(t *testing.T) {
	database := setupRBACRouterDB(t)
	cfg := rbacRouterConfig(t)
	router := New(database, cfg)

	fixture := seedRBACLedger(t, database)
	setRBACMemberRole(t, database, fixture.LedgerID, fixture.UserBID, "viewer")

	reqViewer := httptest.NewRequest(http.MethodGet, "/api/admin/diagnostics", nil)
	reqViewer.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqViewer.AddCookie(authCookie(t, fixture.UserBID))
	rrViewer := httptest.NewRecorder()
	router.ServeHTTP(rrViewer, reqViewer)
	if rrViewer.Code != http.StatusForbidden {
		t.Fatalf("expected viewer diagnostics access to return 403, got %d body: %s", rrViewer.Code, rrViewer.Body.String())
	}

	reqOwner := httptest.NewRequest(http.MethodGet, "/api/admin/diagnostics", nil)
	reqOwner.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqOwner.AddCookie(authCookie(t, fixture.UserAID))
	rrOwner := httptest.NewRecorder()
	router.ServeHTTP(rrOwner, reqOwner)
	if rrOwner.Code != http.StatusOK {
		t.Fatalf("expected owner diagnostics access to return 200, got %d body: %s", rrOwner.Code, rrOwner.Body.String())
	}

	body := rrOwner.Body.String()
	if strings.Contains(body, cfg.JWTSecret) {
		t.Fatalf("diagnostics response must not expose JWT secret, body: %s", body)
	}
	for _, path := range []string{cfg.BackupDir, cfg.UploadDir, cfg.LogDir} {
		if path != "" && strings.Contains(body, path) {
			t.Fatalf("diagnostics response must not expose absolute storage path %q, body: %s", path, body)
		}
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Env      string `json:"env"`
			Database struct {
				Status  string `json:"status"`
				Version int64  `json:"version"`
			} `json:"database"`
			Storage []struct {
				Key        string `json:"key"`
				Status     string `json:"status"`
				Configured bool   `json:"configured"`
			} `json:"storage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rrOwner.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode diagnostics response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success diagnostics response, body: %s", body)
	}
	if resp.Data.Database.Status != "ok" || resp.Data.Database.Version == 0 {
		t.Fatalf("expected ok database diagnostics with schema version, got: %+v", resp.Data.Database)
	}
	if len(resp.Data.Storage) < 4 {
		t.Fatalf("expected storage diagnostics for database/backups/uploads/logs, got: %+v", resp.Data.Storage)
	}
}

func TestRBACAcceptanceBackupEndpointsOwnerOnly(t *testing.T) {
	database := setupRBACRouterDB(t)
	cfg := rbacRouterConfig(t)
	router := New(database, cfg)

	fixture := seedRBACLedger(t, database)

	reqEditorBackup := httptest.NewRequest(http.MethodPost, "/api/admin/backup", nil)
	reqEditorBackup.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqEditorBackup.AddCookie(authCookie(t, fixture.UserBID))
	rrEditorBackup := httptest.NewRecorder()
	router.ServeHTTP(rrEditorBackup, reqEditorBackup)
	if rrEditorBackup.Code != http.StatusForbidden {
		t.Fatalf("expected editor backup to return 403, got %d body: %s", rrEditorBackup.Code, rrEditorBackup.Body.String())
	}

	reqOwnerBackup := httptest.NewRequest(http.MethodPost, "/api/admin/backup", nil)
	reqOwnerBackup.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqOwnerBackup.AddCookie(authCookie(t, fixture.UserAID))
	rrOwnerBackup := httptest.NewRecorder()
	router.ServeHTTP(rrOwnerBackup, reqOwnerBackup)
	if rrOwnerBackup.Code != http.StatusOK {
		t.Fatalf("expected owner backup to return 200, got %d body: %s", rrOwnerBackup.Code, rrOwnerBackup.Body.String())
	}
	var backupResp struct {
		Success bool `json:"success"`
		Data    struct {
			Filename string `json:"filename"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rrOwnerBackup.Body.Bytes(), &backupResp); err != nil {
		t.Fatalf("decode backup response: %v", err)
	}
	if backupResp.Data.Filename == "" {
		t.Fatalf("expected owner backup response to include filename, body: %s", rrOwnerBackup.Body.String())
	}

	reqEditorList := httptest.NewRequest(http.MethodGet, "/api/admin/backups", nil)
	reqEditorList.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqEditorList.AddCookie(authCookie(t, fixture.UserBID))
	rrEditorList := httptest.NewRecorder()
	router.ServeHTTP(rrEditorList, reqEditorList)
	if rrEditorList.Code != http.StatusForbidden {
		t.Fatalf("expected editor backup list to return 403, got %d body: %s", rrEditorList.Code, rrEditorList.Body.String())
	}

	reqOwnerList := httptest.NewRequest(http.MethodGet, "/api/admin/backups", nil)
	reqOwnerList.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqOwnerList.AddCookie(authCookie(t, fixture.UserAID))
	rrOwnerList := httptest.NewRecorder()
	router.ServeHTTP(rrOwnerList, reqOwnerList)
	if rrOwnerList.Code != http.StatusOK {
		t.Fatalf("expected owner backup list to return 200, got %d body: %s", rrOwnerList.Code, rrOwnerList.Body.String())
	}

	reqEditorDownload := httptest.NewRequest(http.MethodGet, "/api/admin/backups/"+backupResp.Data.Filename, nil)
	reqEditorDownload.Header.Set("X-Ledger-Id", fixture.LedgerID)
	reqEditorDownload.AddCookie(authCookie(t, fixture.UserBID))
	rrEditorDownload := httptest.NewRecorder()
	router.ServeHTTP(rrEditorDownload, reqEditorDownload)
	if rrEditorDownload.Code != http.StatusForbidden {
		t.Fatalf("expected editor backup download to return 403, got %d body: %s", rrEditorDownload.Code, rrEditorDownload.Body.String())
	}
}

func setupRBACRouterDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	database.SetMaxOpenConns(1)

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.Up(database, "."); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return database
}

func rbacRouterConfig(t *testing.T) *config.Config {
	t.Helper()

	baseDir := t.TempDir()
	return &config.Config{
		JWTSecret: rbacTestSecret,
		Env:       "test",
		DSN:       filepath.Join(baseDir, "data", "ledger.db"),
		BackupDir: filepath.Join(baseDir, "backups"),
		UploadDir: filepath.Join(baseDir, "uploads"),
		LogDir:    filepath.Join(baseDir, "logs"),
	}
}

func writeRBACAttachmentFixture(t *testing.T, uploadDir string, filename string, content []byte) {
	t.Helper()

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		t.Fatalf("create upload fixture dir: %v", err)
	}
	path := filepath.Join(uploadDir, filename)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write attachment fixture: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(path)
	})
}

func insertRBACAttachmentTransaction(t *testing.T, database *sql.DB, fixture rbacFixture, attachmentPath string) {
	t.Helper()

	now := time.Now().Format(time.RFC3339)
	_, err := database.Exec(`
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id,
			visibility, note, attachment_paths, status, created_at, updated_at
		) VALUES (
			'r03-private-tx', ?, 'expense', 'Private receipt', 3580, 'CNY', ?,
			?, ?, ?, 'private', NULL, ?, 'normal', ?, ?
		)
	`, fixture.LedgerID, now, fixture.UserAID, fixture.UserAID, fixture.UserAID, `["`+attachmentPath+`"]`, now, now)
	if err != nil {
		t.Fatalf("insert private attachment transaction: %v", err)
	}
}

type rbacFixture struct {
	LedgerID   string
	LedgerName string
	UserAID    string
	UserBID    string
}

func seedRBACLedger(t *testing.T, database *sql.DB) rbacFixture {
	t.Helper()

	initRepo := repo.NewInitRepo(database)
	err := initRepo.ExecuteSetupTx(context.Background(), "RBAC Test Ledger", "CNY", []repo.UserPayload{
		{Username: "userA", DisplayName: "User A", PasswordHash: "hash-a"},
		{Username: "userB", DisplayName: "User B", PasswordHash: "hash-b"},
	})
	if err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	var fixture rbacFixture
	fixture.LedgerName = "RBAC Test Ledger"
	err = database.QueryRow("SELECT id FROM ledgers WHERE name = ?", fixture.LedgerName).Scan(&fixture.LedgerID)
	if err != nil {
		t.Fatalf("query ledger id: %v", err)
	}
	err = database.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&fixture.UserAID)
	if err != nil {
		t.Fatalf("query userA id: %v", err)
	}
	err = database.QueryRow("SELECT id FROM users WHERE username = 'userB'").Scan(&fixture.UserBID)
	if err != nil {
		t.Fatalf("query userB id: %v", err)
	}

	assertRBACRole(t, database, fixture.LedgerID, fixture.UserAID, "owner")
	assertRBACRole(t, database, fixture.LedgerID, fixture.UserBID, "editor")
	return fixture
}

func insertRBACUser(t *testing.T, database *sql.DB, username string, displayName string) string {
	t.Helper()

	userID := username + "-id"
	now := time.Now().Format(time.RFC3339)
	_, err := database.Exec(`
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, 'hash', 'user', ?, ?)
	`, userID, username, displayName, now, now)
	if err != nil {
		t.Fatalf("insert user %s: %v", username, err)
	}
	return userID
}

func setRBACMemberRole(t *testing.T, database *sql.DB, ledgerID string, userID string, role string) {
	t.Helper()

	now := time.Now().Format(time.RFC3339)
	_, err := database.Exec(`
		UPDATE ledger_members
		SET role = ?, updated_at = ?
		WHERE ledger_id = ? AND user_id = ?
	`, role, now, ledgerID, userID)
	if err != nil {
		t.Fatalf("update member %s role %s: %v", userID, role, err)
	}
}

func assertRBACRole(t *testing.T, database *sql.DB, ledgerID string, userID string, expected string) {
	t.Helper()

	var role string
	err := database.QueryRow(
		"SELECT role FROM ledger_members WHERE ledger_id = ? AND user_id = ?",
		ledgerID,
		userID,
	).Scan(&role)
	if err != nil {
		t.Fatalf("query member role: %v", err)
	}
	if role != expected {
		t.Fatalf("expected role %s for user %s, got %s", expected, userID, role)
	}
}

func countTransactions(t *testing.T, database *sql.DB) int {
	t.Helper()

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&count); err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	return count
}

func authCookie(t *testing.T, userID string) *http.Cookie {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(rbacTestSecret))
	if err != nil {
		t.Fatalf("sign auth token: %v", err)
	}
	return &http.Cookie{Name: "token", Value: tokenString}
}
