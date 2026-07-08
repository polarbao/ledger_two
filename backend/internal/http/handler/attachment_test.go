package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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

func TestAttachmentUploadAndValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret"

	// 初始化 handler 与 router 依赖
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
		r.Post("/api/attachments", txHandler.HandleUploadAttachment)
		r.Route("/api/transactions", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreate)
			r.Patch("/{id}", txHandler.HandleUpdate)
		})
	})

	// 1. 初始化系统并获取 Cookie
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
	var userAID string
	if err := db.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&userAID); err != nil {
		t.Fatalf("query userA id failed: %v", err)
	}

	// 2. 测试未鉴权拦截
	reqUnauth, _ := http.NewRequest("POST", "/api/attachments", nil)
	rrUnauth := httptest.NewRecorder()
	r.ServeHTTP(rrUnauth, reqUnauth)
	if rrUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for no cookie, got %d", rrUnauth.Code)
	}

	// 3. 测试正常上传（合法 PNG 魔数头）
	var uploadBuf bytes.Buffer
	mpWriter := multipart.NewWriter(&uploadBuf)
	filePart, err := mpWriter.CreateFormFile("file", "test.png")
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	// 写入 PNG 的特征魔数头部，以使 http.DetectContentType 识别为 image/png
	pngHeader := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	_, _ = filePart.Write(pngHeader)
	_, _ = filePart.Write(make([]byte, 100)) // 附加一些假数据
	mpWriter.Close()

	reqUpload, _ := http.NewRequest("POST", "/api/attachments", &uploadBuf)
	reqUpload.Header.Set("Content-Type", mpWriter.FormDataContentType())
	reqUpload.AddCookie(cookieA)

	rrUpload := httptest.NewRecorder()
	r.ServeHTTP(rrUpload, reqUpload)
	if rrUpload.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid image, got %d. Body: %s", rrUpload.Code, rrUpload.Body.String())
	}

	var uploadResp response.SuccessResponse
	if err := json.Unmarshal(rrUpload.Body.Bytes(), &uploadResp); err != nil {
		t.Fatalf("unmarshal upload response failed: %v", err)
	}
	uploadData := uploadResp.Data.(map[string]interface{})
	filePath := uploadData["path"].(string)
	if filePath == "" {
		t.Errorf("expected non-empty attachment file path, got empty")
	}

	// 4. 测试大小超限拦截（大于 10MB）
	var overLimitBuf bytes.Buffer
	mpWriterOver := multipart.NewWriter(&overLimitBuf)
	filePartOver, _ := mpWriterOver.CreateFormFile("file", "large.png")
	_, _ = filePartOver.Write(pngHeader)
	// 写入超过 10MB 的假文件数据
	largeData := make([]byte, 10*1024*1024+1000)
	_, _ = filePartOver.Write(largeData)
	mpWriterOver.Close()

	reqOver, _ := http.NewRequest("POST", "/api/attachments", &overLimitBuf)
	reqOver.Header.Set("Content-Type", mpWriterOver.FormDataContentType())
	reqOver.AddCookie(cookieA)

	rrOver := httptest.NewRecorder()
	r.ServeHTTP(rrOver, reqOver)
	if rrOver.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for oversized file, got %d", rrOver.Code)
	}

	// 5. 测试 Service 创建交易时的附件校验：
	// 5.1 路径穿越与前缀校验失败 (包含 "..")
	reqPayloadErrPath := map[string]interface{}{
		"type":             "expense",
		"title":            "购买午餐",
		"amount_cents":     int64(1500),
		"currency":         "CNY",
		"occurred_at":      time.Now().Format(time.RFC3339),
		"payer_user_id":    userAID,
		"visibility":       "private",
		"attachment_paths": []string{"/uploads/../hack.png"},
	}
	bodyErrPath, _ := json.Marshal(reqPayloadErrPath)
	reqCreateErrPath, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyErrPath))
	reqCreateErrPath.AddCookie(cookieA)
	rrCreateErrPath := httptest.NewRecorder()
	r.ServeHTTP(rrCreateErrPath, reqCreateErrPath)
	if rrCreateErrPath.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for path traversal, got %d", rrCreateErrPath.Code)
	}

	// 5.2 数量超过 5 个校验失败
	reqPayloadErrCount := map[string]interface{}{
		"type":          "expense",
		"title":         "购买午餐",
		"amount_cents":  int64(1500),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"visibility":    "private",
		"attachment_paths": []string{
			"/uploads/1.png",
			"/uploads/2.png",
			"/uploads/3.png",
			"/uploads/4.png",
			"/uploads/5.png",
			"/uploads/6.png",
		},
	}
	bodyErrCount, _ := json.Marshal(reqPayloadErrCount)
	reqCreateErrCount, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyErrCount))
	reqCreateErrCount.AddCookie(cookieA)
	rrCreateErrCount := httptest.NewRecorder()
	r.ServeHTTP(rrCreateErrCount, reqCreateErrCount)
	if rrCreateErrCount.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for >5 attachments, got %d", rrCreateErrCount.Code)
	}

	// 5.3 正常附件路径写入与查询
	reqPayloadOk := map[string]interface{}{
		"type":             "expense",
		"title":            "购买午餐",
		"amount_cents":     int64(1500),
		"currency":         "CNY",
		"occurred_at":      time.Now().Format(time.RFC3339),
		"payer_user_id":    userAID,
		"visibility":       "private",
		"attachment_paths": []string{filePath},
	}
	bodyOk, _ := json.Marshal(reqPayloadOk)
	reqCreateOk, _ := http.NewRequest("POST", "/api/transactions", bytes.NewBuffer(bodyOk))
	reqCreateOk.AddCookie(cookieA)
	rrCreateOk := httptest.NewRecorder()
	r.ServeHTTP(rrCreateOk, reqCreateOk)
	if rrCreateOk.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for valid transaction, got %d. Body: %s", rrCreateOk.Code, rrCreateOk.Body.String())
	}

	var createResp response.SuccessResponse
	json.Unmarshal(rrCreateOk.Body.Bytes(), &createResp)
	txData := createResp.Data.(map[string]interface{})
	txID := txData["id"].(string)

	// 验证返回的 attachment_paths 包含刚才写入的附件
	paths := txData["attachment_paths"].([]interface{})
	if len(paths) != 1 || paths[0].(string) != filePath {
		t.Errorf("expected attachment paths to contain %s, got %+v", filePath, paths)
	}

	// 6. 测试更新交易时的附件校验：
	// 6.1 数量超过 5 个校验失败
	reqPayloadUpdateErr := map[string]interface{}{
		"attachment_paths": []string{
			"/uploads/1.png",
			"/uploads/2.png",
			"/uploads/3.png",
			"/uploads/4.png",
			"/uploads/5.png",
			"/uploads/6.png",
		},
	}
	bodyUpdateErr, _ := json.Marshal(reqPayloadUpdateErr)
	reqUpdateErr, _ := http.NewRequest("PATCH", "/api/transactions/"+txID, bytes.NewBuffer(bodyUpdateErr))
	reqUpdateErr.AddCookie(cookieA)
	rrUpdateErr := httptest.NewRecorder()
	r.ServeHTTP(rrUpdateErr, reqUpdateErr)
	if rrUpdateErr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for update with >5 attachments, got %d", rrUpdateErr.Code)
	}

	// 6.2 正常更新附件
	newFilePath := "/uploads/new_file.jpg"
	reqPayloadUpdateOk := map[string]interface{}{
		"attachment_paths": []string{newFilePath},
	}
	bodyUpdateOk, _ := json.Marshal(reqPayloadUpdateOk)
	reqUpdateOk, _ := http.NewRequest("PATCH", "/api/transactions/"+txID, bytes.NewBuffer(bodyUpdateOk))
	reqUpdateOk.AddCookie(cookieA)
	rrUpdateOk := httptest.NewRecorder()
	r.ServeHTTP(rrUpdateOk, reqUpdateOk)
	if rrUpdateOk.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid transaction update, got %d. Body: %s", rrUpdateOk.Code, rrUpdateOk.Body.String())
	}

	var updateResp response.SuccessResponse
	json.Unmarshal(rrUpdateOk.Body.Bytes(), &updateResp)
	updateData := updateResp.Data.(map[string]interface{})
	updatePaths := updateData["attachment_paths"].([]interface{})
	if len(updatePaths) != 1 || updatePaths[0].(string) != newFilePath {
		t.Errorf("expected updated attachment paths to contain %s, got %+v", newFilePath, updatePaths)
	}
}
