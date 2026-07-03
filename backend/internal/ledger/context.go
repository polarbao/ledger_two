package ledger

import (
	"context"
	"errors"
)

type ledgerContextKey struct{}

var (
	ErrLedgerUserRequired = errors.New("ledger user id is required")
	ErrLedgerIDRequired   = errors.New("ledger id is required")
	ErrLedgerRoleInvalid  = errors.New("ledger role is invalid")
)

type MembershipLookup func(ctx context.Context, ledgerID string, userID string) (Role, error)

func ContextWithLedgerContext(ctx context.Context, lc LedgerContext) context.Context {
	return context.WithValue(ctx, ledgerContextKey{}, lc)
}

func LedgerContextFromContext(ctx context.Context) (LedgerContext, bool) {
	lc, ok := ctx.Value(ledgerContextKey{}).(LedgerContext)
	return lc, ok
}

func ResolveLedgerContext(ctx context.Context, userID string, ledgerID string, isExplicit bool, lookup MembershipLookup) (LedgerContext, error) {
	if userID == "" {
		return LedgerContext{}, ErrLedgerUserRequired
	}
	if ledgerID == "" {
		return LedgerContext{}, ErrLedgerIDRequired
	}

	role, err := lookup(ctx, ledgerID, userID)
	if err != nil {
		return LedgerContext{}, err
	}
	if !IsValidRole(role) {
		return LedgerContext{}, ErrLedgerRoleInvalid
	}

	return LedgerContext{
		UserID:     userID,
		LedgerID:   ledgerID,
		Role:       role,
		IsExplicit: isExplicit,
	}, nil
}
