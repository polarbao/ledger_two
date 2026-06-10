package settlement_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/internal/db/repo"
	"ledger_two/internal/settlement"
	"ledger_two/migrations"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("goose dialect error: %v", err)
	}
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("goose up error: %v", err)
	}
	return db
}

func TestSettlementServiceUnit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// 1. 初始化系统账本和两个用户
	initRepo := repo.NewInitRepo(db)
	err := initRepo.ExecuteSetupTx(context.Background(), "Test Ledger", "CNY", []repo.UserPayload{
		{Username: "userA", DisplayName: "User A", PasswordHash: "hash1"},
		{Username: "userB", DisplayName: "User B", PasswordHash: "hash2"},
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// 查出用户 A 和 B 的实际 UUID
	var userAID, userBID string
	err = db.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&userAID)
	if err != nil {
		t.Fatalf("query userA id failed: %v", err)
	}
	err = db.QueryRow("SELECT id FROM users WHERE username = 'userB'").Scan(&userBID)
	if err != nil {
		t.Fatalf("query userB id failed: %v", err)
	}

	// 查出真实初始化出来的 LedgerID
	var ledgerID string
	err = db.QueryRow("SELECT id FROM ledgers LIMIT 1").Scan(&ledgerID)
	if err != nil {
		t.Fatalf("query ledger id failed: %v", err)
	}

	// 查询默认分类 ID
	var categoryID string
	err = db.QueryRow("SELECT id FROM categories LIMIT 1").Scan(&categoryID)
	if err != nil {
		t.Fatalf("query category failed: %v", err)
	}

	// 实例化 Service
	r := settlement.NewRepository(db)
	svc := settlement.NewService(r)

	// ----------------------------------------------------
	// 场景 1: A 支付 200 元 (20000分)，平摊。
	// ----------------------------------------------------
	_, err = db.Exec(`
		INSERT INTO transactions (id, ledger_id, type, title, amount, occurred_at, owner_user_id, created_by_user_id, payer_user_id, category_id, visibility, split_method, status, created_at, updated_at)
		VALUES ('tx1', ?, 'shared_expense', '日用品', 20000, ?, ?, ?, ?, ?, 'shared', 'equal', 'normal', ?, ?)
	`, ledgerID, time.Now().Format(time.RFC3339), userAID, userAID, userAID, categoryID, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert tx1 failed: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO transaction_splits (id, transaction_id, user_id, share_amount, created_at, updated_at)
		VALUES ('s1', 'tx1', ?, 10000, ?, ?), ('s2', 'tx1', ?, 10000, ?, ?)
	`, userAID, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339), userBID, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert splits failed: %v", err)
	}

	// 验证余额：B 欠 A 10000分 (100.00元)
	balance, err := svc.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("get balance failed: %v", err)
	}
	if balance.AmountCents != 10000 {
		t.Errorf("expected balance = 10000, got %d", balance.AmountCents)
	}
	if balance.FromUserID != userBID || balance.ToUserID != userAID {
		t.Errorf("expected B to owe A, got from %s to %s", balance.FromUserID, balance.ToUserID)
	}

	// ----------------------------------------------------
	// 场景 2: B 支付 80 元 (8000分)，平摊。
	// ----------------------------------------------------
	_, err = db.Exec(`
		INSERT INTO transactions (id, ledger_id, type, title, amount, occurred_at, owner_user_id, created_by_user_id, payer_user_id, category_id, visibility, split_method, status, created_at, updated_at)
		VALUES ('tx2', ?, 'shared_expense', '水果', 8000, ?, ?, ?, ?, ?, 'shared', 'equal', 'normal', ?, ?)
	`, ledgerID, time.Now().Format(time.RFC3339), userBID, userBID, userBID, categoryID, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert tx2 failed: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO transaction_splits (id, transaction_id, user_id, share_amount, created_at, updated_at)
		VALUES ('s3', 'tx2', ?, 4000, ?, ?), ('s4', 'tx2', ?, 4000, ?, ?)
	`, userAID, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339), userBID, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert splits 2 failed: %v", err)
	}

	// 验证累计：B 欠 A 6000分 (60.00元)
	balance2, err := svc.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("get balance 2 failed: %v", err)
	}
	if balance2.AmountCents != 6000 {
		t.Errorf("expected balance = 6000, got %d", balance2.AmountCents)
	}
	if balance2.FromUserID != userBID || balance2.ToUserID != userAID {
		t.Errorf("expected B to owe A, got from %s to %s", balance2.FromUserID, balance2.ToUserID)
	}

	// ----------------------------------------------------
	// 场景 3: B 发起 6000 分结算。
	// ----------------------------------------------------
	_, err = svc.CreateSettlement(context.Background(), userBID, settlement.CreateSettlementRequest{
		FromUserID:  userBID,
		ToUserID:    userAID,
		AmountCents: 6000,
		OccurredAt:  time.Now().Format(time.RFC3339),
		Note:        "微信转账",
	})
	if err != nil {
		t.Fatalf("create settlement failed: %v", err)
	}

	// 验证余额：结清 (0分)
	balance3, err := svc.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("get balance 3 failed: %v", err)
	}
	if balance3.AmountCents != 0 {
		t.Errorf("expected balance = 0 after settlement, got %d", balance3.AmountCents)
	}
	if balance3.FromUserID != "" || balance3.ToUserID != "" {
		t.Errorf("expected from/to user IDs to be empty after settlement, got from: %s, to: %s", balance3.FromUserID, balance3.ToUserID)
	}
}
