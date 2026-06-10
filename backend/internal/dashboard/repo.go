package dashboard

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"ledger_two/internal/transaction"
)

// Repository 首页 Dashboard 统计数据库仓储
// @brief 批量拉取当月流水的原始数据及各关联维度 Map，以保障无 N+1 瓶颈
type Repository struct {
	db *sql.DB
}

// NewRepository 实例化数据仓库
// @brief 创建 Dashboard Repository 实例
// @param db *sql.DB 底层数据库连接句柄
// @return *Repository 仓库实例
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetDashboardRawData 批量获取 Dashboard 计算所需的本月所有流水的原始多维数据
// @brief 通过批量 IN 子句查询防止在循环中执行 SQL 造成的数据库访问压力
// @param ctx context.Context 上下文
// @param ledgerID string 账本 ID
// @param userID string 登录用户 ID (用于可见性安全过滤)
// @param month string 查询月份 (如 '2026-06')
// @return []*transaction.Transaction 当月可见交易列表
// @return map[string][]string 交易 ID 关联标签 Map
// @return map[string][]transaction.SplitResponse 交易 ID 关联分摊 Map
// @return map[string]string 账本内所有分类 Map [category_id] -> name
// @return map[string]string 系统所有用户显示名 Map [user_id] -> display_name
// @return error 错误信息
func (r *Repository) GetDashboardRawData(
	ctx context.Context,
	ledgerID string,
	userID string,
	month string,
) (
	[]*transaction.Transaction,
	map[string][]string,
	map[string][]transaction.SplitResponse,
	map[string]string,
	map[string]string,
	error,
) {
	// 1. 查询账本下的所有分类映射
	categories := make(map[string]string)
	catRows, err := r.db.QueryContext(ctx, "SELECT id, name FROM categories WHERE ledger_id = ?", ledgerID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer catRows.Close()
	for catRows.Next() {
		var id, name string
		if err := catRows.Scan(&id, &name); err == nil {
			categories[id] = name
		}
	}

	// 2. 查询系统内的所有用户 display_name 映射
	users := make(map[string]string)
	userRows, err := r.db.QueryContext(ctx, "SELECT id, display_name FROM users")
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer userRows.Close()
	for userRows.Next() {
		var id, displayName string
		if err := userRows.Scan(&id, &displayName); err == nil {
			users[id] = displayName
		}
	}

	// 3. 查询当月所有满足可见性鉴权的非软删除交易流水
	query := `
		SELECT 
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id,
			visibility, split_method, note, status, created_at, updated_at
		FROM transactions
		WHERE ledger_id = ? AND status != 'deleted' AND occurred_at LIKE ?
		AND (
			created_by_user_id = ? 
			OR owner_user_id = ? 
			OR payer_user_id = ? 
			OR visibility IN ('partner_readable', 'shared')
		)
		ORDER BY occurred_at DESC, created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, ledgerID, month+"%", userID, userID, userID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer rows.Close()

	var list []*transaction.Transaction
	var txIDs []string
	for rows.Next() {
		var tx transaction.Transaction
		var accountID, categoryID, splitMethod, note sql.NullString
		var occurredAtStr, createdAtStr, updatedAtStr string

		err := rows.Scan(
			&tx.ID, &tx.LedgerID, &tx.Type, &tx.Title, &tx.Amount, &tx.Currency, &occurredAtStr,
			&tx.OwnerUserID, &tx.CreatedByUserID, &tx.PayerUserID, &accountID, &categoryID,
			&tx.Visibility, &splitMethod, &note, &tx.Status, &createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		tx.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
		tx.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		tx.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		tx.AccountID = accountID
		tx.CategoryID = categoryID
		tx.SplitMethod = splitMethod
		tx.Note = note

		list = append(list, &tx)
		txIDs = append(txIDs, tx.ID)
	}

	tagMap := make(map[string][]string)
	splitMap := make(map[string][]transaction.SplitResponse)
	if len(txIDs) == 0 {
		return list, tagMap, splitMap, categories, users, nil
	}

	// 4. 批量拉取交易关联的所有 tags 名 (防 N+1)
	placeholders := make([]string, len(txIDs))
	args := make([]interface{}, len(txIDs))
	for i, id := range txIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	tagQuery := fmt.Sprintf(`
		SELECT tt.transaction_id, t.name
		FROM tags t
		JOIN transaction_tags tt ON t.id = tt.tag_id
		WHERE tt.transaction_id IN (%s)
	`, strings.Join(placeholders, ","))

	tagRows, err := r.db.QueryContext(ctx, tagQuery, args...)
	if err == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var txID, tagName string
			if err := tagRows.Scan(&txID, &tagName); err == nil {
				tagMap[txID] = append(tagMap[txID], tagName)
			}
		}
	}

	// 5. 批量拉取交易关联的所有 splits 记录 (防 N+1)
	splitQuery := fmt.Sprintf(`
		SELECT transaction_id, user_id, share_amount
		FROM transaction_splits
		WHERE transaction_id IN (%s)
	`, strings.Join(placeholders, ","))

	splitRows, err := r.db.QueryContext(ctx, splitQuery, args...)
	if err == nil {
		defer splitRows.Close()
		for splitRows.Next() {
			var txID string
			var s transaction.SplitResponse
			if err := splitRows.Scan(&txID, &s.UserID, &s.ShareAmountCents); err == nil {
				splitMap[txID] = append(splitMap[txID], s)
			}
		}
	}

	return list, tagMap, splitMap, categories, users, nil
}
