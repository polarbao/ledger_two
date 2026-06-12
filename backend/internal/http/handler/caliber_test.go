package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"ledger_two/internal/dashboard"
	"ledger_two/internal/db/repo"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/reports"
	"ledger_two/internal/service"
	"ledger_two/internal/settlement"
	"ledger_two/internal/transaction"
)

// TestStatisticsAndSettlementCaliber 验证 docs/tech/12-statistics-caliber.md 规定的统计口径核心业务逻辑
func TestStatisticsAndSettlementCaliber(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret-caliber"

	// 初始化 Handler 与 Service 层
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

	dashRepo := dashboard.NewRepository(db)
	dashSvc := dashboard.NewService(dashRepo, settleSvc)
	dashHandler := dashboard.NewHandler(dashSvc)

	reportsSvc := reports.NewService(db, dashRepo, settleSvc)
	reportsHandler := reports.NewHandler(reportsSvc)

	r := chi.NewRouter()
	r.Post("/api/init/setup", initHandler.HandleSetup)
	r.Post("/api/auth/login", authHandler.HandleLogin)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtSecret))
		r.Get("/api/dashboard", dashHandler.HandleGetDashboard)
		r.Route("/api/transactions", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreate)
			r.Delete("/{id}", txHandler.HandleDelete)
			r.Patch("/{id}", txHandler.HandleUpdate)
		})
		r.Route("/api/shared-expenses", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreateSharedExpense)
		})
		r.Route("/api/settlements", func(r chi.Router) {
			r.Get("/balance", settleHandler.HandleGetBalance)
			r.Post("/", settleHandler.HandleCreate)
		})
		r.Route("/api/reports", func(r chi.Router) {
			r.Get("/monthly-summary", reportsHandler.HandleGetMonthlySummary)
			r.Get("/member-summary", reportsHandler.HandleGetMemberSummary)
		})
	})

	// 1. 初始化系统，注入 A、B 两个用户
	setupPayload := map[string]string{
		"ledger_name":         "Caliber Test Ledger",
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
	_ = db.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&userAID)
	_ = db.QueryRow("SELECT id FROM users WHERE username = 'userB'").Scan(&userBID)

	var categoryID string
	_ = db.QueryRow("SELECT id FROM categories LIMIT 1").Scan(&categoryID)

	// -------------------------------------------------------------------------
	// 2. 校验边界：金额 <= 0 时创建账单失败，且返回统一格式的 VALIDATION_ERROR 错误码
	// -------------------------------------------------------------------------
	badPayload := map[string]interface{}{
		"type":          "expense",
		"title":         "无效金额账单",
		"amount_cents":  int64(0),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
	}
	badBody, _ := json.Marshal(badPayload)
	reqBad, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(badBody))
	reqBad.AddCookie(cookieA)
	rrBad := httptest.NewRecorder()
	r.ServeHTTP(rrBad, reqBad)

	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for zero amount, got %d", rrBad.Code)
	}

	var errResp response.ErrorResponse
	json.Unmarshal(rrBad.Body.Bytes(), &errResp)
	if errResp.Success || errResp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR unified format, got %+v", errResp)
	}

	// -------------------------------------------------------------------------
	// 3. 可见性与统计隔离测试：
	//    - A 创建 private 支出 100元 (10000分)。
	//    - A 视角：本月支出包含此 10000分。
	//    - B 视角：本月支出不包含此 10000分，且 B 尝试 GET 会 404，PATCH 会 403 / 404。
	// -------------------------------------------------------------------------
	payloadPrivA := map[string]interface{}{
		"type":          "expense",
		"title":         "A的私有日记本",
		"amount_cents":  int64(10000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"visibility":    "private",
	}
	bodyPrivA, _ := json.Marshal(payloadPrivA)
	reqPrivA, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyPrivA))
	reqPrivA.AddCookie(cookieA)
	rrPrivA := httptest.NewRecorder()
	r.ServeHTTP(rrPrivA, reqPrivA)
	if rrPrivA.Code != http.StatusCreated {
		t.Fatalf("create private transaction failed, got %d", rrPrivA.Code)
	}

	var createPrivResp response.SuccessResponse
	json.Unmarshal(rrPrivA.Body.Bytes(), &createPrivResp)
	privTxID := createPrivResp.Data.(map[string]interface{})["id"].(string)

	// B 视角拉取 summary，应当为 0
	reqSummaryB, _ := http.NewRequest("GET", "/api/reports/monthly-summary", nil)
	reqSummaryB.AddCookie(cookieB)
	rrSummaryB := httptest.NewRecorder()
	r.ServeHTTP(rrSummaryB, reqSummaryB)
	var respSummaryB response.SuccessResponse
	json.Unmarshal(rrSummaryB.Body.Bytes(), &respSummaryB)
	dataSummaryB := respSummaryB.Data.(map[string]interface{})
	if int64(dataSummaryB["total_expense"].(float64)) != 0 {
		t.Errorf("B should not see A's private transaction, got total_expense=%v", dataSummaryB["total_expense"])
	}

	// B 尝试修改 A 的私有账单，预期 403 Forbidden 或 404 Not Found
	updatePayload := map[string]interface{}{"title": "B恶意修改"}
	bodyUpdate, _ := json.Marshal(updatePayload)
	reqUpdateB, _ := http.NewRequest("PATCH", "/api/transactions/"+privTxID, bytes.NewBuffer(bodyUpdate))
	reqUpdateB.AddCookie(cookieB)
	rrUpdateB := httptest.NewRecorder()
	r.ServeHTTP(rrUpdateB, reqUpdateB)
	if rrUpdateB.Code != http.StatusForbidden && rrUpdateB.Code != http.StatusNotFound {
		t.Errorf("B should be forbidden to update A's private transaction, got %d", rrUpdateB.Code)
	}

	// -------------------------------------------------------------------------
	// 4. 共享分摊与结算逻辑口径验证：
	//    - A 创建共同支出 200元 (20000分)，平摊 (equal)。A/B 各承担 10000分。
	//    - B 创建共同支出 80元 (8000分)，平摊 (equal)。A/B 各承担 4000分。
	//    - B 创建 partner_readable 个人支出 50元 (5000分)。
	// -------------------------------------------------------------------------
	payloadSharedA := map[string]interface{}{
		"title":         "买菜平摊",
		"amount_cents":  int64(20000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"split_method":  "equal",
	}
	bodySharedA, _ := json.Marshal(payloadSharedA)
	reqSharedA, _ := http.NewRequest("POST", "/api/shared-expenses", bytes.NewBuffer(bodySharedA))
	reqSharedA.AddCookie(cookieA)
	rrSharedA := httptest.NewRecorder()
	r.ServeHTTP(rrSharedA, reqSharedA)
	if rrSharedA.Code != http.StatusCreated {
		t.Fatalf("create shared expense A failed, got %d", rrSharedA.Code)
	}

	payloadSharedB := map[string]interface{}{
		"title":         "买日用品平摊",
		"amount_cents":  int64(8000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userBID,
		"category_id":   categoryID,
		"split_method":  "equal",
	}
	bodySharedB, _ := json.Marshal(payloadSharedB)
	reqSharedB, _ := http.NewRequest("POST", "/api/shared-expenses", bytes.NewBuffer(bodySharedB))
	reqSharedB.AddCookie(cookieB)
	rrSharedB := httptest.NewRecorder()
	r.ServeHTTP(rrSharedB, reqSharedB)
	if rrSharedB.Code != http.StatusCreated {
		t.Fatalf("create shared expense B failed, got %d", rrSharedB.Code)
	}

	// B 创建 partner_readable 个人支出 50元 (5000分)。
	payloadReadB := map[string]interface{}{
		"type":          "expense",
		"title":         "B的伙伴可见个人支出",
		"amount_cents":  int64(5000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userBID,
		"category_id":   categoryID,
		"visibility":    "partner_readable",
	}
	bodyReadB, _ := json.Marshal(payloadReadB)
	reqReadB, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyReadB))
	reqReadB.AddCookie(cookieB)
	rrReadB := httptest.NewRecorder()
	r.ServeHTTP(rrReadB, reqReadB)
	if rrReadB.Code != http.StatusCreated {
		t.Fatalf("create partner_readable expense failed, got %d", rrReadB.Code)
	}

	// -------------------------------------------------------------------------
	// 5. 校验月度总额与成员统计字段 (paid_amount, share_amount, raw_net)：
	//    A 的视角拉取 summary：
	//      - 包含 A私有(10000) + 共同A(20000) + 共同B(8000) + B可见个人(5000) = 43000 分
	//    A 的视角拉取 member-summary：
	//      - memberA:
	//         paid_amount  = A私有(10000) + 共同A(20000) = 30000
	//         share_amount = A私有(10000) + 共同A的平摊(10000) + 共同B的平摊(4000) = 24000
	//         raw_net = paid - share = 6000
	//      - memberB:
	//         paid_amount  = 共同B(8000) + B可见个人(5000) = 13000
	//         share_amount = 共同A的平摊(10000) + 共同B的平摊(4000) + B可见个人(5000) = 19000
	//         raw_net = paid - share = -6000
	// -------------------------------------------------------------------------
	reqSummaryA, _ := http.NewRequest("GET", "/api/reports/monthly-summary", nil)
	reqSummaryA.AddCookie(cookieA)
	rrSummaryA := httptest.NewRecorder()
	r.ServeHTTP(rrSummaryA, reqSummaryA)
	var respSummaryA response.SuccessResponse
	json.Unmarshal(rrSummaryA.Body.Bytes(), &respSummaryA)
	dataSummaryA := respSummaryA.Data.(map[string]interface{})
	if int64(dataSummaryA["total_expense"].(float64)) != 43000 {
		t.Errorf("expected total_expense A = 43000, got %v", dataSummaryA["total_expense"])
	}

	reqMemberA, _ := http.NewRequest("GET", "/api/reports/member-summary", nil)
	reqMemberA.AddCookie(cookieA)
	rrMemberA := httptest.NewRecorder()
	r.ServeHTTP(rrMemberA, reqMemberA)
	var respMemberA response.SuccessResponse
	json.Unmarshal(rrMemberA.Body.Bytes(), &respMemberA)
	members := respMemberA.Data.([]interface{})

	var memA, memB map[string]interface{}
	for _, m := range members {
		mMap := m.(map[string]interface{})
		if mMap["user_id"].(string) == userAID {
			memA = mMap
		} else {
			memB = mMap
		}
	}

	if int64(memA["paid_amount"].(float64)) != 30000 || int64(memA["share_amount"].(float64)) != 24000 {
		t.Errorf("A stats caliber error, got paid=%v, share=%v", memA["paid_amount"], memA["share_amount"])
	}
	if int64(memB["paid_amount"].(float64)) != 13000 || int64(memB["share_amount"].(float64)) != 19000 {
		t.Errorf("B stats caliber error, got paid=%v, share=%v", memB["paid_amount"], memB["share_amount"])
	}

	if int64(memA["raw_net"].(float64)) != 6000 {
		t.Errorf("expected A raw_net = 6000, got %v", memA["raw_net"])
	}
	if int64(memB["raw_net"].(float64)) != -6000 {
		t.Errorf("expected B raw_net = -6000, got %v", memB["raw_net"])
	}

	// -------------------------------------------------------------------------
	// 6. 结算对冲抵扣测试：
	//    - B 结算支付 6000 分给 A。
	//    - 校验待结算余额：结清且金额变为 0。
	//    - 校验结算记录本身是否会错误地计入消费支出统计中 (expected: settlement 不属于消费支出，支出仍为 43000 分)
	// -------------------------------------------------------------------------
	payloadSettle := map[string]interface{}{
		"from_user_id": userBID,
		"to_user_id":   userAID,
		"amount_cents": int64(6000),
		"occurred_at":  time.Now().Format(time.RFC3339),
		"note":         "微信结账",
	}
	bodySettle, _ := json.Marshal(payloadSettle)
	reqSettle, _ := http.NewRequest("POST", "/api/settlements", bytes.NewBuffer(bodySettle))
	reqSettle.AddCookie(cookieB)
	rrSettle := httptest.NewRecorder()
	r.ServeHTTP(rrSettle, reqSettle)
	if rrSettle.Code != http.StatusCreated {
		t.Fatalf("create settlement failed, got %d", rrSettle.Code)
	}

	// 待结算金额变为 0
	reqBalance, _ := http.NewRequest("GET", "/api/settlements/balance", nil)
	reqBalance.AddCookie(cookieA)
	rrBalance := httptest.NewRecorder()
	r.ServeHTTP(rrBalance, reqBalance)
	var respBal response.SuccessResponse
	json.Unmarshal(rrBalance.Body.Bytes(), &respBal)
	balData := respBal.Data.(map[string]interface{})
	if int64(balData["amount_cents"].(float64)) != 0 {
		t.Errorf("expected balance to be 0 after settlement, got %v", balData["amount_cents"])
	}

	// A 再次拉取总支出，预期仍为 43000 分（代表 6000分的 settlement 不应作为消费被统计进去）
	rrSummaryA2 := httptest.NewRecorder()
	r.ServeHTTP(rrSummaryA2, reqSummaryA)
	var respSummaryA2 response.SuccessResponse
	json.Unmarshal(rrSummaryA2.Body.Bytes(), &respSummaryA2)
	dataSummaryA2 := respSummaryA2.Data.(map[string]interface{})
	if int64(dataSummaryA2["total_expense"].(float64)) != 43000 {
		t.Errorf("expected total_expense to remain 43000 after settlement, got %v", dataSummaryA2["total_expense"])
	}

	// -------------------------------------------------------------------------
	// 7. Soft Delete 软删除统计实时扣除验证：
	//    - A 软删除自己创建的 A私有 10000 分账单。
	//    - 预期拉取月度支出：实时扣减 10000 分，变为 33000 分。
	// -------------------------------------------------------------------------
	reqDel, _ := http.NewRequest("DELETE", "/api/transactions/"+privTxID, nil)
	reqDel.AddCookie(cookieA)
	rrDel := httptest.NewRecorder()
	r.ServeHTTP(rrDel, reqDel)
	if rrDel.Code != http.StatusOK {
		t.Fatalf("delete private tx failed, got %d", rrDel.Code)
	}

	rrSummaryA3 := httptest.NewRecorder()
	r.ServeHTTP(rrSummaryA3, reqSummaryA)
	var respSummaryA3 response.SuccessResponse
	json.Unmarshal(rrSummaryA3.Body.Bytes(), &respSummaryA3)
	dataSummaryA3 := respSummaryA3.Data.(map[string]interface{})
	if int64(dataSummaryA3["total_expense"].(float64)) != 33000 {
		t.Errorf("expected total_expense after delete private tx to be 33000, got %v", dataSummaryA3["total_expense"])
	}
}

