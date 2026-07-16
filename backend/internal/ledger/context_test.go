package ledger

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestResolveLedgerContext(t *testing.T) {
	lc, err := ResolveLedgerContext(context.Background(), "user-a", "ledger-a", true, func(ctx context.Context, ledgerID string, userID string) (Role, LedgerStatus, int64, error) {
		if ledgerID != "ledger-a" || userID != "user-a" {
			t.Fatalf("unexpected lookup args: %s %s", ledgerID, userID)
		}
		return RoleOwner, LedgerStatusArchived, 7, nil
	})
	if err != nil {
		t.Fatalf("resolve ledger context failed: %v", err)
	}
	if lc.UserID != "user-a" || lc.LedgerID != "ledger-a" || lc.Role != RoleOwner || lc.Status != LedgerStatusArchived || lc.Version != 7 || !lc.IsExplicit {
		t.Fatalf("unexpected ledger context: %+v", lc)
	}
}

func TestResolveLedgerContextRequiresUserAndLedger(t *testing.T) {
	_, err := ResolveLedgerContext(context.Background(), "", "ledger-a", true, nil)
	if !errors.Is(err, ErrLedgerUserRequired) {
		t.Fatalf("expected ErrLedgerUserRequired, got %v", err)
	}

	_, err = ResolveLedgerContext(context.Background(), "user-a", "", true, nil)
	if !errors.Is(err, ErrLedgerIDRequired) {
		t.Fatalf("expected ErrLedgerIDRequired, got %v", err)
	}
}

func TestResolveLedgerContextRejectsInvalidRole(t *testing.T) {
	_, err := ResolveLedgerContext(context.Background(), "user-a", "ledger-a", true, func(ctx context.Context, ledgerID string, userID string) (Role, LedgerStatus, int64, error) {
		return Role("admin"), LedgerStatusActive, 1, nil
	})
	if !errors.Is(err, ErrLedgerRoleInvalid) {
		t.Fatalf("expected ErrLedgerRoleInvalid, got %v", err)
	}
}

func TestResolveLedgerContextReturnsLookupError(t *testing.T) {
	_, err := ResolveLedgerContext(context.Background(), "user-a", "ledger-a", true, func(ctx context.Context, ledgerID string, userID string) (Role, LedgerStatus, int64, error) {
		return "", "", 0, sql.ErrNoRows
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected lookup error, got %v", err)
	}
}

func TestRequireExplicitLedgerContextRejectsMissingAndMismatchedContext(t *testing.T) {
	_, err := RequireExplicitLedgerContext(context.Background(), "user-a")
	if !errors.Is(err, ErrLedgerContextRequired) {
		t.Fatalf("expected ErrLedgerContextRequired, got %v", err)
	}

	ctx := ContextWithLedgerContext(context.Background(), LedgerContext{
		UserID:     "user-b",
		LedgerID:   "ledger-a",
		Role:       RoleOwner,
		Status:     LedgerStatusActive,
		Version:    1,
		IsExplicit: true,
	})
	_, err = RequireExplicitLedgerContext(ctx, "user-a")
	if !errors.Is(err, ErrLedgerContextMismatch) {
		t.Fatalf("expected ErrLedgerContextMismatch, got %v", err)
	}
}
