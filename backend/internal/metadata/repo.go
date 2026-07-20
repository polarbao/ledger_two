package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	errFallbackReplacementRequired = errors.New("fallback replacement category is required")
	errFallbackReplacementInvalid  = errors.New("fallback replacement category is invalid")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *Repository) List(ctx context.Context, kind Kind, ledgerID string, includeArchived bool) ([]Item, error) {
	var (
		items []Item
		err   error
	)
	switch kind {
	case KindCategory:
		query := `
			SELECT id, ledger_id, COALESCE(system_key, ''), name, type, COALESCE(icon, ''), COALESCE(color, ''), sort_order,
				(SELECT COUNT(1) FROM transactions tx WHERE tx.ledger_id = categories.ledger_id AND tx.category_id = categories.id AND tx.status <> 'deleted') AS usage_count,
				is_archived
			FROM categories
			WHERE ledger_id = ?`
		if !includeArchived {
			query += " AND is_archived = 0"
		}
		query += " ORDER BY sort_order ASC, name ASC"
		items, err = r.list(ctx, query, ledgerID)
	case KindTag:
		query := `
			SELECT id, ledger_id, COALESCE(system_key, ''), name, '', '', COALESCE(color, ''), sort_order,
				(SELECT COUNT(1) FROM transaction_tags tt JOIN transactions tx ON tx.id = tt.transaction_id WHERE tx.ledger_id = tags.ledger_id AND tt.tag_id = tags.id AND tx.status <> 'deleted') AS usage_count,
				is_archived
			FROM tags
			WHERE ledger_id = ?`
		if !includeArchived {
			query += " AND is_archived = 0"
		}
		query += " ORDER BY sort_order ASC, name ASC"
		items, err = r.list(ctx, query, ledgerID)
	case KindAccount:
		query := `
			SELECT id, ledger_id, '', name, type, '', '', sort_order,
				(SELECT COUNT(1) FROM transactions tx WHERE tx.ledger_id = accounts.ledger_id AND tx.account_id = accounts.id AND tx.status <> 'deleted') AS usage_count,
				is_archived
			FROM accounts
			WHERE ledger_id = ?`
		if !includeArchived {
			query += " AND is_archived = 0"
		}
		query += " ORDER BY sort_order ASC, name ASC"
		items, err = r.list(ctx, query, ledgerID)
	default:
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	counts, err := r.activeRuleReferenceCounts(ctx, ledgerID)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index].RuleReferenceCount = counts[items[index].ID]
	}
	return items, nil
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
		if err := rows.Scan(&item.ID, &item.LedgerID, &item.SystemKey, &item.Name, &item.Type, &item.Icon, &item.Color, &item.SortOrder, &item.UsageCount, &isArchived); err != nil {
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
		ID:         id,
		LedgerID:   ledgerID,
		Name:       req.Name,
		Type:       req.Type,
		Icon:       req.Icon,
		Color:      req.Color,
		SortOrder:  sortOrder,
		UsageCount: 0,
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

func (r *Repository) ArchiveCategory(
	ctx context.Context,
	ledgerID string,
	actorUserID string,
	id string,
	replacementCategoryID string,
) (*ArchiveResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var categoryType, systemKey string
	var isArchived int
	if err := tx.QueryRowContext(ctx, `
		SELECT type, COALESCE(system_key, ''), is_archived
		FROM categories
		WHERE id = ? AND ledger_id = ?
	`, id, ledgerID).Scan(&categoryType, &systemKey, &isArchived); err != nil {
		return nil, err
	}

	result := &ArchiveResult{ArchivedID: id}
	if systemKey != "expense_other" && systemKey != "income_other" {
		if err := execRequireRows(tx.ExecContext(ctx, `
			UPDATE categories SET is_archived = 1, updated_at = ?
			WHERE id = ? AND ledger_id = ?
		`, time.Now().UTC().Format(time.RFC3339Nano), id, ledgerID)); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return result, nil
	}
	if replacementCategoryID == "" {
		return nil, errFallbackReplacementRequired
	}

	var replacementType, replacementSystemKey string
	var replacementArchived int
	if err := tx.QueryRowContext(ctx, `
		SELECT type, COALESCE(system_key, ''), is_archived
		FROM categories
		WHERE id = ? AND ledger_id = ?
	`, replacementCategoryID, ledgerID).Scan(&replacementType, &replacementSystemKey, &replacementArchived); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errFallbackReplacementInvalid
		}
		return nil, err
	}
	if replacementCategoryID == id || replacementType != categoryType || replacementArchived != 0 || replacementSystemKey != "" {
		return nil, errFallbackReplacementInvalid
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := execRequireRows(tx.ExecContext(ctx, `
		UPDATE categories SET system_key = NULL, updated_at = ?
		WHERE id = ? AND ledger_id = ? AND system_key = ?
	`, now, id, ledgerID, systemKey)); err != nil {
		return nil, errFallbackReplacementInvalid
	}
	if err := execRequireRows(tx.ExecContext(ctx, `
		UPDATE categories SET system_key = ?, updated_at = ?
		WHERE id = ? AND ledger_id = ? AND is_archived = 0 AND type = ? AND system_key IS NULL
	`, systemKey, now, replacementCategoryID, ledgerID, categoryType)); err != nil {
		return nil, errFallbackReplacementInvalid
	}
	if err := execRequireRows(tx.ExecContext(ctx, `
		UPDATE categories SET is_archived = 1, updated_at = ?
		WHERE id = ? AND ledger_id = ? AND system_key IS NULL
	`, now, id, ledgerID)); err != nil {
		return nil, err
	}

	result.FallbackReplaced = true
	result.TransferredSystemKey = systemKey
	result.ReplacementCategoryID = replacementCategoryID
	beforeJSON, err := json.Marshal(map[string]any{
		"archived_id": id, "system_key": systemKey, "is_archived": isArchived == 1,
	})
	if err != nil {
		return nil, err
	}
	afterJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_logs (
			id, ledger_id, actor_user_id, actor_role, action, entity_type,
			entity_id, before_json, after_json, created_at
		) VALUES (?, ?, ?, 'owner', 'metadata_fallback_replace', 'category', ?, ?, ?, ?)
	`, uuid.NewString(), ledgerID, actorUserID, id, string(beforeJSON), string(afterJSON), now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
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

func (r *Repository) activeRuleReferenceCounts(ctx context.Context, ledgerID string) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(result_json, '{}')
		FROM import_rules
		WHERE ledger_id = ? AND COALESCE(status, 'active') = 'active'
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var ruleID, resultJSON string
		if err := rows.Scan(&ruleID, &resultJSON); err != nil {
			return nil, err
		}
		var result struct {
			CategoryID string   `json:"category_id"`
			AccountID  string   `json:"account_id"`
			TagIDs     []string `json:"tag_ids"`
		}
		if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
			return nil, fmt.Errorf("import rule %s has invalid result_json: %w", ruleID, err)
		}
		seen := map[string]struct{}{}
		for _, referenceID := range append([]string{result.CategoryID, result.AccountID}, result.TagIDs...) {
			if referenceID == "" {
				continue
			}
			if _, exists := seen[referenceID]; exists {
				continue
			}
			seen[referenceID] = struct{}{}
			counts[referenceID]++
		}
	}
	return counts, rows.Err()
}

type profileStateItem struct {
	ID         string
	SystemKey  string
	Name       string
	Kind       string
	IsArchived bool
}

func (r *Repository) profileVersionWithTx(ctx context.Context, tx *sql.Tx, ledgerID string) (int, error) {
	var version int
	err := tx.QueryRowContext(ctx, `
		SELECT metadata_profile_version
		FROM ledgers
		WHERE id = ?
	`, ledgerID).Scan(&version)
	return version, err
}

func (r *Repository) listProfileStateWithTx(ctx context.Context, tx *sql.Tx, ledgerID string) ([]profileStateItem, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id,
		       COALESCE(system_key, ''),
		       name,
		       CASE type WHEN 'expense' THEN 'expense_category' ELSE 'income_category' END,
		       is_archived
		FROM categories
		WHERE ledger_id = ?
		UNION ALL
		SELECT id, COALESCE(system_key, ''), name, 'tag', is_archived
		FROM tags
		WHERE ledger_id = ?
	`, ledgerID, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []profileStateItem
	for rows.Next() {
		var item profileStateItem
		var archived int
		if err := rows.Scan(&item.ID, &item.SystemKey, &item.Name, &item.Kind, &archived); err != nil {
			return nil, err
		}
		item.IsArchived = archived == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) updateProfileVersionWithTx(ctx context.Context, tx *sql.Tx, ledgerID string, version int, now time.Time) error {
	return execRequireRows(tx.ExecContext(ctx, `
		UPDATE ledgers
		SET metadata_profile_version = ?, updated_at = ?
		WHERE id = ?
	`, version, now.UTC().Format(time.RFC3339Nano), ledgerID))
}

func (r *Repository) createProfileAuditWithTx(
	ctx context.Context,
	tx *sql.Tx,
	ledgerID string,
	actorUserID string,
	profileKey string,
	beforeJSON []byte,
	afterJSON []byte,
	now time.Time,
) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO audit_logs (
			id, ledger_id, actor_user_id, actor_role, action,
			entity_type, entity_id, before_json, after_json, created_at
		) VALUES (?, ?, ?, 'owner', 'metadata_profile_apply', 'metadata_profile', ?, ?, ?, ?)
	`, uuid.NewString(), ledgerID, actorUserID, profileKey, nullBytes(beforeJSON), nullBytes(afterJSON), now.UTC().Format(time.RFC3339Nano))
	return err
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

func nullBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}
