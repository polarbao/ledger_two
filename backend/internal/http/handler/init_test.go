package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/internal/db/repo"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/service"
	"ledger_two/migrations"
)

// setupTestDB 准备一个全新的内存数据库，并执行真实迁移脚本
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("goose dialect error: %v", err)
	}
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("goose up error: %v", err)
	}
	return db
}

func TestInitFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	initRepo := repo.NewInitRepo(db)
	initSvc := service.NewInitService(initRepo)
	initHandler := handler.NewInitHandler(initSvc)

	// ============================================
	// Test 1: 验证初始状态 (应该为 false)
	// ============================================
	req, _ := http.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()
	initHandler.HandleStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Test 1 failed: expected 200, got %d", rr.Code)
	}
	var statusResp struct {
		Success bool `json:"success"`
		Data    struct {
			Initialized bool `json:"initialized"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&statusResp); err != nil {
		t.Fatal(err)
	}
	if statusResp.Data.Initialized != false {
		t.Errorf("Test 1 failed: expected initialized=false")
	}

	// ============================================
	// Test 2: 执行 Setup 操作
	// ============================================
	payload := map[string]string{
		"ledger_name":         "Test Ledger",
		"user_a_username":     "userA",
		"user_a_display_name": "User A",
		"user_a_password":     "pass123",
		"user_b_username":     "userB",
		"user_b_display_name": "User B",
		"user_b_password":     "pass456",
	}
	body, _ := json.Marshal(payload)
	req2, _ := http.NewRequest("POST", "/setup", bytes.NewBuffer(body))
	rr2 := httptest.NewRecorder()
	initHandler.HandleSetup(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("Test 2 failed: expected 200, got %d body: %s", rr2.Code, rr2.Body.String())
	}

	// ============================================
	// Test 3: 验证设置完成后的状态 (应该为 true)
	// ============================================
	req3, _ := http.NewRequest("GET", "/status", nil)
	rr3 := httptest.NewRecorder()
	initHandler.HandleStatus(rr3, req3)
	json.NewDecoder(rr3.Body).Decode(&statusResp)
	if statusResp.Data.Initialized != true {
		t.Errorf("Test 3 failed: expected initialized=true")
	}

	// ============================================
	// Test 4: 二次调用 Setup (应该拦截并返回 409)
	// ============================================
	req4, _ := http.NewRequest("POST", "/setup", bytes.NewBuffer(body))
	rr4 := httptest.NewRecorder()
	initHandler.HandleSetup(rr4, req4)

	if rr4.Code != http.StatusConflict {
		t.Errorf("Test 4 failed: expected 409 conflict, got %d", rr4.Code)
	}

	// ============================================
	// Test 5: 探查内部数据库密码是否哈希
	// ============================================
	var hash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE username = 'userA'").Scan(&hash)
	if err != nil {
		t.Fatalf("Test 5 failed: query user error %v", err)
	}
	if hash == "pass123" || hash == "" {
		t.Errorf("Test 5 failed: password was not hashed correctly")
	}
}
