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
	batchStatusReady         = "ready"
	batchStatusCommitted     = "committed"
	batchStatusFailed        = "failed"
	batchStatusExpired       = "expired"
	auditActionImportCommit  = "import_commit"
	auditActionImportDiscard = "import_batch_discard"
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

	parserMetadataJSON, err := json.Marshal(batch.ParserMetadata)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO import_batches (
			id, ledger_id, filename, created_by_user_id, status, created_at,
			source_type, file_sha256, total_rows, new_rows, duplicate_rows,
			suspicious_rows, invalid_rows, imported_rows, skipped_rows, updated_at,
			file_format, parser_metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		batch.ID, batch.LedgerID, batch.Filename, batch.CreatedByUserID, batch.Status, batch.CreatedAt,
		batch.SourceType, batch.FileSHA256, batch.TotalRows, batch.NewRows, batch.DuplicateRows,
		batch.SuspiciousRows, batch.InvalidRows, batch.ImportedRows, batch.SkippedRows, batch.UpdatedAt,
		batch.FileFormat, string(parserMetadataJSON),
	)
	if err != nil {
		return err
	}

	for _, row := range batch.Rows {
		normalizedJSON, err := json.Marshal(row)
		if err != nil {
			return err
		}
		reasonJSON, err := json.Marshal(classificationReasonRecord{
			Code: row.Classification.ReasonCode,
			Text: row.Classification.ReasonText,
		})
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
				suggested_rule_id, suggestion_reason,
				selected_category_id, selected_account_id, selected_tag_ids_json, visibility,
				classification_status, classification_confidence, classification_source,
				classification_reason_json, matched_rule_ids_json
			) VALUES (?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			row.ID, batch.ID, calculateImportHash(batch.LedgerID, batch.SourceType, row), row.RowStatus, batch.CreatedAt,
			row.RowNumber, batch.SourceType, nullString(row.ExternalOrderID), nullString(row.OccurredAt), row.Title,
			row.Merchant, nullString(row.Description), row.AmountCents, row.Direction, row.TargetTransactionType,
			row.DuplicateStatus, row.RowStatus, string(normalizedJSON), errorCode, errorMessage,
			nullString(row.SuggestedCategoryID), nullString(row.SuggestedAccountID), jsonString(row.SuggestedTagIDs),
			nullString(row.SuggestedRuleID), nullString(row.SuggestionReason),
			nullString(row.SelectedCategoryID), nullString(row.SelectedAccountID), jsonString(row.SelectedTagIDs), defaultVisibility(row.Visibility),
			row.Classification.Status, row.Classification.Confidence, nullString(row.Classification.Source),
			string(reasonJSON), jsonString(row.Classification.MatchedRuleIDs),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetPreviewBatch(ctx context.Context, ledgerID string, batchID string) (*PreviewBatch, error) {
	var batch PreviewBatch
	var parserMetadataJSON string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, ledger_id, source_type, filename, file_sha256, status,
		       total_rows, new_rows, duplicate_rows, suspicious_rows, invalid_rows,
		       imported_rows, skipped_rows, failed_rows, created_by_user_id, created_at,
		       COALESCE(updated_at, ''), COALESCE(committed_at, ''), COALESCE(expires_at, ''),
		       file_format, parser_metadata_json
		FROM import_batches
		WHERE id = ? AND ledger_id = ?
	`, batchID, ledgerID).Scan(
		&batch.ID, &batch.LedgerID, &batch.SourceType, &batch.Filename, &batch.FileSHA256, &batch.Status,
		&batch.TotalRows, &batch.NewRows, &batch.DuplicateRows, &batch.SuspiciousRows, &batch.InvalidRows,
		&batch.ImportedRows, &batch.SkippedRows, &batch.FailedRows, &batch.CreatedByUserID, &batch.CreatedAt,
		&batch.UpdatedAt, &batch.CommittedAt, &batch.ExpiresAt, &batch.FileFormat, &parserMetadataJSON,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(parserMetadataJSON), &batch.ParserMetadata); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, batch_id, row_number, occurred_at, title, merchant, description,
		       amount_cents, direction, target_transaction_type, duplicate_status,
		       row_status, normalized_json, external_order_id, error_code, error_message,
		       suggested_category_id, suggested_account_id, suggested_tag_ids_json,
		       suggested_rule_id, suggestion_reason,
		       selected_category_id, selected_account_id, selected_tag_ids_json, visibility, import_hash,
		       classification_status, classification_confidence, classification_source,
		       classification_reason_json, matched_rule_ids_json
		FROM import_items
		WHERE batch_id = ?
		  AND EXISTS (
			  SELECT 1
			  FROM import_batches
			  WHERE import_batches.id = import_items.batch_id
			    AND import_batches.ledger_id = ?
		  )
		ORDER BY row_number ASC
	`, batchID, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var row PreviewRow
		var occurredAt, description, normalizedJSON, externalOrderID, errorCode, errorMessage sql.NullString
		var suggestedCategoryID, suggestedAccountID, suggestedTagIDs, suggestedRuleID, suggestionReason sql.NullString
		var selectedCategoryID, selectedAccountID, selectedTagIDs, visibility sql.NullString
		var classificationSource, classificationReasonJSON, matchedRuleIDsJSON sql.NullString
		err := rows.Scan(
			&row.ID, &row.BatchID, &row.RowNumber, &occurredAt, &row.Title, &row.Merchant, &description,
			&row.AmountCents, &row.Direction, &row.TargetTransactionType, &row.DuplicateStatus,
			&row.RowStatus, &normalizedJSON, &externalOrderID, &errorCode, &errorMessage,
			&suggestedCategoryID, &suggestedAccountID, &suggestedTagIDs, &suggestedRuleID, &suggestionReason,
			&selectedCategoryID, &selectedAccountID, &selectedTagIDs, &visibility, &row.ImportHash,
			&row.Classification.Status, &row.Classification.Confidence, &classificationSource,
			&classificationReasonJSON, &matchedRuleIDsJSON,
		)
		if err != nil {
			return nil, err
		}
		row.OccurredAt = valueOf(occurredAt)
		row.Description = valueOf(description)
		if normalizedJSON.Valid && normalizedJSON.String != "" {
			var persisted PreviewRow
			if err := json.Unmarshal([]byte(normalizedJSON.String), &persisted); err != nil {
				return nil, err
			}
			row.SourceAccount = persisted.SourceAccount
			row.SuspiciousReason = persisted.SuspiciousReason
		}
		row.ExternalOrderID = valueOf(externalOrderID)
		row.SuggestedCategoryID = valueOf(suggestedCategoryID)
		row.SuggestedAccountID = valueOf(suggestedAccountID)
		row.SuggestedTagIDs = parseStringList(valueOf(suggestedTagIDs))
		row.SuggestedRuleID = valueOf(suggestedRuleID)
		row.SuggestionReason = valueOf(suggestionReason)
		row.SelectedCategoryID = valueOf(selectedCategoryID)
		row.SelectedAccountID = valueOf(selectedAccountID)
		row.SelectedTagIDs = parseStringList(valueOf(selectedTagIDs))
		row.Visibility = defaultVisibility(valueOf(visibility))
		row.Classification.Source = valueOf(classificationSource)
		if classificationReasonJSON.Valid && classificationReasonJSON.String != "" {
			var reason classificationReasonRecord
			if err := json.Unmarshal([]byte(classificationReasonJSON.String), &reason); err != nil {
				return nil, err
			}
			row.Classification.ReasonCode = reason.Code
			row.Classification.ReasonText = reason.Text
		}
		row.Classification.MatchedRuleIDs = parseStringList(valueOf(matchedRuleIDsJSON))
		row.Classification.SuggestedCategoryID = row.SuggestedCategoryID
		row.Classification.SuggestedAccountID = row.SuggestedAccountID
		row.Classification.SuggestedTagIDs = copyStrings(row.SuggestedTagIDs)
		normalizeClassification(&row.Classification)
		if errorCode.Valid || errorMessage.Valid {
			row.Error = &RowError{Code: errorCode.String, Message: errorMessage.String}
		}
		batch.Rows = append(batch.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	recountClassificationSummary(&batch)
	return &batch, nil
}

func (r *Repository) GetPreviewBatchStatus(ctx context.Context, ledgerID, batchID string) (string, error) {
	var status string
	err := r.db.QueryRowContext(ctx, `
		SELECT status
		FROM import_batches
		WHERE id = ? AND ledger_id = ?
	`, batchID, ledgerID).Scan(&status)
	return status, err
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

func (r *Repository) ValidateMetadataSelections(ctx context.Context, ledgerID string, categoryID string, accountID string, tagIDs []string) error {
	if strings.TrimSpace(categoryID) != "" {
		var exists bool
		if err := r.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM categories WHERE id = ? AND ledger_id = ?)
		`, strings.TrimSpace(categoryID), ledgerID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return sql.ErrNoRows
		}
	}

	if strings.TrimSpace(accountID) != "" {
		var exists bool
		if err := r.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM accounts WHERE id = ? AND ledger_id = ?)
		`, strings.TrimSpace(accountID), ledgerID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return sql.ErrNoRows
		}
	}

	seenTagIDs := make(map[string]struct{}, len(tagIDs))
	for _, rawTagID := range tagIDs {
		tagID := strings.TrimSpace(rawTagID)
		if tagID == "" {
			continue
		}
		if _, exists := seenTagIDs[tagID]; exists {
			continue
		}
		seenTagIDs[tagID] = struct{}{}

		var exists bool
		if err := r.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM tags WHERE id = ? AND ledger_id = ?)
		`, tagID, ledgerID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return sql.ErrNoRows
		}
	}

	return nil
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

	now := time.Now().UTC().Format(time.RFC3339Nano)
	adjustmentJSON, err := json.Marshal(adjustment)
	if err != nil {
		return nil, err
	}
	classificationReasonJSON, err := json.Marshal(classificationReasonRecord{
		Code: row.Classification.ReasonCode,
		Text: row.Classification.ReasonText,
	})
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
		    user_adjustment_json = ?,
		    classification_status = ?,
		    classification_confidence = ?,
		    classification_source = ?,
		    classification_reason_json = ?,
		    matched_rule_ids_json = ?
		WHERE id = ? AND batch_id = ?
		  AND EXISTS (
			  SELECT 1
			  FROM import_batches
			  WHERE import_batches.id = import_items.batch_id
			    AND import_batches.ledger_id = ?
		  )
	`,
		row.TargetTransactionType,
		row.RowStatus,
		row.RowStatus,
		nullString(row.SelectedCategoryID),
		nullString(row.SelectedAccountID),
		jsonString(row.SelectedTagIDs),
		defaultVisibility(row.Visibility),
		string(adjustmentJSON),
		row.Classification.Status,
		row.Classification.Confidence,
		nullString(row.Classification.Source),
		string(classificationReasonJSON),
		jsonString(row.Classification.MatchedRuleIDs),
		row.ID,
		batch.ID,
		batch.LedgerID,
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
		SET status = ?,
		    total_rows = ?,
		    new_rows = ?,
		    duplicate_rows = ?,
		    suspicious_rows = ?,
		    invalid_rows = ?,
		    imported_rows = ?,
		    skipped_rows = ?,
		    failed_rows = ?,
		    updated_at = ?
		WHERE id = ? AND ledger_id = ?
	`,
		batchStatusReady,
		batch.TotalRows,
		batch.NewRows,
		batch.DuplicateRows,
		batch.SuspiciousRows,
		batch.InvalidRows,
		batch.ImportedRows,
		batch.SkippedRows,
		batch.FailedRows,
		now,
		batch.ID,
		batch.LedgerID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetPreviewBatch(ctx, batch.LedgerID, batch.ID)
}

func (r *Repository) MarkPreviewBatchFailed(ctx context.Context, ledgerID string, batchID string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE import_batches
		SET status = ?,
		    failed_rows = CASE WHEN failed_rows > 0 THEN failed_rows ELSE 1 END,
		    updated_at = ?
		WHERE id = ? AND ledger_id = ? AND status = ?
	`, batchStatusFailed, time.Now().Format(time.RFC3339), batchID, ledgerID, batchStatusReady)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
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
			if err := markRowSkipped(ctx, dbTx, lc.LedgerID, row.ID, batch.ID); err != nil {
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
		if err := markRowImported(ctx, dbTx, lc.LedgerID, row.ID, batch.ID, txID); err != nil {
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

func (r *Repository) DiscardPreviewBatch(ctx context.Context, lc ledger.LedgerContext, batchID, reason string) (*DiscardImportBatchResult, error) {
	dbTx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer dbTx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := dbTx.ExecContext(ctx, `
		UPDATE import_batches
		SET status = ?, updated_at = ?, expires_at = ?
		WHERE id = ? AND ledger_id = ? AND status = ?
	`, batchStatusExpired, now, now, batchID, lc.LedgerID, batchStatusReady)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected != 1 {
		return nil, sql.ErrNoRows
	}

	discarded := &DiscardImportBatchResult{
		BatchID:       batchID,
		Status:        batchStatusExpired,
		DiscardReason: reason,
	}
	beforeJSON, err := json.Marshal(map[string]string{"status": batchStatusReady})
	if err != nil {
		return nil, err
	}
	afterJSON, err := json.Marshal(discarded)
	if err != nil {
		return nil, err
	}
	if _, err := dbTx.ExecContext(ctx, `
		INSERT INTO audit_logs (
			id, ledger_id, actor_user_id, actor_role, action, entity_type,
			entity_id, before_json, after_json, created_at
		) VALUES (?, ?, ?, ?, ?, 'import_batch', ?, ?, ?, ?)
	`, uuid.NewString(), lc.LedgerID, lc.UserID, lc.Role, auditActionImportDiscard,
		batchID, string(beforeJSON), string(afterJSON), now); err != nil {
		return nil, err
	}

	if err := dbTx.Commit(); err != nil {
		return nil, err
	}
	return discarded, nil
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

	return attachSelectedTagIDs(ctx, tx, lc.LedgerID, transactionID, row.SelectedTagIDs)
}

func attachSelectedTagIDs(ctx context.Context, tx *sql.Tx, ledgerID string, transactionID string, tagIDs []string) error {
	for _, tagID := range tagIDs {
		if strings.TrimSpace(tagID) == "" {
			continue
		}
		result, err := tx.ExecContext(ctx, `
			INSERT INTO transaction_tags (transaction_id, tag_id)
			SELECT ?, id
			FROM tags
			WHERE id = ? AND ledger_id = ?
		`, transactionID, tagID, ledgerID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return sql.ErrNoRows
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

func markRowImported(ctx context.Context, tx *sql.Tx, ledgerID string, rowID string, batchID string, transactionID string) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE import_items
		SET transaction_id = ?,
		    generated_transaction_id = ?,
		    status = ?,
		    row_status = ?
		WHERE id = ? AND batch_id = ?
		  AND EXISTS (
			  SELECT 1
			  FROM import_batches
			  WHERE import_batches.id = import_items.batch_id
			    AND import_batches.ledger_id = ?
		  )
	`, transactionID, transactionID, RowStatusImported, RowStatusImported, rowID, batchID, ledgerID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func markRowSkipped(ctx context.Context, tx *sql.Tx, ledgerID string, rowID string, batchID string) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE import_items
		SET status = ?,
		    row_status = ?,
		    target_transaction_type = ?
		WHERE id = ? AND batch_id = ?
		  AND EXISTS (
			  SELECT 1
			  FROM import_batches
			  WHERE import_batches.id = import_items.batch_id
			    AND import_batches.ledger_id = ?
		  )
	`, RowStatusSkipped, RowStatusSkipped, TargetTransactionSkipped, rowID, batchID, ledgerID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) CreateImportRule(ctx context.Context, ledgerID string, userID string, ruleID string, req ImportRuleUpsertRequest) (*ImportRuleResponse, error) {
	now := time.Now().Format(time.RFC3339)
	resultJSON, err := json.Marshal(req.Result)
	if err != nil {
		return nil, err
	}
	priority := importRulePriority(req.Priority)
	name := importRuleName(req.Name, req.Pattern)

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO import_rules (
			id, ledger_id, keyword, category_id, tag_names, account_id,
			created_by_user_id, created_at, updated_at,
			name, match_type, pattern, amount_min_cents, amount_max_cents,
			priority, result_json, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active')
	`,
		ruleID,
		ledgerID,
		req.Pattern,
		nullString(req.Result.CategoryID),
		nullString(strings.Join(req.Result.TagIDs, ",")),
		nullString(req.Result.AccountID),
		userID,
		now,
		now,
		name,
		req.MatchType,
		req.Pattern,
		nullInt64(req.AmountMinCents),
		nullInt64(req.AmountMaxCents),
		priority,
		string(resultJSON),
	)
	if err != nil {
		return nil, err
	}

	return r.GetImportRule(ctx, ledgerID, ruleID)
}

func (r *Repository) UpdateImportRule(ctx context.Context, ledgerID string, ruleID string, req ImportRuleUpsertRequest) (*ImportRuleResponse, error) {
	resultJSON, err := json.Marshal(req.Result)
	if err != nil {
		return nil, err
	}
	now := time.Now().Format(time.RFC3339)
	priority := importRulePriority(req.Priority)
	name := importRuleName(req.Name, req.Pattern)

	res, err := r.db.ExecContext(ctx, `
		UPDATE import_rules
		SET keyword = ?,
		    category_id = ?,
		    tag_names = ?,
		    account_id = ?,
		    name = ?,
		    match_type = ?,
		    pattern = ?,
		    amount_min_cents = ?,
		    amount_max_cents = ?,
		    priority = ?,
		    result_json = ?,
		    updated_at = ?
		WHERE id = ? AND ledger_id = ?
	`, req.Pattern, nullString(req.Result.CategoryID), nullString(strings.Join(req.Result.TagIDs, ",")),
		nullString(req.Result.AccountID), name, req.MatchType, req.Pattern, nullInt64(req.AmountMinCents),
		nullInt64(req.AmountMaxCents), priority, string(resultJSON), now, ruleID, ledgerID)
	if err != nil {
		return nil, err
	}
	if affected, err := res.RowsAffected(); err != nil {
		return nil, err
	} else if affected == 0 {
		return nil, sql.ErrNoRows
	}

	return r.GetImportRule(ctx, ledgerID, ruleID)
}

func (r *Repository) ListImportRules(ctx context.Context, ledgerID string, status string) ([]ImportRuleResponse, error) {
	query := `
		SELECT id, ledger_id, COALESCE(name, ''), COALESCE(match_type, ''), COALESCE(pattern, keyword),
		       amount_min_cents, amount_max_cents, priority, COALESCE(result_json, '{}'),
		       COALESCE(status, 'active'), created_by_user_id, created_at, updated_at, COALESCE(archived_at, '')
		FROM import_rules
		WHERE ledger_id = ?
	`
	args := []any{ledgerID}
	if status != "" && status != "all" {
		query += " AND COALESCE(status, 'active') = ?"
		args = append(args, status)
	}
	query += " ORDER BY priority ASC, created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []ImportRuleResponse{}
	for rows.Next() {
		var record importRuleRecord
		if err := rows.Scan(
			&record.ID,
			&record.LedgerID,
			&record.Name,
			&record.MatchType,
			&record.Pattern,
			&record.AmountMinCents,
			&record.AmountMaxCents,
			&record.Priority,
			&record.ResultJSON,
			&record.Status,
			&record.CreatedByUserID,
			&record.CreatedAt,
			&record.UpdatedAt,
			&record.ArchivedAt,
		); err != nil {
			return nil, err
		}
		resp, err := importRuleRecordToResponse(record)
		if err != nil {
			return nil, err
		}
		list = append(list, resp)
	}
	return list, rows.Err()
}

func (r *Repository) GetImportRule(ctx context.Context, ledgerID string, ruleID string) (*ImportRuleResponse, error) {
	var record importRuleRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, ledger_id, COALESCE(name, ''), COALESCE(match_type, ''), COALESCE(pattern, keyword),
		       amount_min_cents, amount_max_cents, priority, COALESCE(result_json, '{}'),
		       COALESCE(status, 'active'), created_by_user_id, created_at, updated_at, COALESCE(archived_at, '')
		FROM import_rules
		WHERE id = ? AND ledger_id = ?
	`, ruleID, ledgerID).Scan(
		&record.ID,
		&record.LedgerID,
		&record.Name,
		&record.MatchType,
		&record.Pattern,
		&record.AmountMinCents,
		&record.AmountMaxCents,
		&record.Priority,
		&record.ResultJSON,
		&record.Status,
		&record.CreatedByUserID,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.ArchivedAt,
	)
	if err != nil {
		return nil, err
	}
	resp, err := importRuleRecordToResponse(record)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (r *Repository) SetImportRuleStatus(ctx context.Context, ledgerID string, ruleID string, status string) (*ImportRuleResponse, error) {
	now := time.Now().Format(time.RFC3339)
	archivedAt := sql.NullString{}
	if status == "archived" {
		archivedAt = sql.NullString{String: now, Valid: true}
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE import_rules
		SET status = ?,
		    archived_at = ?,
		    updated_at = ?
		WHERE id = ? AND ledger_id = ?
	`, status, archivedAt, now, ruleID, ledgerID)
	if err != nil {
		return nil, err
	}
	if affected, err := res.RowsAffected(); err != nil {
		return nil, err
	} else if affected == 0 {
		return nil, sql.ErrNoRows
	}
	return r.GetImportRule(ctx, ledgerID, ruleID)
}

func (r *Repository) CreateImportRuleAudit(ctx context.Context, ledgerID string, userID string, action string, ruleID string, after any) error {
	now := time.Now().Format(time.RFC3339)
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, ledger_id, actor_user_id, action, entity_type, entity_id, before_json, after_json, created_at)
		VALUES (?, ?, ?, ?, 'import_rule', ?, NULL, ?, ?)
	`, uuid.NewString(), ledgerID, userID, action, ruleID, string(afterJSON), now)
	return err
}

func (r *Repository) ActiveMetadataExists(ctx context.Context, ledgerID string, table string, id string) (bool, error) {
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE ledger_id = ? AND id = ? AND COALESCE(is_archived, 0) = 0)", table)
	err := r.db.QueryRowContext(ctx, query, ledgerID, id).Scan(&exists)
	return exists, err
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func importRulePriority(value *int) int {
	if value == nil {
		return 100
	}
	return *value
}

func importRuleName(name string, pattern string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(pattern)
}

func importRuleRecordToResponse(record importRuleRecord) (ImportRuleResponse, error) {
	var result ImportRuleResult
	if record.ResultJSON != "" && record.ResultJSON != "{}" {
		if err := json.Unmarshal([]byte(record.ResultJSON), &result); err != nil {
			return ImportRuleResponse{}, err
		}
	}
	resp := ImportRuleResponse{
		ID:              record.ID,
		Name:            record.Name,
		MatchType:       record.MatchType,
		Pattern:         record.Pattern,
		Priority:        record.Priority,
		Status:          record.Status,
		Result:          result,
		CreatedByUserID: record.CreatedByUserID,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
		ArchivedAt:      record.ArchivedAt,
	}
	if record.AmountMinCents.Valid {
		value := record.AmountMinCents.Int64
		resp.AmountMinCents = &value
	}
	if record.AmountMaxCents.Valid {
		value := record.AmountMaxCents.Int64
		resp.AmountMaxCents = &value
	}
	return resp, nil
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
