package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "请求参数解析失败")
		return
	}

	tokenString, err := h.svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	// 将 Token 签入安全 Cookie 屏障 (HttpOnly防窃取)
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Path:     "/",
		Expires:  time.Now().Add(24 * 7 * time.Hour),
		HttpOnly: true,
		Secure:   false, // 仅在本地 demo 下为 false
		SameSite: http.SameSiteLaxMode,
	})

	// 登录成功：Cookie 已写入，返回成功响应
	// 前端 App 启动逻辑会再调用 /api/auth/me 获取完整用户信息
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// 强制置空和设置负数的失效时间来摧毁 Session
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	// 被 middleware 拦截器包裹，提取身份标识
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
		return
	}

	me, err := h.svc.GetMe(r.Context(), userID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, me)
}
