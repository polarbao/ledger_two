package metadata

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetMemberRole(ctx context.Context, ledgerID string, userID string) (string, error) {
	var role string
	err := r.db.QueryRowContext(ctx, "SELECT role FROM ledger_members WHERE ledger_id = ? AND user_id = ?", ledgerID, userID).Scan(&role)
	return role, err
}

func (r *Repository) GetFirstLedgerRole(ctx context.Context, userID string) (string, string, error) {
	var ledgerID string
	var role string
	err := r.db.QueryRowContext(ctx, "SELECT ledger_id, role FROM ledger_members WHERE user_id = ? LIMIT 1", userID).Scan(&ledgerID, &role)
	return ledgerID, role, err
}

func (r *Repository) List(ctx context.Context, kind Kind, ledgerID string, includeArchived bool) ([]Item, error) {
	switch kind {
	case KindCategory:
		query := "SELECT id, ledger_id, name, type, COALESCE(icon, ''), COALESCE(color, ''), sort_order, is_archived FROM categories WHERE ledger_id = ?"
		if !includeArchived {
			query += " AND is_archived = 0"
		}
		query += " ORDER BY sort_order ASC, name ASC"
		return r.list(ctx, query, ledgerID)
	case KindTag:
		query := "SELECT id, ledger_id, name, '', '', COALESCE(color, ''), sort_order, is_archived FROM tags WHERE ledger_id = ?"
		if !includeArchived {
			query += " AND is_archived = 0"
		}
		query += " ORDER BY sort_order ASC, name ASC"
		return r.list(ctx, query, ledgerID)
	case KindAccount:
		query := "SELECT id, ledger_id, name, type, '', '', sort_order, is_archived FROM accounts WHERE ledger_id = ?"
		if !includeArchived {
			query += " AND is_archived = 0"
		}
		query += " ORDER BY sort_order ASC, name ASC"
		return r.list(ctx, query, ledgerID)
	default:
		return nil, sql.ErrNoRows
	}
}

