package defaults

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ApplyResult struct {
	CreatedCategories int
	CreatedTags       int
	ProfileVersion    int
}

func ApplyFresh(
	ctx context.Context,
	tx *sql.Tx,
	ledgerID string,
	ownerUserID string,
	profileKey string,
	now time.Time,
) (ApplyResult, error) {
	profile, ok := Get(profileKey)
	if !ok {
		return ApplyResult{}, fmt.Errorf("unknown metadata profile %q", profileKey)
	}

	result := ApplyResult{ProfileVersion: profile.Version}
	for _, item := range profile.Items {
		if err := InsertItem(ctx, tx, ledgerID, ownerUserID, item, now); err != nil {
			return ApplyResult{}, err
		}
		if item.Kind == KindTag {
			result.CreatedTags++
		} else {
			result.CreatedCategories++
		}
	}

	formattedNow := now.UTC().Format(time.RFC3339Nano)
	databaseResult, err := tx.ExecContext(ctx, `
		UPDATE ledgers
		SET metadata_profile_version = ?, updated_at = ?
		WHERE id = ?
	`, profile.Version, formattedNow, ledgerID)
	if err != nil {
		return ApplyResult{}, err
	}
	rows, err := databaseResult.RowsAffected()
	if err != nil {
		return ApplyResult{}, err
	}
	if rows != 1 {
		return ApplyResult{}, sql.ErrNoRows
	}
	return result, nil
}

func InsertItem(
	ctx context.Context,
	tx *sql.Tx,
	ledgerID string,
	ownerUserID string,
	item Item,
	now time.Time,
) error {
	formattedNow := now.UTC().Format(time.RFC3339Nano)
	switch item.Kind {
	case KindExpenseCategory, KindIncomeCategory:
		categoryType := "expense"
		if item.Kind == KindIncomeCategory {
			categoryType = "income"
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO categories (
				id, ledger_id, owner_user_id, name, type, icon, color, sort_order,
				is_system, is_archived, system_key, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, 0, ?, ?, ?)
		`, uuid.NewString(), ledgerID, ownerUserID, item.Name, categoryType,
			nullString(item.Icon), nullString(item.Color), item.SortOrder,
			item.SystemKey, formattedNow, formattedNow)
		return err
	case KindTag:
		_, err := tx.ExecContext(ctx, `
			INSERT INTO tags (
				id, ledger_id, owner_user_id, name, color, sort_order,
				is_archived, system_key, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, ?)
		`, uuid.NewString(), ledgerID, ownerUserID, item.Name, nullString(item.Color),
			item.SortOrder, item.SystemKey, formattedNow, formattedNow)
		return err
	default:
		return fmt.Errorf("unsupported metadata profile kind %q", item.Kind)
	}
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
