package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"ledger_two/internal/http/response"
)

type LedgerContext struct {
	UserID     string
	LedgerID   string
	Role       string // "owner", "editor", "viewer"
	IsExplicit bool   // 标记是否由前端显式传入了 X-Ledger-Id
}

type ledgerCtxKey string

const LedgerContextKey ledgerCtxKey = "ledger_context"

// GetLedgerContext 从 Context 中抽取已经解析并注入的 Ledger 上下文
func GetLedgerContext(ctx context.Context) *LedgerContext {
	if val, ok := ctx.Value(LedgerContextKey).(*LedgerContext); ok {
		return val
	}
	return nil
}

// RequireLedgerContext 账本上下文注入与成员身份校验拦截中间件
func RequireLedgerContext(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserIDFromContext(r.Context())
			if userID == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
				return
			}

			// 1. 获取账本 ID
			ledgerID := r.Header.Get("X-Ledger-Id")
			isExplicit := true

			if ledgerID == "" {
				// 智能尝试从路径中匹配 /ledgers/{id}
				parts := strings.Split(r.URL.Path, "/")
				for i, part := range parts {
					if part == "ledgers" && i+1 < len(parts) && parts[i+1] != "" {
						ledgerID = parts[i+1]
						break
					}
				}
			}

			var role string
			if ledgerID != "" {
				// 校验当前用户是否属于该账本
				err := db.QueryRowContext(r.Context(), "SELECT role FROM ledger_members WHERE ledger_id = ? AND user_id = ?", ledgerID, userID).Scan(&role)
				if err != nil {
					response.Error(w, http.StatusForbidden, "FORBIDDEN", "您不是该账本的成员，无权访问该数据")
					return
				}
			} else {
				// Fallback 兜底逻辑：查询该用户所在的第一个账本
				isExplicit = false
				err := db.QueryRowContext(r.Context(), "SELECT ledger_id, role FROM ledger_members WHERE user_id = ? LIMIT 1", userID).Scan(&ledgerID, &role)
				if err != nil {
					// 用户没有加入任何账本
					response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "缺少账本上下文，请先选择或创建账本")
					return
				}
			}

			// 2. 注入上下文
			lc := &LedgerContext{
				UserID:     userID,
				LedgerID:   ledgerID,
				Role:       role,
				IsExplicit: isExplicit,
			}
			ctx := context.WithValue(r.Context(), LedgerContextKey, lc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireLedgerRole 检查账本内角色的权限控制中间件
func RequireLedgerRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lc := GetLedgerContext(r.Context())
			if lc == nil {
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "缺少账本上下文，无法进行角色校验")
				return
			}

			hasRole := false
			for _, allowed := range allowedRoles {
				if lc.Role == allowed {
					hasRole = true
					break
				}
			}

			if !hasRole {
				response.Error(w, http.StatusForbidden, "FORBIDDEN", "当前角色无权执行此操作")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