func (r *Repository) list(ctx context.Context, query string, ledgerID string) ([]Item, error) {
	rows, err := r.db.QueryContext(ctx, query, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		var isArchived int
		if err := rows.Scan(&item.ID, &item.LedgerID, &item.Name, &item.Type, &item.Icon, &item.Color, &item.SortOrder, &isArchived); err != nil {
			return nil, err
		}
		item.IsArchived = isArchived == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) NameExists(ctx context.Context, kind Kind, ledgerID string, itemType string, name string, excludeID string) (bool, error) {
	var query string
	var args []interface{}
	switch kind {
	case KindCategory:
		query = "SELECT 1 FROM categories WHERE ledger_id = ? AND type = ? AND name = ?"
		args = []interface{}{ledgerID, itemType, name}
	case KindTag:
		query = "SELECT 1 FROM tags WHERE ledger_id = ? AND name = ?"
		args = []interface{}{ledgerID, name}
	case KindAccount:
		query = "SELECT 1 FROM accounts WHERE ledger_id = ? AND name = ?"
		args = []interface{}{ledgerID, name}
	default:
		return false, sql.ErrNoRows
	}
	if excludeID != "" {
		query += " AND id <> ?"
		args = append(args, excludeID)
	}

	var exists int
	err := r.db.QueryRowContext(ctx, query+" LIMIT 1", args...).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func (r *Repository) Create(ctx context.Context, kind Kind, ledgerID string, userID string, req UpsertRequest) (*Item, error) {
	id := uuid.NewString()
	now := time.Now().Format(time.RFC3339)
	sortOrder, err := r.nextSortOrder(ctx, kind, ledgerID)
	if err != nil {
		return nil, err
	}
	switch kind {
	case KindCategory:
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO categories (id, ledger_id, owner_user_id, name, type, icon, color, sort_order, is_system, is_archived, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?)
		`, id, ledgerID, userID, req.Name, req.Type, nullString(req.Icon), nullString(req.Color), sortOrder, now, now)
		if err != nil {
			return nil, err
		}
	case KindTag:
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO tags (id, ledger_id, name, owner_user_id, color, sort_order, is_archived, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)
		`, id, ledgerID, req.Name, userID, nullString(req.Color), sortOrder, now, now)
		if err != nil {
			return nil, err
		}
	case KindAccount:
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO accounts (id, ledger_id, owner_user_id, name, type, currency, initial_balance, sort_order, is_archived, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 'CNY', 0, ?, 0, ?, ?)
		`, id, ledgerID, userID, req.Name, req.Type, sortOrder, now, now)
		if err != nil {
			return nil, err
		}
	default:
		return nil, sql.ErrNoRows
	}

	return &Item{
		ID:        id,
		LedgerID:  ledgerID,
		Name:      req.Name,
		Type:      req.Type,
		Icon:      req.Icon,
		Color:     req.Color,
		SortOrder: sortOrder,
	}, nil
}

func (r *Repository) Update(ctx context.Context, kind Kind, ledgerID string, id string, req UpsertRequest) error {
	now := time.Now().Format(time.RFC3339)
	switch kind {
	case KindCategory:
		return execRequireRows(r.db.ExecContext(ctx, `
			UPDATE categories SET name = ?, type = ?, icon = ?, color = ?, updated_at = ?
			WHERE id = ? AND ledger_id = ?
		`, req.Name, req.Type, nullString(req.Icon), nullString(req.Color), now, id, ledgerID))
	case KindTag:
		return execRequireRows(r.db.ExecContext(ctx, `
			UPDATE tags SET name = ?, color = ?, updated_at = ?
			WHERE id = ? AND ledger_id = ?
		`, req.Name, nullString(req.Color), now, id, ledgerID))
	case KindAccount:
		return execRequireRows(r.db.ExecContext(ctx, `
			UPDATE accounts SET name = ?, type = ?, updated_at = ?
			WHERE id = ? AND ledger_id = ?
		`, req.Name, req.Type, now, id, ledgerID))
	default:
		return sql.ErrNoRows
	}
}

func (r *Repository) SetArchived(ctx context.Context, kind Kind, ledgerID string, id string, archived bool) error {
	value := 0
	if archived {
		value = 1
	}
	now := time.Now().Format(time.RFC3339)
	var err error
	switch kind {
	case KindCategory:
		err = execRequireRows(r.db.ExecContext(ctx, "UPDATE categories SET is_archived = ?, updated_at = ? WHERE id = ? AND ledger_id = ?", value, now, id, ledgerID))
	case KindTag:
		err = execRequireRows(r.db.ExecContext(ctx, "UPDATE tags SET is_archived = ?, updated_at = ? WHERE id = ? AND ledger_id = ?", value, now, id, ledgerID))
	case KindAccount:
		err = execRequireRows(r.db.ExecContext(ctx, "UPDATE accounts SET is_archived = ?, updated_at = ? WHERE id = ? AND ledger_id = ?", value, now, id, ledgerID))
	default:
		err = sql.ErrNoRows
	}
	return err
}

func (r *Repository) Reorder(ctx context.Context, kind Kind, ledgerID string, orderedIDs []string) error {
	table, err := tableName(kind)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for index, id := range orderedIDs {
		result, err := tx.ExecContext(ctx, "UPDATE "+table+" SET sort_order = ?, updated_at = ? WHERE id = ? AND ledger_id = ?", index, time.Now().Format(time.RFC3339), id, ledgerID)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return sql.ErrNoRows
		}
	}
	return tx.Commit()
}

func (r *Repository) nextSortOrder(ctx context.Context, kind Kind, ledgerID string) (int, error) {
	table, err := tableName(kind)
	if err != nil {
		return 0, err
	}
	var next int
	err = r.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(sort_order), -1) + 1 FROM "+table+" WHERE ledger_id = ?", ledgerID).Scan(&next)
	return next, err
}

func tableName(kind Kind) (string, error) {
	switch kind {
	case KindCategory:
		return "categories", nil
	case KindTag:
		return "tags", nil
	case KindAccount:
		return "accounts", nil
	default:
		return "", sql.ErrNoRows
	}
}

func execRequireRows(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
