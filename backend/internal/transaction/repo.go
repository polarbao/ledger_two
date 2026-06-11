package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Repository 账单交易数据库仓库
type Repository struct {
	db *sql.DB
}

// NewRepository 实例化数据仓库
// @brief 创建 Transaction 的 Repository 实例
// @param db *sql.DB 数据库连接句柄
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

// CreateWithTx 在事务中创建交易及关联标签
// @brief 事务内写入单条流水及绑定的标签
// @param ctx context.Context 上下文
// @param tx *sql.Tx 事务句柄，为 nil 时使用默认 db 链接
// @param transaction *Transaction 交易对象体
// @param tags []string 关联标签名称列表
// @return error 错误信息
func (r *Repository) CreateWithTx(ctx context.Context, tx *sql.Tx, transaction *Transaction, tags []string) error {
	executor := r.getExecutor(tx)
	now := time.Now().Format(time.RFC3339)

	// 1. 写入交易表
	_, err := executor.ExecContext(ctx, `
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id,
			visibility, split_method, note, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'normal', ?, ?)
	`,
		transaction.ID, transaction.LedgerID, transaction.Type, transaction.Title,
		transaction.Amount, transaction.Currency, transaction.OccurredAt.Format(time.RFC3339),
		transaction.OwnerUserID, transaction.CreatedByUserID, transaction.PayerUserID,
		r.nullString(transaction.AccountID), r.nullString(transaction.CategoryID),
		transaction.Visibility, r.nullString(transaction.SplitMethod),
		r.nullString(transaction.Note), now, now,
	)
	if err != nil {
		return fmt.Errorf("insert transaction failed: %w", err)
	}

	// 2. 插入并关联标签
	if len(tags) > 0 {
		err = r.associateTags(ctx, executor, transaction.ID, transaction.LedgerID, tags, now)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetByID 根据 ID 查询单条交易
// @brief 关联读取交易数据及所有绑定的标签
// @param ctx context.Context 上下文
// @param id string 交易 ID
// @return *Transaction 交易实体
// @return []string 标签名称列表
// @return error 错误信息
func (r *Repository) GetByID(ctx context.Context, id string) (*Transaction, []string, error) {
	var tx Transaction
	var accountID, categoryID, splitMethod, note sql.NullString
	var occurredAtStr, createdAtStr, updatedAtStr string
	var deletedAtStr sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id,
			visibility, split_method, note, status, created_at, updated_at, deleted_at
		FROM transactions
		WHERE id = ?
	`, id).Scan(
		&tx.ID, &tx.LedgerID, &tx.Type, &tx.Title, &tx.Amount, &tx.Currency, &occurredAtStr,
		&tx.OwnerUserID, &tx.CreatedByUserID, &tx.PayerUserID, &accountID, &categoryID,
		&tx.Visibility, &splitMethod, &note, &tx.Status, &createdAtStr, &updatedAtStr, &deletedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errors.New("transaction not found")
		}
		return nil, nil, err
	}

	// 解析时间
	tx.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
	tx.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	tx.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	if deletedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAtStr.String)
		tx.DeletedAt = sql.NullTime{Time: t, Valid: true}
	}

	tx.AccountID = accountID
	tx.CategoryID = categoryID
	tx.SplitMethod = splitMethod
	tx.Note = note

	// 查询标签名
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.name 
		FROM tags t
		JOIN transaction_tags tt ON t.id = tt.tag_id
		WHERE tt.transaction_id = ?
	`, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tags = append(tags, name)
		}
	}

	return &tx, tags, nil
}

