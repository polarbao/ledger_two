package dashboard

import (
	"context"
	"errors"
	"testing"

	appErrors "ledger_two/internal/errors"
	ledgerctx "ledger_two/internal/ledger"
)

func TestGetUserLedgerIDUsesLedgerContext(t *testing.T) {
	svc := &Service{}
	ctx := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID:     "user-a",
		LedgerID:   "ledger-a",
		Role:       ledgerctx.RoleViewer,
		Status:     ledgerctx.LedgerStatusActive,
		Version:    1,
		IsExplicit: true,
	})

	ledgerID, err := svc.getUserLedgerID(ctx, "user-a")
	if err != nil {
		t.Fatalf("get ledger id from context failed: %v", err)
	}
	if ledgerID != "ledger-a" {
		t.Fatalf("expected ledger-a, got %s", ledgerID)
	}
}

func TestGetUserLedgerIDRejectsMissingExplicitContext(t *testing.T) {
	svc := &Service{}
	_, err := svc.getUserLedgerID(context.Background(), "user-a")
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != appErrors.ErrCodeLedgerRequired {
		t.Fatalf("expected LEDGER_REQUIRED, got %v", err)
	}
}
