package ledger

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

type ContextResolver interface {
	ResolveLedgerContext(ctx context.Context, currentUserID string, ledgerID string, isExplicit bool) (LedgerContext, error)
}

func WithLedgerContext(resolver ContextResolver) func(http.Handler) http.Handler {
	return WithRequiredLedgerContext(resolver, "")
}

func WithRequiredLedgerContext(resolver ContextResolver, pathParam string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.GetUserIDFromContext(r.Context())
			headerLedgerID := strings.TrimSpace(middleware.GetHeaderLedgerIDFromContext(r.Context()))
			if headerLedgerID == "" {
				headerLedgerID = strings.TrimSpace(r.Header.Get("X-Ledger-Id"))
			}
			pathLedgerID := ""
			if pathParam != "" {
				pathLedgerID = strings.TrimSpace(chi.URLParam(r, pathParam))
			}
			if pathLedgerID != "" && headerLedgerID != "" && pathLedgerID != headerLedgerID {
				response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerContextMismatch, "路径账本与请求账本不一致"))
				return
			}
			ledgerID := headerLedgerID
			if pathLedgerID != "" {
				ledgerID = pathLedgerID
			}
			if ledgerID == "" {
				response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerRequired, "请选择账本后再执行此操作"))
				return
			}

			lc, err := resolver.ResolveLedgerContext(r.Context(), userID, ledgerID, true)
			if err != nil {
				response.WriteError(w, err)
				return
			}

			ctx := ContextWithLedgerContext(r.Context(), lc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireLedgerContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lc, ok := LedgerContextFromContext(r.Context())
		if !ok || !lc.IsExplicit || lc.LedgerID == "" {
			response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerRequired, "请选择账本后再执行此操作"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireWritableLedger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lc, ok := LedgerContextFromContext(r.Context())
		if !ok || !lc.IsExplicit {
			response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerRequired, "请选择账本后再执行此操作"))
			return
		}
		if !NewLifecyclePolicy().Can(lc.Status, LifecycleWrite) {
			if lc.Status == LedgerStatusArchived {
				response.WriteError(w, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerArchived, "归档账本为只读状态"))
				return
			}
			response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerInvalidState, "账本状态不允许此操作"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireOperation(policy RolePolicy, operation Operation) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lc, ok := LedgerContextFromContext(r.Context())
			if !ok || !lc.IsExplicit {
				response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerRequired, "请选择账本后再执行此操作"))
				return
			}
			if !policy.Can(lc.Role, operation) {
				response.WriteError(w, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeLedgerAccessDenied, "当前角色无权执行此操作"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireInstanceAdmin(policy InstancePolicy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.GetUserIDFromContext(r.Context())
			allowed, err := policy.Can(r.Context(), userID)
			if err != nil {
				response.WriteError(w, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "实例权限校验失败"))
				return
			}
			if !allowed {
				response.WriteError(w, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeInstanceAdminRequired, "需要实例管理员权限"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