// UpdateWithTx 在事务中修改交易及关联标签
// @brief 事务内局部/全部修改流水参数并重构标签绑定
// @param ctx context.Context 上下文
// @param tx *sql.Tx 事务句柄，可为 nil
// @param transaction *Transaction 待更新的交易实体
// @param tags []string 新的标签名称列表
// @return error 错误信息
func (r *Repository) UpdateWithTx(ctx context.Context, tx *sql.Tx, transaction *Transaction, tags []string) error {
	executor := r.getExecutor(tx)
	now := time.Now().Format(time.RFC3339)

	_, err := executor.ExecContext(ctx, `
		UPDATE transactions
		SET type = ?, title = ?, amount = ?, occurred_at = ?,
			payer_user_id = ?, account_id = ?, category_id = ?,
			visibility = ?, note = ?, updated_at = ?
		WHERE id = ?
	`,
		transaction.Type, transaction.Title, transaction.Amount, transaction.OccurredAt.Format(time.RFC3339),
		transaction.PayerUserID, r.nullString(transaction.AccountID), r.nullString(transaction.CategoryID),
		transaction.Visibility, r.nullString(transaction.Note), now, transaction.ID,
	)
	if err != nil {
		return err
	}

	// 重构标签关系：删除旧的，插入新的
	_, err = executor.ExecContext(ctx, "DELETE FROM transaction_tags WHERE transaction_id = ?", transaction.ID)
	if err != nil {
		return err
	}

	if len(tags) > 0 {
		err = r.associateTags(ctx, executor, transaction.ID, transaction.LedgerID, tags, now)
		if err != nil {
			return err
		}
	}

	return nil
}

// SoftDeleteWithTx 软删除账单
// @brief 将指定账单状态置为 deleted 并写下删除时间
// @param ctx context.Context 上下文
// @param tx *sql.Tx 事务句柄
// @param id string 交易 ID
// @param deletedAt time.Time 删除时间
// @return error 错误信息
func (r *Repository) SoftDeleteWithTx(ctx context.Context, tx *sql.Tx, id string, deletedAt time.Time) error {
	executor := r.getExecutor(tx)
	now := deletedAt.Format(time.RFC3339)

	_, err := executor.ExecContext(ctx, `
		UPDATE transactions
		SET status = 'deleted', deleted_at = ?, updated_at = ?
		WHERE id = ?
	`, now, now, id)
	return err
}

// TransactionFilter 查询过滤参数
type TransactionFilter struct {
	Month       string
	Type        string
	CategoryID  string
	Keyword     string
	MinAmount   *int64 // 分为单位
	MaxAmount   *int64 // 分为单位
	PayerUserID string
	Visibility  string
	Tag         string
	Page        int
	PageSize    int
}

