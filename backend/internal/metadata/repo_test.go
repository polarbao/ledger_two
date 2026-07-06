package metadata

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/migrations"
)

func TestRepositoryReorderTags(t *testing.T) {
	database := openMetadataTestDB(t)
	repo := NewRepository(database)

	ctx := context.Background()
	alpha, err := repo.Create(ctx, KindTag, "ledger-1", "user-1", UpsertRequest{Name: "Alpha"})
	if err != nil {
		t.Fatalf("create alpha tag: %v", err)
	}
	beta, err := repo.Create(ctx, KindTag, "ledger-1", "user-1", UpsertRequest{Name: "Beta"})
	if err != nil {
		t.Fatalf("create beta tag: %v", err)
	}
	gamma, err := repo.Create(ctx, KindTag, "ledger-1", "user-1", UpsertRequest{Name: "Gamma"})
	if err != nil {
		t.Fatalf("create gamma tag: %v", err)
	}

	if err := repo.Reorder(ctx, KindTag, "ledger-1", []string{gamma.ID, alpha.ID, beta.ID}); err != nil {
		t.Fatalf("reorder tags: %v", err)
	}

	items, err := repo.List(ctx, KindTag, "ledger-1", true)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	got := []string{items[0].ID, items[1].ID, items[2].ID}
	want := []string{gamma.ID, alpha.ID, beta.ID}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected tag order: got %v want %v", got, want)
		}
	}
	if items[0].SortOrder != 0 || items[1].SortOrder != 1 || items[2].SortOrder != 2 {
		t.Fatalf("unexpected sort order values: %+v", items)
	}
}

func TestRepositoryListUsageCounts(t *testing.T) {
	database := openMetadataTestDB(t)
	repo := NewRepository(database)
	ctx := context.Background()

	category, err := repo.Create(ctx, KindCategory, "ledger-1", "user-1", UpsertRequest{Name: "餐饮", Type: "expense"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	account, err := repo.Create(ctx, KindAccount, "ledger-1", "user-1", UpsertRequest{Name: "支付宝", Type: "alipay"})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	tag, err := repo.Create(ctx, KindTag, "ledger-1", "user-1", UpsertRequest{Name: "聚餐"})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	_, err = database.Exec(`
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id,
			visibility, status, created_at, updated_at
		) VALUES (
			'tx-usage-1', 'ledger-1', 'expense', 'Dinner', 8800, 'CNY', '2026-07-06T12:00:00Z',
			'user-1', 'user-1', 'user-1', ?, ?,
			'private', 'normal', '2026-07-06T12:00:00Z', '2026-07-06T12:00:00Z'
		)
	`, account.ID, category.ID)
	if err != nil {
		t.Fatalf("insert transaction fixture: %v", err)
	}
	_, err = database.Exec("INSERT INTO transaction_tags (transaction_id, tag_id) VALUES ('tx-usage-1', ?)", tag.ID)
	if err != nil {
		t.Fatalf("insert transaction tag fixture: %v", err)
	}

	assertUsageCount(t, repo, KindCategory, "ledger-1", category.ID, 1)
	assertUsageCount(t, repo, KindAccount, "ledger-1", account.ID, 1)
	assertUsageCount(t, repo, KindTag, "ledger-1", tag.ID, 1)
}

func assertUsageCount(t *testing.T, repo *Repository, kind Kind, ledgerID string, id string, expected int) {
	t.Helper()

	items, err := repo.List(context.Background(), kind, ledgerID, true)
	if err != nil {
		t.Fatalf("list %s: %v", kind, err)
	}
	for _, item := range items {
		if item.ID == id {
			if item.UsageCount != expected {
				t.Fatalf("expected %s %s usage_count=%d, got %d", kind, id, expected, item.UsageCount)
			}
			return
		}
	}
	t.Fatalf("expected %s item %s in list", kind, id)
}

func openMetadataTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open memory database: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	database.SetMaxOpenConns(1)

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.Up(database, "."); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return database
}
