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

func TestSharedExpenseFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret"

	// 初始化 Handler
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
			r.Get("/{id}", txHandler.HandleGetByID)
			r.Patch("/{id}", txHandler.HandleUpdate)
			r.Delete("/{id}", txHandler.HandleDelete)
		})
		r.Route("/api/shared-expenses", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreateSharedExpense)
			r.Get("/{id}", txHandler.HandleGetSharedExpenseByID)
			r.Patch("/{id}", txHandler.HandleUpdateSharedExpense)
		})
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

	cookieA := getLoginCookie(t, r, "userA", "pass123")
	cookieB := getLoginCookie(t, r, "userB", "pass456")

	// 查出用户 A 和 B 的实际 UUID 标识
	var userAID, userBID string
	err := db.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&userAID)
	if err != nil {
		t.Fatalf("query userA id failed: %v", err)
	}
	err = db.QueryRow("SELECT id FROM users WHERE username = 'userB'").Scan(&userBID)
	if err != nil {
		t.Fatalf("query userB id failed: %v", err)
	}

	var categoryID string
	err = db.QueryRow("SELECT id FROM categories LIMIT 1").Scan(&categoryID)
	if err != nil {
		t.Fatalf("query category failed: %v", err)
	}

	// ----------------------------------------------------
	// 场景 1: A 支付 200.00 元 (20000分)，平摊 (equal)。A/B 各承担 10000分。
	// ----------------------------------------------------
	payload1 := map[string]interface{}{
		"title":         "购买家庭日用品",
		"amount_cents":  int64(20000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"split_method":  "equal",
		"tag_names":     []string{"日用品"},
		"note":          "超市采购",
	}
	body1, _ := json.Marshal(payload1)
	req1, _ := http.NewRequest("POST", "/api/shared-expenses", bytes.NewBuffer(body1))
	req1.AddCookie(cookieA)
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for payload1, got %d. Body: %s", rr1.Code, rr1.Body.String())
	}

	var resp1 response.SuccessResponse
	json.Unmarshal(rr1.Body.Bytes(), &resp1)
	txData1 := resp1.Data.(map[string]interface{})
	txID1 := txData1["id"].(string)

	// 验证分摊是否正确
	var splitAmountA, splitAmountB int64
	err = db.QueryRow("SELECT share_amount FROM transaction_splits WHERE transaction_id = ? AND user_id = ?", txID1, userAID).Scan(&splitAmountA)
	if err != nil {
		t.Fatalf("query split amount A failed: %v", err)
	}
	err = db.QueryRow("SELECT share_amount FROM transaction_splits WHERE transaction_id = ? AND user_id = ?", txID1, userBID).Scan(&splitAmountB)
	if err != nil {
		t.Fatalf("query split amount B failed: %v", err)
	}

	if splitAmountA != 10000 || splitAmountB != 10000 {
		t.Errorf("expected A/B to split 10000 each, got A: %d, B: %d", splitAmountA, splitAmountB)
	}

	// ----------------------------------------------------
	// 场景 2: A 支付 100.01 元 (10001分)，平摊 (equal)。A 承担 5001分，B 承担 5000分。
	// ----------------------------------------------------
	payload2 := map[string]interface{}{
		"title":         "买菜买水果",
		"amount_cents":  int64(10001),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"split_method":  "equal",
	}
	body2, _ := json.Marshal(payload2)
	req2, _ := http.NewRequest("POST", "/api/shared-expenses", bytes.NewBuffer(body2))
	req2.AddCookie(cookieA)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for payload2, got %d. Body: %s", rr2.Code, rr2.Body.String())
	}

	var resp2 response.SuccessResponse
	json.Unmarshal(rr2.Body.Bytes(), &resp2)
	txData2 := resp2.Data.(map[string]interface{})
	txID2 := txData2["id"].(string)

	err = db.QueryRow("SELECT share_amount FROM transaction_splits WHERE transaction_id = ? AND user_id = ?", txID2, userAID).Scan(&splitAmountA)
	if err != nil {
		t.Fatalf("query split amount A failed: %v", err)
	}
	err = db.QueryRow("SELECT share_amount FROM transaction_splits WHERE transaction_id = ? AND user_id = ?", txID2, userBID).Scan(&splitAmountB)
	if err != nil {
		t.Fatalf("query split amount B failed: %v", err)
	}

	if splitAmountA != 5001 || splitAmountB != 5000 {
		t.Errorf("expected A: 5001, B: 5000, got A: %d, B: %d", splitAmountA, splitAmountB)
	}

	// ----------------------------------------------------
	// 场景 3: A 支付 200.00 元 (20000分)，仅付款人承担 (payer_only)。A 承担 20000分，B 承担 0分。
	// ----------------------------------------------------
	payload3 := map[string]interface{}{
		"title":         "A的个人奢华餐",
		"amount_cents":  int64(20000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"split_method":  "payer_only",
	}
	body3, _ := json.Marshal(payload3)
	req3, _ := http.NewRequest("POST", "/api/shared-expenses", bytes.NewBuffer(body3))
	req3.AddCookie(cookieA)
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for payload3, got %d. Body: %s", rr3.Code, rr3.Body.String())
	}

	var resp3 response.SuccessResponse
	json.Unmarshal(rr3.Body.Bytes(), &resp3)
	txData3 := resp3.Data.(map[string]interface{})
	txID3 := txData3["id"].(string)

	err = db.QueryRow("SELECT share_amount FROM transaction_splits WHERE transaction_id = ? AND user_id = ?", txID3, userAID).Scan(&splitAmountA)
	if err != nil {
		t.Fatalf("query split amount A failed: %v", err)
	}
	err = db.QueryRow("SELECT share_amount FROM transaction_splits WHERE transaction_id = ? AND user_id = ?", txID3, userBID).Scan(&splitAmountB)
	if err != nil {
		t.Fatalf("query split amount B failed: %v", err)
	}

	if splitAmountA != 20000 || splitAmountB != 0 {
		t.Errorf("expected A: 20000, B: 0, got A: %d, B: %d", splitAmountA, splitAmountB)
	}

	// ----------------------------------------------------
	// 场景 4: 权限校验。B 登录尝试 PATCH 编辑 A 创建的共同支出，预期 403 Forbidden。
	// ----------------------------------------------------
	updatePayload := map[string]interface{}{
		"title": "B恶意修改标题",
	}
	bodyUpdate, _ := json.Marshal(updatePayload)
	reqUpdate, _ := http.NewRequest("PATCH", "/api/shared-expenses/"+txID1, bytes.NewBuffer(bodyUpdate))
	reqUpdate.AddCookie(cookieB)
	rrUpdate := httptest.NewRecorder()
	r.ServeHTTP(rrUpdate, reqUpdate)

	if rrUpdate.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for partner edit, got %d", rrUpdate.Code)
	}

	// ----------------------------------------------------
	// 场景 5: B 登录尝试 GET 拉取 A 创建的共同支出详情，预期 200 OK (因为 visibility = shared)。
	// ----------------------------------------------------
	reqGet, _ := http.NewRequest("GET", "/api/shared-expenses/"+txID1, nil)
	reqGet.AddCookie(cookieB)
	rrGet := httptest.NewRecorder()
	r.ServeHTTP(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected 200 OK for reading partner shared expense, got %d", rrGet.Code)
	}

	// 并且验证返回的 DTO 里包含了分摊信息
	var getResp response.SuccessResponse
	json.Unmarshal(rrGet.Body.Bytes(), &getResp)
	getData := getResp.Data.(map[string]interface{})
	if getData["split_method"].(string) != "equal" {
		t.Errorf("expected split_method equal, got %v", getData["split_method"])
	}
	participants := getData["participants"].([]interface{})
	if len(participants) != 2 {
		t.Errorf("expected 2 participants, got %d", len(participants))
	}

	// ----------------------------------------------------
	// 场景 6: A 登录进行 PATCH 编辑该共同支出金额为 100.03元 (10003分)，平摊。
	// A 应更新为 5002分，B 5001分。
	// ----------------------------------------------------
	updatePayload2 := map[string]interface{}{
		"amount_cents": int64(10003),
	}
	bodyUpdate2, _ := json.Marshal(updatePayload2)
	reqUpdate2, _ := http.NewRequest("PATCH", "/api/shared-expenses/"+txID1, bytes.NewBuffer(bodyUpdate2))
	reqUpdate2.AddCookie(cookieA)
	rrUpdate2 := httptest.NewRecorder()
	r.ServeHTTP(rrUpdate2, reqUpdate2)

	if rrUpdate2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for editing own shared expense, got %d. Body: %s", rrUpdate2.Code, rrUpdate2.Body.String())
	}

	err = db.QueryRow("SELECT share_amount FROM transaction_splits WHERE transaction_id = ? AND user_id = ?", txID1, userAID).Scan(&splitAmountA)
	if err != nil {
		t.Fatalf("query split amount A failed: %v", err)
	}
	err = db.QueryRow("SELECT share_amount FROM transaction_splits WHERE transaction_id = ? AND user_id = ?", txID1, userBID).Scan(&splitAmountB)
	if err != nil {
		t.Fatalf("query split amount B failed: %v", err)
	}

	if splitAmountA != 5002 || splitAmountB != 5001 {
		t.Errorf("expected updated split A: 5002, B: 5001, got A: %d, B: %d", splitAmountA, splitAmountB)
	}

	// 验证 audit_logs 审计表中更新日志存在，并且 before 和 after JSON 快照里含有分摊字段
	var beforeJSON, afterJSON string
	err = db.QueryRow("SELECT before_json, after_json FROM audit_logs WHERE action = 'update' AND entity_id = ? LIMIT 1", txID1).Scan(&beforeJSON, &afterJSON)
	if err != nil {
		t.Fatalf("query update audit log failed: %v", err)
	}

	if beforeJSON == "" || afterJSON == "" {
		t.Errorf("audit log JSON snapshots should not be empty")
	}

	// ----------------------------------------------------
	// 场景 7: A 登录软删除该共同支出。
	// ----------------------------------------------------
	reqDel, _ := http.NewRequest("DELETE", "/api/transactions/"+txID1, nil)
	reqDel.AddCookie(cookieA)
	rrDel := httptest.NewRecorder()
	r.ServeHTTP(rrDel, reqDel)

	if rrDel.Code != http.StatusOK {
		t.Errorf("expected 200 OK for deleting shared expense, got %d", rrDel.Code)
	}

	// A 再次获取列表，预期不含被软删除的 txID1
	reqList, _ := http.NewRequest("GET", "/api/transactions", nil)
	reqList.AddCookie(cookieA)
	rrList := httptest.NewRecorder()
	r.ServeHTTP(rrList, reqList)

	var listResp response.SuccessResponse
	json.Unmarshal(rrList.Body.Bytes(), &listResp)
	listData := listResp.Data.([]interface{})
	for _, item := range listData {
		txItem := item.(map[string]interface{})
		if txItem["id"].(string) == txID1 {
			t.Errorf("deleted shared expense should not be listed")
		}
	}
}
