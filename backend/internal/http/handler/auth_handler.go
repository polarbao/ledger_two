package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"ledger_two/internal/config"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
	cfg *config.Config
}

func (h *AuthHandler) SetConfig(cfg *config.Config) {
	h.cfg = cfg
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
		Secure:   h.isCookieSecure(r),
		SameSite: h.getCookieSameSite(),
	})

	// 登录成功：Cookie 已写入，返回成功响应
	// 前端 App 启动逻辑会再调用 /api/auth/me 获取完整用户信息
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// 强制置空和设置负数的失效时间来摧毁 Session，需与登录 Cookie 属性保持完全对齐
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.isCookieSecure(r),
		SameSite: h.getCookieSameSite(),
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

func (h *AuthHandler) isCookieSecure(r *http.Request) bool {
	if h.cfg == nil {
		return false
	}
	secure := h.cfg.AppEnv == "production"
	if h.cfg.CookieSecure == "false" {
		return false
	} else if h.cfg.CookieSecure == "true" {
		return true
	}
	// 如果在生产环境下使用非加密的 HTTP（例如内网 NAS 直接访问），自动将 Secure 降级为 false 允许保存 Cookie
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		return false
	}
	return secure
}

func (h *AuthHandler) getCookieSameSite() http.SameSite {
	if h.cfg == nil {
		return http.SameSiteLaxMode
	}
	switch strings.ToLower(h.cfg.CookieSameSite) {
	case "lax":
		return http.SameSiteLaxMode
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
