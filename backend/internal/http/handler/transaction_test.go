package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"ledger_two/internal/db/repo"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/service"
	"ledger_two/internal/transaction"
)

func TestTransactionFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret"

	// 初始化各模块 Handler
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
		r.Use(middleware.RequireAuth(jwtSecret))
		r.Route("/api/transactions", func(r chi.Router) {
			r.Get("/", txHandler.HandleList)
			r.Post("/", txHandler.HandleCreate)
			r.Get("/{id}", txHandler.HandleGetByID)
			r.Patch("/{id}", txHandler.HandleUpdate)
			r.Delete("/{id}", txHandler.HandleDelete)
		})
		r.Get("/api/categories", txHandler.HandleListCategories)
		r.Get("/api/transaction-defaults", txHandler.HandleGetTransactionDefault)
	})

	// 1. 初始化系统，注入 A、B 两个用户
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

	var userAID string
	err := db.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&userAID)
	if err != nil {
		t.Fatalf("query userA id failed: %v", err)
	}

	// 提取分类 ID 以供记账关联
	var categoryID string
	err = db.QueryRow("SELECT id FROM categories LIMIT 1").Scan(&categoryID)
	if err != nil {
		t.Fatalf("query category failed: %v", err)
	}

	// 3. 测试创建：正常支出
	reqPayload := map[string]interface{}{
		"type":          "expense",
		"title":         "购买午餐",
		"amount_cents":  int64(1500), // 15元
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"visibility":    "private",
		"tag_names":     []string{"外卖", "工作餐"},
		"note":          "好吃的便当",
	}
	bodyA, _ := json.Marshal(reqPayload)
	reqCreate, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyA))
	reqCreate.AddCookie(cookieA)
	rrCreate := httptest.NewRecorder()
	r.ServeHTTP(rrCreate, reqCreate)

	if rrCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", rrCreate.Code, rrCreate.Body.String())
	}

	var createResp response.SuccessResponse
	json.Unmarshal(rrCreate.Body.Bytes(), &createResp)
	txData := createResp.Data.(map[string]interface{})
	txID := txData["id"].(string)

	var categoryName string
	err = db.QueryRow("SELECT name FROM categories WHERE id = ?", categoryID).Scan(&categoryName)
	if err != nil {
		t.Fatalf("query category name failed: %v", err)
	}
	if _, err = db.Exec("UPDATE categories SET is_archived = 1 WHERE id = ?", categoryID); err != nil {
		t.Fatalf("archive category fixture failed: %v", err)
	}

	reqCategoriesActive, _ := http.NewRequest("GET", "/api/categories", nil)
	reqCategoriesActive.AddCookie(cookieA)
	rrCategoriesActive := httptest.NewRecorder()
	r.ServeHTTP(rrCategoriesActive, reqCategoriesActive)
	if rrCategoriesActive.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for active categories, got %d. Body: %s", rrCategoriesActive.Code, rrCategoriesActive.Body.String())
	}
	var categoriesActiveResp response.SuccessResponse
	json.Unmarshal(rrCategoriesActive.Body.Bytes(), &categoriesActiveResp)
	for _, item := range categoriesActiveResp.Data.([]interface{}) {
		cat := item.(map[string]interface{})
		if cat["id"].(string) == categoryID {
			t.Fatalf("archived category should not be returned by default /api/categories")
		}
	}

	reqCategoriesAll, _ := http.NewRequest("GET", "/api/categories?include_archived=true", nil)
	reqCategoriesAll.AddCookie(cookieA)
	rrCategoriesAll := httptest.NewRecorder()
	r.ServeHTTP(rrCategoriesAll, reqCategoriesAll)
	if rrCategoriesAll.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for include_archived categories, got %d. Body: %s", rrCategoriesAll.Code, rrCategoriesAll.Body.String())
	}
	var categoriesAllResp response.SuccessResponse
	json.Unmarshal(rrCategoriesAll.Body.Bytes(), &categoriesAllResp)
	foundArchivedCategory := false
	for _, item := range categoriesAllResp.Data.([]interface{}) {
		cat := item.(map[string]interface{})
		if cat["id"].(string) == categoryID {
			foundArchivedCategory = true
			if cat["name"].(string) != categoryName {
				t.Fatalf("expected archived category name %q, got %q", categoryName, cat["name"])
			}
			if cat["is_archived"].(bool) != true {
				t.Fatalf("expected archived category is_archived=true, got %+v", cat)
			}
		}
	}
	if !foundArchivedCategory {
		t.Fatalf("expected include_archived categories to contain historical category %s", categoryID)
	}

	reqListWithArchivedCategory, _ := http.NewRequest("GET", "/api/transactions", nil)
	reqListWithArchivedCategory.AddCookie(cookieA)
	rrListWithArchivedCategory := httptest.NewRecorder()
	r.ServeHTTP(rrListWithArchivedCategory, reqListWithArchivedCategory)
	if rrListWithArchivedCategory.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for transaction list with archived category, got %d. Body: %s", rrListWithArchivedCategory.Code, rrListWithArchivedCategory.Body.String())
	}
	var listWithArchivedCategoryResp response.SuccessResponse
	json.Unmarshal(rrListWithArchivedCategory.Body.Bytes(), &listWithArchivedCategoryResp)
	foundTxWithCategoryName := false
	for _, item := range listWithArchivedCategoryResp.Data.([]interface{}) {
		tx := item.(map[string]interface{})
		if tx["id"].(string) == txID {
			foundTxWithCategoryName = true
			if tx["category_name"].(string) != categoryName {
				t.Fatalf("expected transaction category_name %q, got %v", categoryName, tx["category_name"])
			}
			if tx["category_is_archived"].(bool) != true {
				t.Fatalf("expected transaction category_is_archived=true, got %+v", tx)
			}
		}
	}
	if !foundTxWithCategoryName {
		t.Fatalf("expected transaction list to contain transaction %s with archived category display fields", txID)
	}

	// 4. 测试创建：正常收入
	reqPayloadIncome := map[string]interface{}{
		"type":          "income",
		"title":         "工资发放",
		"amount_cents":  int64(500000), // 5000元
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"visibility":    "partner_readable",
	}
	bodyIncome, _ := json.Marshal(reqPayloadIncome)
	reqCreateInc, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyIncome))
	reqCreateInc.AddCookie(cookieA)
	rrCreateInc := httptest.NewRecorder()
	r.ServeHTTP(rrCreateInc, reqCreateInc)
	if rrCreateInc.Code != http.StatusCreated {
		t.Errorf("expected 201 Created for income, got %d", rrCreateInc.Code)
	}

	reqDefaults, _ := http.NewRequest("GET", "/api/transaction-defaults", nil)
	reqDefaults.AddCookie(cookieA)
	rrDefaults := httptest.NewRecorder()
	r.ServeHTTP(rrDefaults, reqDefaults)
	if rrDefaults.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for transaction defaults, got %d. Body: %s", rrDefaults.Code, rrDefaults.Body.String())
	}

	var defaultsResp response.SuccessResponse
	json.Unmarshal(rrDefaults.Body.Bytes(), &defaultsResp)
	defaultsData := defaultsResp.Data.(map[string]interface{})
	if defaultsData["type"].(string) != "income" {
		t.Errorf("expected last transaction type income in defaults, got %v", defaultsData["type"])
	}
	if defaultsData["payer_user_id"].(string) != userAID {
		t.Errorf("expected payer userA in defaults, got %v", defaultsData["payer_user_id"])
	}
	if defaultsData["visibility"].(string) != "partner_readable" {
		t.Errorf("expected partner_readable visibility in defaults, got %v", defaultsData["visibility"])
	}

	// 5. 测试校验边界：金额为 0 报错
	reqPayloadErr := map[string]interface{}{
		"type":          "expense",
		"amount_cents":  int64(0),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": "userA",
	}
	bodyErr, _ := json.Marshal(reqPayloadErr)
	reqCreateErr, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyErr))
	reqCreateErr.AddCookie(cookieA)
	rrCreateErr := httptest.NewRecorder()
	r.ServeHTTP(rrCreateErr, reqCreateErr)

	if rrCreateErr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for 0 amount, got %d", rrCreateErr.Code)
	}

	var errResp response.ErrorResponse
	json.Unmarshal(rrCreateErr.Body.Bytes(), &errResp)
	if errResp.Success || errResp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR code, got %+v", errResp)
	}

	// 6. 测试可见性隔离：private 账单对方不可见
	// 用户 B 尝试获取用户 A 的那个 private 账单，预期返回 404 Not Found 以防越权探测
	reqGetB, _ := http.NewRequest("GET", "/api/transactions/"+txID, nil)
	reqGetB.AddCookie(cookieB)
	rrGetB := httptest.NewRecorder()
	r.ServeHTTP(rrGetB, reqGetB)

	if rrGetB.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for partner private bill, got %d", rrGetB.Code)
	}

	// 用户 B 拉取列表，预期 private 账单不应该在列表中出现
	reqListB, _ := http.NewRequest("GET", "/api/transactions", nil)
	reqListB.AddCookie(cookieB)
	rrListB := httptest.NewRecorder()
	r.ServeHTTP(rrListB, reqListB)

	var listBResp response.SuccessResponse
	json.Unmarshal(rrListB.Body.Bytes(), &listBResp)
	listBData := listBResp.Data.([]interface{})

	// 用户 B 列表里应该只看到 partner_readable 的收入账单，看不到 private 支出
	for _, item := range listBData {
		txItem := item.(map[string]interface{})
		if txItem["id"].(string) == txID {
			t.Errorf("userB should not be able to list userA's private transaction")
		}
	}

	// 7. 测试只读隔离：partner_readable 对方可见但不能编辑
	var incomeID string
	var listResp response.SuccessResponse
	reqListA, _ := http.NewRequest("GET", "/api/transactions", nil)
	reqListA.AddCookie(cookieA)
	rrListA := httptest.NewRecorder()
	r.ServeHTTP(rrListA, reqListA)
	json.Unmarshal(rrListA.Body.Bytes(), &listResp)
	listAData := listResp.Data.([]interface{})
	for _, item := range listAData {
		txItem := item.(map[string]interface{})
		if txItem["type"].(string) == "income" {
			incomeID = txItem["id"].(string)
		}
	}

	// 用户 B 获取 partner_readable 的收入账单详情，预期 200
	reqGetIncB, _ := http.NewRequest("GET", "/api/transactions/"+incomeID, nil)
	reqGetIncB.AddCookie(cookieB)
	rrGetIncB := httptest.NewRecorder()
	r.ServeHTTP(rrGetIncB, reqGetIncB)
	if rrGetIncB.Code != http.StatusOK {
		t.Errorf("userB should be able to view partner_readable bill, got %d", rrGetIncB.Code)
	}

	// 用户 B 尝试编辑 A 的 partner_readable 账单，预期 403
	updatePayload := map[string]interface{}{
		"title": "修改后的工资",
	}
	bodyUpdate, _ := json.Marshal(updatePayload)
	reqUpdateB, _ := http.NewRequest("PATCH", "/api/transactions/"+incomeID, bytes.NewBuffer(bodyUpdate))
	reqUpdateB.AddCookie(cookieB)
	rrUpdateB := httptest.NewRecorder()
	r.ServeHTTP(rrUpdateB, reqUpdateB)
	if rrUpdateB.Code != http.StatusForbidden {
		t.Errorf("userB should be forbidden to edit userA's bill, got %d", rrUpdateB.Code)
	}

	// 8. 测试删除逻辑
	// 用户 A 删除自己创建的账单
	reqDelete, _ := http.NewRequest("DELETE", "/api/transactions/"+txID, nil)
	reqDelete.AddCookie(cookieA)
	rrDelete := httptest.NewRecorder()
	r.ServeHTTP(rrDelete, reqDelete)
	if rrDelete.Code != http.StatusOK {
		t.Errorf("expected 200 for deletion, got %d", rrDelete.Code)
	}

	// 用户 A 再次拉取列表，预期该账单不显示
	reqListAAfter, _ := http.NewRequest("GET", "/api/transactions", nil)
	reqListAAfter.AddCookie(cookieA)
	rrListAAfter := httptest.NewRecorder()
	r.ServeHTTP(rrListAAfter, reqListAAfter)
	var listAAfterResp response.SuccessResponse
	json.Unmarshal(rrListAAfter.Body.Bytes(), &listAAfterResp)
	listAAfterData := listAAfterResp.Data.([]interface{})
	for _, item := range listAAfterData {
		txItem := item.(map[string]interface{})
		if txItem["id"].(string) == txID {
			t.Errorf("transaction should not be returned after soft deletion")
		}
	}

	// 探查底层的 audit_logs 数量，确保产生了对应的删除审计行
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action = 'delete' AND entity_id = ?", txID).Scan(&count)
	if err != nil {
		t.Fatalf("query audit logs error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 audit log for delete action, got %d", count)
	}
}

// 辅助登录提取 Cookie
func getLoginCookie(t *testing.T, r http.Handler, username, password string) *http.Cookie {
	loginPayload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(loginPayload)
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("helper login failed for %s: %v", username, rr.Body.String())
	}
	for _, c := range rr.Result().Cookies() {
		if c.Name == "token" {
			return c
		}
	}
	t.Fatalf("token cookie not found for %s", username)
	return nil
}
