package defaults

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/migrations"
)

func TestApplyFreshCreatesBasicProfileInsideCallerTransaction(t *testing.T) {
	database := openDefaultsTestDB(t)
	seedDefaultsLedger(t, database, "ledger-basic")

	tx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin defaults transaction: %v", err)
	}
	result, err := ApplyFresh(context.Background(), tx, "ledger-basic", "user-defaults", ProfileBasicCNV1, time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("apply fresh profile: %v", err)
	}
	if result.CreatedCategories != 19 || result.CreatedTags != 8 || result.ProfileVersion != 1 {
		t.Fatalf("unexpected fresh result: %+v", result)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit defaults transaction: %v", err)
	}

	var categoryCount, tagCount, profileVersion int
	if err := database.QueryRow("SELECT COUNT(*) FROM categories WHERE ledger_id = 'ledger-basic'").Scan(&categoryCount); err != nil {
		t.Fatalf("count categories: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM tags WHERE ledger_id = 'ledger-basic'").Scan(&tagCount); err != nil {
		t.Fatalf("count tags: %v", err)
	}
	if err := database.QueryRow("SELECT metadata_profile_version FROM ledgers WHERE id = 'ledger-basic'").Scan(&profileVersion); err != nil {
		t.Fatalf("read profile version: %v", err)
	}
	if categoryCount != 19 || tagCount != 8 || profileVersion != 1 {
		t.Fatalf("unexpected persisted defaults: categories=%d tags=%d version=%d", categoryCount, tagCount, profileVersion)
	}

	var fallbackCount int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM categories
		WHERE ledger_id = 'ledger-basic' AND system_key IN ('expense_other', 'income_other')
	`).Scan(&fallbackCount); err != nil {
		t.Fatalf("count fallback categories: %v", err)
	}
	if fallbackCount != 2 {
		t.Fatalf("fallback category count = %d, want 2", fallbackCount)
	}
}

func TestApplyFreshEmptyProfileAndRollbackLeaveNoPartialMetadata(t *testing.T) {
	database := openDefaultsTestDB(t)
	seedDefaultsLedger(t, database, "ledger-empty")
	seedDefaultsLedger(t, database, "ledger-rollback")

	emptyTx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin empty transaction: %v", err)
	}
	result, err := ApplyFresh(context.Background(), emptyTx, "ledger-empty", "user-defaults", ProfileEmpty, time.Now())
	if err != nil {
		t.Fatalf("apply empty profile: %v", err)
	}
	if result.CreatedCategories != 0 || result.CreatedTags != 0 || result.ProfileVersion != 0 {
		t.Fatalf("unexpected empty result: %+v", result)
	}
	if err := emptyTx.Commit(); err != nil {
		t.Fatalf("commit empty profile: %v", err)
	}

	rollbackTx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin rollback transaction: %v", err)
	}
	if _, err := ApplyFresh(context.Background(), rollbackTx, "ledger-rollback", "user-defaults", ProfileBasicCNV1, time.Now()); err != nil {
		t.Fatalf("apply rollback profile: %v", err)
	}
	if err := rollbackTx.Rollback(); err != nil {
		t.Fatalf("rollback profile: %v", err)
	}

	for _, ledgerID := range []string{"ledger-empty", "ledger-rollback"} {
		var count int
		if err := database.QueryRow(`
			SELECT (SELECT COUNT(*) FROM categories WHERE ledger_id = ?) +
			       (SELECT COUNT(*) FROM tags WHERE ledger_id = ?)
		`, ledgerID, ledgerID).Scan(&count); err != nil {
			t.Fatalf("count metadata for %s: %v", ledgerID, err)
		}
		if count != 0 {
			t.Fatalf("ledger %s contains %d unexpected metadata rows", ledgerID, count)
		}
	}
}

func openDefaultsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open defaults database: %v", err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.Up(database, "."); err != nil {
		t.Fatalf("run defaults migrations: %v", err)
	}
	return database
}

func seedDefaultsLedger(t *testing.T, database *sql.DB, ledgerID string) {
	t.Helper()
	if _, err := database.Exec(`
		INSERT OR IGNORE INTO users (id, username, display_name, password_hash, role, created_at, updated_at)
		VALUES ('user-defaults', 'defaults', 'Defaults', 'hash', 'user', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z');
		INSERT INTO ledgers (id, name, default_currency, created_at, updated_at)
		VALUES (?, ?, 'CNY', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z');
	`, ledgerID, ledgerID); err != nil {
		t.Fatalf("seed defaults ledger %s: %v", ledgerID, err)
	}
}
