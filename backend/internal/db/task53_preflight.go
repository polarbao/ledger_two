package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	task53UpgradeSourceVersion int64 = 21
	task53TargetSchemaVersion  int64 = 22
)

type task53MigrationSnapshot struct {
	metrics map[string]int64
}

func prepareTask53Upgrade(ctx context.Context, database *sql.DB, currentVersion int64) (*task53MigrationSnapshot, error) {
	switch currentVersion {
	case 0, task53TargetSchemaVersion:
		return nil, nil
	case task53UpgradeSourceVersion:
	default:
		return nil, fmt.Errorf("task53 upgrade requires schema version %d, got %d", task53UpgradeSourceVersion, currentVersion)
	}

	if err := runTask53IntegrityChecks(ctx, database); err != nil {
		return nil, err
	}
	metrics, err := captureTask53MigrationMetrics(ctx, database)
	if err != nil {
		return nil, fmt.Errorf("capture task53 migration snapshot: %w", err)
	}
	return &task53MigrationSnapshot{metrics: metrics}, nil
}

func runTask53IntegrityChecks(ctx context.Context, database *sql.DB) error {
	results, err := readQuickCheckResults(ctx, database)
	if err != nil {
		return fmt.Errorf("task53 preflight quick_check: %w", err)
	}
	if err := validateQuickCheckResults(results); err != nil {
		return fmt.Errorf("task53 preflight: %w", err)
	}
	if err := validateForeignKeys(ctx, database); err != nil {
		return fmt.Errorf("task53 preflight: %w", err)
	}
	for _, table := range []string{"categories", "tags"} {
		exists, err := task53ColumnExists(ctx, database, table, "system_key")
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("task53 preflight: unexpected existing %s.system_key column", table)
		}
	}
	if err := validateTask53RuleJSON(ctx, database); err != nil {
		return err
	}
	if err := validateTask53ImportItemJSON(ctx, database); err != nil {
		return err
	}
	if err := validateTask53ImportMetadataReferences(ctx, database); err != nil {
		return err
	}
	return nil
}

