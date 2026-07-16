package dashboard

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/migrations"
)

func TestTask503BDashboardResolvesOnlyCurrentLedgerReferencedHistoricalUsers(t *testing.T) {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open dashboard database: %v", err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.Up(database, "."); err != nil {
		t.Fatalf("run dashboard migrations: %v", err)
	}

	if _, err := database.Exec(`
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at) VALUES
			('user-current', 'current', 'Current Member', 'hash', 'user', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-history', 'history', 'Historical Member', 'hash', 'user', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z'),
			('user-unrelated', 'unrelated', 'Unrelated User', 'hash', 'user', '2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z');

		INSERT INTO ledgers (id, name, default_currency, status, version, created_at, updated_at)
		VALUES ('ledger-history', 'History Ledger', 'CNY', 'active', 4, '2026-01-01T00:00:00Z', '2026-07-01T00:00:00Z');

		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES ('ledger-history', 'user-current', 'owner', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');

		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, visibility,
			status, created_at, updated_at
		) VALUES (
			'txn-history', 'ledger-history', 'expense', 'Historical expense', 456, 'CNY',
			'2026-07-02T00:00:00Z', 'user-history', 'user-history', 'user-history',
			'partner_readable', 'normal', '2026-07-02T00:00:00Z', '2026-07-02T00:00:00Z'
		);
	`); err != nil {
		t.Fatalf("seed dashboard history fixture: %v", err)
	}

	transactions, _, _, _, users, err := NewRepository(database).GetDashboardRawData(
		context.Background(),
		"ledger-history",
		"user-current",
		"2026-07",
	)
	if err != nil {
		t.Fatalf("get dashboard history data: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("expected one visible historical transaction, got %d", len(transactions))
	}
	if users["user-current"] != "Current Member" || users["user-history"] != "Historical Member" {
		t.Fatalf("missing current or historical user names: %+v", users)
	}
	if _, leaked := users["user-unrelated"]; leaked {
		t.Fatalf("unrelated global user leaked into dashboard map: %+v", users)
	}
}
