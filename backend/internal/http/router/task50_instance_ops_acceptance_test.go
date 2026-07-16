package router

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTask503CInstanceAdminIsIndependentAndDiagnosticsAreAudited(t *testing.T) {
	database := setupRBACRouterDB(t)
	cfg := rbacRouterConfig(t)
	handler := New(database, cfg)
	fixture := seedRBACLedger(t, database)

	nonAdminOwnerID := fixture.UserBID
	insertTask50Ledger(t, database, "task50-instance-owner-ledger", nonAdminOwnerID)
	denied := performTask50InstanceRequest(
		t,
		handler,
		nonAdminOwnerID,
		http.MethodGet,
		"/api/admin/diagnostics",
		fixture.LedgerID,
		"",
	)
	assertRouterError(t, denied, http.StatusForbidden, "INSTANCE_ADMIN_REQUIRED")

	adminOnlyUserID := insertRBACUser(t, database, "instance-only-admin", "Instance Only Admin")
	if _, err := database.Exec(`
		INSERT INTO instance_admins (user_id, granted_at, granted_by_user_id)
		VALUES (?, ?, ?)
	`, adminOnlyUserID, time.Now().UTC().Format(time.RFC3339Nano), fixture.UserAID); err != nil {
		t.Fatalf("grant instance administrator: %v", err)
	}

	diagnostics := performTask50InstanceRequest(
		t,
		handler,
		adminOnlyUserID,
		http.MethodGet,
		"/api/admin/diagnostics",
		"unknown-ledger-that-must-be-ignored",
		"",
	)
	if diagnostics.Code != http.StatusOK {
		t.Fatalf("instance-only admin diagnostics: status=%d body=%s", diagnostics.Code, diagnostics.Body.String())
	}
	assertTask50InstanceAudit(t, database, adminOnlyUserID, "system_diagnostics", 1)

	ledgerRequest := httptest.NewRequest(http.MethodGet, "/api/ledgers/"+fixture.LedgerID, nil)
	ledgerRequest.AddCookie(authCookie(t, adminOnlyUserID))
	ledgerResponse := httptest.NewRecorder()
	handler.ServeHTTP(ledgerResponse, ledgerRequest)
	assertRouterError(t, ledgerResponse, http.StatusForbidden, "LEDGER_ACCESS_DENIED")

	var ledgerAuditCount int
	if err := database.QueryRow(`
		SELECT COUNT(*)
		FROM audit_logs
		WHERE actor_user_id = ?
	`, adminOnlyUserID).Scan(&ledgerAuditCount); err != nil {
		t.Fatalf("count instance-only administrator ledger audits: %v", err)
	}
	if ledgerAuditCount != 0 {
		t.Fatalf("instance operations wrote %d ledger audit rows", ledgerAuditCount)
	}
}