func task53ColumnExists(ctx context.Context, database *sql.DB, table string, column string) (bool, error) {
	rows, err := database.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return false, fmt.Errorf("task53 preflight inspect %s columns: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

type task53RuleResult struct {
	CategoryID string   `json:"category_id"`
	AccountID  string   `json:"account_id"`
	TagIDs     []string `json:"tag_ids"`
}

func validateTask53RuleJSON(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, "SELECT id, result_json FROM import_rules ORDER BY id")
	if err != nil {
		return fmt.Errorf("task53 preflight read import rules: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var raw sql.NullString
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}
		trimmed := strings.TrimSpace(raw.String)
		var result task53RuleResult
		if !raw.Valid || trimmed == "" || trimmed[0] != '{' || !json.Valid([]byte(trimmed)) || json.Unmarshal([]byte(trimmed), &result) != nil {
			return fmt.Errorf("task53 preflight: import rule %s has invalid result_json", id)
		}
	}
	return rows.Err()
}

func validateTask53ImportItemJSON(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, `
		SELECT id, suggested_tag_ids_json, selected_tag_ids_json
		FROM import_items
		ORDER BY id
	`)
	if err != nil {
		return fmt.Errorf("task53 preflight read import item tag json: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var suggested, selected sql.NullString
		if err := rows.Scan(&id, &suggested, &selected); err != nil {
			return err
		}
		for field, raw := range map[string]sql.NullString{
			"suggested_tag_ids_json": suggested,
			"selected_tag_ids_json":  selected,
		} {
			if !raw.Valid || strings.TrimSpace(raw.String) == "" {
				continue
			}
			var tagIDs []string
			if !json.Valid([]byte(raw.String)) || json.Unmarshal([]byte(raw.String), &tagIDs) != nil {
				return fmt.Errorf("task53 preflight: import item %s has invalid %s", id, field)
			}
		}
	}
	return rows.Err()
}

func validateTask53ImportMetadataReferences(ctx context.Context, database *sql.DB) error {
	queries := []struct {
		name  string
		query string
	}{
		{
			name: "suggested category",
			query: `SELECT ii.id FROM import_items ii JOIN import_batches ib ON ib.id = ii.batch_id
				LEFT JOIN categories c ON c.id = ii.suggested_category_id AND c.ledger_id = ib.ledger_id
				WHERE ii.suggested_category_id IS NOT NULL AND c.id IS NULL LIMIT 1`,
		},
		{
			name: "selected category",
			query: `SELECT ii.id FROM import_items ii JOIN import_batches ib ON ib.id = ii.batch_id
				LEFT JOIN categories c ON c.id = ii.selected_category_id AND c.ledger_id = ib.ledger_id
				WHERE ii.selected_category_id IS NOT NULL AND c.id IS NULL LIMIT 1`,
		},
		{
			name: "suggested account",
			query: `SELECT ii.id FROM import_items ii JOIN import_batches ib ON ib.id = ii.batch_id
				LEFT JOIN accounts a ON a.id = ii.suggested_account_id AND a.ledger_id = ib.ledger_id
				WHERE ii.suggested_account_id IS NOT NULL AND a.id IS NULL LIMIT 1`,
		},
		{
			name: "selected account",
			query: `SELECT ii.id FROM import_items ii JOIN import_batches ib ON ib.id = ii.batch_id
				LEFT JOIN accounts a ON a.id = ii.selected_account_id AND a.ledger_id = ib.ledger_id
				WHERE ii.selected_account_id IS NOT NULL AND a.id IS NULL LIMIT 1`,
		},
	}
	for _, check := range queries {
		var id string
		err := database.QueryRowContext(ctx, check.query).Scan(&id)
		if err == nil {
			return fmt.Errorf("task53 preflight: import item %s has orphan %s", id, check.name)
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("task53 preflight check %s: %w", check.name, err)
		}
	}
	if err := validateTask53RuleMetadataReferences(ctx, database); err != nil {
		return err
	}
	if err := validateTask53ImportItemTagReferences(ctx, database); err != nil {
		return err
	}
	return nil
}

func validateTask53RuleMetadataReferences(ctx context.Context, database *sql.DB) error {
	categories, err := task53LoadMetadataReferences(ctx, database, "categories")
	if err != nil {
		return err
	}
	accounts, err := task53LoadMetadataReferences(ctx, database, "accounts")
	if err != nil {
		return err
	}
	tags, err := task53LoadMetadataReferences(ctx, database, "tags")
	if err != nil {
		return err
	}
	rows, err := database.QueryContext(ctx, "SELECT id, ledger_id, result_json FROM import_rules ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, ledgerID, raw string
		if err := rows.Scan(&id, &ledgerID, &raw); err != nil {
			return err
		}
		var result task53RuleResult
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			return fmt.Errorf("task53 preflight: import rule %s has invalid result_json", id)
		}
		for _, reference := range []struct {
			kind  string
			items map[string]bool
			id    string
		}{
			{kind: "categories", items: categories, id: result.CategoryID},
			{kind: "accounts", items: accounts, id: result.AccountID},
		} {
			if reference.id == "" {
				continue
			}
			if !reference.items[task53MetadataReferenceKey(ledgerID, reference.id)] {
				return fmt.Errorf("task53 preflight: import rule %s references missing or cross-ledger %s %s", id, reference.kind, reference.id)
			}
		}
		for _, tagID := range result.TagIDs {
			if !tags[task53MetadataReferenceKey(ledgerID, tagID)] {
				return fmt.Errorf("task53 preflight: import rule %s references missing or cross-ledger tag %s", id, tagID)
			}
		}
	}
	return rows.Err()
}

