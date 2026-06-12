package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"golang.org/x/text/encoding/simplifiedchinese"

	"ledger_two/internal/db/repo"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/service"
	"ledger_two/internal/transaction"
)

// constructMultipartRequest 构造用于测试上传 CSV 文件的 Multipart Form 请求
func constructMultipartRequest(t *testing.T, fieldName, fileName string, fileContent []byte) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		t.Fatalf("write file content failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	req, err := http.NewRequest("POST", "/api/transactions/import/parse", body)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestCSVImportParse(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret-import"

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
		r.Post("/api/transactions/import/parse", txHandler.HandleParseCSV)
	})

	// 初始化系统并注入用户
	setupPayload := map[string]string{
		"ledger_name":         "Import Test Ledger",
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

	t.Run("UTF-8 CSV Parse Success", func(t *testing.T) {
		csvContent := `交易时间,交易类型,交易对方,商品,金额(元),备注
2026-06-12 10:00:00,支出,星巴克,拿铁咖啡,32.00,下午茶
2026-06-12 11:30:00,支出,滴滴出行,打车,45.50,商务出行
`
		req := constructMultipartRequest(t, "file", "statement.csv", []byte(csvContent))
		req.AddCookie(cookieA)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				Headers []string   `json:"headers"`
				Rows    [][]string `json:"rows"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal response failed: %v", err)
		}

		if !res.Success {
			t.Fatalf("expected success to be true")
		}

		if len(res.Data.Headers) != 6 {
			t.Errorf("expected 6 headers, got %d", len(res.Data.Headers))
		}
		if res.Data.Headers[2] != "交易对方" {
			t.Errorf("expected header[2] to be '交易对方', got '%s'", res.Data.Headers[2])
		}

		if len(res.Data.Rows) != 2 {
			t.Errorf("expected 2 rows, got %d", len(res.Data.Rows))
		}
		if res.Data.Rows[0][3] != "拿铁咖啡" {
			t.Errorf("expected row[0][3] to be '拿铁咖啡', got '%s'", res.Data.Rows[0][3])
		}
	})

	t.Run("GBK CSV Parse Success", func(t *testing.T) {
		utf8Content := `交易时间,商品,金额
2026-06-12 10:00:00,星巴克咖啡,28.50
`
		// 将 UTF-8 内容编码为 GBK 字节数组，模拟支付宝/微信中文导出
		gbkContent, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(utf8Content))
		if err != nil {
			t.Fatalf("encode GBK failed: %v", err)
		}

		req := constructMultipartRequest(t, "file", "alipay_gbk.csv", gbkContent)
		req.AddCookie(cookieA)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				Headers []string   `json:"headers"`
				Rows    [][]string `json:"rows"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal response failed: %v", err)
		}

		if len(res.Data.Headers) != 3 || res.Data.Headers[1] != "商品" {
			t.Errorf("expected headers to be [交易时间, 商品, 金额], got %v", res.Data.Headers)
		}
		if len(res.Data.Rows) != 1 || res.Data.Rows[0][1] != "星巴克咖啡" {
			t.Errorf("expected data row to contain '星巴克咖啡', got %v", res.Data.Rows)
		}
	})

	t.Run("Invalid File Type Error (Non-CSV)", func(t *testing.T) {
		req := constructMultipartRequest(t, "file", "statement.txt", []byte("some normal text data"))
		req.AddCookie(cookieA)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}

		var res struct {
			Success bool `json:"success"`
			Error   struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(rr.Body.Bytes(), &res)
		if res.Success || res.Error.Code != "IMPORT_FILE_INVALID" {
			t.Errorf("expected IMPORT_FILE_INVALID error code, got %s", res.Error.Code)
		}
	})

	t.Run("WeChat Alipay Format with Intro and Summary", func(t *testing.T) {
		// 模拟微信账单头部带描述干扰，以及尾部带汇总行
		mixedContent := `微信支付账单明细
微信支付（中国）网络技术有限公司 电子账单明细
----------------------------------------
交易时间,交易类型,交易对方,商品,金额(元)
2026-06-12 10:00:00,支出,美团外卖,黄焖鸡米饭,25.00
2026-06-12 14:00:00,支出,瑞幸咖啡,生椰拿铁,16.00
----------------------------------------
生成时间: 2026-06-12
数据条数: 2条
`
		req := constructMultipartRequest(t, "file", "wechat.csv", []byte(mixedContent))
		req.AddCookie(cookieA)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				Headers []string   `json:"headers"`
				Rows    [][]string `json:"rows"`
			} `json:"data"`
		}
		json.Unmarshal(rr.Body.Bytes(), &res)

		// 检查表头定位与数据行数，是否排除了前三行干扰，以及结尾干扰
		if len(res.Data.Headers) != 5 || res.Data.Headers[3] != "商品" {
			t.Errorf("expected 5 headers, got %v", res.Data.Headers)
		}
		if len(res.Data.Rows) != 2 {
			t.Errorf("expected 2 data rows, got %d rows: %v", len(res.Data.Rows), res.Data.Rows)
		}
		if res.Data.Rows[0][3] != "黄焖鸡米饭" || res.Data.Rows[1][3] != "生椰拿铁" {
			t.Errorf("data content parsed mismatch: %v", res.Data.Rows)
		}
	})
}