// List 拉取流水列表 (加入安全可见性审计)
// @brief 根据当前登录用户和可见性策略拉取账单流水列表，并支持过滤与分页
// @param ctx context.Context 上下文
// @param ledgerID string 当前账本 ID
// @param userID string 登录用户 ID (用于 private 可见性鉴权)
// @param filter TransactionFilter 过滤条件参数
// @return []*Transaction 账单交易实体列表
// @return map[string][]string 关联标签名映射 map [transaction_id] -> [tag_name1, tag_name2]
// @return error 错误信息
func (r *Repository) List(ctx context.Context, ledgerID string, userID string, filter TransactionFilter) ([]*Transaction, map[string][]string, error) {
	query := `
		SELECT 
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id,
			visibility, split_method, note, status, created_at, updated_at
		FROM transactions
		WHERE ledger_id = ? AND status != 'deleted'
		AND (
			created_by_user_id = ? 
			OR owner_user_id = ? 
			OR payer_user_id = ? 
			OR visibility IN ('partner_readable', 'shared')
		)
	`
	args := []interface{}{ledgerID, userID, userID, userID}

	// 月度过滤 (occurred_at 是 ISO 字符串，通过 LIKE '2025-04%' 匹配)
	if filter.Month != "" {
		query += " AND occurred_at LIKE ?"
		args = append(args, filter.Month+"%")
	}

	// 类型过滤
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}

	// 分类过滤
	if filter.CategoryID != "" {
		query += " AND category_id = ?"
		args = append(args, filter.CategoryID)
	}

	// 关键字搜索
	if filter.Keyword != "" {
		query += " AND (title LIKE ? OR note LIKE ?)"
		args = append(args, "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}

	// 金额下限过滤
	if filter.MinAmount != nil {
		query += " AND amount >= ?"
		args = append(args, *filter.MinAmount)
	}

	// 金额上限过滤
	if filter.MaxAmount != nil {
		query += " AND amount <= ?"
		args = append(args, *filter.MaxAmount)
	}

	// 付款人过滤
	if filter.PayerUserID != "" {
		query += " AND payer_user_id = ?"
		args = append(args, filter.PayerUserID)
	}

	// 可见性过滤
	if filter.Visibility != "" {
		query += " AND visibility = ?"
		args = append(args, filter.Visibility)
	}

	// 标签名称过滤
	if filter.Tag != "" {
		query += " AND id IN (SELECT tt.transaction_id FROM transaction_tags tt JOIN tags tg ON tt.tag_id = tg.id WHERE tg.name = ?)"
		args = append(args, filter.Tag)
	}

	// 按时间倒序排序
	query += " ORDER BY occurred_at DESC, created_at DESC"

	// 分页处理
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	query += " LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var list []*Transaction
	var ids []string
	for rows.Next() {
		var tx Transaction
		var accountID, categoryID, splitMethod, note sql.NullString
		var occurredAtStr, createdAtStr, updatedAtStr string

		err := rows.Scan(
			&tx.ID, &tx.LedgerID, &tx.Type, &tx.Title, &tx.Amount, &tx.Currency, &occurredAtStr,
			&tx.OwnerUserID, &tx.CreatedByUserID, &tx.PayerUserID, &accountID, &categoryID,
			&tx.Visibility, &splitMethod, &note, &tx.Status, &createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, nil, err
		}

		tx.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
		tx.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		tx.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		tx.AccountID = accountID
		tx.CategoryID = categoryID
		tx.SplitMethod = splitMethod
		tx.Note = note

		list = append(list, &tx)
		ids = append(ids, tx.ID)
	}

	// 批量拉取标签以防 N+1 查询问题
	tagMap := make(map[string][]string)
	if len(ids) == 0 {
		return list, tagMap, nil
	}

	// SQLite IN 子句绑定
	placeholders := make([]string, len(ids))
	tagArgs := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		tagArgs[i] = id
	}

	tagQuery := fmt.Sprintf(`
		SELECT tt.transaction_id, t.name
		FROM tags t
		JOIN transaction_tags tt ON t.id = tt.tag_id
		WHERE tt.transaction_id IN (%s)
	`, strings.Join(placeholders, ","))

	tagRows, err := r.db.QueryContext(ctx, tagQuery, tagArgs...)
	if err != nil {
		return list, tagMap, nil // 标签拉取失败不阻断账单列表的呈现
	}
	defer tagRows.Close()

	for tagRows.Next() {
		var txID, tagName string
		if err := tagRows.Scan(&txID, &tagName); err == nil {
			tagMap[txID] = append(tagMap[txID], tagName)
		}
	}

	return list, tagMap, nil
}

