package middleware

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"

	"ledger_two/internal/http/response"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// authError 向客户端返回符合统一 {success, error} 规范的 401 响应
func authError(w http.ResponseWriter) {
	response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
}

// RequireAuth 是拦截器，从 HttpOnly Cookie 获取鉴权标识并验证 Token
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				authError(w)
				return
			}

			tokenStr := cookie.Value
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				authError(w)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				authError(w)
				return
			}

			userID, ok := claims["user_id"].(string)
			if !ok {
				authError(w)
				return
			}

			// 将 user_id 注入上下文，向下游透传
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext 从上游拦截器存放的 Context 中抽取 user_id
func GetUserIDFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(UserIDKey).(string); ok {
		return val
	}
	return ""
}
