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
)

func TestAuthFlow(t *testing.T) {
	// 复用 init_test.go 中的 setupTestDB
	db := setupTestDB(t)
	defer db.Close()

	jwtSecret := "test-secret"

	initRepo := repo.NewInitRepo(db)
	initSvc := service.NewInitService(initRepo)
	initHandler := handler.NewInitHandler(initSvc)

	authRepo := repo.NewAuthRepo(db)
	authSvc := service.NewAuthService(authRepo, jwtSecret)
	authHandler := handler.NewAuthHandler(authSvc)

	r := chi.NewRouter()
	r.Post("/api/init/setup", initHandler.HandleSetup)
	r.Post("/api/auth/login", authHandler.HandleLogin)
	r.Post("/api/auth/logout", authHandler.HandleLogout)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtSecret))
		r.Get("/api/auth/me", authHandler.HandleMe)
	})

	// 1. 初始化系统数据
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

	// 2. 测试拦截：无凭证访问受保护的 /me
	reqMeFail, _ := http.NewRequest("GET", "/api/auth/me", nil)
	rrMeFail := httptest.NewRecorder()
	r.ServeHTTP(rrMeFail, reqMeFail)
	if rrMeFail.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %v", rrMeFail.Code)
	}

	// 3. 测试错误凭证登录：密码错误
	loginFailPayload := map[string]string{
		"username": "userA",
		"password": "wrong_password",
	}
	body2, _ := json.Marshal(loginFailPayload)
	reqLoginFail, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body2))
	rrLoginFail := httptest.NewRecorder()
	r.ServeHTTP(rrLoginFail, reqLoginFail)
	if rrLoginFail.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %v", rrLoginFail.Code)
	}

	// 4. 正确登录
	loginSuccessPayload := map[string]string{
		"username": "userA",
		"password": "pass123",
	}
	body3, _ := json.Marshal(loginSuccessPayload)
	reqLoginSuccess, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body3))
	rrLoginSuccess := httptest.NewRecorder()
	r.ServeHTTP(rrLoginSuccess, reqLoginSuccess)
	if rrLoginSuccess.Code != http.StatusOK {
		t.Fatalf("expected 200 for correct login, got %v", rrLoginSuccess.Code)
	}

	// 断言生成的 HttpOnly Cookie
	cookies := rrLoginSuccess.Result().Cookies()
	var authCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "token" {
			authCookie = c
			break
		}
	}
	if authCookie == nil {
		t.Fatalf("expected token cookie to be set")
	}
	if !authCookie.HttpOnly {
		t.Errorf("token cookie must be HttpOnly")
	}

	// 5. 验证鉴权放行：携带有效 Cookie 获取个人信息
	reqMeSuccess, _ := http.NewRequest("GET", "/api/auth/me", nil)
	reqMeSuccess.AddCookie(authCookie)
	rrMeSuccess := httptest.NewRecorder()
	r.ServeHTTP(rrMeSuccess, reqMeSuccess)
	if rrMeSuccess.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated request, got %v", rrMeSuccess.Code)
	}

	var meResp struct {
		Success bool `json:"success"`
		Data    struct {
			Username    string `json:"username"`
			DisplayName string `json:"display_name"`
			LedgerID    string `json:"ledger_id"`
		} `json:"data"`
	}
	json.NewDecoder(rrMeSuccess.Body).Decode(&meResp)
	if !meResp.Success || meResp.Data.Username != "userA" || meResp.Data.DisplayName != "User A" {
		t.Errorf("expected userA info, got %+v", meResp.Data)
	}

	// 6. 测试登出清理 Session
	reqLogout, _ := http.NewRequest("POST", "/api/auth/logout", nil)
	rrLogout := httptest.NewRecorder()
	r.ServeHTTP(rrLogout, reqLogout)
	if rrLogout.Code != http.StatusOK {
		t.Errorf("expected 200 for logout, got %v", rrLogout.Code)
	}
	logoutCookies := rrLogout.Result().Cookies()
	cleared := false
	for _, c := range logoutCookies {
		if c.Name == "token" && c.MaxAge == -1 {
			cleared = true
		}
	}
	if !cleared {
		t.Errorf("expected token cookie to be cleared via MaxAge -1")
	}
}
