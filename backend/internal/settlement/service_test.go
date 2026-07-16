package settlement_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/internal/db/repo"
	ledgerctx "ledger_two/internal/ledger"
	"ledger_two/internal/settlement"
	"ledger_two/migrations"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	db.SetMaxOpenConns(1)

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
	ctxA := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID: userAID, LedgerID: ledgerID, Role: ledgerctx.RoleOwner,
		Status: ledgerctx.LedgerStatusActive, Version: 1, IsExplicit: true,
	})
	ctxB := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID: userBID, LedgerID: ledgerID, Role: ledgerctx.RoleEditor,
		Status: ledgerctx.LedgerStatusActive, Version: 1, IsExplicit: true,
	})

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
	balance, err := svc.GetBalance(ctxA, userAID)
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
	balance2, err := svc.GetBalance(ctxB, userBID)
	if err != nil {
		t.Fatalf("get balance 2 failed: %v", err)
	}
	if balance2.AmountCents != 6000 {
		t.Errorf("expected balance = 6000, got %d", balance2.AmountCents)
	}
	if balance2.FromUserID != userBID || balance2.ToUserID != userAID {
		t.Errorf("expected B to owe A, got from %s to %s", balance2.FromUserID, balance2.ToUserID)
	}
	for _, ub := range balance2.UserBalances {
		if ub.UserID == userAID {
			if ub.PaidCents != 20000 || ub.ShareCents != 14000 || ub.RawNetCents != 6000 || ub.FinalNetCents != 6000 || ub.NetCents != 6000 {
				t.Errorf("unexpected A explain fields: %+v", ub)
			}
		}
		if ub.UserID == userBID {
			if ub.PaidCents != 8000 || ub.ShareCents != 14000 || ub.RawNetCents != -6000 || ub.FinalNetCents != -6000 || ub.NetCents != -6000 {
				t.Errorf("unexpected B explain fields: %+v", ub)
			}
		}
	}

	// ----------------------------------------------------
	// 场景 3: B 发起 6000 分结算。
	// ----------------------------------------------------
	_, err = svc.CreateSettlement(ctxB, userBID, settlement.CreateSettlementRequest{
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
	balance3, err := svc.GetBalance(ctxA, userAID)
	if err != nil {
		t.Fatalf("get balance 3 failed: %v", err)
	}
	if balance3.AmountCents != 0 {
		t.Errorf("expected balance = 0 after settlement, got %d", balance3.AmountCents)
	}
	if balance3.FromUserID != "" || balance3.ToUserID != "" {
		t.Errorf("expected from/to user IDs to be empty after settlement, got from: %s, to: %s", balance3.FromUserID, balance3.ToUserID)
	}
	for _, ub := range balance3.UserBalances {
		if ub.UserID == userAID {
			if ub.RawNetCents != 6000 || ub.SettlementNetCents != -6000 || ub.FinalNetCents != 0 || ub.NetCents != 0 {
				t.Errorf("unexpected A settled explain fields: %+v", ub)
			}
		}
		if ub.UserID == userBID {
			if ub.RawNetCents != -6000 || ub.SettlementNetCents != 6000 || ub.FinalNetCents != 0 || ub.NetCents != 0 {
				t.Errorf("unexpected B settled explain fields: %+v", ub)
			}
		}
	}

	// ----------------------------------------------------
	// 场景 4: 月份范围只统计该月的共同支出和结算记录。
	// ----------------------------------------------------
	previousMonthAt := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	currentMonthAt := time.Now().Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO transactions (id, ledger_id, type, title, amount, occurred_at, owner_user_id, created_by_user_id, payer_user_id, category_id, visibility, split_method, status, created_at, updated_at)
		VALUES
			('tx-old-month', ?, 'shared_expense', '上月共同支出', 10000, ?, ?, ?, ?, ?, 'shared', 'equal', 'normal', ?, ?),
			('tx-current-month', ?, 'shared_expense', '本月共同支出', 6000, ?, ?, ?, ?, ?, 'shared', 'equal', 'normal', ?, ?)
	`, ledgerID, previousMonthAt, userAID, userAID, userAID, categoryID, previousMonthAt, previousMonthAt,
		ledgerID, currentMonthAt, userBID, userBID, userBID, categoryID, currentMonthAt, currentMonthAt)
	if err != nil {
		t.Fatalf("insert scoped transactions failed: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO transaction_splits (id, transaction_id, user_id, share_amount, created_at, updated_at)
		VALUES
			('s-old-a', 'tx-old-month', ?, 5000, ?, ?),
			('s-old-b', 'tx-old-month', ?, 5000, ?, ?),
			('s-current-a', 'tx-current-month', ?, 3000, ?, ?),
			('s-current-b', 'tx-current-month', ?, 3000, ?, ?)
	`, userAID, previousMonthAt, previousMonthAt, userBID, previousMonthAt, previousMonthAt,
		userAID, currentMonthAt, currentMonthAt, userBID, currentMonthAt, currentMonthAt)
	if err != nil {
		t.Fatalf("insert scoped transaction splits failed: %v", err)
	}

	allBalance, err := svc.GetBalance(ctxA, userAID)
	if err != nil {
		t.Fatalf("get all-time balance failed: %v", err)
	}
	if allBalance.AmountCents != 2000 || allBalance.FromUserID != userBID || allBalance.ToUserID != userAID {
		t.Errorf("unexpected all-time balance: %+v", allBalance)
	}

	monthBalance, err := svc.GetBalanceForMonth(ctxA, userAID, time.Now().Format("2006-01"))
	if err != nil {
		t.Fatalf("get current-month balance failed: %v", err)
	}
	if monthBalance.AmountCents != 3000 || monthBalance.FromUserID != userAID || monthBalance.ToUserID != userBID {
		t.Errorf("unexpected current-month balance: %+v", monthBalance)
	}
	for _, ub := range monthBalance.UserBalances {
		if ub.UserID == userAID && (ub.PaidCents != 20000 || ub.ShareCents != 17000 || ub.SettlementNetCents != -6000 || ub.FinalNetCents != -3000) {
			t.Errorf("unexpected scoped A fields: %+v", ub)
		}
		if ub.UserID == userBID && (ub.PaidCents != 14000 || ub.ShareCents != 17000 || ub.SettlementNetCents != 6000 || ub.FinalNetCents != 3000) {
			t.Errorf("unexpected scoped B fields: %+v", ub)
		}
	}

	if _, err := svc.GetBalanceForMonth(ctxA, userAID, "2026-13"); err == nil {
		t.Error("expected invalid month to be rejected")
	}
}

