package settlement

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository 结算模块数据库仓库
// @brief 提供结算明细表的增删改查及账务数据多表汇总逻辑
type Repository struct {
	db *sql.DB
}

// NewRepository 实例化数据仓库
// @brief 创建 Settlement 的 Repository 实例
// @param db *sql.DB 底层数据库连接句柄
// @return *Repository 仓库实例
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetDB 获取底层 db 连接
// @brief 返回底层数据库实例，供 Service 层启动事务使用
// @return *sql.DB 数据库连接实例
func (r *Repository) GetDB() *sql.DB {
	return r.db
}

// CreateWithTx 在数据库事务中插入结算明细
// @brief 事务内物理写入单笔补款结算数据
// @param ctx context.Context 上下文
// @param tx *sql.Tx 数据库事务句柄
// @param s *Settlement 结算数据模型
// @return error 错误信息
func (r *Repository) CreateWithTx(ctx context.Context, tx *sql.Tx, s *Settlement) error {
	var executor dbExecutor = r.db
	if tx != nil {
		executor = tx
	}

	now := time.Now().Format(time.RFC3339)

	var noteVal interface{}
	if s.Note.Valid {
		noteVal = s.Note.String
	} else {
		noteVal = nil
	}

	_, err := executor.ExecContext(ctx, `
		INSERT INTO settlements (
			id, ledger_id, from_user_id, to_user_id, amount, currency, 
			occurred_at, note, created_by_user_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		s.ID, s.LedgerID, s.FromUserID, s.ToUserID, s.Amount, s.Currency,
		s.OccurredAt.Format(time.RFC3339), noteVal, s.CreatedByUserID, now,
	)
	if err != nil {
		return fmt.Errorf("insert settlement failed: %w", err)
	}
	return nil
}

// List 拉取历史结算明细列表
// @brief 支持按月份模糊检索及时间逆序排列
// @param ctx context.Context 上下文
// @param ledgerID string 账本 ID
// @param month string 过滤月份 (如 '2026-06')
// @return []*Settlement 结算明细列表
// @return error 错误信息
func (r *Repository) List(ctx context.Context, ledgerID string, month string) ([]*Settlement, error) {
	query := `
		SELECT 
			id, ledger_id, from_user_id, to_user_id, amount, currency, 
			occurred_at, note, created_by_user_id, created_at
		FROM settlements
		WHERE ledger_id = ?
	`
	args := []interface{}{ledgerID}

	if month != "" {
		query += " AND occurred_at LIKE ?"
		args = append(args, month+"%")
	}

	query += " ORDER BY occurred_at DESC, created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Settlement
	for rows.Next() {
		var s Settlement
		var note sql.NullString
		var occurredAtStr, createdAtStr string

		err := rows.Scan(
			&s.ID, &s.LedgerID, &s.FromUserID, &s.ToUserID, &s.Amount, &s.Currency,
			&occurredAtStr, &note, &s.CreatedByUserID, &createdAtStr,
		)
		if err != nil {
			return nil, err
		}

		s.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		s.Note = note

		list = append(list, &s)
	}
	return list, nil
}

// GetSharedExpensesNetStats 多表汇总统计各自共同支出的已付、应摊及已结总额
// @brief 运行参数化 SQL 进行聚合统计
// @param ctx context.Context 上下文
// @param ledgerID string 账本 ID
// @return map[string]int64 各自支付共同支出的总和 Map [user_id] -> cents
// @return map[string]int64 各自应分摊共同支出的总和 Map [user_id] -> cents
// @return map[string]int64 各自已付出的结算补款总和 Map [user_id] -> cents
// @return map[string]int64 各自收到的结算补款总和 Map [user_id] -> cents
// @return error 错误信息
func (r *Repository) GetSharedExpensesNetStats(ctx context.Context, ledgerID string) (map[string]int64, map[string]int64, map[string]int64, map[string]int64, error) {
	paidMap := make(map[string]int64)
	shareMap := make(map[string]int64)
	settledOutMap := make(map[string]int64)
	settledInMap := make(map[string]int64)

	// 1. 汇总统计各自实际支付的共同支出（排除软删除）
	paidRows, err := r.db.QueryContext(ctx, `
		SELECT payer_user_id, SUM(amount)
		FROM transactions
		WHERE ledger_id = ? AND type = 'shared_expense' AND status != 'deleted'
		GROUP BY payer_user_id
	`, ledgerID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer paidRows.Close()

	for paidRows.Next() {
		var userID string
		var amount int64
		if err := paidRows.Scan(&userID, &amount); err == nil {
			paidMap[userID] = amount
		}
	}

	// 2. 汇总统计各自实际应该分摊的共同支出（排除软删除交易）
	shareRows, err := r.db.QueryContext(ctx, `
		SELECT ts.user_id, SUM(ts.share_amount)
		FROM transaction_splits ts
		JOIN transactions t ON ts.transaction_id = t.id
		WHERE t.ledger_id = ? AND t.status != 'deleted'
		GROUP BY ts.user_id
	`, ledgerID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer shareRows.Close()

	for shareRows.Next() {
		var userID string
		var shareAmount int64
		if err := shareRows.Scan(&userID, &shareAmount); err == nil {
			shareMap[userID] = shareAmount
		}
	}

	// 3. 汇总统计各自作为付款人向对方发起的结算补款总额
	settleOutRows, err := r.db.QueryContext(ctx, `
		SELECT from_user_id, SUM(amount)
		FROM settlements
		WHERE ledger_id = ?
		GROUP BY from_user_id
	`, ledgerID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer settleOutRows.Close()

	for settleOutRows.Next() {
		var userID string
		var amount int64
		if err := settleOutRows.Scan(&userID, &amount); err == nil {
			settledOutMap[userID] = amount
		}
	}

	// 4. 汇总统计各自作为收款人收到的结算补款总额
	settleInRows, err := r.db.QueryContext(ctx, `
		SELECT to_user_id, SUM(amount)
		FROM settlements
		WHERE ledger_id = ?
		GROUP BY to_user_id
	`, ledgerID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer settleInRows.Close()

	for settleInRows.Next() {
		var userID string
		var amount int64
		if err := settleInRows.Scan(&userID, &amount); err == nil {
			settledInMap[userID] = amount
		}
	}

	return paidMap, shareMap, settledOutMap, settledInMap, nil
}

// CreateAuditLogWithTx 在事务内物理写入单笔审计日志
func (r *Repository) CreateAuditLogWithTx(ctx context.Context, tx *sql.Tx, ledgerID, actorUserID, action, entityType, entityID, beforeJSON, afterJSON string) error {
	var executor dbExecutor = r.db
	if tx != nil {
		executor = tx
	}

	id := uuid.NewString()
	now := time.Now().Format(time.RFC3339)

	var beforeVal, afterVal interface{}
	if beforeJSON != "" {
		beforeVal = beforeJSON
	} else {
		beforeVal = nil
	}
	if afterJSON != "" {
		afterVal = afterJSON
	} else {
		afterVal = nil
	}

	_, err := executor.ExecContext(ctx, `
		INSERT INTO audit_logs (
			id, ledger_id, actor_user_id, action, entity_type, entity_id, 
			before_json, after_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		id, ledgerID, actorUserID, action, entityType, entityID,
		beforeVal, afterVal, now,
	)
	if err != nil {
		return fmt.Errorf("insert audit log failed: %w", err)
	}
	return nil
}

type dbExecutor interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}