// CreateAuditLogWithTx 写入审计日志
// @brief 记录账目变更审计明细
// @param ctx context.Context 上下文
// @param tx *sql.Tx 事务句柄
// @param log *AuditLog 审计行模型
// @return error 错误信息
func (r *Repository) CreateAuditLogWithTx(ctx context.Context, tx *sql.Tx, log *AuditLog) error {
	executor := r.getExecutor(tx)
	now := time.Now().Format(time.RFC3339)

	_, err := executor.ExecContext(ctx, `
		INSERT INTO audit_logs (id, ledger_id, actor_user_id, action, entity_type, entity_id, before_json, after_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.NewString(), log.LedgerID, log.ActorUserID, log.Action, log.EntityType, log.EntityID, log.BeforeJSON, log.AfterJSON, now)
	return err
}

type dbExecutor interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// 辅助方法：判断并获取 executor
func (r *Repository) getExecutor(tx *sql.Tx) dbExecutor {
	if tx != nil {
		return tx
	}
	return r.db
}

// 辅助方法：处理 sql.NullString
func (r *Repository) nullString(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// 辅助方法：关联并插入标签
func (r *Repository) associateTags(ctx context.Context, executor dbExecutor, transactionID, ledgerID string, tags []string, now string) error {
	for _, name := range tags {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// 查询标签是否存在 (强制复用事务 executor 防止 SQLite 内存数据库多连接池分流报错)
		var tagID string
		err := executor.QueryRowContext(ctx, "SELECT id FROM tags WHERE ledger_id = ? AND name = ?", ledgerID, name).Scan(&tagID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// 插入新标签
				tagID = uuid.NewString()
				_, err = executor.ExecContext(ctx, `
					INSERT INTO tags (id, ledger_id, name, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?)
				`, tagID, ledgerID, name, now, now)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// 插入中间关联表
		_, err = executor.ExecContext(ctx, `
			INSERT OR IGNORE INTO transaction_tags (transaction_id, tag_id)
			VALUES (?, ?)
		`, transactionID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateSplitsWithTx 在事务中批量写入分摊数据
// @brief 事务内写入多条分摊拆分流水
// @param ctx context.Context 上下文
// @param tx *sql.Tx 事务句柄
// @param splits []TransactionSplit 分摊实体列表
// @return error 错误信息
func (r *Repository) CreateSplitsWithTx(ctx context.Context, tx *sql.Tx, splits []TransactionSplit) error {
	executor := r.getExecutor(tx)
	now := time.Now().Format(time.RFC3339)

	for _, split := range splits {
		_, err := executor.ExecContext(ctx, `
			INSERT INTO transaction_splits (id, transaction_id, user_id, share_amount, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, split.ID, split.TransactionID, split.UserID, split.ShareAmount, now, now)
		if err != nil {
			return fmt.Errorf("insert transaction split failed: %w", err)
		}
	}
	return nil
}

