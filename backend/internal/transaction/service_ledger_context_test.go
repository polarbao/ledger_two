package transaction

import (
	"context"
	"testing"

	ledgerctx "ledger_two/internal/ledger"
)

func TestGetUserLedgerIDUsesLedgerContext(t *testing.T) {
	svc := &Service{}
	ctx := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID:   "user-a",
		LedgerID: "ledger-a",
		Role:     ledgerctx.RoleOwner,
	})

	ledgerID, err := svc.getUserLedgerID(ctx, "user-a")
	if err != nil {
		t.Fatalf("get ledger id from context failed: %v", err)
	}
	if ledgerID != "ledger-a" {
		t.Fatalf("expected ledger-a, got %s", ledgerID)
	}
}

func TestCheckRoleUsesLedgerContext(t *testing.T) {
	svc := &Service{}
	ctx := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID:   "user-a",
		LedgerID: "ledger-a",
		Role:     ledgerctx.RoleEditor,
	})

	if err := svc.checkRole(ctx, "ledger-a", "user-a", "owner", "editor"); err != nil {
		t.Fatalf("check role from context failed: %v", err)
	}
}

func TestCheckRoleRejectsLedgerContextRole(t *testing.T) {
	svc := &Service{}
	ctx := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID:   "user-a",
		LedgerID: "ledger-a",
		Role:     ledgerctx.RoleViewer,
	})

	if err := svc.checkRole(ctx, "ledger-a", "user-a", "owner", "editor"); err == nil {
		t.Fatalf("expected forbidden error")
	}
}
