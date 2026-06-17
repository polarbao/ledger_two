package middleware_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"ledger_two/internal/http/middleware"
)

// setupMockDB 创建测试所需的临时内存数据库，并注入测试数据
func setupMockDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// 1. 建表
	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE
		);
		CREATE TABLE ledger_members (
			ledger_id TEXT,
			user_id TEXT,
			role TEXT,
			created_at TEXT,
			updated_at TEXT,
			PRIMARY KEY (ledger_id, user_id)
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	// 2. 插入测试数据
	_, err = db.Exec(`
		INSERT INTO users (id, username) VALUES ('user1', 'owner_user');
		INSERT INTO users (id, username) VALUES ('user2', 'no_ledger_user');
		INSERT INTO users (id, username) VALUES ('user3', 'editor_user');

		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at) 
		VALUES ('ledgerA', 'user1', 'owner', '2026-06-17T00:00:00Z', '2026-06-17T00:00:00Z');

		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at) 
		VALUES ('ledgerB', 'user1', 'viewer', '2026-06-17T00:00:00Z', '2026-06-17T00:00:00Z');

		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at) 
		VALUES ('ledgerA', 'user3', 'editor', '2026-06-17T00:00:00Z', '2026-06-17T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("failed to seed test data: %v", err)
	}

	return db
}

func TestRequireLedgerContext(t *testing.T) {
	db := setupMockDB(t)
	defer db.Close()

	// 准备中间件
	requireCtxMiddleware := middleware.RequireLedgerContext(db)

	// 一个简单的 Handler，用于检查 Context 中是否注入了预期的 LedgerContext
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lc := middleware.GetLedgerContext(r.Context())
		if lc == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-Test-Role", lc.Role)
		w.Header().Set("X-Test-Ledger", lc.LedgerID)
		if lc.IsExplicit {
			w.Header().Set("X-Test-Explicit", "true")
		} else {
			w.Header().Set("X-Test-Explicit", "false")
		}
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Valid explicit X-Ledger-Id Header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/transactions", nil)
		req.Header.Set("X-Ledger-Id", "ledgerA")
		// 模拟 RequireAuth 注入的 userID
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user1")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		requireCtxMiddleware(dummyHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if role := rr.Header().Get("X-Test-Role"); role != "owner" {
			t.Errorf("expected role 'owner', got '%s'", role)
		}
		if ledger := rr.Header().Get("X-Test-Ledger"); ledger != "ledgerA" {
			t.Errorf("expected ledger 'ledgerA', got '%s'", ledger)
		}
		if explicit := rr.Header().Get("X-Test-Explicit"); explicit != "true" {
			t.Errorf("expected explicit 'true', got '%s'", explicit)
		}
	})

	t.Run("URL Path smart matching for /ledgers/{id}", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/ledgers/ledgerB/members", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user1")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		requireCtxMiddleware(dummyHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if role := rr.Header().Get("X-Test-Role"); role != "viewer" {
			t.Errorf("expected role 'viewer', got '%s'", role)
		}
		if ledger := rr.Header().Get("X-Test-Ledger"); ledger != "ledgerB" {
			t.Errorf("expected ledger 'ledgerB', got '%s'", ledger)
		}
		if explicit := rr.Header().Get("X-Test-Explicit"); explicit != "true" {
			t.Errorf("expected explicit 'true', got '%s'", explicit)
		}
	})

	t.Run("Non-membership explicit access yields 403", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/transactions", nil)
		req.Header.Set("X-Ledger-Id", "ledgerB") // user3 不在 ledgerB 里
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user3")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		requireCtxMiddleware(dummyHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("No ledger Header/Path causes Fallback lookup", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/transactions", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user1")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		requireCtxMiddleware(dummyHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		// user1 有 ledgerA 和 ledgerB，LIMIT 1 可能会取得其中一个
		ledger := rr.Header().Get("X-Test-Ledger")
		if ledger != "ledgerA" && ledger != "ledgerB" {
			t.Errorf("expected ledger to be fallback ledgerA or ledgerB, got '%s'", ledger)
		}
		if explicit := rr.Header().Get("X-Test-Explicit"); explicit != "false" {
			t.Errorf("expected explicit 'false', got '%s'", explicit)
		}
	})

	t.Run("No ledger found for user yields 400", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/transactions", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user2") // user2 无账本
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		requireCtxMiddleware(dummyHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestRequireLedgerRole(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Matched role bypasses check", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/transactions", nil)
		lc := &middleware.LedgerContext{
			UserID:   "user1",
			LedgerID: "ledgerA",
			Role:     "owner",
		}
		ctx := context.WithValue(req.Context(), middleware.LedgerContextKey, lc)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		middleware.RequireLedgerRole("owner", "editor")(dummyHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("Unmatched role gets blocked with 403", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/transactions", nil)
		lc := &middleware.LedgerContext{
			UserID:   "user1",
			LedgerID: "ledgerB",
			Role:     "viewer",
		}
		ctx := context.WithValue(req.Context(), middleware.LedgerContextKey, lc)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		middleware.RequireLedgerRole("owner", "editor")(dummyHandler).ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})
}