func TestTask503BSettlementKeepsHistoricalParticipantBalanceAfterMembershipRemoval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	initRepo := repo.NewInitRepo(db)
	if err := initRepo.ExecuteSetupTx(context.Background(), "History Balance", "CNY", []repo.UserPayload{
		{Username: "owner", DisplayName: "Owner", PasswordHash: "hash1"},
		{Username: "former", DisplayName: "Former Member", PasswordHash: "hash2"},
	}); err != nil {
		t.Fatalf("setup history balance: %v", err)
	}

	var ledgerID, ownerID, formerID string
	if err := db.QueryRow("SELECT id FROM ledgers LIMIT 1").Scan(&ledgerID); err != nil {
		t.Fatalf("query history ledger: %v", err)
	}
	if err := db.QueryRow("SELECT id FROM users WHERE username = 'owner'").Scan(&ownerID); err != nil {
		t.Fatalf("query history owner: %v", err)
	}
	if err := db.QueryRow("SELECT id FROM users WHERE username = 'former'").Scan(&formerID); err != nil {
		t.Fatalf("query former member: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id,
			visibility, split_method, status, created_at, updated_at
		) VALUES (
			'history-shared', ?, 'shared_expense', 'Historical shared expense', 10000, ?,
			?, ?, ?, 'shared', 'equal', 'normal', ?, ?
		)
	`, ledgerID, now, ownerID, ownerID, ownerID, now, now); err != nil {
		t.Fatalf("insert historical shared expense: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO transaction_splits (
			id, transaction_id, user_id, share_amount, created_at, updated_at
		) VALUES
			('history-owner-split', 'history-shared', ?, 5000, ?, ?),
			('history-former-split', 'history-shared', ?, 5000, ?, ?)
	`, ownerID, now, now, formerID, now, now); err != nil {
		t.Fatalf("insert historical splits: %v", err)
	}
	if _, err := db.Exec(`
		DELETE FROM ledger_members
		WHERE ledger_id = ? AND user_id = ?
	`, ledgerID, formerID); err != nil {
		t.Fatalf("remove former membership fixture: %v", err)
	}

	ctx := ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID:     ownerID,
		LedgerID:   ledgerID,
		Role:       ledgerctx.RoleOwner,
		Status:     ledgerctx.LedgerStatusActive,
		Version:    2,
		IsExplicit: true,
	})
	balance, err := settlement.NewService(settlement.NewRepository(db)).GetBalance(ctx, ownerID)
	if err != nil {
		t.Fatalf("get historical participant balance: %v", err)
	}
	if balance.AmountCents != 5000 || balance.FromUserID != formerID || balance.ToUserID != ownerID {
		t.Fatalf("historical balance lost after membership removal: %+v", balance)
	}
	if len(balance.UserBalances) != 2 {
		t.Fatalf("expected current and historical participant balances, got %+v", balance.UserBalances)
	}
}

