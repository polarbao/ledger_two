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
	"ledger_two/internal/http/response"
	"ledger_two/internal/service"
	"ledger_two/internal/settlement"
	"ledger_two/internal/transaction"
)

func TestDashboardFlow(t *testing.T) {
	db := setupTestDB(t)
	db.SetMaxOpenConns(1)
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

	dashRepo := dashboard.NewRepository(db)
	dashSvc := dashboard.NewService(dashRepo, settleSvc)
	dashHandler := dashboard.NewHandler(dashSvc)

	r := chi.NewRouter()
	r.Post("/api/init/setup", initHandler.HandleSetup)
	r.Post("/api/auth/login", authHandler.HandleLogin)

	r.Group(func(r chi.Router) {
		r.Use(testAuthenticatedLedgerContext(db, jwtSecret))
		r.Route("/api/transactions", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreate)
			r.Delete("/{id}", txHandler.HandleDelete)
		})
		r.Route("/api/shared-expenses", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreateSharedExpense)
		})
		r.Get("/api/dashboard", dashHandler.HandleGetDashboard)
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

	var cat1, cat2 string
	rows, err := db.Query("SELECT id FROM categories LIMIT 2")
	if err != nil {
		t.Fatalf("query categories failed: %v", err)
	}
	if rows.Next() {
		rows.Scan(&cat1)
	}
	if rows.Next() {
		rows.Scan(&cat2)
	}
	rows.Close()

	currentMonth := time.Now().Format("2006-01")

	t.Run("empty dashboard serializes collections as arrays", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/dashboard?month="+currentMonth, nil)
		req.AddCookie(cookieA)
		setTestLedgerHeader(t, db, req, "Test Ledger")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("empty dashboard failed: %s", rr.Body.String())
		}
		var body struct {
			Data struct {
				RecentTransactions []json.RawMessage `json:"recent_transactions"`
				CategorySummary    []json.RawMessage `json:"category_summary"`
				TagSummary         []json.RawMessage `json:"tag_summary"`
				UserStats          []json.RawMessage `json:"user_stats"`
				SharedBalance      struct {
					SuggestedTransfers []json.RawMessage `json:"suggested_transfers"`
				} `json:"shared_balance"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode empty dashboard: %v", err)
		}
		if body.Data.RecentTransactions == nil {
			t.Fatalf("expected recent_transactions to be [], got null: %s", rr.Body.String())
		}
		if body.Data.CategorySummary == nil {
			t.Fatalf("expected category_summary to be [], got null: %s", rr.Body.String())
		}
		if body.Data.TagSummary == nil {
			t.Fatalf("expected tag_summary to be [], got null: %s", rr.Body.String())
		}
		if body.Data.UserStats == nil {
			t.Fatalf("expected user_stats to be an array, got null: %s", rr.Body.String())
		}
		if body.Data.SharedBalance.SuggestedTransfers == nil {
			t.Fatalf("expected suggested_transfers to be [], got null: %s", rr.Body.String())
		}
	})

	// ----------------------------------------------------
	// 场景 1: A 录入一笔 150.00元 (15000分) 的个人普通支出 (分类: cat1, 标签: 外卖)
	// ----------------------------------------------------
	payload1 := map[string]interface{}{
		"type":          "expense",
		"title":         "购买外卖午餐",
		"amount_cents":  int64(15000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   cat1,
		"visibility":    "private",
		"tag_names":     []string{"外卖"},
	}
	body1, _ := json.Marshal(payload1)
	req1, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(body1))
	req1.AddCookie(cookieA)
	setTestLedgerHeader(t, db, req1, "Test Ledger")
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("create expense 1 failed: %v", rr1.Body.String())
	}

	// ----------------------------------------------------
	// 场景 2: A 录入一笔 5000.00元 (500000分) 的个人普通收入 (分类: cat2)
	// ----------------------------------------------------
	payload2 := map[string]interface{}{
		"type":          "income",
		"title":         "发放月度奖金",
		"amount_cents":  int64(500000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   cat2,
		"visibility":    "private",
	}
	body2, _ := json.Marshal(payload2)
	req2, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(body2))
	req2.AddCookie(cookieA)
	setTestLedgerHeader(t, db, req2, "Test Ledger")
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("create income failed: %v", rr2.Body.String())
	}

	// ----------------------------------------------------
	// 场景 3: B 录入一笔 100.00元 (10000分) 的共同支出 (分类: cat2, 标签: 超市, 平摊 equal)
	// ----------------------------------------------------
	payload3 := map[string]interface{}{
		"title":         "超市日用品采购",
		"amount_cents":  int64(10000),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userBID,
		"category_id":   cat2,
		"split_method":  "equal",
		"tag_names":     []string{"超市"},
	}
	body3, _ := json.Marshal(payload3)
	req3, _ := http.NewRequest("POST", "/api/shared-expenses", bytes.NewBuffer(body3))
	req3.AddCookie(cookieB)
	setTestLedgerHeader(t, db, req3, "Test Ledger")
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusCreated {
		t.Fatalf("create shared expense failed: %v", rr3.Body.String())
	}

	var resp3 response.SuccessResponse
	json.Unmarshal(rr3.Body.Bytes(), &resp3)
	txData3 := resp3.Data.(map[string]interface{})
	sharedTxID := txData3["id"].(string)

	// ----------------------------------------------------
	// 场景 4: 调用 GET /api/dashboard 验证聚合结果 (从 A 登录的角度)
	// ----------------------------------------------------
	reqDash, _ := http.NewRequest("GET", "/api/dashboard?month="+currentMonth, nil)
	reqDash.AddCookie(cookieA)
	setTestLedgerHeader(t, db, reqDash, "Test Ledger")
	rrDash := httptest.NewRecorder()
	r.ServeHTTP(rrDash, reqDash)
	if rrDash.Code != http.StatusOK {
		t.Fatalf("get dashboard failed: %v", rrDash.Body.String())
	}

	var dashResp response.SuccessResponse
	json.Unmarshal(rrDash.Body.Bytes(), &dashResp)
	dashData := dashResp.Data.(map[string]interface{})

	// 验证支出与收入
	if int64(dashData["total_expense_cents"].(float64)) != 25000 { // 15000 (A普通) + 10000 (共同)
		t.Errorf("expected total_expense_cents = 25000, got %v", dashData["total_expense_cents"])
	}
	if int64(dashData["total_income_cents"].(float64)) != 500000 {
		t.Errorf("expected total_income_cents = 500000, got %v", dashData["total_income_cents"])
	}
	if int64(dashData["my_paid_cents"].(float64)) != 15000 { // 我 (A) 掏了 15000 垫付
		t.Errorf("expected my_paid_cents = 15000, got %v", dashData["my_paid_cents"])
	}
	if int64(dashData["partner_paid_cents"].(float64)) != 10000 { // 对方 (B) 掏了 10000 垫付
		t.Errorf("expected partner_paid_cents = 10000, got %v", dashData["partner_paid_cents"])
	}

	// 验证成员承担
	userStats := dashData["user_stats"].([]interface{})
	if len(userStats) != 2 {
		t.Fatalf("expected 2 user stats, got %d", len(userStats))
	}
	// 按 display_name 排序：dispA 应该在前面，dispB 在后面
	uStatA := userStats[0].(map[string]interface{})
	uStatB := userStats[1].(map[string]interface{})

	// A: 掏了 15000；承担了 15000 (个人) + 5000 (平摊) = 20000
	if int64(uStatA["paid_cents"].(float64)) != 15000 {
		t.Errorf("expected userA paid = 15000, got %v", uStatA["paid_cents"])
	}
	if int64(uStatA["share_cents"].(float64)) != 20000 {
		t.Errorf("expected userA share = 20000, got %v", uStatA["share_cents"])
	}

	// B: 掏了 10000；承担了 5000 (平摊) = 5000
	if int64(uStatB["paid_cents"].(float64)) != 10000 {
		t.Errorf("expected userB paid = 10000, got %v", uStatB["paid_cents"])
	}
	if int64(uStatB["share_cents"].(float64)) != 5000 {
		t.Errorf("expected userB share = 5000, got %v", uStatB["share_cents"])
	}

	// 验证分类汇总占比与排序
	catSummary := dashData["category_summary"].([]interface{})
	if len(catSummary) != 2 {
		t.Errorf("expected 2 category summary items, got %d", len(catSummary))
	}
	// cat1 消费额是 15000 分，占比 60%，应当列在第一位
	c1 := catSummary[0].(map[string]interface{})
	if c1["id"].(string) != cat1 || int64(c1["amount_cents"].(float64)) != 15000 || c1["percent"].(float64) != 60.0 {
		t.Errorf("cat1 stats error: %+v", c1)
	}

	// 验证标签统计与排序
	tagSummary := dashData["tag_summary"].([]interface{})
	if len(tagSummary) != 2 {
		t.Errorf("expected 2 tag summary items, got %d", len(tagSummary))
	}
	// "外卖" 标签额 15000 分，占比 60%，应当列在第一位
	t1 := tagSummary[0].(map[string]interface{})
	if t1["name"].(string) != "外卖" || int64(t1["amount_cents"].(float64)) != 15000 || t1["percent"].(float64) != 60.0 {
		t.Errorf("tag '外卖' stats error: %+v", t1)
	}

	// 最近流水应包含 3 条记录
	recent := dashData["recent_transactions"].([]interface{})
	if len(recent) != 3 {
		t.Errorf("expected recent transactions count = 3, got %d", len(recent))
	}

	// ----------------------------------------------------
	// 场景 5: 软删除 B 的共同支出账单，再次请求，验证软删除隔离
	// ----------------------------------------------------
	reqDel, _ := http.NewRequest("DELETE", "/api/transactions/"+sharedTxID, nil)
	reqDel.AddCookie(cookieB)
	setTestLedgerHeader(t, db, reqDel, "Test Ledger")
	rrDel := httptest.NewRecorder()
	r.ServeHTTP(rrDel, reqDel)
	if rrDel.Code != http.StatusOK {
		t.Fatalf("delete shared expense failed: %v", rrDel.Body.String())
	}

	rrDash2 := httptest.NewRecorder()
	r.ServeHTTP(rrDash2, reqDash)
	if rrDash2.Code != http.StatusOK {
		t.Fatalf("get dashboard after delete failed: %v", rrDash2.Body.String())
	}

	var dashResp2 response.SuccessResponse
	json.Unmarshal(rrDash2.Body.Bytes(), &dashResp2)
	dashData2 := dashResp2.Data.(map[string]interface{})

	// 软删除后，总支出应降为 15000，且只剩 1 个分类和 1 个标签统计
	if int64(dashData2["total_expense_cents"].(float64)) != 15000 {
		t.Errorf("expected total_expense_cents = 15000, got %v", dashData2["total_expense_cents"])
	}
	catSummary2 := dashData2["category_summary"].([]interface{})
	if len(catSummary2) != 1 {
		t.Errorf("expected 1 category summary item, got %d", len(catSummary2))
	}
	tagSummary2 := dashData2["tag_summary"].([]interface{})
	if len(tagSummary2) != 1 {
		t.Errorf("expected 1 tag summary item, got %d", len(tagSummary2))
	}

	// 成员统计中，B的支付与承担均归为 0
	userStats2 := dashData2["user_stats"].([]interface{})
	uStatB2 := userStats2[1].(map[string]interface{})
	if int64(uStatB2["paid_cents"].(float64)) != 0 || int64(uStatB2["share_cents"].(float64)) != 0 {
		t.Errorf("expected userB paid/share to be 0 after deleting shared expense, got paid: %v, share: %v", uStatB2["paid_cents"], uStatB2["share_cents"])
	}
}