// GetSplitsByTxID 查询单笔交易的分摊行
// @brief 获取交易对应的所有用户分摊详情
// @param ctx context.Context 上下文
// @param txID string 交易 ID
// @return []SplitResponse 分摊详情 DTO 列表
// @return error 错误信息
func (r *Repository) GetSplitsByTxID(ctx context.Context, txID string) ([]SplitResponse, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, share_amount 
		FROM transaction_splits 
		WHERE transaction_id = ?
	`, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var splits []SplitResponse
	for rows.Next() {
		var s SplitResponse
		if err := rows.Scan(&s.UserID, &s.ShareAmountCents); err == nil {
			splits = append(splits, s)
		}
	}
	return splits, nil
}

// GetSplitsByTxIDs 批量查询分摊列表以防止 N+1
// @brief 根据交易 ID 集合批量抓取分摊数据
// @param ctx context.Context 上下文
// @param txIDs []string 交易 ID 列表
// @return map[string][]SplitResponse 关联分摊 Map [transaction_id] -> []SplitResponse
// @return error 错误信息
func (r *Repository) GetSplitsByTxIDs(ctx context.Context, txIDs []string) (map[string][]SplitResponse, error) {
	splitMap := make(map[string][]SplitResponse)
	if len(txIDs) == 0 {
		return splitMap, nil
	}

	placeholders := make([]string, len(txIDs))
	args := make([]interface{}, len(txIDs))
	for i, id := range txIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT transaction_id, user_id, share_amount
		FROM transaction_splits
		WHERE transaction_id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var txID string
		var s SplitResponse
		if err := rows.Scan(&txID, &s.UserID, &s.ShareAmountCents); err == nil {
			splitMap[txID] = append(splitMap[txID], s)
		}
	}
	return splitMap, nil
}

// Category 分类列表返回 DTO
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListCategories 查询账本下所有的系统分类列表
// @brief 从 categories 数据库表中拉取当前 ledger 对应的分类
// @param ctx context.Context 上下文
// @param ledgerID string 账本 ID
// @return []Category 分类数据列表
// @return error 错误信息
func (r *Repository) ListCategories(ctx context.Context, ledgerID string) ([]Category, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name FROM categories WHERE ledger_id = ? ORDER BY sort_order ASC", ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

// CreateTemplate 创建账单模板
func (r *Repository) CreateTemplate(ctx context.Context, tmpl *TransactionTemplate) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO transaction_templates (
			id, ledger_id, name, type, title, amount_cents,
			category_id, account_id, payer_user_id, split_method,
			tag_names, note, created_by_user_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tmpl.ID, tmpl.LedgerID, tmpl.Name, tmpl.Type, tmpl.Title, tmpl.AmountCents,
		tmpl.CategoryID, tmpl.AccountID, tmpl.PayerUserID, tmpl.SplitMethod,
		tmpl.TagNames, tmpl.Note, tmpl.CreatedByUserID, now, now)
	return err
}

// GetTemplateByID 根据 ID 查询单个账单模板
func (r *Repository) GetTemplateByID(ctx context.Context, id string) (*TransactionTemplate, error) {
	var tmpl TransactionTemplate
	var occurredAtStr string // 未使用，占位符或为类型兼容性
	_ = occurredAtStr
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, ledger_id, name, type, title, amount_cents,
			category_id, account_id, payer_user_id, split_method,
			tag_names, note, created_by_user_id, created_at, updated_at
		FROM transaction_templates
		WHERE id = ?
	`, id).Scan(
		&tmpl.ID, &tmpl.LedgerID, &tmpl.Name, &tmpl.Type, &tmpl.Title, &tmpl.AmountCents,
		&tmpl.CategoryID, &tmpl.AccountID, &tmpl.PayerUserID, &tmpl.SplitMethod,
		&tmpl.TagNames, &tmpl.Note, &tmpl.CreatedByUserID, &occurredAtStr, &occurredAtStr, // 实际上是 createdAt/updatedAt 对应 TEXT，以格式兼容存入即可
	)
	if err != nil {
		return nil, err
	}
	// 解析时间
	tmpl.CreatedAt, _ = time.Parse(time.RFC3339, occurredAtStr)
	tmpl.UpdatedAt, _ = time.Parse(time.RFC3339, occurredAtStr)
	return &tmpl, nil
}

// ListTemplates 获取指定账本下的所有模板列表
func (r *Repository) ListTemplates(ctx context.Context, ledgerID string) ([]*TransactionTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			id, ledger_id, name, type, title, amount_cents,
			category_id, account_id, payer_user_id, split_method,
			tag_names, note, created_by_user_id, created_at, updated_at
		FROM transaction_templates
		WHERE ledger_id = ?
		ORDER BY created_at DESC
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*TransactionTemplate
	for rows.Next() {
		var tmpl TransactionTemplate
		var createdAtStr, updatedAtStr string
		err := rows.Scan(
			&tmpl.ID, &tmpl.LedgerID, &tmpl.Name, &tmpl.Type, &tmpl.Title, &tmpl.AmountCents,
			&tmpl.CategoryID, &tmpl.AccountID, &tmpl.PayerUserID, &tmpl.SplitMethod,
			&tmpl.TagNames, &tmpl.Note, &tmpl.CreatedByUserID, &createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}
		tmpl.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		tmpl.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		templates = append(templates, &tmpl)
	}
	return templates, nil
}

