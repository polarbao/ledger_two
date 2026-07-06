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
