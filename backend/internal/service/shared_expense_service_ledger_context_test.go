package service

import (
	"context"
	"errors"
	"testing"

	appErrors "ledger_two/internal/errors"
	ledgerctx "ledger_two/internal/ledger"
)

func TestSharedExpenseGetUserLedgerIDUsesExplicitContext(t *testing.T) {
	svc := &SharedExpenseService{}
	ctx := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID:     "user-a",
		LedgerID:   "ledger-a",
		Role:       ledgerctx.RoleOwner,
		Status:     ledgerctx.LedgerStatusActive,
		Version:    1,
		IsExplicit: true,
	})

	ledgerID, err := svc.GetUserLedgerID(ctx, "user-a")
	if err != nil || ledgerID != "ledger-a" {
		t.Fatalf("got ledger=%q err=%v", ledgerID, err)
	}
}

func TestSharedExpenseGetUserLedgerIDRejectsMissingContext(t *testing.T) {
	svc := &SharedExpenseService{}
	_, err := svc.GetUserLedgerID(context.Background(), "user-a")
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != appErrors.ErrCodeLedgerRequired {
		t.Fatalf("expected LEDGER_REQUIRED, got %v", err)
	}
}