// UpdateTemplate 全量更新指定模板属性
func (r *Repository) UpdateTemplate(ctx context.Context, tmpl *TransactionTemplate) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE transaction_templates
		SET 
			name = ?, type = ?, title = ?, amount_cents = ?,
			category_id = ?, account_id = ?, payer_user_id = ?,
			split_method = ?, tag_names = ?, note = ?, updated_at = ?
		WHERE id = ? AND ledger_id = ?
	`, tmpl.Name, tmpl.Type, tmpl.Title, tmpl.AmountCents,
		tmpl.CategoryID, tmpl.AccountID, tmpl.PayerUserID,
		tmpl.SplitMethod, tmpl.TagNames, tmpl.Note, now,
		tmpl.ID, tmpl.LedgerID)
	return err
}

// DeleteTemplate 删除指定模板
func (r *Repository) DeleteTemplate(ctx context.Context, id string, ledgerID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM transaction_templates
		WHERE id = ? AND ledger_id = ?
	`, id, ledgerID)
	return err
}

// CreateRecurringRule 创建周期账单规则
func (r *Repository) CreateRecurringRule(ctx context.Context, rule *RecurringRule) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO recurring_rules (
			id, ledger_id, name, type, title, amount_cents,
			category_id, payer_user_id, split_method, tag_names,
			note, frequency, next_due_date, created_by_user_id,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rule.ID, rule.LedgerID, rule.Name, rule.Type, rule.Title, rule.AmountCents,
		rule.CategoryID, rule.PayerUserID, rule.SplitMethod, rule.TagNames,
		rule.Note, rule.Frequency, rule.NextDueDate, rule.CreatedByUserID,
		now, now)
	return err
}

// GetRecurringRuleByID 根据 ID 获取周期账单规则
func (r *Repository) GetRecurringRuleByID(ctx context.Context, id string) (*RecurringRule, error) {
	var rule RecurringRule
	var createdAtStr, updatedAtStr string
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, ledger_id, name, type, title, amount_cents,
			category_id, payer_user_id, split_method, tag_names,
			note, frequency, next_due_date, created_by_user_id,
			created_at, updated_at
		FROM recurring_rules
		WHERE id = ?
	`, id).Scan(
		&rule.ID, &rule.LedgerID, &rule.Name, &rule.Type, &rule.Title, &rule.AmountCents,
		&rule.CategoryID, &rule.PayerUserID, &rule.SplitMethod, &rule.TagNames,
		&rule.Note, &rule.Frequency, &rule.NextDueDate, &rule.CreatedByUserID,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	rule.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	rule.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &rule, nil
}

// ListRecurringRules 获取账本下的周期账单规则列表
func (r *Repository) ListRecurringRules(ctx context.Context, ledgerID string) ([]*RecurringRule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			id, ledger_id, name, type, title, amount_cents,
			category_id, payer_user_id, split_method, tag_names,
			note, frequency, next_due_date, created_by_user_id,
			created_at, updated_at
		FROM recurring_rules
		WHERE ledger_id = ?
		ORDER BY created_at DESC
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*RecurringRule
	for rows.Next() {
		var rule RecurringRule
		var createdAtStr, updatedAtStr string
		err := rows.Scan(
			&rule.ID, &rule.LedgerID, &rule.Name, &rule.Type, &rule.Title, &rule.AmountCents,
			&rule.CategoryID, &rule.PayerUserID, &rule.SplitMethod, &rule.TagNames,
			&rule.Note, &rule.Frequency, &rule.NextDueDate, &rule.CreatedByUserID,
			&createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}
		rule.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		rule.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		rules = append(rules, &rule)
	}
	return rules, nil
}

// DeleteRecurringRule 删除指定的周期规则
func (r *Repository) DeleteRecurringRule(ctx context.Context, id string, ledgerID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM recurring_rules
		WHERE id = ? AND ledger_id = ?
	`, id, ledgerID)
	return err
}

