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
	"ledger_two/internal/service"
	"ledger_two/internal/transaction"
)

// TestTransactionTemplates CRUD 核心业务逻辑及参数边界防御性测试
func TestTransactionTemplates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret-templates"

	// 初始化各模块
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
		r.Route("/api/transaction-templates", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreateTemplate)
			r.Get("/", txHandler.HandleListTemplates)
			r.Get("/{id}", txHandler.HandleGetTemplate)
			r.Put("/{id}", txHandler.HandleUpdateTemplate)
			r.Post("/{id}/archive", txHandler.HandleArchiveTemplate)
			r.Post("/{id}/restore", txHandler.HandleRestoreTemplate)
			r.Delete("/{id}", txHandler.HandleDeleteTemplate)
		})
		r.Route("/api/transactions", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreate)
		})
	})

	// 1. 初始化系统，注入 userA 和 userB
	setupPayload := map[string]string{
		"ledger_name":         "Template Test Ledger",
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
	var userAID string
	if err := db.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&userAID); err != nil {
		t.Fatalf("query userA id failed: %v", err)
	}

	// 2. 测试创建模板：参数边界防错
	// 2.1 模板名为空失败
	badPayload1 := map[string]interface{}{
		"name": "",
		"type": "expense",
	}
	body1, _ := json.Marshal(badPayload1)
	req1, _ := http.NewRequest("POST", "/api/transaction-templates", bytes.NewBuffer(body1))
	req1.AddCookie(cookieA)
	setTestLedgerHeader(t, db, req1, "Template Test Ledger")
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty template name, got %d", rr1.Code)
	}

	// 2.2 类型无效失败
	badPayload2 := map[string]interface{}{
		"name": "日常午餐",
		"type": "transfer", // 无效类型
	}
	body2, _ := json.Marshal(badPayload2)
	req2, _ := http.NewRequest("POST", "/api/transaction-templates", bytes.NewBuffer(body2))
	req2.AddCookie(cookieA)
	setTestLedgerHeader(t, db, req2, "Template Test Ledger")
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid template type, got %d", rr2.Code)
	}

	// 2.3 金额小于 0 失败
	negAmount := int64(-500)
	badPayload3 := map[string]interface{}{
		"name":         "日常午餐",
		"type":         "expense",
		"amount_cents": &negAmount,
	}
	body3, _ := json.Marshal(badPayload3)
	req3, _ := http.NewRequest("POST", "/api/transaction-templates", bytes.NewBuffer(body3))
	req3.AddCookie(cookieA)
	setTestLedgerHeader(t, db, req3, "Template Test Ledger")
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative amount, got %d", rr3.Code)
	}

	// 2.4 正确创建模板
	validAmount := int64(1500) // 15元
	noteVal := "每周固定吃黄焖鸡"
	titleVal := "吃黄焖鸡米饭"
	okPayload := map[string]interface{}{
		"name":         "美味午餐模板",
		"type":         "expense",
		"title":        &titleVal,
		"amount_cents": &validAmount,
		"tag_names":    []string{"工作餐", "餐饮"},
		"note":         &noteVal,
	}
	bodyOk, _ := json.Marshal(okPayload)
	reqOk, _ := http.NewRequest("POST", "/api/transaction-templates", bytes.NewBuffer(bodyOk))
	reqOk.AddCookie(cookieA)
	setTestLedgerHeader(t, db, reqOk, "Template Test Ledger")
	rrOk := httptest.NewRecorder()
	r.ServeHTTP(rrOk, reqOk)
	if rrOk.Code != http.StatusCreated {
		t.Fatalf("failed to create valid template: %v", rrOk.Body.String())
	}

	var createdTmpl struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rrOk.Body.Bytes(), &createdTmpl)

	// 3. 测试获取模板列表
	reqList, _ := http.NewRequest("GET", "/api/transaction-templates", nil)
	reqList.AddCookie(cookieA)
	setTestLedgerHeader(t, db, reqList, "Template Test Ledger")
	rrList := httptest.NewRecorder()
	r.ServeHTTP(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Errorf("failed to list templates: %v", rrList.Body.String())
	}

	var listResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(rrList.Body.Bytes(), &listResp)
	list := listResp.Data
	if len(list) != 1 {
		t.Errorf("expected 1 template in list, got %d. Response: %s", len(list), rrList.Body.String())
	}

	// 4. 测试更新模板
	updatedName := "美味午餐模板修改版"
	updatePayload := map[string]interface{}{
		"name":         updatedName,
		"type":         "expense",
		"title":        &titleVal,
		"amount_cents": &validAmount,
		"tag_names":    []string{"工作餐", "修改版"},
		"note":         &noteVal,
	}
	bodyUpdate, _ := json.Marshal(updatePayload)
	reqUpdate, _ := http.NewRequest("PUT", "/api/transaction-templates/"+createdTmpl.Data.ID, bytes.NewBuffer(bodyUpdate))
	reqUpdate.AddCookie(cookieA)
	setTestLedgerHeader(t, db, reqUpdate, "Template Test Ledger")
	rrUpdate := httptest.NewRecorder()
	r.ServeHTTP(rrUpdate, reqUpdate)
	if rrUpdate.Code != http.StatusOK {
		t.Errorf("failed to update template: %v", rrUpdate.Body.String())
	}

	// 5. 校验从模板回填的数据可以正确写入普通支出交易
	txPayload := map[string]interface{}{
		"type":          "expense",
		"title":         "吃黄焖鸡米饭",
		"amount_cents":  1500,
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"tag_names":     []string{"工作餐", "修改版"},
		"note":          "每周固定吃黄焖鸡",
	}
	bodyTx, _ := json.Marshal(txPayload)
	reqTx, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyTx))
	reqTx.AddCookie(cookieA)
	setTestLedgerHeader(t, db, reqTx, "Template Test Ledger")
	rrTx := httptest.NewRecorder()
	r.ServeHTTP(rrTx, reqTx)
	if rrTx.Code != http.StatusCreated {
		t.Fatalf("failed to create transaction from template: %v", rrTx.Body.String())
	}

	// 6. 测试 DELETE 兼容旧调用，但实际执行软归档
	reqDel, _ := http.NewRequest("DELETE", "/api/transaction-templates/"+createdTmpl.Data.ID, nil)
	reqDel.AddCookie(cookieA)
	setTestLedgerHeader(t, db, reqDel, "Template Test Ledger")
	rrDel := httptest.NewRecorder()
	r.ServeHTTP(rrDel, reqDel)
	if rrDel.Code != http.StatusOK {
		t.Errorf("failed to archive template by delete endpoint: %v", rrDel.Body.String())
	}

	// 默认列表不返回已归档模板
	reqList2, _ := http.NewRequest("GET", "/api/transaction-templates", nil)
	reqList2.AddCookie(cookieA)
	setTestLedgerHeader(t, db, reqList2, "Template Test Ledger")
	rrList2 := httptest.NewRecorder()
	r.ServeHTTP(rrList2, reqList2)
	var listResp2 struct {
		Data []map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(rrList2.Body.Bytes(), &listResp2)
	list2 := listResp2.Data
	if len(list2) != 0 {
		t.Errorf("expected 0 templates after deletion, got %d", len(list2))
	}

	// 管理列表可以显式包含已归档模板
	reqListArchived, _ := http.NewRequest("GET", "/api/transaction-templates?include_archived=true", nil)
	reqListArchived.AddCookie(cookieA)
	setTestLedgerHeader(t, db, reqListArchived, "Template Test Ledger")
	rrListArchived := httptest.NewRecorder()
	r.ServeHTTP(rrListArchived, reqListArchived)
	var archivedResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(rrListArchived.Body.Bytes(), &archivedResp)
	if len(archivedResp.Data) != 1 || archivedResp.Data[0]["is_archived"] != true {
		t.Fatalf("expected archived template in include_archived list, got %s", rrListArchived.Body.String())
	}

	// 恢复后默认列表重新可见
	reqRestore, _ := http.NewRequest("POST", "/api/transaction-templates/"+createdTmpl.Data.ID+"/restore", nil)
	reqRestore.AddCookie(cookieA)
	setTestLedgerHeader(t, db, reqRestore, "Template Test Ledger")
	rrRestore := httptest.NewRecorder()
	r.ServeHTTP(rrRestore, reqRestore)
	if rrRestore.Code != http.StatusOK {
		t.Fatalf("failed to restore template: %s", rrRestore.Body.String())
	}

	reqList3, _ := http.NewRequest("GET", "/api/transaction-templates", nil)
	reqList3.AddCookie(cookieA)
	setTestLedgerHeader(t, db, reqList3, "Template Test Ledger")
	rrList3 := httptest.NewRecorder()
	r.ServeHTTP(rrList3, reqList3)
	var listResp3 struct {
		Data []map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(rrList3.Body.Bytes(), &listResp3)
	if len(listResp3.Data) != 1 || listResp3.Data[0]["is_archived"] != false {
		t.Errorf("expected restored template in default list, got %s", rrList3.Body.String())
	}
}
