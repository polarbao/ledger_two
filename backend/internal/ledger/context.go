package ledger

import (
	"context"
	"errors"
)

type ledgerContextKey struct{}

var (
	ErrLedgerUserRequired    = errors.New("ledger user id is required")
	ErrLedgerIDRequired      = errors.New("ledger id is required")
	ErrLedgerRoleInvalid     = errors.New("ledger role is invalid")
	ErrLedgerStateInvalid    = errors.New("ledger state is invalid")
	ErrLedgerContextRequired = errors.New("explicit ledger context is required")
	ErrLedgerContextMismatch = errors.New("ledger context does not match current user")
)

type LedgerAccessLookup func(ctx context.Context, ledgerID string, userID string) (Role, LedgerStatus, int64, error)

func ContextWithLedgerContext(ctx context.Context, lc LedgerContext) context.Context {
	return context.WithValue(ctx, ledgerContextKey{}, lc)
}

func LedgerContextFromContext(ctx context.Context) (LedgerContext, bool) {
	lc, ok := ctx.Value(ledgerContextKey{}).(LedgerContext)
	return lc, ok
}

func RequireExplicitLedgerContext(ctx context.Context, userID string) (LedgerContext, error) {
	lc, ok := LedgerContextFromContext(ctx)
	if !ok || !lc.IsExplicit || lc.LedgerID == "" {
		return LedgerContext{}, ErrLedgerContextRequired
	}
	if lc.UserID != userID {
		return LedgerContext{}, ErrLedgerContextMismatch
	}
	return lc, nil
}

func ResolveLedgerContext(ctx context.Context, userID string, ledgerID string, isExplicit bool, lookup LedgerAccessLookup) (LedgerContext, error) {
	if userID == "" {
		return LedgerContext{}, ErrLedgerUserRequired
	}
	if ledgerID == "" {
		return LedgerContext{}, ErrLedgerIDRequired
	}

	role, status, version, err := lookup(ctx, ledgerID, userID)
	if err != nil {
		return LedgerContext{}, err
	}
	if !IsValidRole(role) {
		return LedgerContext{}, ErrLedgerRoleInvalid
	}
	if (status != LedgerStatusActive && status != LedgerStatusArchived) || version < 1 {
		return LedgerContext{}, ErrLedgerStateInvalid
	}

	return LedgerContext{
		UserID:     userID,
		LedgerID:   ledgerID,
		Role:       role,
		Status:     status,
		Version:    version,
		IsExplicit: isExplicit,
	}, nil
}