// TestAdvancedFilterAndBatchTag 验证 Task 15 高级筛选与批量打标签功能
func TestAdvancedFilterAndBatchTag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret-caliber-advanced"

	// 初始化 Handler 与 Service 层
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

	dashRepo := dashboard.NewRepository(db)
	dashSvc := dashboard.NewService(dashRepo, settleSvc)
	dashHandler := dashboard.NewHandler(dashSvc)

	reportsSvc := reports.NewService(db, dashRepo, settleSvc)
	reportsHandler := reports.NewHandler(reportsSvc)

	r := chi.NewRouter()
	r.Post("/api/init/setup", initHandler.HandleSetup)
	r.Post("/api/auth/login", authHandler.HandleLogin)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtSecret))
		r.Get("/api/dashboard", dashHandler.HandleGetDashboard)
		r.Route("/api/transactions", func(r chi.Router) {
			r.Get("/", txHandler.HandleList)
			r.Post("/", txHandler.HandleCreate)
			r.Post("/batch-tag", txHandler.HandleBatchTag)
			r.Get("/{id}", txHandler.HandleGetByID)
			r.Delete("/{id}", txHandler.HandleDelete)
			r.Patch("/{id}", txHandler.HandleUpdate)
		})
		r.Route("/api/shared-expenses", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreateSharedExpense)
		})
		r.Route("/api/settlements", func(r chi.Router) {
			r.Get("/balance", settleHandler.HandleGetBalance)
			r.Post("/", settleHandler.HandleCreate)
		})
		r.Route("/api/reports", func(r chi.Router) {
			r.Get("/monthly-summary", reportsHandler.HandleGetMonthlySummary)
			r.Get("/member-summary", reportsHandler.HandleGetMemberSummary)
		})
	})

	// 1. 初始化系统，注入 A、B 两个用户
	setupPayload := map[string]string{
		"ledger_name":         "Advanced Filter Ledger",
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
	_ = db.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&userAID)
	_ = db.QueryRow("SELECT id FROM users WHERE username = 'userB'").Scan(&userBID)

	var categoryID string
	_ = db.QueryRow("SELECT id FROM categories LIMIT 1").Scan(&categoryID)

	// 2. 创建 5 笔测试数据
	// 交易 1：A 创建, type: expense, amount: 1000, private, tags: ["food", "lunch"]
	tx1Payload := map[string]interface{}{
		"type":          "expense",
		"title":         "A私有午餐",
		"amount_cents":  int64(1000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"visibility":    "private",
		"tag_names":     []string{"food", "lunch"},
	}
	bodyTx1, _ := json.Marshal(tx1Payload)
	reqTx1, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyTx1))
	reqTx1.AddCookie(cookieA)
	rrTx1 := httptest.NewRecorder()
	r.ServeHTTP(rrTx1, reqTx1)
	if rrTx1.Code != http.StatusCreated {
		t.Fatalf("create transaction 1 failed: %d", rrTx1.Code)
	}
	var resTx1 response.SuccessResponse
	json.Unmarshal(rrTx1.Body.Bytes(), &resTx1)
	tx1ID := resTx1.Data.(map[string]interface{})["id"].(string)

	// 交易 2：A 创建, type: expense, amount: 2000, partner_readable, tags: ["food"]
	tx2Payload := map[string]interface{}{
		"type":          "expense",
		"title":         "A共享晚餐",
		"amount_cents":  int64(2000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"visibility":    "partner_readable",
		"tag_names":     []string{"food"},
	}
	bodyTx2, _ := json.Marshal(tx2Payload)
	reqTx2, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyTx2))
	reqTx2.AddCookie(cookieA)
	rrTx2 := httptest.NewRecorder()
	r.ServeHTTP(rrTx2, reqTx2)
	if rrTx2.Code != http.StatusCreated {
		t.Fatalf("create transaction 2 failed: %d", rrTx2.Code)
	}
	var resTx2 response.SuccessResponse
	json.Unmarshal(rrTx2.Body.Bytes(), &resTx2)
	tx2ID := resTx2.Data.(map[string]interface{})["id"].(string)

	// 交易 3：B 创建, type: shared_expense, amount: 3000, split_method: equal, tags: ["rent"]
	tx3Payload := map[string]interface{}{
		"title":         "B平摊租房",
		"amount_cents":  int64(3000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userBID,
		"category_id":   categoryID,
		"split_method":  "equal",
		"tag_names":     []string{"rent"},
	}
	bodyTx3, _ := json.Marshal(tx3Payload)
	reqTx3, _ := http.NewRequest("POST", "/api/shared-expenses", bytes.NewBuffer(bodyTx3))
	reqTx3.AddCookie(cookieB)
	rrTx3 := httptest.NewRecorder()
	r.ServeHTTP(rrTx3, reqTx3)
	if rrTx3.Code != http.StatusCreated {
		t.Fatalf("create transaction 3 failed: %d", rrTx3.Code)
	}
	var resTx3 response.SuccessResponse
	json.Unmarshal(rrTx3.Body.Bytes(), &resTx3)
	_ = resTx3.Data.(map[string]interface{})["id"].(string)

	// 交易 4：B 创建, type: expense, amount: 5000, private, tags: ["game"]
	tx4Payload := map[string]interface{}{
		"type":          "expense",
		"title":         "B私有游戏",
		"amount_cents":  int64(5000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userBID,
		"category_id":   categoryID,
		"visibility":    "private",
		"tag_names":     []string{"game"},
	}
	bodyTx4, _ := json.Marshal(tx4Payload)
	reqTx4, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyTx4))
	reqTx4.AddCookie(cookieB)
	rrTx4 := httptest.NewRecorder()
	r.ServeHTTP(rrTx4, reqTx4)
	if rrTx4.Code != http.StatusCreated {
		t.Fatalf("create transaction 4 failed: %d", rrTx4.Code)
	}

	// 交易 5：A 创建, type: income, amount: 10000, partner_readable, tags: ["salary"]
	tx5Payload := map[string]interface{}{
		"type":          "income",
		"title":         "A个人工资",
		"amount_cents":  int64(10000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"visibility":    "partner_readable",
		"tag_names":     []string{"salary"},
	}
	bodyTx5, _ := json.Marshal(tx5Payload)
	reqTx5, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyTx5))
	reqTx5.AddCookie(cookieA)
	rrTx5 := httptest.NewRecorder()
	r.ServeHTTP(rrTx5, reqTx5)
	if rrTx5.Code != http.StatusCreated {
		t.Fatalf("create transaction 5 failed: %d", rrTx5.Code)
	}

	// 3. 多维度高级筛选测试 (GET /api/transactions)
	// (a) A 视角拉取全量，应该看到 1, 2, 3, 5。看不到 4 (B的private)。共 4 笔
	reqListAllA, _ := http.NewRequest("GET", "/api/transactions", nil)
	reqListAllA.AddCookie(cookieA)
	rrListAllA := httptest.NewRecorder()
	r.ServeHTTP(rrListAllA, reqListAllA)
	var resListAllA response.SuccessResponse
	json.Unmarshal(rrListAllA.Body.Bytes(), &resListAllA)
	listAllA := resListAllA.Data.([]interface{})
	if len(listAllA) != 4 {
		t.Errorf("expected A to see 4 transactions, got %d", len(listAllA))
	}

	// (b) A 视角筛选金额：min_amount=1500&max_amount=4000。预期得到 2 (2000), 3 (3000)。共 2 笔
	reqListAmountA, _ := http.NewRequest("GET", "/api/transactions?min_amount=1500&max_amount=4000", nil)
	reqListAmountA.AddCookie(cookieA)
	rrListAmountA := httptest.NewRecorder()
	r.ServeHTTP(rrListAmountA, reqListAmountA)
	var resListAmountA response.SuccessResponse
	json.Unmarshal(rrListAmountA.Body.Bytes(), &resListAmountA)
	listAmountA := resListAmountA.Data.([]interface{})
	if len(listAmountA) != 2 {
		t.Errorf("expected A to see 2 transactions in amount range [1500, 4000], got %d", len(listAmountA))
	}

	// (c) A 视角过滤标签：tag=food。预期得到 1 (food, lunch), 2 (food)。共 2 笔
	reqListTagA, _ := http.NewRequest("GET", "/api/transactions?tag=food", nil)
	reqListTagA.AddCookie(cookieA)
	rrListTagA := httptest.NewRecorder()
	r.ServeHTTP(rrListTagA, reqListTagA)
	var resListTagA response.SuccessResponse
	json.Unmarshal(rrListTagA.Body.Bytes(), &resListTagA)
	listTagA := resListTagA.Data.([]interface{})
	if len(listTagA) != 2 {
		t.Errorf("expected A to see 2 transactions with tag=food, got %d", len(listTagA))
	}

	// (d) A 视角过滤付款人与可见性：payer_user_id=userBID, visibility=shared。预期得到 3 (shared)。共 1 笔
	reqListMultiA, _ := http.NewRequest("GET", "/api/transactions?payer_user_id="+userBID+"&visibility=shared", nil)
	reqListMultiA.AddCookie(cookieA)
	rrListMultiA := httptest.NewRecorder()
	r.ServeHTTP(rrListMultiA, reqListMultiA)
	var resListMultiA response.SuccessResponse
	json.Unmarshal(rrListMultiA.Body.Bytes(), &resListMultiA)
	listMultiA := resListMultiA.Data.([]interface{})
	if len(listMultiA) != 1 {
		t.Errorf("expected A to see 1 transaction with B payer and shared visibility, got %d", len(listMultiA))
	}

	// (e) B 视角拉取全量，应该看不到 1 (A的private)。共 4 笔 (2, 3, 4, 5)
	reqListAllB, _ := http.NewRequest("GET", "/api/transactions", nil)
	reqListAllB.AddCookie(cookieB)
	rrListAllB := httptest.NewRecorder()
	r.ServeHTTP(rrListAllB, reqListAllB)
	var resListAllB response.SuccessResponse
	json.Unmarshal(rrListAllB.Body.Bytes(), &resListAllB)
	listAllB := resListAllB.Data.([]interface{})
	if len(listAllB) != 4 {
		t.Errorf("expected B to see 4 transactions, got %d", len(listAllB))
	}

	// 4. 批量打标签越权测试 (POST /api/transactions/batch-tag)
	// (a) B 试图给 A 的私有交易 1 打标签，预期返回 403 Forbidden
	batchIllegalPayload := map[string]interface{}{
		"transaction_ids": []string{tx1ID},
		"tag_names":       []string{"illegal"},
	}
	bodyIllegal, _ := json.Marshal(batchIllegalPayload)
	reqIllegal, _ := http.NewRequest("POST", "/api/transactions/batch-tag", bytes.NewBuffer(bodyIllegal))
	reqIllegal.AddCookie(cookieB)
	rrIllegal := httptest.NewRecorder()
	r.ServeHTTP(rrIllegal, reqIllegal)
	if rrIllegal.Code != http.StatusForbidden {
		t.Errorf("expected B tagging A's private transaction to return 403 Forbidden, got %d", rrIllegal.Code)
	}

	// 5. 批量打标签正常追加测试
	// A 批量给交易 1 (原有: ["food", "lunch"]) 和交易 2 (原有: ["food"]) 打上标签 ["batch1", "food"]
	batchPayload := map[string]interface{}{
		"transaction_ids": []string{tx1ID, tx2ID},
		"tag_names":       []string{"batch1", "food"},
	}
	bodyBatch, _ := json.Marshal(batchPayload)
	reqBatch, _ := http.NewRequest("POST", "/api/transactions/batch-tag", bytes.NewBuffer(bodyBatch))
	reqBatch.AddCookie(cookieA)
	rrBatch := httptest.NewRecorder()
	r.ServeHTTP(rrBatch, reqBatch)
	if rrBatch.Code != http.StatusOK {
		t.Fatalf("batch tag failed, got code %d, body: %s", rrBatch.Code, rrBatch.Body.String())
	}

	// 校验交易 1 和交易 2 的标签状态
	// 交易 1 应包含：food, lunch, batch1 (去重追加，无重复)
	reqTx1Detail, _ := http.NewRequest("GET", "/api/transactions/"+tx1ID, nil)
	reqTx1Detail.AddCookie(cookieA)
	rrTx1Detail := httptest.NewRecorder()
	r.ServeHTTP(rrTx1Detail, reqTx1Detail)
	var resTx1Detail response.SuccessResponse
	json.Unmarshal(rrTx1Detail.Body.Bytes(), &resTx1Detail)
	tx1Data := resTx1Detail.Data.(map[string]interface{})
	tx1Tags := tx1Data["tags"].([]interface{})
	if len(tx1Tags) != 3 {
		t.Errorf("expected tx1 tags length to be 3, got %d (tags: %v)", len(tx1Tags), tx1Tags)
	}

	// 交易 2 应包含：food, batch1 (去重追加，无重复)
	reqTx2Detail, _ := http.NewRequest("GET", "/api/transactions/"+tx2ID, nil)
	reqTx2Detail.AddCookie(cookieA)
	rrTx2Detail := httptest.NewRecorder()
	r.ServeHTTP(rrTx2Detail, reqTx2Detail)
	var resTx2Detail response.SuccessResponse
	json.Unmarshal(rrTx2Detail.Body.Bytes(), &resTx2Detail)
	tx2Data := resTx2Detail.Data.(map[string]interface{})
	tx2Tags := tx2Data["tags"].([]interface{})
	if len(tx2Tags) != 2 {
		t.Errorf("expected tx2 tags length to be 2, got %d (tags: %v)", len(tx2Tags), tx2Tags)
	}

	// 校验审计日志：应该新增了 2 条 action = 'batch_tag' 的审计记录
	var auditCount int
	err := db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action = 'batch_tag'").Scan(&auditCount)
	if err != nil {
		t.Fatalf("query audit logs count failed: %v", err)
	}
	if auditCount != 2 {
		t.Errorf("expected 2 batch_tag audit logs, got %d", auditCount)
	}
}