func TestTask503AUnsettledBalanceSnapshotUsesCallerTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	initRepo := repo.NewInitRepo(db)
	if err := initRepo.ExecuteSetupTx(context.Background(), "Lifecycle Ledger", "CNY", []repo.UserPayload{
		{Username: "owner", DisplayName: "Owner", PasswordHash: "hash1"},
		{Username: "partner", DisplayName: "Partner", PasswordHash: "hash2"},
	}); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	var ledgerID, ownerID, partnerID, categoryID string
	for query, target := range map[string]*string{
		"SELECT id FROM ledgers LIMIT 1":                        &ledgerID,
		"SELECT id FROM users WHERE username = 'owner'":         &ownerID,
		"SELECT id FROM users WHERE username = 'partner'":       &partnerID,
		"SELECT id FROM categories ORDER BY created_at LIMIT 1": &categoryID,
	} {
		if err := db.QueryRow(query).Scan(target); err != nil {
			t.Fatalf("load fixture with %q: %v", query, err)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, occurred_at, owner_user_id,
			created_by_user_id, payer_user_id, category_id, visibility,
			split_method, status, created_at, updated_at
		) VALUES ('lifecycle-shared', ?, 'shared_expense', 'Lifecycle', 2400, ?, ?, ?, ?, ?, 'shared', 'equal', 'normal', ?, ?)
	`, ledgerID, now, ownerID, ownerID, ownerID, categoryID, now, now); err != nil {
		t.Fatalf("insert shared expense: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO transaction_splits (id, transaction_id, user_id, share_amount, created_at, updated_at)
		VALUES ('lifecycle-owner', 'lifecycle-shared', ?, 1200, ?, ?),
		       ('lifecycle-partner', 'lifecycle-shared', ?, 1200, ?, ?)
	`, ownerID, now, now, partnerID, now, now); err != nil {
		t.Fatalf("insert shared splits: %v", err)
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin lifecycle transaction: %v", err)
	}
	defer tx.Rollback()

	service := settlement.NewService(settlement.NewRepository(db))
	snapshot, err := service.GetUnsettledBalance(context.Background(), tx, ledgerctx.LedgerContext{
		UserID: ownerID, LedgerID: ledgerID, Role: ledgerctx.RoleOwner,
		Status: ledgerctx.LedgerStatusActive, Version: 1, IsExplicit: true,
	})
	if err != nil {
		t.Fatalf("get transaction-scoped balance: %v", err)
	}
	if snapshot.AmountCents != 1200 || snapshot.FromUserID == nil || *snapshot.FromUserID != partnerID || snapshot.ToUserID == nil || *snapshot.ToUserID != ownerID {
		t.Fatalf("unexpected lifecycle balance snapshot: %+v", snapshot)
	}
}