func validateTask53ImportItemTagReferences(ctx context.Context, database *sql.DB) error {
	tags, err := task53LoadMetadataReferences(ctx, database, "tags")
	if err != nil {
		return err
	}
	rows, err := database.QueryContext(ctx, `
		SELECT ii.id, ib.ledger_id, ii.suggested_tag_ids_json, ii.selected_tag_ids_json
		FROM import_items ii
		JOIN import_batches ib ON ib.id = ii.batch_id
		ORDER BY ii.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, ledgerID string
		var suggested, selected sql.NullString
		if err := rows.Scan(&id, &ledgerID, &suggested, &selected); err != nil {
			return err
		}
		for _, raw := range []sql.NullString{suggested, selected} {
			if !raw.Valid || strings.TrimSpace(raw.String) == "" {
				continue
			}
			var tagIDs []string
			if err := json.Unmarshal([]byte(raw.String), &tagIDs); err != nil {
				return fmt.Errorf("task53 preflight: import item %s has invalid tag reference json", id)
			}
			for _, tagID := range tagIDs {
				if !tags[task53MetadataReferenceKey(ledgerID, tagID)] {
					return fmt.Errorf("task53 preflight: import item %s references missing or cross-ledger tag %s", id, tagID)
				}
			}
		}
	}
	return rows.Err()
}

func task53LoadMetadataReferences(ctx context.Context, database *sql.DB, table string) (map[string]bool, error) {
	rows, err := database.QueryContext(ctx, "SELECT ledger_id, id FROM "+table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make(map[string]bool)
	for rows.Next() {
		var ledgerID, id string
		if err := rows.Scan(&ledgerID, &id); err != nil {
			return nil, err
		}
		items[task53MetadataReferenceKey(ledgerID, id)] = true
	}
	return items, rows.Err()
}

func task53MetadataReferenceKey(ledgerID string, id string) string {
	return ledgerID + "\x00" + id
}

func captureTask53MigrationMetrics(ctx context.Context, database *sql.DB) (map[string]int64, error) {
	metrics, err := captureTask50MigrationMetrics(ctx, database)
	if err != nil {
		return nil, err
	}

	queries := []struct {
		name  string
		query string
	}{
		{name: "categories", query: "SELECT ledger_id, COUNT(*) FROM categories GROUP BY ledger_id"},
		{name: "tags", query: "SELECT ledger_id, COUNT(*) FROM tags GROUP BY ledger_id"},
		{name: "import_rules", query: "SELECT ledger_id, COUNT(*) FROM import_rules GROUP BY ledger_id"},
		{name: "import_items", query: "SELECT ib.ledger_id, COUNT(*) FROM import_items ii JOIN import_batches ib ON ib.id = ii.batch_id GROUP BY ib.ledger_id"},
		{name: "transaction_tags", query: "SELECT tx.ledger_id, COUNT(*) FROM transaction_tags tt JOIN transactions tx ON tx.id = tt.transaction_id GROUP BY tx.ledger_id"},
	}
	for _, metricQuery := range queries {
		rows, err := database.QueryContext(ctx, metricQuery.query)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var ledgerID string
			var count int64
			if err := rows.Scan(&ledgerID, &count); err != nil {
				rows.Close()
				return nil, err
			}
			metrics["task53:"+metricQuery.name+":"+ledgerID] = count
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	rows, err := database.QueryContext(ctx, `
		SELECT tx.ledger_id, COUNT(*), COALESCE(SUM(ts.share_amount), 0)
		FROM transaction_splits ts
		JOIN transactions tx ON tx.id = ts.transaction_id
		GROUP BY tx.ledger_id
	`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var ledgerID string
		var count, amount int64
		if err := rows.Scan(&ledgerID, &count, &amount); err != nil {
			rows.Close()
			return nil, err
		}
		metrics["task53:splits:"+ledgerID+":count"] = count
		metrics["task53:splits:"+ledgerID+":amount"] = amount
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	hashQueries := []struct {
		name  string
		query string
	}{
		{name: "item_hash", query: "SELECT ib.ledger_id, ii.import_hash FROM import_items ii JOIN import_batches ib ON ib.id = ii.batch_id ORDER BY ib.ledger_id, ii.import_hash"},
		{name: "ref_hash", query: "SELECT ledger_id, import_hash FROM transaction_import_refs ORDER BY ledger_id, import_hash"},
	}
	for _, hashQuery := range hashQueries {
		rows, err := database.QueryContext(ctx, hashQuery.query)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var ledgerID, hash string
			if err := rows.Scan(&ledgerID, &hash); err != nil {
				rows.Close()
				return nil, err
			}
			metrics["task53:"+hashQuery.name+":"+ledgerID+":"+hash]++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return metrics, nil
}

func verifyTask53MigrationSnapshot(ctx context.Context, database *sql.DB, before *task53MigrationSnapshot) error {
	if before == nil {
		return nil
	}
	results, err := readQuickCheckResults(ctx, database)
	if err != nil {
		return err
	}
	if err := validateQuickCheckResults(results); err != nil {
		return err
	}
	if err := validateForeignKeys(ctx, database); err != nil {
		return err
	}
	after, err := captureTask53MigrationMetrics(ctx, database)
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(before.metrics)+len(after))
	seen := make(map[string]bool, len(before.metrics)+len(after))
	for key := range before.metrics {
		seen[key] = true
		keys = append(keys, key)
	}
	for key := range after {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		beforeValue, beforeFound := before.metrics[key]
		afterValue, afterFound := after[key]
		if !beforeFound || !afterFound || beforeValue != afterValue {
			return fmt.Errorf("task53 migration changed %s: before=%d (present=%t), after=%d (present=%t)", key, beforeValue, beforeFound, afterValue, afterFound)
		}
	}
	return nil
}
