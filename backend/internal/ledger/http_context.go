package ledger

import (
	"context"
	"net/http"

	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

type ContextResolver interface {
	ResolveLedgerContext(ctx context.Context, currentUserID string, ledgerID string, isExplicit bool) (LedgerContext, error)
}

func WithLedgerContext(resolver ContextResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.GetUserIDFromContext(r.Context())
			ledgerID := middleware.GetHeaderLedgerIDFromContext(r.Context())
			if ledgerID == "" {
				ledgerID = r.Header.Get("X-Ledger-Id")
			}
			if ledgerID == "" {
				next.ServeHTTP(w, r)
				return
			}

			lc, err := resolver.ResolveLedgerContext(r.Context(), userID, ledgerID, true)
			if err != nil {
				response.WriteError(w, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithLedgerContext(r.Context(), lc)))
		})
	}
}