func TestTask503CBackupCreateListAndDownloadMatchContractAndAudit(t *testing.T) {
	database := setupRBACRouterDB(t)
	cfg := rbacRouterConfig(t)
	handler := New(database, cfg)
	fixture := seedRBACLedger(t, database)

	created := performTask50InstanceRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodPost,
		"/api/admin/backup",
		fixture.LedgerID,
		"",
	)
	if created.Code != http.StatusOK {
		t.Fatalf("create backup: status=%d body=%s", created.Code, created.Body.String())
	}
	var createdEnvelope task50BackupEnvelope
	if err := json.Unmarshal(created.Body.Bytes(), &createdEnvelope); err != nil {
		t.Fatalf("decode backup create response: %v", err)
	}
	if !createdEnvelope.Success ||
		createdEnvelope.Data.Filename == "" ||
		createdEnvelope.Data.SizeBytes <= 0 ||
		createdEnvelope.Data.CreatedAt.IsZero() {
		t.Fatalf("backup create response does not match BackupInfo: %+v", createdEnvelope)
	}
	assertTask50InstanceAudit(t, database, fixture.UserAID, "manual_database_backup", 1)

	listed := performTask50InstanceRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodGet,
		"/api/admin/backups",
		"ignored-ledger-header",
		"",
	)
	if listed.Code != http.StatusOK {
		t.Fatalf("list backups: status=%d body=%s", listed.Code, listed.Body.String())
	}
	var listEnvelope struct {
		Success bool               `json:"success"`
		Data    []task50BackupInfo `json:"data"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("decode backup list response: %v", err)
	}
	if len(listEnvelope.Data) != 1 || listEnvelope.Data[0].Filename != createdEnvelope.Data.Filename {
		t.Fatalf("unexpected backup list: %+v", listEnvelope.Data)
	}
	assertTask50InstanceAudit(t, database, fixture.UserAID, "list_database_backups", 1)

	downloaded := performTask50InstanceRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodGet,
		"/api/admin/backups/"+createdEnvelope.Data.Filename,
		fixture.LedgerID,
		"",
	)
	if downloaded.Code != http.StatusOK {
		t.Fatalf("download backup: status=%d body=%s", downloaded.Code, downloaded.Body.String())
	}
	if !strings.Contains(downloaded.Header().Get("Content-Disposition"), filepath.Base(createdEnvelope.Data.Filename)) {
		t.Fatalf("download response missing backup filename: %s", downloaded.Header().Get("Content-Disposition"))
	}
	if downloaded.Body.Len() == 0 {
		t.Fatal("downloaded backup body is empty")
	}
	assertTask50InstanceAudit(t, database, fixture.UserAID, "download_database_backup", 1)
}

func TestTask503CRestorePrepareReturnsFrozenDTOAndRejectsSiblingTraversal(t *testing.T) {
	database := setupRBACRouterDB(t)
	cfg := rbacRouterConfig(t)
	handler := New(database, cfg)
	fixture := seedRBACLedger(t, database)

	created := performTask50InstanceRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodPost,
		"/api/admin/backup",
		"",
		"",
	)
	if created.Code != http.StatusOK {
		t.Fatalf("create restore target: status=%d body=%s", created.Code, created.Body.String())
	}
	var createdEnvelope task50BackupEnvelope
	if err := json.Unmarshal(created.Body.Bytes(), &createdEnvelope); err != nil {
		t.Fatalf("decode restore target: %v", err)
	}

	restoreBody, _ := json.Marshal(map[string]string{"filename": createdEnvelope.Data.Filename})
	prepared := performTask50InstanceRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodPost,
		"/api/admin/restore",
		"ignored-ledger-header",
		string(restoreBody),
	)
	if prepared.Code != http.StatusOK {
		t.Fatalf("prepare restore: status=%d body=%s", prepared.Code, prepared.Body.String())
	}
	var restoreEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Filename         string `json:"filename"`
			Instructions     string `json:"instructions"`
			RequiresDowntime bool   `json:"requires_downtime"`
		} `json:"data"`
	}
	if err := json.Unmarshal(prepared.Body.Bytes(), &restoreEnvelope); err != nil {
		t.Fatalf("decode restore prepare response: %v", err)
	}
	if restoreEnvelope.Data.Filename != createdEnvelope.Data.Filename ||
		restoreEnvelope.Data.Instructions == "" ||
		!restoreEnvelope.Data.RequiresDowntime {
		t.Fatalf("restore response does not match frozen DTO: %+v", restoreEnvelope.Data)
	}
	assertTask50InstanceAudit(t, database, fixture.UserAID, "prepare_database_restore", 1)

	siblingDir := cfg.BackupDir + "-evil"
	if err := os.MkdirAll(siblingDir, 0755); err != nil {
		t.Fatalf("create sibling traversal directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siblingDir, "outside.db"), []byte("not a managed backup"), 0644); err != nil {
		t.Fatalf("write sibling traversal fixture: %v", err)
	}
	traversalKey := "../" + filepath.Base(siblingDir) + "/outside.db"
	traversalBody, _ := json.Marshal(map[string]string{"filename": traversalKey})
	rejected := performTask50InstanceRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodPost,
		"/api/admin/restore",
		"",
		string(traversalBody),
	)
	assertRouterError(t, rejected, http.StatusBadRequest, "VALIDATION_ERROR")
	assertTask50InstanceAudit(t, database, fixture.UserAID, "prepare_database_restore", 1)

	symlinkDir := filepath.Join(cfg.BackupDir, "linked")
	if err := os.Symlink(siblingDir, symlinkDir); err == nil {
		symlinkBody, _ := json.Marshal(map[string]string{"filename": "linked/outside.db"})
		symlinkRejected := performTask50InstanceRequest(
			t,
			handler,
			fixture.UserAID,
			http.MethodPost,
			"/api/admin/restore",
			"",
			string(symlinkBody),
		)
		assertRouterError(t, symlinkRejected, http.StatusBadRequest, "VALIDATION_ERROR")
		assertTask50InstanceAudit(t, database, fixture.UserAID, "prepare_database_restore", 1)
	}
}

type task50BackupInfo struct {
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

type task50BackupEnvelope struct {
	Success bool             `json:"success"`
	Data    task50BackupInfo `json:"data"`
}

func performTask50InstanceRequest(
	t *testing.T,
	handler http.Handler,
	userID, method, path, ledgerID, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if ledgerID != "" {
		req.Header.Set("X-Ledger-Id", ledgerID)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(authCookie(t, userID))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func assertTask50InstanceAudit(t *testing.T, database interface {
	QueryRow(query string, args ...any) *sql.Row
}, actorUserID, action string, expected int) {
	t.Helper()
	var count int
	if err := database.QueryRow(`
		SELECT COUNT(*)
		FROM instance_audit_logs
		WHERE actor_user_id = ? AND action = ?
	`, actorUserID, action).Scan(&count); err != nil {
		t.Fatalf("count instance audit %s: %v", action, err)
	}
	if count != expected {
		t.Fatalf("expected %d %s instance audits, got %d", expected, action, count)
	}
}
