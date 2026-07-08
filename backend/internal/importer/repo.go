package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreatePreviewBatch(ctx context.Context, batch *PreviewBatch) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO import_batches (
			id, ledger_id, filename, created_by_user_id, status, created_at,
			source_type, file_sha256, total_rows, new_rows, duplicate_rows,
			suspicious_rows, invalid_rows, imported_rows, skipped_rows, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		batch.ID, batch.LedgerID, batch.Filename, batch.CreatedByUserID, batch.Status, batch.CreatedAt,
		batch.SourceType, batch.FileSHA256, batch.TotalRows, batch.NewRows, batch.DuplicateRows,
		batch.SuspiciousRows, batch.InvalidRows, batch.ImportedRows, batch.SkippedRows, batch.UpdatedAt,
	)
	if err != nil {
		return err
	}

	for _, row := range batch.Rows {
		normalizedJSON, err := json.Marshal(row)
		if err != nil {
			return err
		}
		errorCode := sql.NullString{}
		errorMessage := sql.NullString{}
		if row.Error != nil {
			errorCode = sql.NullString{String: row.Error.Code, Valid: true}
			errorMessage = sql.NullString{String: row.Error.Message, Valid: true}
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO import_items (
				id, batch_id, transaction_id, import_hash, status, created_at,
				row_number, source_type, external_order_id, occurred_at, title,
				merchant, description, amount_cents, direction, target_transaction_type,
				duplicate_status, row_status, normalized_json, error_code, error_message,
				suggested_category_id, suggested_account_id, suggested_tag_ids_json,
				selected_category_id, selected_account_id, selected_tag_ids_json, visibility
			) VALUES (?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			row.ID, batch.ID, calculateImportHash(batch.LedgerID, batch.SourceType, row), row.RowStatus, batch.CreatedAt,
			row.RowNumber, batch.SourceType, nullString(row.ExternalOrderID), nullString(row.OccurredAt), row.Title,
			row.Merchant, nullString(row.Description), row.AmountCents, row.Direction, row.TargetTransactionType,
			row.DuplicateStatus, row.RowStatus, string(normalizedJSON), errorCode, errorMessage,
			nullString(row.SuggestedCategoryID), nullString(row.SuggestedAccountID), jsonString(row.SuggestedTagIDs),
			nullString(row.SelectedCategoryID), nullString(row.SelectedAccountID), jsonString(row.SelectedTagIDs), defaultVisibility(row.Visibility),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetPreviewBatch(ctx context.Context, ledgerID string, batchID string) (*PreviewBatch, error) {
	var batch PreviewBatch
	err := r.db.QueryRowContext(ctx, `
		SELECT id, ledger_id, source_type, filename, file_sha256, status,
		       total_rows, new_rows, duplicate_rows, suspicious_rows, invalid_rows,
		       imported_rows, skipped_rows, created_by_user_id, created_at, COALESCE(updated_at, '')
		FROM import_batches
		WHERE id = ? AND ledger_id = ?
	`, batchID, ledgerID).Scan(
		&batch.ID, &batch.LedgerID, &batch.SourceType, &batch.Filename, &batch.FileSHA256, &batch.Status,
		&batch.TotalRows, &batch.NewRows, &batch.DuplicateRows, &batch.SuspiciousRows, &batch.InvalidRows,
		&batch.ImportedRows, &batch.SkippedRows, &batch.CreatedByUserID, &batch.CreatedAt, &batch.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, batch_id, row_number, occurred_at, title, merchant, description,
		       amount_cents, direction, target_transaction_type, duplicate_status,
		       row_status, source_type, external_order_id, error_code, error_message,
		       suggested_category_id, suggested_account_id, suggested_tag_ids_json,
		       selected_category_id, selected_account_id, selected_tag_ids_json, visibility
		FROM import_items
		WHERE batch_id = ?
		ORDER BY row_number ASC
	`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var row PreviewRow
		var occurredAt, description, sourceAccount, externalOrderID, errorCode, errorMessage sql.NullString
		var suggestedCategoryID, suggestedAccountID, suggestedTagIDs sql.NullString
		var selectedCategoryID, selectedAccountID, selectedTagIDs, visibility sql.NullString
		err := rows.Scan(
			&row.ID, &row.BatchID, &row.RowNumber, &occurredAt, &row.Title, &row.Merchant, &description,
			&row.AmountCents, &row.Direction, &row.TargetTransactionType, &row.DuplicateStatus,
			&row.RowStatus, &sourceAccount, &externalOrderID, &errorCode, &errorMessage,
			&suggestedCategoryID, &suggestedAccountID, &suggestedTagIDs,
			&selectedCategoryID, &selectedAccountID, &selectedTagIDs, &visibility,
		)
		if err != nil {
			return nil, err
		}
		row.OccurredAt = valueOf(occurredAt)
		row.Description = valueOf(description)
		row.SourceAccount = valueOf(sourceAccount)
		row.ExternalOrderID = valueOf(externalOrderID)
		row.SuggestedCategoryID = valueOf(suggestedCategoryID)
		row.SuggestedAccountID = valueOf(suggestedAccountID)
		row.SuggestedTagIDs = parseStringList(valueOf(suggestedTagIDs))
		row.SelectedCategoryID = valueOf(selectedCategoryID)
		row.SelectedAccountID = valueOf(selectedAccountID)
		row.SelectedTagIDs = parseStringList(valueOf(selectedTagIDs))
		row.Visibility = defaultVisibility(valueOf(visibility))
		if errorCode.Valid || errorMessage.Valid {
			row.Error = &RowError{Code: errorCode.String, Message: errorMessage.String}
		}
		batch.Rows = append(batch.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &batch, nil
}

func (r *Repository) ExistingImportedHashes(ctx context.Context, ledgerID string, hashes []string) (map[string]bool, error) {
	existing := map[string]bool{}
	if len(hashes) == 0 {
		return existing, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(hashes)), ",")
	args := make([]any, 0, len(hashes)+1)
	args = append(args, ledgerID)
	for _, hash := range hashes {
		args = append(args, hash)
	}

	query := `
		SELECT i.import_hash
		FROM import_items i
		JOIN import_batches b ON b.id = i.batch_id
		WHERE b.ledger_id = ? AND i.status = 'imported' AND i.import_hash IN (` + placeholders + `)
	`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		existing[hash] = true
	}
	return existing, rows.Err()
}

func (r *Repository) GetPreviewRow(ctx context.Context, ledgerID string, batchID string, rowID string) (*PreviewBatch, *PreviewRow, error) {
	batch, err := r.GetPreviewBatch(ctx, ledgerID, batchID)
	if err != nil {
		return nil, nil, err
	}
	for i := range batch.Rows {
		if batch.Rows[i].ID == rowID {
			return batch, &batch.Rows[i], nil
		}
	}
	return batch, nil, sql.ErrNoRows
}

func (r *Repository) UpdatePreviewRow(ctx context.Context, batch *PreviewBatch, row PreviewRow, adjustment RowAdjustment) (*PreviewBatch, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)
	adjustmentJSON, err := json.Marshal(adjustment)
	if err != nil {
		return nil, err
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE import_items
		SET target_transaction_type = ?,
		    row_status = ?,
		    status = ?,
		    selected_category_id = ?,
		    selected_account_id = ?,
		    selected_tag_ids_json = ?,
		    visibility = ?,
		    user_adjustment_json = ?
		WHERE id = ? AND batch_id = ?
	`,
		row.TargetTransactionType,
		row.RowStatus,
		row.RowStatus,
		nullString(row.SelectedCategoryID),
		nullString(row.SelectedAccountID),
		jsonString(row.SelectedTagIDs),
		defaultVisibility(row.Visibility),
		string(adjustmentJSON),
		row.ID,
		batch.ID,
	)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, sql.ErrNoRows
	}

	recountBatch(batch)
	_, err = tx.ExecContext(ctx, `
		UPDATE import_batches
		SET total_rows = ?,
		    new_rows = ?,
		    duplicate_rows = ?,
		    suspicious_rows = ?,
		    invalid_rows = ?,
		    imported_rows = ?,
		    skipped_rows = ?,
		    updated_at = ?
		WHERE id = ?
	`,
		batch.TotalRows,
		batch.NewRows,
		batch.DuplicateRows,
		batch.SuspiciousRows,
		batch.InvalidRows,
		batch.ImportedRows,
		batch.SkippedRows,
		now,
		batch.ID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetPreviewBatch(ctx, batch.LedgerID, batch.ID)
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func valueOf(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func jsonString(values []string) string {
	if values == nil {
		return "[]"
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func parseStringList(value string) []string {
	if value == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(value), &list); err != nil {
		return nil
	}
	return list
}

func defaultVisibility(value string) string {
	if value == "" {
		return "private"
	}
	return value
}
