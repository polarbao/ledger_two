package ledger

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/migrations"
)

func TestRepositoryListsLedgersByLifecycleStatus(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	repository := NewRepository(database)

	active, err := repository.ListUserLedgersByStatus(context.Background(), "user-a", LedgerListActive)
	if err != nil {
		t.Fatalf("list active ledgers: %v", err)
	}
	if len(active) != 1 || active[0].ID != "ledger-active" {
		t.Fatalf("expected only active ledger, got %+v", active)
	}
	if active[0].Status != LedgerStatusActive || active[0].Version != 3 || active[0].MemberCount != 2 {
		t.Fatalf("unexpected active ledger lifecycle data: %+v", active[0])
	}

	archived, err := repository.ListUserLedgersByStatus(context.Background(), "user-a", LedgerListArchived)
	if err != nil {
		t.Fatalf("list archived ledgers: %v", err)
	}
	if len(archived) != 1 || archived[0].ID != "ledger-archived" {
		t.Fatalf("expected only archived ledger, got %+v", archived)
	}
	if archived[0].ArchivedAt == nil || archived[0].ArchivedByUserID == nil || *archived[0].ArchivedByUserID != "user-a" {
		t.Fatalf("expected archived metadata, got %+v", archived[0])
	}

	all, err := repository.ListUserLedgersByStatus(context.Background(), "user-a", LedgerListAll)
	if err != nil {
		t.Fatalf("list all ledgers: %v", err)
	}
	if len(all) != 2 || all[0].ID != "ledger-active" || all[1].ID != "ledger-archived" {
		t.Fatalf("expected active ledger before archived ledger, got %+v", all)
	}
}

func TestRepositoryLoadsLifecycleAndMembershipFacts(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	repository := NewRepository(database)

	loaded, err := repository.GetLedgerByID(context.Background(), "ledger-archived")
	if err != nil {
		t.Fatalf("load archived ledger: %v", err)
	}
	if loaded.Status != LedgerStatusArchived || loaded.Version != 8 || loaded.MemberCount != 1 {
		t.Fatalf("unexpected loaded ledger: %+v", loaded)
	}

	members, err := repository.CountMembers(context.Background(), "ledger-active")
	if err != nil || members != 2 {
		t.Fatalf("expected 2 members, got %d, err=%v", members, err)
	}
	owners, err := repository.CountOwners(context.Background(), "ledger-active")
	if err != nil || owners != 1 {
		t.Fatalf("expected 1 owner, got %d, err=%v", owners, err)
	}

	isAdmin, err := repository.IsInstanceAdmin(context.Background(), "user-a")
	if err != nil || !isAdmin {
		t.Fatalf("expected user-a instance admin, got %t, err=%v", isAdmin, err)
	}
	isAdmin, err = repository.IsInstanceAdmin(context.Background(), "user-b")
	if err != nil || isAdmin {
		t.Fatalf("expected user-b non-admin, got %t, err=%v", isAdmin, err)
	}
}

func TestRepositoryClaimsLedgerVersionConditionally(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	repository := NewRepository(database)

	tx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	claimed, err := repository.ClaimLedgerVersion(context.Background(), tx, "ledger-active", 3)
	if err != nil {
		t.Fatalf("claim current version: %v", err)
	}
	if !claimed {
		t.Fatal("expected current version to be claimed")
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit version claim: %v", err)
	}

	staleTx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin stale transaction: %v", err)
	}
	defer staleTx.Rollback()
	claimed, err = repository.ClaimLedgerVersion(context.Background(), staleTx, "ledger-active", 3)
	if err != nil {
		t.Fatalf("claim stale version: %v", err)
	}
	if claimed {
		t.Fatal("expected stale version claim to affect no rows")
	}
	if err := staleTx.Rollback(); err != nil {
		t.Fatalf("rollback stale transaction: %v", err)
	}

	var version int64
	if err := database.QueryRow("SELECT version FROM ledgers WHERE id = 'ledger-active'").Scan(&version); err != nil {
		t.Fatalf("query claimed version: %v", err)
	}
	if version != 4 {
		t.Fatalf("expected version 4, got %d", version)
	}
}

func TestMigrationEnforcesOwnerAndMemberDatabaseConstraints(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)

	_, err := database.Exec(`
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES ('ledger-active', 'user-c', 'viewer', '2026-07-03T00:00:00Z', '2026-07-03T00:00:00Z')
	`)
	if err == nil {
		t.Fatal("expected third member insert to be rejected")
	}

	_, err = database.Exec(`
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES ('ledger-archived', 'user-b', 'owner', '2026-07-03T00:00:00Z', '2026-07-03T00:00:00Z')
	`)
	if err == nil {
		t.Fatal("expected second owner insert to be rejected")
	}
}

func openLedgerRepositoryTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open repository database: %v", err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = database.Close()
	})

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.Up(database, "."); err != nil {
		t.Fatalf("run repository migrations: %v", err)
	}
	return database
}

func seedLedgerRepositoryFixtures(t *testing.T, database *sql.DB) {
	t.Helper()

	_, err := database.Exec(`
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at) VALUES
			('user-a', 'alice', 'Alice', 'hash-a', 'user', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('user-b', 'bob', 'Bob', 'hash-b', 'user', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z'),
			('user-c', 'cara', 'Cara', 'hash-c', 'user', '2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z');

		INSERT INTO ledgers (
			id, name, default_currency, status, archived_at, archived_by_user_id,
			version, created_at, updated_at
		) VALUES
			('ledger-active', 'Active Ledger', 'CNY', 'active', NULL, NULL, 3,
			 '2026-01-01T00:00:00Z', '2026-07-01T00:00:00Z'),
			('ledger-archived', 'Archived Ledger', 'CNY', 'archived',
			 '2026-06-30T12:00:00Z', 'user-a', 8,
			 '2026-02-01T00:00:00Z', '2026-06-30T12:00:00Z');

		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at) VALUES
			('ledger-active', 'user-a', 'owner', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('ledger-active', 'user-b', 'editor', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z'),
			('ledger-archived', 'user-a', 'owner', '2026-02-01T00:00:00Z', '2026-02-01T00:00:00Z');

		INSERT INTO instance_admins (user_id, granted_at, granted_by_user_id)
		VALUES ('user-a', '2026-01-01T00:00:00Z', NULL);
	`)
	if err != nil {
		t.Fatalf("seed repository fixtures: %v", err)
	}
}
