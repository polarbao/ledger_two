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
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/service"
	"ledger_two/internal/transaction"
)

func TestRecurringBilling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret-recurring"

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
		r.Route("/api/recurring-rules", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreateRecurringRule)
			r.Get("/", txHandler.HandleListRecurringRules)
			r.Delete("/{id}", txHandler.HandleDeleteRecurringRule)
		})
		r.Route("/api/recurring-reminders", func(r chi.Router) {
			r.Get("/", txHandler.HandleListRecurringReminders)
			r.Post("/{id}/confirm", txHandler.HandleConfirmReminder)
			r.Post("/{id}/ignore", txHandler.HandleIgnoreReminder)
		})
		r.Route("/api/transactions", func(r chi.Router) {
			r.Get("/", txHandler.HandleList)
		})
	})

	// 1. 初始化系统，注入用户
	setupPayload := map[string]string{
		"ledger_name":         "Recurring Test Ledger",
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

	// 2. 测试创建周期规则参数拦截
	// 2.1 名称为空失败
	badPayload1 := map[string]interface{}{
		"name":          "",
		"type":          "expense",
		"frequency":     "monthly",
		"next_due_date": "2026-06-01",
	}
	body1, _ := json.Marshal(badPayload1)
	req1, _ := http.NewRequest("POST", "/api/recurring-rules", bytes.NewBuffer(body1))
	req1.AddCookie(cookieA)
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty rule name, got %d", rr1.Code)
	}

	// 2.2 频率无效失败
	badPayload2 := map[string]interface{}{
		"name":          "房租",
		"type":          "expense",
		"frequency":     "every-two-weeks",
		"next_due_date": "2026-06-01",
	}
	body2, _ := json.Marshal(badPayload2)
	req2, _ := http.NewRequest("POST", "/api/recurring-rules", bytes.NewBuffer(body2))
	req2.AddCookie(cookieA)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid frequency, got %d", rr2.Code)
	}

	// 2.3 成功创建周期规则 (设定首次到期时间为 2026-06-01，即过去的时间，用来测试懒触发生成提醒)
	amountVal := int64(300000) // 3000元
	noteVal := "每月固定交房租房东"
	titleVal := "房租提醒实例"
	okPayload := map[string]interface{}{
		"name":          "每月房租规则",
		"type":          "shared_expense",
		"title":         &titleVal,
		"amount_cents":  &amountVal,
		"payer_user_id": "userA",
		"split_method":  "equal",
		"tag_names":     []string{"住房", "固定支出"},
		"note":          &noteVal,
		"frequency":     "monthly",
		"next_due_date": "2026-06-01",
	}
	bodyOk, _ := json.Marshal(okPayload)
	reqOk, _ := http.NewRequest("POST", "/api/recurring-rules", bytes.NewBuffer(bodyOk))
	reqOk.AddCookie(cookieA)
	rrOk := httptest.NewRecorder()
	r.ServeHTTP(rrOk, reqOk)
	if rrOk.Code != http.StatusCreated {
		t.Fatalf("failed to create valid recurring rule: %v", rrOk.Body.String())
	}

	var createdRule struct {
		Data struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			NextDueDate string `json:"next_due_date"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rrOk.Body.Bytes(), &createdRule)

	// 3. 触发懒扫描：拉取到期提醒列表，今天日期是 2026-06-11
	// 首次到期日是 2026-06-01，它小于 2026-06-11。
	// 检测扫描器会自动插入一条 2026-06-01 的待处理 Reminder。
	// 然后，NextDueDate 推进 1 个月至 2026-07-01。
	// 由于 2026-07-01 > 2026-06-11，扫描停止。
	reqList, _ := http.NewRequest("GET", "/api/recurring-reminders", nil)
	reqList.AddCookie(cookieA)
	rrList := httptest.NewRecorder()
	r.ServeHTTP(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Fatalf("failed to list reminders: %v", rrList.Body.String())
	}

	var listResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(rrList.Body.Bytes(), &listResp)
	reminders := listResp.Data

	if len(reminders) != 1 {
		t.Fatalf("expected 1 reminder triggered, got %d. Response: %s", len(reminders), rrList.Body.String())
	}

	reminder := reminders[0]
	if reminder["due_date"].(string) != "2026-06-01" {
		t.Errorf("expected due date 2026-06-01, got %v", reminder["due_date"])
	}
	if reminder["status"].(string) != "pending" {
		t.Errorf("expected reminder status pending, got %v", reminder["status"])
	}

	// 验证规则下次到期日已经被更新到 2026-07-01
	reqRules, _ := http.NewRequest("GET", "/api/recurring-rules", nil)
	reqRules.AddCookie(cookieA)
	rrRules := httptest.NewRecorder()
	r.ServeHTTP(rrRules, reqRules)
	var rulesResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(rrRules.Body.Bytes(), &rulesResp)
	rulesList := rulesResp.Data
	if len(rulesList) != 1 {
		t.Fatalf("expected 1 rule in list, got %d", len(rulesList))
	}
	retrievedRule := rulesList[0]
	if retrievedRule["next_due_date"].(string) != "2026-07-01" {
		t.Errorf("expected rule next_due_date advanced to 2026-07-01, got %v", retrievedRule["next_due_date"])
	}

	// 4. 确认到期提醒：生成真实交易账单
	reminderID := reminder["id"].(string)
	reqConfirm, _ := http.NewRequest("POST", "/api/recurring-reminders/"+reminderID+"/confirm", nil)
	reqConfirm.AddCookie(cookieA)
	rrConfirm := httptest.NewRecorder()
	r.ServeHTTP(rrConfirm, reqConfirm)
	if rrConfirm.Code != http.StatusOK {
		t.Fatalf("failed to confirm reminder: %v", rrConfirm.Body.String())
	}

	// 5. 校验真实流水是否增加，且数据一致
	reqTx, _ := http.NewRequest("GET", "/api/transactions", nil)
	reqTx.AddCookie(cookieA)
	rrTx := httptest.NewRecorder()
	r.ServeHTTP(rrTx, reqTx)
	var txResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(rrTx.Body.Bytes(), &txResp)
	transactions := txResp.Data
	if len(transactions) != 1 {
		t.Fatalf("expected 1 transaction created by confirmation, got %d", len(transactions))
	}

	createdTx := transactions[0]
	if createdTx["type"].(string) != "shared_expense" {
		t.Errorf("expected transaction type shared_expense, got %v", createdTx["type"])
	}
	// 校验金额 (cents) 是否对齐
	if int64(createdTx["amount_cents"].(float64)) != 300000 {
		t.Errorf("expected transaction amount_cents 300000, got %v", createdTx["amount_cents"])
	}
	// 校验 occurred_at 应该与 reminder.due_date 相同，即 2026-06-01 左右
	occurredAtStr := createdTx["occurred_at"].(string)
	if occurredAtStr[:10] != "2026-06-01" {
		t.Errorf("expected transaction occurred_at 2026-06-01, got %s", occurredAtStr)
	}

	// 6. 测试删除规则
	reqDel, _ := http.NewRequest("DELETE", "/api/recurring-rules/"+createdRule.Data.ID, nil)
	reqDel.AddCookie(cookieA)
	rrDel := httptest.NewRecorder()
	r.ServeHTTP(rrDel, reqDel)
	if rrDel.Code != http.StatusOK {
		t.Errorf("failed to delete rule: %v", rrDel.Body.String())
	}
}
