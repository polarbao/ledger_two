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
