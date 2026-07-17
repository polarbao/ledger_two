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
	db.SetMaxOpenConns(1)

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

	// The first initialized user is the only initial instance administrator.
	var adminUsername string
	err = db.QueryRow(`
		SELECT u.username
		FROM instance_admins ia
		JOIN users u ON u.id = ia.user_id
	`).Scan(&adminUsername)
	if err != nil {
		t.Fatalf("Test 6 failed: query instance administrator %v", err)
	}
	if adminUsername != "userA" {
		t.Errorf("Test 6 failed: expected userA to be instance administrator, got %s", adminUsername)
	}

	var adminCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM instance_admins").Scan(&adminCount); err != nil {
		t.Fatalf("Test 6 failed: count instance administrators %v", err)
	}
	if adminCount != 1 {
		t.Errorf("Test 6 failed: expected exactly one instance administrator, got %d", adminCount)
	}

	var categoryCount, tagCount, profileVersion int
	if err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&categoryCount); err != nil {
		t.Fatalf("Test 7 failed: count default categories %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount); err != nil {
		t.Fatalf("Test 7 failed: count default tags %v", err)
	}
	if err := db.QueryRow("SELECT metadata_profile_version FROM ledgers LIMIT 1").Scan(&profileVersion); err != nil {
		t.Fatalf("Test 7 failed: read metadata profile version %v", err)
	}
	if categoryCount != 19 || tagCount != 8 || profileVersion != 1 {
		t.Errorf("Test 7 failed: categories=%d tags=%d profile_version=%d", categoryCount, tagCount, profileVersion)
	}
}

func TestTask531InitSetupRollsBackAllStateWhenDefaultProfileFails(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TRIGGER task53_fail_init_profile
		BEFORE INSERT ON categories
		FOR EACH ROW WHEN NEW.system_key = 'expense_health'
		BEGIN
			SELECT RAISE(ABORT, 'injected init profile failure');
		END;
	`); err != nil {
		t.Fatalf("create init failure trigger: %v", err)
	}

	initRepo := repo.NewInitRepo(db)
	err := initRepo.ExecuteSetupTx(t.Context(), "Rollback Ledger", "CNY", []repo.UserPayload{
		{Username: "owner", DisplayName: "Owner", PasswordHash: "hash-owner"},
		{Username: "partner", DisplayName: "Partner", PasswordHash: "hash-partner"},
	})
	if err == nil {
		t.Fatal("expected injected init profile failure")
	}

	for _, table := range []string{"users", "ledgers", "ledger_members", "accounts", "categories", "tags", "instance_admins", "app_settings"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatalf("count rolled back table %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("init failure left %d rows in %s", count, table)
		}
	}
}
