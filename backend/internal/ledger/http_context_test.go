package ledger

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
)

type fakeContextResolver struct {
	lc  LedgerContext
	err error
}

func (f fakeContextResolver) ResolveLedgerContext(ctx context.Context, currentUserID string, ledgerID string, isExplicit bool) (LedgerContext, error) {
	if currentUserID != "user-a" {
		return LedgerContext{}, errors.New("unexpected user id")
	}
	if ledgerID != "ledger-a" {
		return LedgerContext{}, errors.New("unexpected ledger id")
	}
	if !isExplicit {
		return LedgerContext{}, errors.New("expected explicit ledger context")
	}
	return f.lc, f.err
}

func TestWithLedgerContextSkipsWhenLedgerHeaderMissing(t *testing.T) {
	called := false
	handler := WithLedgerContext(fakeContextResolver{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := LedgerContextFromContext(r.Context()); ok {
			t.Fatalf("ledger context should not be injected without ledger header")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user-a"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if !called {
		t.Fatalf("expected next handler to be called")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
}

func TestWithLedgerContextInjectsResolvedContext(t *testing.T) {
	expected := LedgerContext{UserID: "user-a", LedgerID: "ledger-a", Role: RoleOwner, IsExplicit: true}
	handler := WithLedgerContext(fakeContextResolver{lc: expected})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lc, ok := LedgerContextFromContext(r.Context())
		if !ok {
			t.Fatalf("expected ledger context")
		}
		if lc != expected {
			t.Fatalf("unexpected ledger context: %+v", lc)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	req.Header.Set("X-Ledger-Id", "ledger-a")
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user-a"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
}

func TestWithLedgerContextStopsOnResolverError(t *testing.T) {
	handler := WithLedgerContext(fakeContextResolver{
		err: appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "您不是该账本的成员"),
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	req.Header.Set("X-Ledger-Id", "ledger-a")
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user-a"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rr.Code)
	}
}
