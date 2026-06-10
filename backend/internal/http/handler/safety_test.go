package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"ledger_two/internal/config"
	"ledger_two/internal/db/repo"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/safety"
	"ledger_two/internal/service"
	"ledger_two/internal/transaction"
)

func TestSafetyFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret"
	// 为测试动态创建一个可写的备份目录
	tmpBackupDir, err := os.MkdirTemp("", "ledger_two_backup_test_*")
	if err != nil {
		t.Fatalf("failed to create temp backup dir: %v", err)
	}
	defer os.RemoveAll(tmpBackupDir)

	cfg := &config.Config{
		JWTSecret: jwtSecret,
		BackupDir: tmpBackupDir,
	}

	// 初始化依赖
	initRepo := repo.NewInitRepo(db)
	initSvc := service.NewInitService(initRepo)
	initHandler := handler.NewInitHandler(initSvc)

	authRepo := repo.NewAuthRepo(db)
	authSvc := service.NewAuthService(authRepo, jwtSecret)
	authHandler := handler.NewAuthHandler(authSvc)

	txRepo := transaction.NewRepository(db)
	txSvc := transaction.NewService(txRepo)
	txHandler := transaction.NewHandler(txSvc)

	safetySvc := safety.NewService(db, cfg)
	safetyHandler := safety.NewHandler(safetySvc)

	r := chi.NewRouter()
	r.Post("/api/init/setup", initHandler.HandleSetup)
	r.Post("/api/auth/login", authHandler.HandleLogin)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtSecret))
		r.Route("/api/transactions", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreate)
		})

		r.Route("/api/admin", func(r chi.Router) {
			r.Post("/backup", safetyHandler.HandleManualBackup)
			r.Get("/backups", safetyHandler.HandleGetBackups)
			r.Get("/backups/{filename}", safetyHandler.HandleDownloadBackup)
		})

		r.Route("/api/export", func(r chi.Router) {
			r.Get("/transactions.csv", safetyHandler.HandleExportCSV)
			r.Get("/full.json", safetyHandler.HandleExportJSON)
		})
	})

	// 1. 初始化系统，注入 A、B 用户
	setupPayload := map[string]string{
		"ledger_name":         "Test Ledger",
		"user_a_username":     "userA",
		"user_a_display_name": "User A",
		"user_a_password":     "pass123",
		"user_b_username":     "userB",
		"user_b_display_name": "User B",
		"user_b_password":     "pass456",
	}
	body, _ := json.Marshal(setupPayload)
	reqSetup, _ := http.NewRequest("POST", "/api/init/setup", bytes.NewBuffer(body))
	rrSetup := httptest.NewRecorder()
	r.ServeHTTP(rrSetup, reqSetup)
	if rrSetup.Code != http.StatusOK {
		t.Fatalf("setup failed: %v", rrSetup.Body.String())
	}

	// 2. 获取用户 A 和用户 B 的登录 Token
	cookieA := getLoginCookie(t, r, "userA", "pass123")
	cookieB := getLoginCookie(t, r, "userB", "pass456")

	// 3. 拦截未登录测试
	reqNoAuth, _ := http.NewRequest("POST", "/api/admin/backup", nil)
	rrNoAuth := httptest.NewRecorder()
	r.ServeHTTP(rrNoAuth, reqNoAuth)
	if rrNoAuth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthorized backup access, got %d", rrNoAuth.Code)
	}

	// 4. 手动备份成功测试
	reqBackup, _ := http.NewRequest("POST", "/api/admin/backup", nil)
	reqBackup.AddCookie(cookieA)
	rrBackup := httptest.NewRecorder()
	r.ServeHTTP(rrBackup, reqBackup)
	if rrBackup.Code != http.StatusOK {
		t.Fatalf("expected 200 for manual backup, got %d. Body: %s", rrBackup.Code, rrBackup.Body.String())
	}

	var backupResp response.SuccessResponse
	json.Unmarshal(rrBackup.Body.Bytes(), &backupResp)
	if !backupResp.Success {
		t.Fatalf("expected success response, got: %s", rrBackup.Body.String())
	}
	backupData, ok := backupResp.Data.(map[string]interface{})
	if !ok || backupData["filename"] == nil {
		t.Fatalf("backup response structure invalid, got: %s", rrBackup.Body.String())
	}
	filename := backupData["filename"].(string)

	// 验证备份物理文件在 backups/manual 目录下成功生成
	expectedPhysicalFile := filepath.Join(tmpBackupDir, filename)
	if _, err := os.Stat(expectedPhysicalFile); os.IsNotExist(err) {
		t.Errorf("physical backup file not found at %s", expectedPhysicalFile)
	}

	// 验证审计日志记录写入
	var backupAuditCount int
	err = db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action = 'backup' AND entity_type = 'database'").Scan(&backupAuditCount)
	if err != nil {
		t.Fatalf("query backup audit count failed: %v", err)
	}
	if backupAuditCount != 1 {
		t.Errorf("expected 1 backup audit log, got %d", backupAuditCount)
	}

	// 5. 备份列表查询测试
	reqBackupsList, _ := http.NewRequest("GET", "/api/admin/backups", nil)
	reqBackupsList.AddCookie(cookieA)
	rrBackupsList := httptest.NewRecorder()
	r.ServeHTTP(rrBackupsList, reqBackupsList)
	if rrBackupsList.Code != http.StatusOK {
		t.Fatalf("get backups list failed, got %d", rrBackupsList.Code)
	}

	var listResp response.SuccessResponse
	json.Unmarshal(rrBackupsList.Body.Bytes(), &listResp)
	if !listResp.Success {
		t.Fatalf("expected list success")
	}
	backupsList, ok := listResp.Data.([]interface{})
	if !ok {
		t.Fatalf("invalid backups list format, got %s", rrBackupsList.Body.String())
	}
	if len(backupsList) != 1 {
		t.Errorf("expected 1 item in backups list, got %d", len(backupsList))
	} else {
		item := backupsList[0].(map[string]interface{})
		if item["filename"].(string) != filename {
			t.Errorf("expected filename %s, got %s", filename, item["filename"])
		}
	}

	// 6. 物理不可写备份路径报错测试
	// 使用不合法的系统级目录路径测试
	cfg.BackupDir = "/sys_non_writable_dir_for_test/invalid_path"
	reqBackupErr, _ := http.NewRequest("POST", "/api/admin/backup", nil)
	reqBackupErr.AddCookie(cookieA)
	rrBackupErr := httptest.NewRecorder()
	r.ServeHTTP(rrBackupErr, reqBackupErr)
	if rrBackupErr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 status code for invalid backup path, got %d", rrBackupErr.Code)
	}

	var errResp struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(rrBackupErr.Body.Bytes(), &errResp)
	if errResp.Error.Code != "BACKUP_PATH_INVALID" {
		t.Errorf("expected BACKUP_PATH_INVALID error code, got %s", errResp.Error.Code)
	}

	// 恢复配置以防后续测试失败
	cfg.BackupDir = tmpBackupDir

	// 7. 用户权限隔离导出测试
	// 7.1 用户 A 创建一笔 private 账单
	var categoryID string
	err = db.QueryRow("SELECT id FROM categories LIMIT 1").Scan(&categoryID)
	if err != nil {
		t.Fatalf("query category failed: %v", err)
	}

	reqPayload := map[string]interface{}{
		"type":          "expense",
		"title":         "用户A的私密日记本",
		"amount_cents":  int64(9900), // 99元
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": "userA",
		"category_id":   categoryID,
		"visibility":    "private",
		"note":          "只有A可见",
	}
	bodyA, _ := json.Marshal(reqPayload)
	reqCreateA, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyA))
	reqCreateA.AddCookie(cookieA)
	rrCreateA := httptest.NewRecorder()
	r.ServeHTTP(rrCreateA, reqCreateA)
	if rrCreateA.Code != http.StatusCreated {
		t.Fatalf("create private transaction for A failed, got %d", rrCreateA.Code)
	}

	// 7.2 用户 A 导出 JSON，应当包含该账单，且包含脱敏的用户信息，不含 password_hash
	reqExportJSONA, _ := http.NewRequest("GET", "/api/export/full.json", nil)
	reqExportJSONA.AddCookie(cookieA)
	rrExportJSONA := httptest.NewRecorder()
	r.ServeHTTP(rrExportJSONA, reqExportJSONA)
	if rrExportJSONA.Code != http.StatusOK {
		t.Fatalf("user A export json failed, got %d", rrExportJSONA.Code)
	}

	var exportA map[string]interface{}
	json.Unmarshal(rrExportJSONA.Body.Bytes(), &exportA)

	// 验证脱敏
	usersListA := exportA["users"].([]interface{})
	for _, u := range usersListA {
		userMap := u.(map[string]interface{})
		if _, exists := userMap["password_hash"]; exists {
			t.Errorf("JSON export contains sensitive password_hash!")
		}
	}

	// 验证包含 A 自己的私有账单
	txsListA := exportA["transactions"].([]interface{})
	foundPrivateBillInA := false
	for _, tx := range txsListA {
		txMap := tx.(map[string]interface{})
		if txMap["title"].(string) == "用户A的私密日记本" {
			foundPrivateBillInA = true
		}
	}
	if !foundPrivateBillInA {
		t.Errorf("expected user A's JSON export to include their own private bill, but not found")
	}

	// 7.3 用户 B 导出 JSON，应当绝不包含 A 的 private 账单
	reqExportJSONB, _ := http.NewRequest("GET", "/api/export/full.json", nil)
	reqExportJSONB.AddCookie(cookieB)
	rrExportJSONB := httptest.NewRecorder()
	r.ServeHTTP(rrExportJSONB, reqExportJSONB)
	if rrExportJSONB.Code != http.StatusOK {
		t.Fatalf("user B export json failed, got %d", rrExportJSONB.Code)
	}

	var exportB map[string]interface{}
	json.Unmarshal(rrExportJSONB.Body.Bytes(), &exportB)

	txsListB := exportB["transactions"].([]interface{})
	foundPrivateBillInB := false
	for _, tx := range txsListB {
		txMap := tx.(map[string]interface{})
		if txMap["title"].(string) == "用户A的私密日记本" {
			foundPrivateBillInB = true
		}
	}
	if foundPrivateBillInB {
		t.Errorf("SECURITY VULNERABILITY: User B's JSON export includes user A's private bill!")
	}

	// 7.4 用户 A 导出 CSV，应当包含该账单，且验证 CSV 审计日志写入
	reqExportCSVA, _ := http.NewRequest("GET", "/api/export/transactions.csv", nil)
	reqExportCSVA.AddCookie(cookieA)
	rrExportCSVA := httptest.NewRecorder()
	r.ServeHTTP(rrExportCSVA, reqExportCSVA)
	if rrExportCSVA.Code != http.StatusOK {
		t.Fatalf("user A export csv failed, got %d", rrExportCSVA.Code)
	}

	csvContent := rrExportCSVA.Body.String()
	if !strings.Contains(csvContent, "用户A的私密日记本") {
		t.Errorf("expected user A's CSV export to include their own private bill")
	}

	// 7.5 用户 B 导出 CSV，应当不包含 A 的 private 账单
	reqExportCSVB, _ := http.NewRequest("GET", "/api/export/transactions.csv", nil)
	reqExportCSVB.AddCookie(cookieB)
	rrExportCSVB := httptest.NewRecorder()
	r.ServeHTTP(rrExportCSVB, reqExportCSVB)
	if rrExportCSVB.Code != http.StatusOK {
		t.Fatalf("user B export csv failed, got %d", rrExportCSVB.Code)
	}

	csvContentB := rrExportCSVB.Body.String()
	if strings.Contains(csvContentB, "用户A的私密日记本") {
		t.Errorf("SECURITY VULNERABILITY: User B's CSV export includes user A's private bill!")
	}

	// 验证导出审计日志成功记录
	var exportAuditCount int
	err = db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action IN ('export_csv', 'export_json')").Scan(&exportAuditCount)
	if err != nil {
		t.Fatalf("query export audit count failed: %v", err)
	}
	// A 导出 JSON + A 导出 CSV + B 导出 JSON + B 导出 CSV = 4 次导出审计
	if exportAuditCount != 4 {
		t.Errorf("expected 4 export audit logs, got %d", exportAuditCount)
	}
}