// CreateRecurringReminder 创建待确认周期账单提醒
func (r *Repository) CreateRecurringReminder(ctx context.Context, tx *sql.Tx, reminder *RecurringReminder) error {
	executor := r.getExecutor(tx)
	now := time.Now().Format(time.RFC3339)
	_, err := executor.ExecContext(ctx, `
		INSERT INTO recurring_reminders (
			id, ledger_id, rule_id, due_date, status,
			transaction_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, reminder.ID, reminder.LedgerID, reminder.RuleID, reminder.DueDate,
		reminder.Status, reminder.TransactionID, now, now)
	return err
}

// GetRecurringReminderByID 查询单条周期提醒
func (r *Repository) GetRecurringReminderByID(ctx context.Context, id string) (*RecurringReminder, error) {
	var reminder RecurringReminder
	var createdAtStr, updatedAtStr string
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, ledger_id, rule_id, due_date, status,
			transaction_id, created_at, updated_at
		FROM recurring_reminders
		WHERE id = ?
	`, id).Scan(
		&reminder.ID, &reminder.LedgerID, &reminder.RuleID, &reminder.DueDate,
		&reminder.Status, &reminder.TransactionID, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	reminder.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	reminder.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &reminder, nil
}

// ListRecurringRemindersWithDetails 获取账本下的周期账单提醒（JOIN 包含规则详细参数，以便前端渲染）
func (r *Repository) ListRecurringRemindersWithDetails(ctx context.Context, ledgerID string) ([]*RecurringReminderDetail, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			r.id, r.ledger_id, r.rule_id, r.due_date, r.status, r.transaction_id, r.created_at, r.updated_at,
			u.name, u.type, u.title, u.amount_cents, u.category_id, c.name, u.payer_user_id, u.split_method,
			u.tag_names, u.note, u.frequency
		FROM recurring_reminders r
		INNER JOIN recurring_rules u ON r.rule_id = u.id
		LEFT JOIN categories c ON u.category_id = c.id
		WHERE r.ledger_id = ? AND r.status = 'pending'
		ORDER BY r.due_date DESC, r.created_at DESC
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var details []*RecurringReminderDetail
	for rows.Next() {
		var reminder RecurringReminder
		var detail RecurringReminderDetail
		var catName sql.NullString
		var createdAtStr, updatedAtStr string
		err := rows.Scan(
			&reminder.ID, &reminder.LedgerID, &reminder.RuleID, &reminder.DueDate, &reminder.Status, &reminder.TransactionID, &createdAtStr, &updatedAtStr,
			&detail.RuleName, &detail.Type, &detail.Title, &detail.AmountCents, &detail.CategoryID, &catName, &detail.PayerUserID, &detail.SplitMethod,
			&detail.TagNames, &detail.Note, &detail.Frequency,
		)
		if err != nil {
			return nil, err
		}
		reminder.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		reminder.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		detail.Reminder = &reminder
		detail.CategoryName = catName
		details = append(details, &detail)
	}
	return details, nil
}

// UpdateRecurringReminderStatusWithTx 在事务（或非事务）中更新提醒状态
func (r *Repository) UpdateRecurringReminderStatusWithTx(ctx context.Context, tx *sql.Tx, id string, ledgerID string, status string, txID sql.NullString) error {
	executor := r.getExecutor(tx)
	now := time.Now().Format(time.RFC3339)
	_, err := executor.ExecContext(ctx, `
		UPDATE recurring_reminders
		SET status = ?, transaction_id = ?, updated_at = ?
		WHERE id = ? AND ledger_id = ?
	`, status, txID, now, id, ledgerID)
	return err
}

// UpdateRecurringRuleNextDueDateWithTx 更新规则的下一次触发时间
func (r *Repository) UpdateRecurringRuleNextDueDateWithTx(ctx context.Context, tx *sql.Tx, id string, nextDueDate string) error {
	executor := r.getExecutor(tx)
	now := time.Now().Format(time.RFC3339)
	_, err := executor.ExecContext(ctx, `
		UPDATE recurring_rules
		SET next_due_date = ?, updated_at = ?
		WHERE id = ?
	`, nextDueDate, now, id)
	return err
}
