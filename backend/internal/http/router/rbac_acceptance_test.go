package router

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	router := New(database, &config.Config{JWTSecret: rbacTestSecret})

	fixture := seedRBACLedger(t, database)
	guestID := insertRBACUser(t, database, "guest", "Guest")
	addRBACMember(t, database, fixture.LedgerID, guestID, "viewer")

	beforeCount := countTransactions(t, database)
	payload := map[string]interface{}{
		"type":          "expense",
		"title":         "viewer should not write",
		"amount_cents":  int64(1234),
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": guestID,
		"visibility":    "private",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewReader(body))
	req.Header.Set("X-Ledger-Id", fixture.LedgerID)
	req.AddCookie(authCookie(t, guestID))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected viewer write to return 403, got %d body: %s", rr.Code, rr.Body.String())
	}
	if afterCount := countTransactions(t, database); afterCount != beforeCount {
		t.Fatalf("viewer write should not create transactions, before=%d after=%d", beforeCount, afterCount)
	}
}

func TestRBACAcceptanceNonMemberCannotReadLedgerTransactions(t *testing.T) {
	database := setupRBACRouterDB(t)
	router := New(database, &config.Config{JWTSecret: rbacTestSecret})

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

func addRBACMember(t *testing.T, database *sql.DB, ledgerID string, userID string, role string) {
	t.Helper()

	now := time.Now().Format(time.RFC3339)
	_, err := database.Exec(`
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, ledgerID, userID, role, now, now)
	if err != nil {
		t.Fatalf("insert member %s role %s: %v", userID, role, err)
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
