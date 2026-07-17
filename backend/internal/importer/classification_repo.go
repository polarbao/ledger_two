package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"ledger_two/internal/importer/classifier"
)

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