func TestCSVImportDeduplicationAndCommit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret-commit-import"

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
		r.Post("/api/transactions/import/analyze", txHandler.HandleAnalyzeImport)
		r.Post("/api/transactions/import/commit", txHandler.HandleCommitImport)
	})

	// 初始化系统并注入用户
	setupPayload := map[string]string{
		"ledger_name":         "Import Commit Test Ledger",
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

	// 获取用户 A 和 B 的 ID
	var userAID, userBID string
	db.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&userAID)
	db.QueryRow("SELECT id FROM users WHERE username = 'userB'").Scan(&userBID)

	t.Run("Analyze and Commit Import Success with Deduplication", func(t *testing.T) {
		// 1. 模拟第一次提交导入：包含两笔交易
		item1 := transaction.ImportItemRequest{
			OccurredAt:  "2026-06-12T10:00:00Z",
			AmountCents: 3200,
			Title:       "拿铁咖啡",
			Merchant:    "星巴克",
			CategoryID:  categoryID,
			PayerUserID: userAID,
			Type:        "expense",
			TagNames:    []string{"coffee", "drink"},
			Note:        "下午茶",
		}

		item2 := transaction.ImportItemRequest{
			OccurredAt:  "2026-06-12T11:30:00Z",
			AmountCents: 4550,
			Title:       "打车",
			Merchant:    "滴滴出行",
			CategoryID:  categoryID,
			PayerUserID: userAID,
			Type:        "shared_expense", // 共同支出
			TagNames:    []string{"taxi"},
			Note:        "商务出行",
		}

		commitPayload1 := transaction.CommitImportRequest{
			Filename: "statement_1.csv",
			Items:    []transaction.ImportItemRequest{item1, item2},
		}

		bodyCommit1, _ := json.Marshal(commitPayload1)
		reqCommit1, _ := http.NewRequest("POST", "/api/transactions/import/commit", bytes.NewBuffer(bodyCommit1))
		reqCommit1.AddCookie(cookieA)
		rrCommit1 := httptest.NewRecorder()
		r.ServeHTTP(rrCommit1, reqCommit1)

		if rrCommit1.Code != http.StatusOK {
			t.Fatalf("expected commit 1 to succeed, got %d. Body: %s", rrCommit1.Code, rrCommit1.Body.String())
		}

		// 验证第一次导入后 transactions、splits 和 tags 的写入情况
		var txCount int
		db.QueryRow("SELECT COUNT(*) FROM transactions WHERE status = 'normal'").Scan(&txCount)
		if txCount != 2 {
			t.Errorf("expected 2 transactions in db, got %d", txCount)
		}

		var splitCount int
		db.QueryRow("SELECT COUNT(*) FROM transaction_splits").Scan(&splitCount)
		if splitCount != 2 { // 共同支出 1 笔产生 2 条分摊
			t.Errorf("expected 2 splits in db, got %d", splitCount)
		}

		// 验证 audit_logs 中记录了 import 批次操作
		var auditCount int
		db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action = 'import'").Scan(&auditCount)
		if auditCount != 1 {
			t.Errorf("expected 1 import audit log, got %d", auditCount)
		}

		// 2. 模拟第二次去重分析：包含一笔重复记录（item1）和一笔新记录（item3）
		item3 := transaction.ImportItemRequest{
			OccurredAt:  "2026-06-12T15:00:00Z",
			AmountCents: 9900,
			Title:       "火锅",
			Merchant:    "海底捞",
			CategoryID:  categoryID,
			PayerUserID: userAID,
			Type:        "shared_expense",
			TagNames:    []string{"food"},
			Note:        "聚餐",
		}

		analyzePayload := transaction.AnalyzeImportRequest{
			Items: []transaction.ImportItemRequest{item1, item3},
		}

		bodyAnalyze, _ := json.Marshal(analyzePayload)
		reqAnalyze, _ := http.NewRequest("POST", "/api/transactions/import/analyze", bytes.NewBuffer(bodyAnalyze))
		reqAnalyze.AddCookie(cookieA)
		rrAnalyze := httptest.NewRecorder()
		r.ServeHTTP(rrAnalyze, reqAnalyze)

		if rrAnalyze.Code != http.StatusOK {
			t.Fatalf("expected analyze to succeed, got %d. Body: %s", rrAnalyze.Code, rrAnalyze.Body.String())
		}

		var analyzeRes struct {
			Success bool                             `json:"success"`
			Data    transaction.AnalyzeImportResponse `json:"data"`
		}
		json.Unmarshal(rrAnalyze.Body.Bytes(), &analyzeRes)
		if !analyzeRes.Success {
			t.Fatalf("expected analyze response success to be true")
		}
		if analyzeRes.Data.TotalCount != 2 {
			t.Errorf("expected analyze total 2, got %d", analyzeRes.Data.TotalCount)
		}
		if analyzeRes.Data.SkipCount != 1 {
			t.Errorf("expected analyze skip 1, got %d", analyzeRes.Data.SkipCount)
		}
		if analyzeRes.Data.ImportCount != 1 {
			t.Errorf("expected analyze import 1, got %d", analyzeRes.Data.ImportCount)
		}

		// 3. 模拟第二次确认提交导入：包含一笔重复记录（item1）和一笔新记录（item3）
		commitPayload2 := transaction.CommitImportRequest{
			Filename: "statement_2.csv",
			Items:    []transaction.ImportItemRequest{item1, item3},
		}

		bodyCommit2, _ := json.Marshal(commitPayload2)
		reqCommit2, _ := http.NewRequest("POST", "/api/transactions/import/commit", bytes.NewBuffer(bodyCommit2))
		reqCommit2.AddCookie(cookieA)
		rrCommit2 := httptest.NewRecorder()
		r.ServeHTTP(rrCommit2, reqCommit2)

		if rrCommit2.Code != http.StatusOK {
			t.Fatalf("expected commit 2 to succeed, got %d. Body: %s", rrCommit2.Code, rrCommit2.Body.String())
		}

		// 检查第二次提交后，transactions 应该只有 3 笔（新加入了海底捞，星巴克去重跳过）
		db.QueryRow("SELECT COUNT(*) FROM transactions WHERE status = 'normal'").Scan(&txCount)
		if txCount != 3 {
			t.Errorf("expected 3 transactions in db after deduplication commit, got %d", txCount)
		}

		// 验证 import_items 的状态标记是否正确（有 1 个 skipped，3 个 imported）
		var importedCount, skippedCount int
		db.QueryRow("SELECT COUNT(*) FROM import_items WHERE status = 'imported'").Scan(&importedCount)
		db.QueryRow("SELECT COUNT(*) FROM import_items WHERE status = 'skipped'").Scan(&skippedCount)
		if importedCount != 3 {
			t.Errorf("expected 3 imported items, got %d", importedCount)
		}
		if skippedCount != 1 {
			t.Errorf("expected 1 skipped items, got %d", skippedCount)
		}
	})

	t.Run("Transaction Rollback on Validation Error", func(t *testing.T) {
		// 记录开始前的数据行数
		var startTxCount int
		db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&startTxCount)

		// 构造一个会触发错误的导入项（使用不存在的 category_id 或是无效的 payer_user_id）
		badItem := transaction.ImportItemRequest{
			OccurredAt:  "2026-06-12T16:00:00Z",
			AmountCents: 1500,
			Title:       "错误分类账单",
			CategoryID:  "non-existent-category-id", // 错误的分类ID，在 Commit 逻辑中虽然没有级联校验，但若是其它报错或者无效的 payer_user_id...
			PayerUserID: "invalid-user-id",          // 无效付款人 ID 会触发验证错误并返回 400
			Type:        "expense",
		}

		goodItem := transaction.ImportItemRequest{
			OccurredAt:  "2026-06-12T17:00:00Z",
			AmountCents: 5000,
			Title:       "未入库的好账单",
			CategoryID:  categoryID,
			PayerUserID: userAID,
			Type:        "expense",
		}

		commitPayload := transaction.CommitImportRequest{
			Filename: "rollback_test.csv",
			Items:    []transaction.ImportItemRequest{goodItem, badItem}, // goodItem在前，badItem在后，验证badItem出错时goodItem会不会被回滚
		}

		bodyCommit, _ := json.Marshal(commitPayload)
		reqCommit, _ := http.NewRequest("POST", "/api/transactions/import/commit", bytes.NewBuffer(bodyCommit))
		reqCommit.AddCookie(cookieA)
		rrCommit := httptest.NewRecorder()
		r.ServeHTTP(rrCommit, reqCommit)

		if rrCommit.Code != http.StatusBadRequest {
			t.Errorf("expected bad request (400) for rollback test, got %d. Body: %s", rrCommit.Code, rrCommit.Body.String())
		}

		// 检查 transactions 行数，应该完全没有发生变化，代表 goodItem 完全被回滚了
		var endTxCount int
		db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&endTxCount)
		if endTxCount != startTxCount {
			t.Errorf("expected transaction count to remain %d, got %d. Atomic rollback failed!", startTxCount, endTxCount)
		}

		// 检查 import_batches 也没有生成脏批次
		var batchCount int
		db.QueryRow("SELECT COUNT(*) FROM import_batches WHERE filename = 'rollback_test.csv'").Scan(&batchCount)
		if batchCount != 0 {
			t.Errorf("expected 0 import batch records for rollback_test, got %d", batchCount)
		}
	})
}

