package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

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

func TestWithRequiredLedgerContextRejectsMissingLedgerID(t *testing.T) {
	handler := WithRequiredLedgerContext(fakeContextResolver{}, "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user-a"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
	if code := responseErrorCode(t, rr); code != appErrors.ErrCodeLedgerRequired {
		t.Fatalf("expected %s, got %s", appErrors.ErrCodeLedgerRequired, code)
	}
}

func TestWithLedgerContextInjectsResolvedContext(t *testing.T) {
	expected := LedgerContext{UserID: "user-a", LedgerID: "ledger-a", Role: RoleOwner, IsExplicit: true}
	handler := WithRequiredLedgerContext(fakeContextResolver{lc: expected}, "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := WithRequiredLedgerContext(fakeContextResolver{
		err: appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "您不是该账本的成员"),
	}, "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestWithRequiredLedgerContextRejectsPathHeaderMismatch(t *testing.T) {
	router := chi.NewRouter()
	router.Route("/api/ledgers/{ledgerId}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "user-a")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		r.Use(WithRequiredLedgerContext(fakeContextResolver{}, "ledgerId"))
		r.Get("/members", func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("next handler should not be called")
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ledgers/ledger-a/members", nil)
	req.Header.Set("X-Ledger-Id", "ledger-b")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
	if code := responseErrorCode(t, rr); code != appErrors.ErrCodeLedgerContextMismatch {
		t.Fatalf("expected %s, got %s", appErrors.ErrCodeLedgerContextMismatch, code)
	}
}

func TestRequireWritableLedgerRejectsArchivedLedger(t *testing.T) {
	handler := RequireWritableLedger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next handler should not be called")
	}))
	ctx := ContextWithLedgerContext(context.Background(), LedgerContext{
		UserID:     "user-a",
		LedgerID:   "ledger-a",
		Role:       RoleOwner,
		Status:     LedgerStatusArchived,
		Version:    2,
		IsExplicit: true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/transactions", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}
	if code := responseErrorCode(t, rr); code != appErrors.ErrCodeLedgerArchived {
		t.Fatalf("expected %s, got %s", appErrors.ErrCodeLedgerArchived, code)
	}
}

func responseErrorCode(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return payload.Error.Code
}
