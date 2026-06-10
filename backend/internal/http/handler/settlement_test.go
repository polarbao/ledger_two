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
	"ledger_two/internal/settlement"
	"ledger_two/internal/transaction"
)

func TestSettlementFlow(t *testing.T) {
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

	settleRepo := settlement.NewRepository(db)
	settleSvc := settlement.NewService(settleRepo)
	settleHandler := settlement.NewHandler(settleSvc)

	r := chi.NewRouter()
	r.Post("/api/init/setup", initHandler.HandleSetup)
	r.Post("/api/auth/login", authHandler.HandleLogin)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtSecret))
		r.Route("/api/transactions", func(r chi.Router) {
			r.Get("/", txHandler.HandleList)
		})
		r.Route("/api/shared-expenses", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreateSharedExpense)
		})
		r.Route("/api/settlements", func(r chi.Router) {
			r.Get("/balance", settleHandler.HandleGetBalance)
			r.Get("/", settleHandler.HandleList)
			r.Post("/", settleHandler.HandleCreate)
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
	// 场景 1: A 支付 200.00 元（20000分）共同支出，平摊（equal）。
	// ----------------------------------------------------
	payload1 := map[string]interface{}{
		"title":         "家庭日用品采购",
		"amount_cents":  int64(20000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"split_method":  "equal",
	}
	body1, _ := json.Marshal(payload1)
	req1, _ := http.NewRequest("POST", "/api/shared-expenses", bytes.NewBuffer(body1))
	req1.AddCookie(cookieA)
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rr1.Code)
	}

	// ----------------------------------------------------
	// 场景 2: B 支付 80.00 元（8000分）共同支出，平摊（equal）。
	// ----------------------------------------------------
	payload2 := map[string]interface{}{
		"title":         "买水果蔬菜",
		"amount_cents":  int64(8000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userBID,
		"category_id":   categoryID,
		"split_method":  "equal",
	}
	body2, _ := json.Marshal(payload2)
	req2, _ := http.NewRequest("POST", "/api/shared-expenses", bytes.NewBuffer(body2))
	req2.AddCookie(cookieB)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rr2.Code)
	}

	// ----------------------------------------------------
	// 场景 3: 检查待结算余额。预期为 B 欠 A 60.00 元（6000分）。
	// ----------------------------------------------------
	reqBalance, _ := http.NewRequest("GET", "/api/settlements/balance", nil)
	reqBalance.AddCookie(cookieA)
	rrBalance := httptest.NewRecorder()
	r.ServeHTTP(rrBalance, reqBalance)
	if rrBalance.Code != http.StatusOK {
		t.Fatalf("get balance failed: %v", rrBalance.Body.String())
	}

	var balanceResp response.SuccessResponse
	json.Unmarshal(rrBalance.Body.Bytes(), &balanceResp)
	balanceData := balanceResp.Data.(map[string]interface{})

	// 验证已付与应摊金额
	// 由于 userA 排序可能与 ID 的 userAID 对应，我们直接利用 JSON 数据校验
	// users 按照 username ASC 排序，因此 users[0] 为 userA（userAID），users[1] 为 userB（userBID）
	if int64(balanceData["user_a_paid_cents"].(float64)) != 20000 {
		t.Errorf("expected user_a_paid_cents = 20000, got %v", balanceData["user_a_paid_cents"])
	}
	if int64(balanceData["user_a_share_cents"].(float64)) != 14000 { // 10000 + 4000
		t.Errorf("expected user_a_share_cents = 14000, got %v", balanceData["user_a_share_cents"])
	}
	if int64(balanceData["user_b_paid_cents"].(float64)) != 8000 {
		t.Errorf("expected user_b_paid_cents = 8000, got %v", balanceData["user_b_paid_cents"])
	}
	if int64(balanceData["user_b_share_cents"].(float64)) != 14000 { // 10000 + 4000
		t.Errorf("expected user_b_share_cents = 14000, got %v", balanceData["user_b_share_cents"])
	}

	// 最终应该 B 欠 A 6000
	if balanceData["from_user_id"].(string) != userBID {
		t.Errorf("expected from_user_id (debtor) to be B, got %v", balanceData["from_user_id"])
	}
	if balanceData["to_user_id"].(string) != userAID {
		t.Errorf("expected to_user_id (creditor) to be A, got %v", balanceData["to_user_id"])
	}
	if int64(balanceData["amount_cents"].(float64)) != 6000 {
		t.Errorf("expected amount_cents = 6000, got %v", balanceData["amount_cents"])
	}

	// ----------------------------------------------------
	// 场景 4: B 发起对 A 的结算，支付 6000分。
	// ----------------------------------------------------
	payloadSettle := map[string]interface{}{
		"from_user_id": userBID,
		"to_user_id":   userAID,
		"amount_cents": int64(6000),
		"occurred_at":  time.Now().Format(time.RFC3339),
		"note":         "微信转账差额结清",
	}
	bodySettle, _ := json.Marshal(payloadSettle)
	reqSettle, _ := http.NewRequest("POST", "/api/settlements", bytes.NewBuffer(bodySettle))
	reqSettle.AddCookie(cookieB)
	rrSettle := httptest.NewRecorder()
	r.ServeHTTP(rrSettle, reqSettle)
	if rrSettle.Code != http.StatusCreated {
		t.Fatalf("create settlement failed: %v", rrSettle.Body.String())
	}

	// 验证在 transactions 交易流水表中是否成功生成了一条 type = 'settlement' 的流水
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM transactions WHERE type = 'settlement' AND payer_user_id = ? AND amount = ?", userBID, 6000).Scan(&count)
	if err != nil {
		t.Fatalf("query settlement transaction stream failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 settlement transaction stream record, got %d", count)
	}

	// 验证在 audit_logs 审计表中是否成功生成了一条 action = 'create' 且 entity_type = 'settlement' 的审计行
	var auditCount int
	err = db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action = 'create' AND entity_type = 'settlement'").Scan(&auditCount)
	if err != nil {
		t.Fatalf("query audit logs error: %v", err)
	}
	if auditCount != 1 {
		t.Errorf("expected 1 audit log for settlement creation, got %d", auditCount)
	}

	// ----------------------------------------------------
	// 场景 5: 结算后重新查询待结算余额，预期结清（金额为 0）。
	// ----------------------------------------------------
	rrBalance2 := httptest.NewRecorder()
	r.ServeHTTP(rrBalance2, reqBalance)
	if rrBalance2.Code != http.StatusOK {
		t.Fatalf("get balance after settlement failed: %v", rrBalance2.Body.String())
	}

	var balanceResp2 response.SuccessResponse
	json.Unmarshal(rrBalance2.Body.Bytes(), &balanceResp2)
	balanceData2 := balanceResp2.Data.(map[string]interface{})

	if int64(balanceData2["amount_cents"].(float64)) != 0 {
		t.Errorf("expected amount_cents = 0 after settlement, got %v", balanceData2["amount_cents"])
	}
	if balanceData2["from_user_id"].(string) != "" || balanceData2["to_user_id"].(string) != "" {
		t.Errorf("expected from/to user ids to be empty after settlement, got from: %v, to: %v", balanceData2["from_user_id"], balanceData2["to_user_id"])
	}

	// ----------------------------------------------------
	// 场景 6: 拉取结算历史并进行月份过滤。
	// ----------------------------------------------------
	currentMonth := time.Now().Format("2006-01")
	reqHistory, _ := http.NewRequest("GET", "/api/settlements?month="+currentMonth, nil)
	reqHistory.AddCookie(cookieA)
	rrHistory := httptest.NewRecorder()
	r.ServeHTTP(rrHistory, reqHistory)
	if rrHistory.Code != http.StatusOK {
		t.Fatalf("get settlements history failed: %v", rrHistory.Body.String())
	}

	var historyResp response.SuccessResponse
	json.Unmarshal(rrHistory.Body.Bytes(), &historyResp)
	historyData := historyResp.Data.([]interface{})
	if len(historyData) != 1 {
		t.Errorf("expected 1 history record, got %d", len(historyData))
	}
}
