package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ledger_two/internal/ledger"
)

const (
	batchStatusCommitted    = "committed"
	auditActionImportCommit = "import_commit"
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
		       imported_rows, skipped_rows, failed_rows, created_by_user_id, created_at,
		       COALESCE(updated_at, ''), COALESCE(committed_at, ''), COALESCE(expires_at, '')
		FROM import_batches
		WHERE id = ? AND ledger_id = ?
	`, batchID, ledgerID).Scan(
		&batch.ID, &batch.LedgerID, &batch.SourceType, &batch.Filename, &batch.FileSHA256, &batch.Status,
		&batch.TotalRows, &batch.NewRows, &batch.DuplicateRows, &batch.SuspiciousRows, &batch.InvalidRows,
		&batch.ImportedRows, &batch.SkippedRows, &batch.FailedRows, &batch.CreatedByUserID, &batch.CreatedAt,
		&batch.UpdatedAt, &batch.CommittedAt, &batch.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, batch_id, row_number, occurred_at, title, merchant, description,
		       amount_cents, direction, target_transaction_type, duplicate_status,
		       row_status, source_type, external_order_id, error_code, error_message,
		       suggested_category_id, suggested_account_id, suggested_tag_ids_json,
		       selected_category_id, selected_account_id, selected_tag_ids_json, visibility, import_hash
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
			&selectedCategoryID, &selectedAccountID, &selectedTagIDs, &visibility, &row.ImportHash,
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
	args := make([]any, 0, len(hashes)*2+2)
	args = append(args, ledgerID)
	for _, hash := range hashes {
		args = append(args, hash)
	}
	args = append(args, ledgerID)
	for _, hash := range hashes {
		args = append(args, hash)
	}

	query := `
		SELECT import_hash
		FROM transaction_import_refs
		WHERE ledger_id = ? AND import_hash IN (` + placeholders + `)
		UNION
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

func (r *Repository) CommitPreviewBatch(ctx context.Context, lc ledger.LedgerContext, batch *PreviewBatch) (*CommitResult, error) {
	dbTx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer dbTx.Rollback()

	var currentStatus string
	err = dbTx.QueryRowContext(ctx, `
		SELECT status
		FROM import_batches
		WHERE id = ? AND ledger_id = ?
	`, batch.ID, lc.LedgerID).Scan(&currentStatus)
	if err != nil {
		return nil, err
	}
	if currentStatus != "ready" {
		return nil, fmt.Errorf("import batch status is %s", currentStatus)
	}

	now := time.Now().Format(time.RFC3339)
	result := &CommitResult{
		BatchID:      batch.ID,
		Status:       batchStatusCommitted,
		FailedRows:   0,
		SkippedRows:  0,
		ImportedRows: 0,
	}

	for _, row := range batch.Rows {
		if row.RowStatus == RowStatusSkipped || row.TargetTransactionType == TargetTransactionSkipped {
			if err := markRowSkipped(ctx, dbTx, row.ID, batch.ID); err != nil {
				return nil, err
			}
			result.SkippedRows++
			continue
		}

		txID := uuid.NewString()
		if err := insertImportedTransaction(ctx, dbTx, lc, txID, row, now); err != nil {
			return nil, err
		}
		if err := insertTransactionImportRef(ctx, dbTx, lc.LedgerID, txID, batch.ID, batch.SourceType, row, now); err != nil {
			return nil, err
		}
		if err := markRowImported(ctx, dbTx, row.ID, batch.ID, txID); err != nil {
			return nil, err
		}

		result.ImportedRows++
		result.GeneratedTransactionIDs = append(result.GeneratedTransactionIDs, txID)
	}

	_, err = dbTx.ExecContext(ctx, `
		UPDATE import_batches
		SET status = ?,
		    imported_rows = ?,
		    skipped_rows = ?,
		    failed_rows = ?,
		    updated_at = ?,
		    committed_at = ?
		WHERE id = ? AND ledger_id = ?
	`, batchStatusCommitted, result.ImportedRows, result.SkippedRows, result.FailedRows, now, now, batch.ID, lc.LedgerID)
	if err != nil {
		return nil, err
	}

	auditJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	_, err = dbTx.ExecContext(ctx, `
		INSERT INTO audit_logs (id, ledger_id, actor_user_id, action, entity_type, entity_id, before_json, after_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?)
	`, uuid.NewString(), lc.LedgerID, lc.UserID, auditActionImportCommit, "import_batch", batch.ID, string(auditJSON), now)
	if err != nil {
		return nil, err
	}

	if err := dbTx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func insertImportedTransaction(ctx context.Context, tx *sql.Tx, lc ledger.LedgerContext, transactionID string, row PreviewRow, now string) error {
	visibility := defaultVisibility(row.Visibility)
	note := strings.TrimSpace(row.Description)
	if row.Merchant != "" && note != "" {
		note = row.Merchant + " | " + note
	} else if row.Merchant != "" {
		note = row.Merchant
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id,
			visibility, split_method, note, attachment_paths, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 'CNY', ?, ?, ?, ?, ?, ?, ?, NULL, ?, NULL, 'normal', ?, ?)
	`,
		transactionID,
		lc.LedgerID,
		row.TargetTransactionType,
		row.Title,
		row.AmountCents,
		row.OccurredAt,
		lc.UserID,
		lc.UserID,
		lc.UserID,
		nullString(row.SelectedAccountID),
		nullString(row.SelectedCategoryID),
		visibility,
		nullString(note),
		now,
		now,
	)
	if err != nil {
		return err
	}

	return attachSelectedTagIDs(ctx, tx, transactionID, row.SelectedTagIDs)
}

func attachSelectedTagIDs(ctx context.Context, tx *sql.Tx, transactionID string, tagIDs []string) error {
	for _, tagID := range tagIDs {
		if strings.TrimSpace(tagID) == "" {
			continue
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO transaction_tags (transaction_id, tag_id)
			VALUES (?, ?)
		`, transactionID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertTransactionImportRef(ctx context.Context, tx *sql.Tx, ledgerID string, transactionID string, batchID string, sourceType string, row PreviewRow, now string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO transaction_import_refs (
			id, ledger_id, transaction_id, import_batch_id, import_row_id,
			import_hash, external_order_id, source_type, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.NewString(), ledgerID, transactionID, batchID, row.ID, row.ImportHash, nullString(row.ExternalOrderID), sourceType, now)
	return err
}

func markRowImported(ctx context.Context, tx *sql.Tx, rowID string, batchID string, transactionID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE import_items
		SET transaction_id = ?,
		    generated_transaction_id = ?,
		    status = ?,
		    row_status = ?
		WHERE id = ? AND batch_id = ?
	`, transactionID, transactionID, RowStatusImported, RowStatusImported, rowID, batchID)
	return err
}

func markRowSkipped(ctx context.Context, tx *sql.Tx, rowID string, batchID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE import_items
		SET status = ?,
		    row_status = ?,
		    target_transaction_type = ?
		WHERE id = ? AND batch_id = ?
	`, RowStatusSkipped, RowStatusSkipped, TargetTransactionSkipped, rowID, batchID)
	return err
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
