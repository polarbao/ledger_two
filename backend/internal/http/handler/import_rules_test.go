package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"ledger_two/internal/db/repo"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/service"
	"ledger_two/internal/transaction"
)

func TestImportRulesCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret-rules"

	initRepo := repo.NewInitRepo(db)
	initSvc := service.NewInitService(initRepo)
	initHandler := handler.NewInitHandler(initSvc)

	authRepo := repo.NewAuthRepo(db)
	authSvc := service.NewAuthService(authRepo, jwtSecret)
	authHandler := handler.NewAuthHandler(authSvc)

	txRepo := transaction.NewRepository(db)
	txSvc := transaction.NewService(txRepo)
	txHandler := transaction.NewHandler(txSvc)

	r := chi.NewRouter()
	r.Post("/api/init/setup", initHandler.HandleSetup)
	r.Post("/api/auth/login", authHandler.HandleLogin)

	r.Group(func(r chi.Router) {
		r.Use(testAuthenticatedLedgerContext(db, jwtSecret))
		r.Get("/api/accounts", txHandler.HandleListAccounts)
		r.Route("/api/import-rules", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreateImportRule)
			r.Get("/", txHandler.HandleListImportRules)
			r.Delete("/{id}", txHandler.HandleDeleteImportRule)
		})
	})

	// 初始化系统并注入用户
	setupPayload := map[string]string{
		"ledger_name":         "Import Rules Test Ledger",
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

	cookieA := getLoginCookie(t, r, "userA", "pass123")

	// 获取默认分类 ID
	var categoryID string
	err := db.QueryRow("SELECT id FROM categories LIMIT 1").Scan(&categoryID)
	if err != nil {
		t.Fatalf("failed to get category: %v", err)
	}

	// 1. 测试拉取账户列表
	t.Run("List Accounts", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/accounts", nil)
		req.AddCookie(cookieA)
		setTestLedgerHeader(t, db, req, "Import Rules Test Ledger")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var res struct {
			Success bool                  `json:"success"`
			Data    []transaction.Account `json:"data"`
		}
		json.Unmarshal(rr.Body.Bytes(), &res)
		if !res.Success {
			t.Fatalf("expected success response")
		}
		if len(res.Data) != 2 {
			t.Errorf("expected 2 accounts, got %d", len(res.Data))
		}
		if res.Data[0].Name != "User A日常账户" {
			t.Errorf("expected account name 'User A日常账户', got %s", res.Data[0].Name)
		}
	})

	// 获取其中一个账户的 ID
	var accountID string
	err = db.QueryRow("SELECT id FROM accounts LIMIT 1").Scan(&accountID)
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}

	var createdRuleID string

	// 2. 测试创建匹配规则
	t.Run("Create Import Rule", func(t *testing.T) {
		rulePayload := map[string]interface{}{
			"keyword":     "星巴克",
			"category_id": categoryID,
			"account_id":  accountID,
			"tag_names":   []string{"咖啡", "下午茶"},
		}

		body, _ := json.Marshal(rulePayload)
		req, _ := http.NewRequest("POST", "/api/import-rules", bytes.NewBuffer(body))
		req.AddCookie(cookieA)
		setTestLedgerHeader(t, db, req, "Import Rules Test Ledger")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var res struct {
			Success bool                           `json:"success"`
			Data    transaction.ImportRuleResponse `json:"data"`
		}
		json.Unmarshal(rr.Body.Bytes(), &res)
		if !res.Success {
			t.Fatalf("expected success response")
		}
		if res.Data.Keyword != "星巴克" {
			t.Errorf("expected keyword '星巴克', got %s", res.Data.Keyword)
		}
		if len(res.Data.TagNames) != 2 || res.Data.TagNames[0] != "咖啡" {
			t.Errorf("expected tags [咖啡, 下午茶], got %v", res.Data.TagNames)
		}
		createdRuleID = res.Data.ID
	})

	// 3. 测试创建验证失败
	t.Run("Create Import Rule Validation Fail", func(t *testing.T) {
		// 缺少 keyword
		rulePayload := map[string]interface{}{
			"category_id": categoryID,
		}

		body, _ := json.Marshal(rulePayload)
		req, _ := http.NewRequest("POST", "/api/import-rules", bytes.NewBuffer(body))
		req.AddCookie(cookieA)
		setTestLedgerHeader(t, db, req, "Import Rules Test Ledger")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rr.Code)
		}
	})

	// 4. 测试列出导入规则
	t.Run("List Import Rules", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/import-rules", nil)
		req.AddCookie(cookieA)
		setTestLedgerHeader(t, db, req, "Import Rules Test Ledger")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var res struct {
			Success bool                             `json:"success"`
			Data    []transaction.ImportRuleResponse `json:"data"`
		}
		json.Unmarshal(rr.Body.Bytes(), &res)
		if len(res.Data) != 1 {
			t.Errorf("expected 1 rule, got %d", len(res.Data))
		}
		if res.Data[0].ID != createdRuleID {
			t.Errorf("expected rule ID %s, got %s", createdRuleID, res.Data[0].ID)
		}
	})

	// 5. 测试删除规则越权拦截
	t.Run("Delete Import Rule Non-Existent", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/import-rules/non-existent-id", nil)
		req.AddCookie(cookieA)
		setTestLedgerHeader(t, db, req, "Import Rules Test Ledger")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})

	// 6. 测试成功删除规则
	t.Run("Delete Import Rule Success", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/import-rules/"+createdRuleID, nil)
		req.AddCookie(cookieA)
		setTestLedgerHeader(t, db, req, "Import Rules Test Ledger")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		// 再次拉取规则，应当为空
		reqList, _ := http.NewRequest("GET", "/api/import-rules", nil)
		reqList.AddCookie(cookieA)
		setTestLedgerHeader(t, db, reqList, "Import Rules Test Ledger")
		rrList := httptest.NewRecorder()
		r.ServeHTTP(rrList, reqList)

		var res struct {
			Success bool                             `json:"success"`
			Data    []transaction.ImportRuleResponse `json:"data"`
		}
		json.Unmarshal(rrList.Body.Bytes(), &res)
		if len(res.Data) != 0 {
			t.Errorf("expected 0 rules after deletion, got %d", len(res.Data))
		}
	})
}
