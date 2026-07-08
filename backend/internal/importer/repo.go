package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
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
				duplicate_status, row_status, normalized_json, error_code, error_message
			) VALUES (?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			row.ID, batch.ID, calculateImportHash(batch.LedgerID, batch.SourceType, row), row.RowStatus, batch.CreatedAt,
			row.RowNumber, batch.SourceType, nullString(row.ExternalOrderID), nullString(row.OccurredAt), row.Title,
			row.Merchant, nullString(row.Description), row.AmountCents, row.Direction, row.TargetTransactionType,
			row.DuplicateStatus, row.RowStatus, string(normalizedJSON), errorCode, errorMessage,
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
		       row_status, source_type, external_order_id, error_code, error_message
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
		err := rows.Scan(
			&row.ID, &row.BatchID, &row.RowNumber, &occurredAt, &row.Title, &row.Merchant, &description,
			&row.AmountCents, &row.Direction, &row.TargetTransactionType, &row.DuplicateStatus,
			&row.RowStatus, &sourceAccount, &externalOrderID, &errorCode, &errorMessage,
		)
		if err != nil {
			return nil, err
		}
		row.OccurredAt = valueOf(occurredAt)
		row.Description = valueOf(description)
		row.SourceAccount = valueOf(sourceAccount)
		row.ExternalOrderID = valueOf(externalOrderID)
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
