package ledger

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestResolveLedgerContext(t *testing.T) {
	lc, err := ResolveLedgerContext(context.Background(), "user-a", "ledger-a", true, func(ctx context.Context, ledgerID string, userID string) (Role, error) {
		if ledgerID != "ledger-a" || userID != "user-a" {
			t.Fatalf("unexpected lookup args: %s %s", ledgerID, userID)
		}
		return RoleOwner, nil
	})
	if err != nil {
		t.Fatalf("resolve ledger context failed: %v", err)
	}
	if lc.UserID != "user-a" || lc.LedgerID != "ledger-a" || lc.Role != RoleOwner || !lc.IsExplicit {
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
	_, err := ResolveLedgerContext(context.Background(), "user-a", "ledger-a", true, func(ctx context.Context, ledgerID string, userID string) (Role, error) {
		return Role("admin"), nil
	})
	if !errors.Is(err, ErrLedgerRoleInvalid) {
		t.Fatalf("expected ErrLedgerRoleInvalid, got %v", err)
	}
}

func TestResolveLedgerContextReturnsLookupError(t *testing.T) {
	_, err := ResolveLedgerContext(context.Background(), "user-a", "ledger-a", true, func(ctx context.Context, ledgerID string, userID string) (Role, error) {
		return "", sql.ErrNoRows
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected lookup error, got %v", err)
	}
}
