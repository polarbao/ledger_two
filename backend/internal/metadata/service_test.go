package metadata

import (
	"context"
	"errors"
	"testing"

	appErrors "ledger_two/internal/errors"
	ledgerctx "ledger_two/internal/ledger"
)

func TestParseKind(t *testing.T) {
	tests := []struct {
		value string
		ok    bool
	}{
		{"categories", true},
		{"tags", true},
		{"accounts", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		_, ok := ParseKind(tt.value)
		if ok != tt.ok {
			t.Fatalf("ParseKind(%q) ok=%v, want %v", tt.value, ok, tt.ok)
		}
	}
}

func TestCanManage(t *testing.T) {
	if !CanManage("owner") {
		t.Fatalf("owner should manage metadata")
	}
	if CanManage("editor") {
		t.Fatalf("editor should not manage metadata by default")
	}
	if CanManage("viewer") {
		t.Fatalf("viewer should not manage metadata")
	}
}

func TestResolveLedgerUsesExplicitContext(t *testing.T) {
	svc := &Service{}
	ctx := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID:     "user-a",
		LedgerID:   "ledger-a",
		Role:       ledgerctx.RoleOwner,
		Status:     ledgerctx.LedgerStatusActive,
		Version:    1,
		IsExplicit: true,
	})

	ledgerID, role, err := svc.resolveLedger(ctx, "user-a")
	if err != nil {
		t.Fatalf("resolve explicit ledger: %v", err)
	}
	if ledgerID != "ledger-a" || role != "owner" {
		t.Fatalf("got ledger=%q role=%q", ledgerID, role)
	}
}

func TestResolveLedgerRejectsMissingExplicitContext(t *testing.T) {
	svc := &Service{}
	_, _, err := svc.resolveLedger(context.Background(), "user-a")
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != appErrors.ErrCodeLedgerRequired {
		t.Fatalf("expected LEDGER_REQUIRED, got %v", err)
	}
}
