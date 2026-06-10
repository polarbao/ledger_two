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

func TestReportsFlow(t *testing.T) {
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
		})
		r.Route("/api/shared-expenses", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreateSharedExpense)
		})
		r.Route("/api/settlements", func(r chi.Router) {
			r.Post("/", settleHandler.HandleCreate)
		})
		r.Route("/api/reports", func(r chi.Router) {
			r.Get("/monthly-summary", reportsHandler.HandleGetMonthlySummary)
			r.Get("/category-summary", reportsHandler.HandleGetCategorySummary)
			r.Get("/tag-summary", reportsHandler.HandleGetTagSummary)
			r.Get("/member-summary", reportsHandler.HandleGetMemberSummary)
		})
	})

	// 1. 初始化
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

	// 获取真实 ID
	var userAID, userBID string
	_ = db.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&userAID)
	_ = db.QueryRow("SELECT id FROM users WHERE username = 'userB'").Scan(&userBID)

	var categoryID string
	_ = db.QueryRow("SELECT id FROM categories LIMIT 1").Scan(&categoryID)

	// 2. 鉴权测试
	reqNoAuth, _ := http.NewRequest("GET", "/api/reports/monthly-summary", nil)
	rrNoAuth := httptest.NewRecorder()
	r.ServeHTTP(rrNoAuth, reqNoAuth)
	if rrNoAuth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthorized summary query, got %d", rrNoAuth.Code)
	}

	// 3. 创建测试账单：A 记账共同支出 200元 (20000分)，B 记账个人支出 50元 (5000分)
	payload1 := map[string]interface{}{
		"title":         "买菜平摊",
		"amount_cents":  int64(20000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"split_method":  "equal",
		"tag_names":     []string{"餐饮", "买菜"},
	}
	body1, _ := json.Marshal(payload1)
	req1, _ := http.NewRequest("POST", "/api/shared-expenses", bytes.NewBuffer(body1))
	req1.AddCookie(cookieA)
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("create shared expense failed, got %d", rr1.Code)
	}

	payload2 := map[string]interface{}{
		"type":          "expense",
		"title":         "B的私人日记本",
		"amount_cents":  int64(5000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userBID,
		"category_id":   categoryID,
		"visibility":    "private",
	}
	body2, _ := json.Marshal(payload2)
	req2, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(body2))
	req2.AddCookie(cookieB)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("create transaction failed, got %d", rr2.Code)
	}

	// 4. 验证月度汇总：B 的个人账单对 A 不可见，但对 B 可见
	// A 视角拉取月度汇总，预期 A 只能看到共同支出 20000 分
	reqSummaryA, _ := http.NewRequest("GET", "/api/reports/monthly-summary", nil)
	reqSummaryA.AddCookie(cookieA)
	rrSummaryA := httptest.NewRecorder()
	r.ServeHTTP(rrSummaryA, reqSummaryA)
	if rrSummaryA.Code != http.StatusOK {
		t.Fatalf("get monthly summary for A failed, got %d", rrSummaryA.Code)
	}

	var respSummaryA response.SuccessResponse
	json.Unmarshal(rrSummaryA.Body.Bytes(), &respSummaryA)
	dataSummaryA := respSummaryA.Data.(map[string]interface{})
	if int64(dataSummaryA["total_expense"].(float64)) != 20000 {
		t.Errorf("expected A total_expense = 20000, got %v", dataSummaryA["total_expense"])
	}
	if int64(dataSummaryA["shared_expense"].(float64)) != 20000 {
		t.Errorf("expected A shared_expense = 20000, got %v", dataSummaryA["shared_expense"])
	}

	// B 视角拉取月度汇总，预期 B 能看到共同支出 20000 分 + 个人私有 5000 分 = 25000 分
	reqSummaryB, _ := http.NewRequest("GET", "/api/reports/monthly-summary", nil)
	reqSummaryB.AddCookie(cookieB)
	rrSummaryB := httptest.NewRecorder()
	r.ServeHTTP(rrSummaryB, reqSummaryB)
	if rrSummaryB.Code != http.StatusOK {
		t.Fatalf("get monthly summary for B failed, got %d", rrSummaryB.Code)
	}

	var respSummaryB response.SuccessResponse
	json.Unmarshal(rrSummaryB.Body.Bytes(), &respSummaryB)
	dataSummaryB := respSummaryB.Data.(map[string]interface{})
	if int64(dataSummaryB["total_expense"].(float64)) != 25000 {
		t.Errorf("expected B total_expense = 25000, got %v", dataSummaryB["total_expense"])
	}

	// 验证与 Dashboard 是否绝对一致
	reqDashB, _ := http.NewRequest("GET", "/api/dashboard", nil)
	reqDashB.AddCookie(cookieB)
	rrDashB := httptest.NewRecorder()
	r.ServeHTTP(rrDashB, reqDashB)
	var respDashB response.SuccessResponse
	json.Unmarshal(rrDashB.Body.Bytes(), &respDashB)
	dataDashB := respDashB.Data.(map[string]interface{})
	if dataDashB["total_expense_cents"].(float64) != dataSummaryB["total_expense"].(float64) {
		t.Errorf("Dashboard total expense mismatch with monthly report summary!")
	}

	// 5. 验证成员统计：B 视角拉取，预期：
	// A: paid = 20000, share = 10000 (平摊的一半)
	// B: paid = 5000 (个人), share = 15000 (B平摊的10000 + 个人5000)
	reqMemberB, _ := http.NewRequest("GET", "/api/reports/member-summary", nil)
	reqMemberB.AddCookie(cookieB)
	rrMemberB := httptest.NewRecorder()
	r.ServeHTTP(rrMemberB, reqMemberB)
	if rrMemberB.Code != http.StatusOK {
		t.Fatalf("get member summary failed, got %d", rrMemberB.Code)
	}

	var respMemberB response.SuccessResponse
	json.Unmarshal(rrMemberB.Body.Bytes(), &respMemberB)
	membersData := respMemberB.Data.([]interface{})
	if len(membersData) != 2 {
		t.Fatalf("expected 2 members, got %d", len(membersData))
	}

	var memberA, memberB map[string]interface{}
	for _, m := range membersData {
		mMap := m.(map[string]interface{})
		if mMap["user_id"].(string) == userAID {
			memberA = mMap
		} else {
			memberB = mMap
		}
	}

	if int64(memberA["paid_amount"].(float64)) != 20000 || int64(memberA["share_amount"].(float64)) != 10000 {
		t.Errorf("memberA stats mismatch, got paid: %v, share: %v", memberA["paid_amount"], memberA["share_amount"])
	}
	if int64(memberB["paid_amount"].(float64)) != 5000 || int64(memberB["share_amount"].(float64)) != 15000 {
		t.Errorf("memberB stats mismatch, got paid: %v, share: %v", memberB["paid_amount"], memberB["share_amount"])
	}

	// 验证 raw_net, final_net 均正确
	if int64(memberA["raw_net"].(float64)) != 10000 {
		t.Errorf("expected raw_net A = 10000, got %v", memberA["raw_net"])
	}
	if int64(memberB["raw_net"].(float64)) != -10000 {
		t.Errorf("expected raw_net B = -10000, got %v", memberB["raw_net"])
	}

	// 6. 软删除 A 账单后，统计额度是否扣除更新
	// 获取 A 创建的那个 shared_expense 的 ID
	var txID string
	err := db.QueryRow("SELECT id FROM transactions WHERE type = 'shared_expense'").Scan(&txID)
	if err != nil {
		t.Fatalf("query shared expense id failed: %v", err)
	}

	reqDel, _ := http.NewRequest("DELETE", "/api/transactions/"+txID, nil)
	reqDel.AddCookie(cookieA)
	rrDel := httptest.NewRecorder()
	r.ServeHTTP(rrDel, reqDel)
	if rrDel.Code != http.StatusOK {
		t.Fatalf("delete shared expense failed, got %d", rrDel.Code)
	}

	// B 再次拉取汇总，应只剩 B 的个人支出 5000 分 (总支出 20000 分已被扣除)
	rrSummaryB2 := httptest.NewRecorder()
	r.ServeHTTP(rrSummaryB2, reqSummaryB)
	var respSummaryB2 response.SuccessResponse
	json.Unmarshal(rrSummaryB2.Body.Bytes(), &respSummaryB2)
	dataSummaryB2 := respSummaryB2.Data.(map[string]interface{})
	if int64(dataSummaryB2["total_expense"].(float64)) != 5000 {
		t.Errorf("expected total_expense after delete to be 5000, got %v", dataSummaryB2["total_expense"])
	}
}
