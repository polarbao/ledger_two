package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ledger_two/internal/importer/classifier"
	"ledger_two/internal/ledger"
)

var errReclassifyBatchChanged = errors.New("import batch is no longer ready for reclassification")

func (r *Repository) LoadClassificationContext(ctx context.Context, ledgerID string) (classifier.Context, error) {
	metadata, err := r.loadClassificationMetadata(ctx, ledgerID)
	if err != nil {
		return classifier.Context{}, err
	}
	rules, err := r.loadClassificationRules(ctx, ledgerID)
	if err != nil {
		return classifier.Context{}, err
	}
	return classifier.Context{
		LedgerID: ledgerID,
		Rules:    rules,
		Metadata: metadata,
		Builtins: classifier.BuiltinV1(),
	}, nil
}

func (r *Repository) loadClassificationMetadata(ctx context.Context, ledgerID string) ([]classifier.MetadataItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ledger_id, COALESCE(system_key, ''),
		       CASE type
		           WHEN 'expense' THEN 'expense_category'
		           WHEN 'income' THEN 'income_category'
		           ELSE 'invalid_category'
		       END,
		       is_archived
		FROM categories
		WHERE ledger_id = ?
		UNION ALL
		SELECT id, ledger_id, COALESCE(system_key, ''), 'tag', is_archived
		FROM tags
		WHERE ledger_id = ?
		UNION ALL
		SELECT id, ledger_id, '', 'account', is_archived
		FROM accounts
		WHERE ledger_id = ?
		ORDER BY 4, 1
	`, ledgerID, ledgerID, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("load classification metadata: %w", err)
	}
	defer rows.Close()

	var result []classifier.MetadataItem
	for rows.Next() {
		var item classifier.MetadataItem
		var archived int
		if err := rows.Scan(&item.ID, &item.LedgerID, &item.SystemKey, &item.Kind, &archived); err != nil {
			return nil, err
		}
		item.IsArchived = archived == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) loadClassificationRules(ctx context.Context, ledgerID string) ([]classifier.Rule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ledger_id, COALESCE(origin, 'manual'), COALESCE(source_type, ''),
		       COALESCE(apply_mode, 'suggest'), COALESCE(confidence, 'high'),
		       COALESCE(match_type, ''), COALESCE(pattern, keyword),
		       amount_min_cents, amount_max_cents, priority, COALESCE(result_json, '{}'),
		       COALESCE(status, 'active'), created_at, created_by_user_id
		FROM import_rules
		WHERE ledger_id = ? AND COALESCE(status, 'active') = 'active'
		ORDER BY priority ASC,
		         CASE COALESCE(origin, 'manual') WHEN 'manual' THEN 0 WHEN 'learned' THEN 1 ELSE 2 END,
		         CASE COALESCE(match_type, '')
		             WHEN 'merchant_equals' THEN 0
		             WHEN 'source_account' THEN 0
		             WHEN 'merchant_contains' THEN 1
		             WHEN 'description_contains' THEN 1
		             WHEN 'amount_range' THEN 2
		             ELSE 99
		         END,
		         created_at DESC,
		         id ASC
	`, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("load classification rules: %w", err)
	}
	defer rows.Close()

	var result []classifier.Rule
	for rows.Next() {
		var rule classifier.Rule
		var minAmount, maxAmount sql.NullInt64
		var resultJSON string
		if err := rows.Scan(
			&rule.ID,
			&rule.LedgerID,
			&rule.Origin,
			&rule.SourceType,
			&rule.ApplyMode,
			&rule.Confidence,
			&rule.MatchType,
			&rule.Pattern,
			&minAmount,
			&maxAmount,
			&rule.Priority,
			&resultJSON,
			&rule.Status,
			&rule.CreatedAt,
			&rule.CreatedByUserID,
		); err != nil {
			return nil, err
		}
		trimmed := strings.TrimSpace(resultJSON)
		if trimmed == "" || trimmed[0] != '{' || json.Unmarshal([]byte(trimmed), &rule.Result) != nil {
			return nil, fmt.Errorf("classification rule %s has invalid result_json", rule.ID)
		}
		rule.AmountMinCents = nullableInt64Pointer(minAmount)
		rule.AmountMaxCents = nullableInt64Pointer(maxAmount)
		result = append(result, rule)
	}
	return result, rows.Err()
}

func nullableInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func (r *Repository) ApplyReclassification(
	ctx context.Context,
	lc ledger.LedgerContext,
	batchID string,
	expectedUpdatedAt string,
	rows []PreviewRow,
	result *ReclassifyResult,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status, updatedAt string
	var expiresAt sql.NullString
	if err := tx.QueryRowContext(ctx, `
		SELECT status, expires_at, updated_at
		FROM import_batches
		WHERE id = ? AND ledger_id = ?
	`, batchID, lc.LedgerID).Scan(&status, &expiresAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errReclassifyBatchChanged
		}
		return err
	}
	if status != batchStatusReady || updatedAt != expectedUpdatedAt {
		return errReclassifyBatchChanged
	}
	if expiresAt.Valid {
		parsed, err := time.Parse(time.RFC3339, expiresAt.String)
		if err != nil || !parsed.After(time.Now()) {
			return errReclassifyBatchChanged
		}
	}

	for _, row := range rows {
		reasonJSON, err := json.Marshal(classificationReasonRecord{
			Code: row.Classification.ReasonCode,
			Text: row.Classification.ReasonText,
		})
		if err != nil {
			return err
		}
		updated, err := tx.ExecContext(ctx, `
			UPDATE import_items
			SET suggested_category_id = ?,
			    suggested_account_id = ?,
			    suggested_tag_ids_json = ?,
			    suggested_rule_id = ?,
			    suggestion_reason = ?,
			    selected_category_id = ?,
			    selected_account_id = ?,
			    selected_tag_ids_json = ?,
			    classification_status = ?,
			    classification_confidence = ?,
			    classification_source = ?,
			    classification_reason_json = ?,
			    matched_rule_ids_json = ?
			WHERE id = ? AND batch_id = ?
			  AND row_status <> ?
			  AND classification_status NOT IN (?, ?)
			  AND COALESCE(classification_source, '') NOT IN (?, ?)
			  AND EXISTS (
			      SELECT 1 FROM import_batches
			      WHERE import_batches.id = import_items.batch_id
			        AND import_batches.ledger_id = ?
			        AND import_batches.status = ?
			  )
		`,
			nullString(row.SuggestedCategoryID), nullString(row.SuggestedAccountID), jsonString(row.SuggestedTagIDs),
			nullString(row.SuggestedRuleID), nullString(row.SuggestionReason),
			nullString(row.SelectedCategoryID), nullString(row.SelectedAccountID), jsonString(row.SelectedTagIDs),
			row.Classification.Status, row.Classification.Confidence, nullString(row.Classification.Source),
			string(reasonJSON), jsonString(row.Classification.MatchedRuleIDs),
			row.ID, batchID,
			RowStatusAdjusted,
			ClassificationStatusManual, ClassificationStatusBulk,
			string(classifier.SourceManual), string(classifier.SourceBulk),
			lc.LedgerID, batchStatusReady,
		)
		if err != nil {
			return err
		}
		affected, err := updated.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return errReclassifyBatchChanged
		}
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	updated, err := tx.ExecContext(ctx, `
		UPDATE import_batches
		SET updated_at = ?
		WHERE id = ? AND ledger_id = ? AND status = ? AND updated_at = ?
	`, now, batchID, lc.LedgerID, batchStatusReady, expectedUpdatedAt)
	if err != nil {
		return err
	}
	affected, err := updated.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errReclassifyBatchChanged
	}

	afterJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_logs (
			id, ledger_id, actor_user_id, actor_role, action, entity_type,
			entity_id, before_json, after_json, created_at
		) VALUES (?, ?, ?, ?, 'import_reclassify', 'import_batch', ?, NULL, ?, ?)
	`, uuid.NewString(), lc.LedgerID, lc.UserID, lc.Role, batchID, string(afterJSON), now); err != nil {
		return err
	}

	return tx.Commit()
}

type classificationReasonRecord struct {
	Code string `json:"code,omitempty"`
	Text string `json:"text,omitempty"`
}
