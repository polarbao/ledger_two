package handler_test

import (
	"database/sql"
	"net/http"
	"testing"

	"ledger_two/internal/http/middleware"
	ledgerctx "ledger_two/internal/ledger"
)

// testAuthenticatedLedgerContext requires each handler test request to provide
// its ledger explicitly; tests must not authorize requests via a first-ledger fallback.
func testAuthenticatedLedgerContext(db *sql.DB, jwtSecret string) func(http.Handler) http.Handler {
	ledgerService := ledgerctx.NewService(ledgerctx.NewRepository(db))
	requireLedger := ledgerctx.WithRequiredLedgerContext(ledgerService, "")

	return func(next http.Handler) http.Handler {
		contextualized := requireLedger(next)
		return middleware.RequireAuth(jwtSecret)(contextualized)
	}
}

func setTestLedgerHeader(t *testing.T, db *sql.DB, request *http.Request, ledgerName string) {
	t.Helper()
	var ledgerID string
	if err := db.QueryRowContext(request.Context(), "SELECT id FROM ledgers WHERE name = ?", ledgerName).Scan(&ledgerID); err != nil {
		t.Fatalf("resolve explicit test ledger %q: %v", ledgerName, err)
	}
	request.Header.Set("X-Ledger-Id", ledgerID)
}
