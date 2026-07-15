package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

const (
	task50UpgradeSourceVersion int64 = 19
	task50TargetSchemaVersion  int64 = 21
)

type task50MigrationSnapshot struct {
	metrics map[string]int64
}

func prepareTask50Upgrade(ctx context.Context, database *sql.DB, currentVersion int64) (*task50MigrationSnapshot, error) {
	switch currentVersion {
	case 0:
		var applicationTableCount int
		if err := database.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM sqlite_master
			WHERE type = 'table'
			  AND name NOT LIKE 'sqlite_%'
			  AND name <> 'goose_db_version'
		`).Scan(&applicationTableCount); err != nil {
			return nil, fmt.Errorf("inspect empty database before task50 migration: %w", err)
		}
		if applicationTableCount != 0 {
			return nil, fmt.Errorf("task50 upgrade rejected unversioned non-empty database")
		}
		return nil, nil
	case task50TargetSchemaVersion:
		return nil, nil
	case task50UpgradeSourceVersion:
	default:
		return nil, fmt.Errorf(
			"task50 upgrade requires schema version %d, got %d",
			task50UpgradeSourceVersion,
			currentVersion,
		)
	}

	if err := runTask50IntegrityChecks(ctx, database); err != nil {
		return nil, err
	}

	metrics, err := captureTask50MigrationMetrics(ctx, database)
	if err != nil {
		return nil, fmt.Errorf("capture task50 migration snapshot: %w", err)
	}
	return &task50MigrationSnapshot{metrics: metrics}, nil
}

func runTask50IntegrityChecks(ctx context.Context, database *sql.DB) error {
	results, err := readQuickCheckResults(ctx, database)
	if err != nil {
		return fmt.Errorf("task50 preflight quick_check: %w", err)
	}
	if err := validateQuickCheckResults(results); err != nil {
		return err
	}

	var ledgerID string
	var count int
	err = database.QueryRowContext(ctx, `
		SELECT ledger_id, COUNT(*)
		FROM ledger_members
		GROUP BY ledger_id
		HAVING COUNT(*) > 2
		ORDER BY ledger_id
		LIMIT 1
	`).Scan(&ledgerID, &count)
	if err == nil {
		return fmt.Errorf("task50 preflight: ledger %s has %d members; expected at most 2", ledgerID, count)
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("task50 preflight member count: %w", err)
	}

	err = database.QueryRowContext(ctx, `
		SELECT l.id, SUM(CASE WHEN lm.role = 'owner' THEN 1 ELSE 0 END)
		FROM ledgers l
		LEFT JOIN ledger_members lm ON lm.ledger_id = l.id
		GROUP BY l.id
		HAVING SUM(CASE WHEN lm.role = 'owner' THEN 1 ELSE 0 END) <> 1
		ORDER BY l.id
		LIMIT 1
	`).Scan(&ledgerID, &count)
	if err == nil {
		return fmt.Errorf("task50 preflight: ledger %s has %d owners; expected exactly 1", ledgerID, count)
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("task50 preflight owner count: %w", err)
	}

	var userID string
	var role string
	err = database.QueryRowContext(ctx, `
		SELECT ledger_id, user_id, role
		FROM ledger_members
		WHERE role NOT IN ('owner', 'editor', 'viewer')
		ORDER BY ledger_id, user_id
		LIMIT 1
	`).Scan(&ledgerID, &userID, &role)
	if err == nil {
		return fmt.Errorf("task50 preflight: ledger %s user %s has invalid member role %s", ledgerID, userID, role)
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("task50 preflight member roles: %w", err)
	}

	if err := validateForeignKeys(ctx, database); err != nil {
		return err
	}
	return nil
}

func readQuickCheckResults(ctx context.Context, database *sql.DB) ([]string, error) {
	rows, err := database.QueryContext(ctx, "PRAGMA quick_check")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func validateQuickCheckResults(results []string) error {
	if len(results) == 1 && strings.EqualFold(strings.TrimSpace(results[0]), "ok") {
		return nil
	}
	if len(results) == 0 {
		return fmt.Errorf("task50 preflight quick_check failed: no result returned")
	}
	return fmt.Errorf("task50 preflight quick_check failed: %s", strings.Join(results, "; "))
}

func validateForeignKeys(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("task50 preflight foreign_key_check: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		var table string
		var rowID sql.NullInt64
		var parent string
		var foreignKeyID int
		if err := rows.Scan(&table, &rowID, &parent, &foreignKeyID); err != nil {
			return fmt.Errorf("task50 preflight foreign_key_check scan: %w", err)
		}
		return fmt.Errorf(
			"task50 preflight: foreign key violation in %s row %s referencing %s (fk %d)",
			table,
			nullableRowID(rowID),
			parent,
			foreignKeyID,
		)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("task50 preflight foreign_key_check: %w", err)
	}
	return nil
}

func nullableRowID(rowID sql.NullInt64) string {
	if !rowID.Valid {
		return "unknown"
	}
	return fmt.Sprintf("%d", rowID.Int64)
}

func captureTask50MigrationMetrics(ctx context.Context, database *sql.DB) (map[string]int64, error) {
	metrics := make(map[string]int64)
	tables := []string{
		"users",
		"ledgers",
		"ledger_members",
		"accounts",
		"categories",
		"tags",
		"transactions",
		"transaction_splits",
		"transaction_tags",
		"settlements",
		"audit_logs",
		"app_settings",
		"transaction_templates",
		"recurring_rules",
		"recurring_reminders",
		"import_batches",
		"import_items",
		"import_rules",
		"transaction_import_refs",
		"transaction_defaults",
	}
	for _, table := range tables {
		var count int64
		if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s: %w", table, err)
		}
		metrics["table:"+table] = count
	}

	ledgerMetricQueries := []struct {
		name  string
		query string
	}{
		{
			name:  "transactions",
			query: "SELECT ledger_id, COUNT(*), COALESCE(SUM(amount), 0) FROM transactions GROUP BY ledger_id",
		},
		{
			name:  "settlements",
			query: "SELECT ledger_id, COUNT(*), COALESCE(SUM(amount), 0) FROM settlements GROUP BY ledger_id",
		},
	}
	for _, metricQuery := range ledgerMetricQueries {
		rows, err := database.QueryContext(ctx, metricQuery.query)
		if err != nil {
			return nil, fmt.Errorf("capture %s metrics: %w", metricQuery.name, err)
		}
		for rows.Next() {
			var ledgerID string
			var count int64
			var amount int64
			if err := rows.Scan(&ledgerID, &count, &amount); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan %s metrics: %w", metricQuery.name, err)
			}
			metrics[metricQuery.name+":"+ledgerID+":count"] = count
			metrics[metricQuery.name+":"+ledgerID+":amount"] = amount
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate %s metrics: %w", metricQuery.name, err)
		}
		rows.Close()
	}

	perLedgerCountQueries := []struct {
		name  string
		query string
	}{
		{name: "members", query: "SELECT ledger_id, COUNT(*) FROM ledger_members GROUP BY ledger_id"},
		{name: "import_batches", query: "SELECT ledger_id, COUNT(*) FROM import_batches GROUP BY ledger_id"},
		{name: "audit_logs", query: "SELECT ledger_id, COUNT(*) FROM audit_logs GROUP BY ledger_id"},
		{name: "import_refs", query: "SELECT ledger_id, COUNT(*) FROM transaction_import_refs GROUP BY ledger_id"},
		{
			name: "attachment_refs",
			query: `
				SELECT ledger_id,
				       COALESCE(SUM(CASE
				           WHEN attachment_paths IS NOT NULL
				            AND TRIM(attachment_paths) NOT IN ('', '[]') THEN 1
				           ELSE 0
				       END), 0)
				FROM transactions
				GROUP BY ledger_id
			`,
		},
	}
	for _, metricQuery := range perLedgerCountQueries {
		rows, err := database.QueryContext(ctx, metricQuery.query)
		if err != nil {
			return nil, fmt.Errorf("capture %s metrics: %w", metricQuery.name, err)
		}
		for rows.Next() {
			var ledgerID string
			var count int64
			if err := rows.Scan(&ledgerID, &count); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan %s metrics: %w", metricQuery.name, err)
			}
			metrics[metricQuery.name+":"+ledgerID] = count
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate %s metrics: %w", metricQuery.name, err)
		}
		rows.Close()
	}

	return metrics, nil
}

func verifyTask50MigrationSnapshot(ctx context.Context, database *sql.DB, before *task50MigrationSnapshot) error {
	if before == nil {
		return nil
	}
	if err := runTask50IntegrityChecks(ctx, database); err != nil {
		return fmt.Errorf("post-migration integrity check: %w", err)
	}
	after, err := captureTask50MigrationMetrics(ctx, database)
	if err != nil {
		return fmt.Errorf("capture post-migration snapshot: %w", err)
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
			return fmt.Errorf(
				"task50 migration changed %s: before=%d (present=%t), after=%d (present=%t)",
				key,
				beforeValue,
				beforeFound,
				afterValue,
				afterFound,
			)
		}
	}
	return nil
}
