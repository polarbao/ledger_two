package transaction

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	appErrors "ledger_two/internal/errors"
)

// Service 交易明细核心业务逻辑服务
type Service struct {
	repo *Repository
}

// NewService 实例化业务服务
// @brief 创建 Transaction 的 Service 实例
// @param repo *Repository 数据库仓库句柄
// @return *Service 服务实例
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create 记账流水业务实现
// @brief 业务校验并安全创建普通收支账单
// @param ctx context.Context 上下文
// @param currentUserID string 当前操作用户 ID
// @param req CreateTransactionRequest 创建参数
// @return *TransactionResponse 创建成功后的 DTO
// @return error 错误对象
func (s *Service) Create(ctx context.Context, currentUserID string, req CreateTransactionRequest) (*TransactionResponse, error) {
	// 1. 金额校验
	if req.AmountCents <= 0 {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "金额必须大于 0")
	}

	// 2. 类型校验
	if req.Type != "expense" && req.Type != "income" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "记账类型必须为 expense 或 income")
	}

	// 3. 时间校验
	occurredAt, err := time.Parse(time.RFC3339, req.OccurredAt)
	if err != nil {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "交易时间格式必须符合 ISO8601 标准")
	}

	// 4. 用户存在校验
	if req.PayerUserID == "" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "付款人用户 ID 不能为空")
	}

	// 5. 获取全局唯一 LedgerID
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 6. 可见性处理
	visibility := req.Visibility
	if visibility == "" {
		visibility = "private"
	}
	if visibility != "private" && visibility != "partner_readable" && visibility != "shared" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "无效的可见性属性值")
	}

	// 7. 标题 fallback
	title := req.Title
	if title == "" {
		if req.CategoryID != nil && *req.CategoryID != "" {
			title = s.getCategoryName(ctx, *req.CategoryID)
		}
		if title == "" {
			title = "未分类流水"
		}
	}

	// 8. 构造实体并执行写入
	txID := uuid.NewString()
	var accountVal sql.NullString
	if req.AccountID != nil {
		accountVal = sql.NullString{String: *req.AccountID, Valid: true}
	}
	var categoryVal sql.NullString
	if req.CategoryID != nil {
		categoryVal = sql.NullString{String: *req.CategoryID, Valid: true}
	}

	txModel := &Transaction{
		ID:              txID,
		LedgerID:        ledgerID,
		Type:            req.Type,
		Title:           title,
		Amount:          req.AmountCents,
		Currency:        "CNY",
		OccurredAt:      occurredAt,
		OwnerUserID:     currentUserID, // 谁记的，所有人默认是记账人自己
		CreatedByUserID: currentUserID,
		PayerUserID:     req.PayerUserID,
		AccountID:       accountVal,
		CategoryID:      categoryVal,
		Visibility:      visibility,
		Note:            sql.NullString{String: req.Note, Valid: req.Note != ""},
	}

	dbConn := s.repo.GetDB()
	dbTx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer dbTx.Rollback()

	err = s.repo.CreateWithTx(ctx, dbTx, txModel, req.TagNames)
	if err != nil {
		return nil, err
	}

	err = dbTx.Commit()
	if err != nil {
		return nil, err
	}

	return s.toDTO(txModel, req.TagNames), nil
}

// GetByID 单条流水详情查询
// @brief 获取流水详情并完成可见性鉴权隔离
// @param ctx context.Context 上下文
// @param currentUserID string 访问者用户 ID
// @param id string 交易账单 ID
// @return *TransactionResponse 账单明细数据
// @return error 异常对象
func (s *Service) GetByID(ctx context.Context, currentUserID string, id string) (*TransactionResponse, error) {
	tx, tags, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.NewAppError(404, "NOT_FOUND", "账单未找到或已删除")
	}

	// 校验查看权限
	if !s.CanViewTransaction(currentUserID, tx) {
		return nil, appErrors.NewAppError(404, "NOT_FOUND", "账单未找到或已删除")
	}

	dto := s.toDTO(tx, tags)
	if tx.Type == "shared_expense" {
		splits, err := s.repo.GetSplitsByTxID(ctx, id)
		if err == nil {
			dto.SplitMethod = &tx.SplitMethod.String
			dto.Participants = splits
		}
	}

	return dto, nil
}

// Update 局部修改账单流水
// @brief 业务校验并更新单条账单，更新前后的变更数据同步写入审计日志
// @param ctx context.Context 上下文
// @param currentUserID string 编辑操作人用户 ID
// @param id string 待修改账单 ID
// @param req UpdateTransactionRequest 变动行属性
// @return *TransactionResponse 更新后的账单 DTO
// @return error 异常对象
func (s *Service) Update(ctx context.Context, currentUserID string, id string, req UpdateTransactionRequest) (*TransactionResponse, error) {
	tx, oldTags, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.NewAppError(404, "NOT_FOUND", "账单未找到")
	}

	// 校验查看权限以防越权探测
	if !s.CanViewTransaction(currentUserID, tx) {
		return nil, appErrors.NewAppError(404, "NOT_FOUND", "账单未找到")
	}

	// 校验编辑权限：谁创建谁编辑，被删除的拒绝编辑
	if !s.CanEditTransaction(currentUserID, tx) {
		return nil, appErrors.NewAppError(403, "FORBIDDEN", "无权编辑此账单")
	}

	isShared := tx.Type == "shared_expense"
	var oldSplits []SplitResponse
	if isShared {
		oldSplits, _ = s.repo.GetSplitsByTxID(ctx, id)
	}

	oldDTO := s.toDTO(tx, oldTags)
	if isShared {
		oldDTO.SplitMethod = &tx.SplitMethod.String
		oldDTO.Participants = oldSplits
	}
	beforeJSONBytes, _ := json.Marshal(oldDTO)

	// 应用更新属性并校验
	if req.Title != nil {
		tx.Title = *req.Title
	}
	if req.AmountCents != nil {
		if *req.AmountCents <= 0 {
			return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "金额必须大于 0")
		}
		tx.Amount = *req.AmountCents
	}
	if req.OccurredAt != nil {
		occurredAt, err := time.Parse(time.RFC3339, *req.OccurredAt)
		if err != nil {
			return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "交易时间格式错误")
		}
		tx.OccurredAt = occurredAt
	}
	if req.PayerUserID != nil {
		if *req.PayerUserID == "" {
			return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "付款人用户 ID 不能为空")
		}
		tx.PayerUserID = *req.PayerUserID
	}
	if req.AccountID != nil {
		if *req.AccountID == nil {
			tx.AccountID = sql.NullString{Valid: false}
		} else {
			tx.AccountID = sql.NullString{String: **req.AccountID, Valid: true}
		}
	}
	if req.CategoryID != nil {
		if *req.CategoryID == nil {
			tx.CategoryID = sql.NullString{Valid: false}
		} else {
			tx.CategoryID = sql.NullString{String: **req.CategoryID, Valid: true}
		}
	}
	if req.Visibility != nil {
		val := *req.Visibility
		if val != "private" && val != "partner_readable" && val != "shared" {
			return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "无效的可见性属性值")
		}
		tx.Visibility = val
	}
	if req.Note != nil {
		tx.Note = sql.NullString{String: *req.Note, Valid: *req.Note != ""}
	}

	tags := oldTags
	if req.TagNames != nil {
		tags = *req.TagNames
	}

	var newSplits []TransactionSplit
	var splitMethodVal string
	if isShared {
		if req.SplitMethod != nil {
			splitMethodVal = *req.SplitMethod
			if splitMethodVal != "equal" && splitMethodVal != "payer_only" {
				return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "无效的分摊方式")
			}
			tx.SplitMethod = sql.NullString{String: splitMethodVal, Valid: true}
		} else if tx.SplitMethod.Valid {
			splitMethodVal = tx.SplitMethod.String
		}

		// 重新计算分摊金额
		users, err := s.getSystemUsers(ctx)
		if err != nil {
			return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统用户失败")
		}

		payerID := tx.PayerUserID
		var otherID string
		foundPayer := false
		for _, u := range users {
			if u == payerID {
				foundPayer = true
			} else {
				otherID = u
			}
		}
		if !foundPayer {
			return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "无效的付款人")
		}

		var payerShare, otherShare int64
		if splitMethodVal == "equal" {
			base := tx.Amount / 2
			rem := tx.Amount % 2
			payerShare = base + rem
			otherShare = base
		} else if splitMethodVal == "payer_only" {
			payerShare = tx.Amount
			otherShare = 0
		}

		newSplits = []TransactionSplit{
			{ID: uuid.NewString(), TransactionID: tx.ID, UserID: payerID, ShareAmount: payerShare},
			{ID: uuid.NewString(), TransactionID: tx.ID, UserID: otherID, ShareAmount: otherShare},
		}
	}

	// 事务内提交修改与审计
	dbConn := s.repo.GetDB()
	dbTx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer dbTx.Rollback()

	err = s.repo.UpdateWithTx(ctx, dbTx, tx, tags)
	if err != nil {
		return nil, err
	}

	if isShared {
		// 删除旧分摊
		_, err = dbTx.ExecContext(ctx, "DELETE FROM transaction_splits WHERE transaction_id = ?", tx.ID)
		if err != nil {
			return nil, err
		}
		// 写入新分摊
		err = s.repo.CreateSplitsWithTx(ctx, dbTx, newSplits)
		if err != nil {
			return nil, err
		}
	}

	afterDTO := s.toDTO(tx, tags)
	var newSplitsDTO []SplitResponse
	if isShared {
		afterDTO.SplitMethod = &splitMethodVal
		for _, ns := range newSplits {
			newSplitsDTO = append(newSplitsDTO, SplitResponse{
				UserID:           ns.UserID,
				ShareAmountCents: ns.ShareAmount,
			})
		}
		afterDTO.Participants = newSplitsDTO
	}
	afterJSONBytes, _ := json.Marshal(afterDTO)

	// 审计日志
	auditLog := &AuditLog{
		LedgerID:    tx.LedgerID,
		ActorUserID: currentUserID,
		Action:      "update",
		EntityType:  "transaction",
		EntityID:    tx.ID,
		BeforeJSON:  sql.NullString{String: string(beforeJSONBytes), Valid: true},
		AfterJSON:   sql.NullString{String: string(afterJSONBytes), Valid: true},
	}
	err = s.repo.CreateAuditLogWithTx(ctx, dbTx, auditLog)
	if err != nil {
		return nil, err
	}

	err = dbTx.Commit()
	if err != nil {
		return nil, err
	}

	return afterDTO, nil
}

// Delete 软删除账单明细
// @brief 执行软删除并入库对应的删除审计记录
// @param ctx context.Context 上下文
// @param currentUserID string 执行删除者 ID
// @param id string 交易账单 ID
// @return error 异常对象
func (s *Service) Delete(ctx context.Context, currentUserID string, id string) error {
	tx, tags, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return appErrors.NewAppError(404, "NOT_FOUND", "账单未找到")
	}

	// 校验查看权限以防越权探测
	if !s.CanViewTransaction(currentUserID, tx) {
		return appErrors.NewAppError(404, "NOT_FOUND", "账单未找到")
	}

	// 校验编辑/删除权限：谁创建谁删除
	if !s.CanEditTransaction(currentUserID, tx) {
		return appErrors.NewAppError(403, "FORBIDDEN", "无权删除此账单")
	}

	beforeDTO := s.toDTO(tx, tags)
	if tx.Type == "shared_expense" {
		splits, _ := s.repo.GetSplitsByTxID(ctx, id)
		beforeDTO.SplitMethod = &tx.SplitMethod.String
		beforeDTO.Participants = splits
	}
	beforeJSONBytes, _ := json.Marshal(beforeDTO)

	dbConn := s.repo.GetDB()
	dbTx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer dbTx.Rollback()

	now := time.Now()
	err = s.repo.SoftDeleteWithTx(ctx, dbTx, id, now)
	if err != nil {
		return err
	}

	// 写入审计
	auditLog := &AuditLog{
		LedgerID:    tx.LedgerID,
		ActorUserID: currentUserID,
		Action:      "delete",
		EntityType:  "transaction",
		EntityID:    tx.ID,
		BeforeJSON:  sql.NullString{String: string(beforeJSONBytes), Valid: true},
		AfterJSON:   sql.NullString{Valid: false},
	}
	err = s.repo.CreateAuditLogWithTx(ctx, dbTx, auditLog)
	if err != nil {
		return err
	}

	return dbTx.Commit()
}

// List 流水列表查询
// @brief 分页拉取列表并根据权限规则安全组装 DTO
// @param ctx context.Context 上下文
// @param currentUserID string 访问者 ID
// @param filter TransactionFilter 过滤条件体
// @return []*TransactionResponse 明细列表
// @return error 异常
func (s *Service) List(ctx context.Context, currentUserID string, filter TransactionFilter) ([]*TransactionResponse, error) {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	list, tagMap, err := s.repo.List(ctx, ledgerID, currentUserID, filter)
	if err != nil {
		return nil, err
	}

	var sharedTxIDs []string
	for _, tx := range list {
		if tx.Type == "shared_expense" {
			sharedTxIDs = append(sharedTxIDs, tx.ID)
		}
	}

	var splitMap map[string][]SplitResponse
	if len(sharedTxIDs) > 0 {
		splitMap, _ = s.repo.GetSplitsByTxIDs(ctx, sharedTxIDs)
	}

	var res []*TransactionResponse
	for _, tx := range list {
		tags := tagMap[tx.ID]
		dto := s.toDTO(tx, tags)
		if tx.Type == "shared_expense" && splitMap != nil {
			if splits, ok := splitMap[tx.ID]; ok {
				dto.SplitMethod = &tx.SplitMethod.String
				dto.Participants = splits
			}
		}
		res = append(res, dto)
	}
	return res, nil
}

// CanViewTransaction 可见性校验规则
// @brief 校验指定用户是否有权限查看某账单
// @param currentUserID string 访问用户 ID
// @param tx *Transaction 账单交易模型
// @return bool 可见返回 true
func (s *Service) CanViewTransaction(currentUserID string, tx *Transaction) bool {
	if tx.Status == "deleted" {
		return false
	}
	// 创建者、拥有者、付款者自己可见
	if tx.CreatedByUserID == currentUserID || tx.OwnerUserID == currentUserID || tx.PayerUserID == currentUserID {
		return true
	}
	// 伙伴可见
	if tx.Visibility == "partner_readable" || tx.Visibility == "shared" {
		return true
	}
	return false
}

// CanEditTransaction 编辑校验规则
// @brief 校验指定用户是否有权修改/删除某账单
// @param currentUserID string 操作人 ID
// @param tx *Transaction 账单交易模型
// @return bool 权限通过返回 true
func (s *Service) CanEditTransaction(currentUserID string, tx *Transaction) bool {
	return tx.Status != "deleted" && tx.CreatedByUserID == currentUserID
}

// 辅助转换
func (s *Service) toDTO(tx *Transaction, tags []string) *TransactionResponse {
	var account *string
	if tx.AccountID.Valid {
		account = &tx.AccountID.String
	}
	var category *string
	if tx.CategoryID.Valid {
		category = &tx.CategoryID.String
	}
	note := ""
	if tx.Note.Valid {
		note = tx.Note.String
	}

	return &TransactionResponse{
		ID:              tx.ID,
		Type:            tx.Type,
		Title:           tx.Title,
		AmountCents:     tx.Amount,
		Currency:        tx.Currency,
		OccurredAt:      tx.OccurredAt.Format(time.RFC3339),
		OwnerUserID:     tx.OwnerUserID,
		CreatedByUserID: tx.CreatedByUserID,
		PayerUserID:     tx.PayerUserID,
		AccountID:       account,
		CategoryID:      category,
		Visibility:      tx.Visibility,
		Note:            note,
		Status:          tx.Status,
		Tags:            tags,
		CreatedAt:       tx.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       tx.UpdatedAt.Format(time.RFC3339),
	}
}

// 辅助方法：查询唯一 LedgerID
func (s *Service) getLedgerID(ctx context.Context) (string, error) {
	var id string
	dbConn := s.repo.GetDB()
	err := dbConn.QueryRowContext(ctx, "SELECT id FROM ledgers LIMIT 1").Scan(&id)
	return id, err
}

// 辅助方法：查询分类名称
func (s *Service) getCategoryName(ctx context.Context, catID string) string {
	var name string
	dbConn := s.repo.GetDB()
	err := dbConn.QueryRowContext(ctx, "SELECT name FROM categories WHERE id = ?", catID).Scan(&name)
	if err != nil {
		return ""
	}
	return name
}

// 辅助方法：查询系统内的两个用户ID
func (s *Service) getSystemUsers(ctx context.Context) ([]string, error) {
	dbConn := s.repo.GetDB()
	rows, err := dbConn.QueryContext(ctx, "SELECT id FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			users = append(users, id)
		}
	}
	return users, nil
}

// CreateSharedExpense 共同支出记账业务实现
// @brief 业务校验、分摊计算并写入 transactions 及 transaction_splits 表
// @param ctx context.Context 上下文
// @param currentUserID string 记账人 ID
// @param req CreateSharedExpenseRequest 创建参数
// @return *TransactionResponse 创建后的 DTO
// @return error 错误对象
func (s *Service) CreateSharedExpense(ctx context.Context, currentUserID string, req CreateSharedExpenseRequest) (*TransactionResponse, error) {
	// 1. 金额校验
	if req.AmountCents <= 0 {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "金额必须大于 0")
	}

	// 2. 时间校验
	occurredAt, err := time.Parse(time.RFC3339, req.OccurredAt)
	if err != nil {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "交易时间格式必须符合 ISO8601 标准")
	}

	// 3. 分摊方式校验
	if req.SplitMethod != "equal" && req.SplitMethod != "payer_only" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "分摊类型必须为 equal 或 payer_only")
	}

	// 4. 用户校验与分摊计算
	users, err := s.getSystemUsers(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统用户失败")
	}

	// 校验付款人是否合法
	var otherUserID string
	foundPayer := false
	for _, u := range users {
		if u == req.PayerUserID {
			foundPayer = true
		} else {
			otherUserID = u
		}
	}

	if !foundPayer {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "付款人用户不在当前账本成员中")
	}

	// 计算分摊金额 (不能使用 float)
	var payerShare, otherShare int64
	if req.SplitMethod == "equal" {
		base := req.AmountCents / 2
		rem := req.AmountCents % 2
		payerShare = base + rem
		otherShare = base
	} else if req.SplitMethod == "payer_only" {
		payerShare = req.AmountCents
		otherShare = 0
	}

	// 5. 获取全局唯一 LedgerID
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 6. 标题 fallback
	title := req.Title
	if title == "" {
		if req.CategoryID != nil && *req.CategoryID != "" {
			title = s.getCategoryName(ctx, *req.CategoryID)
		}
		if title == "" {
			title = "未分类共同支出"
		}
	}

	// 7. 构造交易实体 (Type = shared_expense, Visibility = shared)
	txID := uuid.NewString()
	var categoryVal sql.NullString
	if req.CategoryID != nil {
		categoryVal = sql.NullString{String: *req.CategoryID, Valid: true}
	}

	txModel := &Transaction{
		ID:              txID,
		LedgerID:        ledgerID,
		Type:            "shared_expense",
		Title:           title,
		Amount:          req.AmountCents,
		Currency:        "CNY",
		OccurredAt:      occurredAt,
		OwnerUserID:     currentUserID,
		CreatedByUserID: currentUserID,
		PayerUserID:     req.PayerUserID,
		CategoryID:      categoryVal,
		Visibility:      "shared",
		SplitMethod:     sql.NullString{String: req.SplitMethod, Valid: true},
		Note:            sql.NullString{String: req.Note, Valid: req.Note != ""},
	}

	// 8. 构造分摊实体
	splits := []TransactionSplit{
		{
			ID:            uuid.NewString(),
			TransactionID: txID,
			UserID:        req.PayerUserID,
			ShareAmount:   payerShare,
		},
		{
			ID:            uuid.NewString(),
			TransactionID: txID,
			UserID:        otherUserID,
			ShareAmount:   otherShare,
		},
	}

	// 9. 事务内打包写入
	dbConn := s.repo.GetDB()
	dbTx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer dbTx.Rollback()

	// 写入 transactions & tags
	err = s.repo.CreateWithTx(ctx, dbTx, txModel, req.TagNames)
	if err != nil {
		return nil, err
	}

	// 写入 transaction_splits
	err = s.repo.CreateSplitsWithTx(ctx, dbTx, splits)
	if err != nil {
		return nil, err
	}

	err = dbTx.Commit()
	if err != nil {
		return nil, err
	}

	// 组装 Response DTO
	dto := s.toDTO(txModel, req.TagNames)
	dto.SplitMethod = &req.SplitMethod
	dto.Participants = []SplitResponse{
		{UserID: req.PayerUserID, ShareAmountCents: payerShare},
		{UserID: otherUserID, ShareAmountCents: otherShare},
	}

	return dto, nil
}

// ListCategories 获取当前账本下所有的系统分类列表
// @brief 获取当前账本的 ledgerID 并通过 repo 读取消费分类
// @param ctx context.Context 上下文
// @return []Category 分类数据列表
// @return error 错误信息
func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}
	return s.repo.ListCategories(ctx, ledgerID)
}

// toTemplateResponse 辅助实体转换为统一 DTO 输出模型
func (s *Service) toTemplateResponse(tmpl *TransactionTemplate) *TemplateResponse {
	var amount *int64
	if tmpl.AmountCents.Valid {
		val := tmpl.AmountCents.Int64
		amount = &val
	}

	tags := []string{}
	if tmpl.TagNames.Valid && tmpl.TagNames.String != "" {
		tags = strings.Split(tmpl.TagNames.String, ",")
	}

	return &TemplateResponse{
		ID:              tmpl.ID,
		Name:            tmpl.Name,
		Type:            tmpl.Type,
		Title:           tmpl.Title.String,
		AmountCents:     amount,
		CategoryID:      tmpl.CategoryID.String,
		AccountID:       tmpl.AccountID.String,
		PayerUserID:     tmpl.PayerUserID.String,
		SplitMethod:     tmpl.SplitMethod.String,
		TagNames:        tags,
		Note:            tmpl.Note.String,
		CreatedByUserID: tmpl.CreatedByUserID,
		CreatedAt:       tmpl.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       tmpl.UpdatedAt.Format(time.RFC3339),
	}
}

// CreateTemplate 创建模板业务逻辑
func (s *Service) CreateTemplate(ctx context.Context, currentUserID string, req CreateTemplateRequest) (*TemplateResponse, error) {
	if req.Name == "" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "模板名称不能为空")
	}

	if req.Type != "expense" && req.Type != "income" && req.Type != "shared_expense" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "无效的记账模板类型")
	}

	if req.AmountCents != nil && *req.AmountCents < 0 {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "模板金额必须大于或等于 0")
	}

	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 拼装实体
	tmplID := uuid.NewString()
	var titleVal sql.NullString
	if req.Title != nil {
		titleVal = sql.NullString{String: *req.Title, Valid: true}
	}
	var amountVal sql.NullInt64
	if req.AmountCents != nil {
		amountVal = sql.NullInt64{Int64: *req.AmountCents, Valid: true}
	}
	var categoryVal sql.NullString
	if req.CategoryID != nil {
		categoryVal = sql.NullString{String: *req.CategoryID, Valid: true}
	}
	var accountVal sql.NullString
	if req.AccountID != nil {
		accountVal = sql.NullString{String: *req.AccountID, Valid: true}
	}
	var payerVal sql.NullString
	if req.PayerUserID != nil {
		payerVal = sql.NullString{String: *req.PayerUserID, Valid: true}
	}
	var splitVal sql.NullString
	if req.SplitMethod != nil {
		splitVal = sql.NullString{String: *req.SplitMethod, Valid: true}
	}
	var noteVal sql.NullString
	if req.Note != nil {
		noteVal = sql.NullString{String: *req.Note, Valid: true}
	}

	tagsStr := strings.Join(req.TagNames, ",")
	var tagsVal sql.NullString
	if tagsStr != "" {
		tagsVal = sql.NullString{String: tagsStr, Valid: true}
	}

	tmpl := &TransactionTemplate{
		ID:              tmplID,
		LedgerID:        ledgerID,
		Name:            req.Name,
		Type:            req.Type,
		Title:           titleVal,
		AmountCents:     amountVal,
		CategoryID:      categoryVal,
		AccountID:       accountVal,
		PayerUserID:     payerVal,
		SplitMethod:     splitVal,
		TagNames:        tagsVal,
		Note:            noteVal,
		CreatedByUserID: currentUserID,
	}

	if err := s.repo.CreateTemplate(ctx, tmpl); err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "创建交易模板失败")
	}

	tmpl.CreatedAt = time.Now()
	tmpl.UpdatedAt = time.Now()

	return s.toTemplateResponse(tmpl), nil
}

// GetTemplate 查询单个模板并进行越权校验
func (s *Service) GetTemplate(ctx context.Context, currentUserID string, id string) (*TemplateResponse, error) {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	tmpl, err := s.repo.GetTemplateByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.NewAppError(404, "NOT_FOUND", "账单模板未找到")
		}
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "读取模板失败")
	}

	if tmpl.LedgerID != ledgerID {
		return nil, appErrors.NewAppError(403, "FORBIDDEN", "无权查看该模板")
	}

	return s.toTemplateResponse(tmpl), nil
}

// ListTemplates 获取该账本下的所有模板列表
func (s *Service) ListTemplates(ctx context.Context, currentUserID string) ([]*TemplateResponse, error) {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	templates, err := s.repo.ListTemplates(ctx, ledgerID)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取模板列表失败")
	}

	var res []*TemplateResponse
	for _, t := range templates {
		res = append(res, s.toTemplateResponse(t))
	}
	return res, nil
}

// UpdateTemplate 更新模板业务逻辑
func (s *Service) UpdateTemplate(ctx context.Context, currentUserID string, id string, req CreateTemplateRequest) (*TemplateResponse, error) {
	if req.Name == "" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "模板名称不能为空")
	}

	if req.Type != "expense" && req.Type != "income" && req.Type != "shared_expense" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "无效的记账模板类型")
	}

	if req.AmountCents != nil && *req.AmountCents < 0 {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "模板金额必须大于或等于 0")
	}

	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 1. 先读取并校验越权
	tmpl, err := s.repo.GetTemplateByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.NewAppError(404, "NOT_FOUND", "欲更新的模板不存在")
		}
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "读取模板失败")
	}

	if tmpl.LedgerID != ledgerID {
		return nil, appErrors.NewAppError(403, "FORBIDDEN", "无权更新该模板")
	}

	// 2. 覆盖更新
	tmpl.Name = req.Name
	tmpl.Type = req.Type

	if req.Title != nil {
		tmpl.Title = sql.NullString{String: *req.Title, Valid: true}
	} else {
		tmpl.Title = sql.NullString{Valid: false}
	}
	if req.AmountCents != nil {
		tmpl.AmountCents = sql.NullInt64{Int64: *req.AmountCents, Valid: true}
	} else {
		tmpl.AmountCents = sql.NullInt64{Valid: false}
	}
	if req.CategoryID != nil {
		tmpl.CategoryID = sql.NullString{String: *req.CategoryID, Valid: true}
	} else {
		tmpl.CategoryID = sql.NullString{Valid: false}
	}
	if req.AccountID != nil {
		tmpl.AccountID = sql.NullString{String: *req.AccountID, Valid: true}
	} else {
		tmpl.AccountID = sql.NullString{Valid: false}
	}
	if req.PayerUserID != nil {
		tmpl.PayerUserID = sql.NullString{String: *req.PayerUserID, Valid: true}
	} else {
		tmpl.PayerUserID = sql.NullString{Valid: false}
	}
	if req.SplitMethod != nil {
		tmpl.SplitMethod = sql.NullString{String: *req.SplitMethod, Valid: true}
	} else {
		tmpl.SplitMethod = sql.NullString{Valid: false}
	}
	if req.Note != nil {
		tmpl.Note = sql.NullString{String: *req.Note, Valid: true}
	} else {
		tmpl.Note = sql.NullString{Valid: false}
	}

	tagsStr := strings.Join(req.TagNames, ",")
	if tagsStr != "" {
		tmpl.TagNames = sql.NullString{String: tagsStr, Valid: true}
	} else {
		tmpl.TagNames = sql.NullString{Valid: false}
	}

	if err := s.repo.UpdateTemplate(ctx, tmpl); err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "更新账单模板失败")
	}

	tmpl.UpdatedAt = time.Now()

	return s.toTemplateResponse(tmpl), nil
}

// DeleteTemplate 删除模板逻辑
func (s *Service) DeleteTemplate(ctx context.Context, currentUserID string, id string) error {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	tmpl, err := s.repo.GetTemplateByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return appErrors.NewAppError(404, "NOT_FOUND", "欲删除的模板不存在")
		}
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "读取模板失败")
	}

	if tmpl.LedgerID != ledgerID {
		return appErrors.NewAppError(403, "FORBIDDEN", "无权删除该模板")
	}

	if err := s.repo.DeleteTemplate(ctx, id, ledgerID); err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "删除模板失败")
	}

	return nil
}

// CreateRecurringRule 创建周期账单规则
func (s *Service) CreateRecurringRule(ctx context.Context, currentUserID string, req CreateRecurringRuleRequest) (*RecurringRuleResponse, error) {
	if req.Name == "" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "规则名称不能为空")
	}
	if req.Type != "expense" && req.Type != "income" && req.Type != "shared_expense" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "无效的记账规则类型")
	}
	if req.Frequency != "weekly" && req.Frequency != "monthly" && req.Frequency != "yearly" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "无效的周期频次频率")
	}
	if _, err := time.Parse("2006-01-02", req.NextDueDate); err != nil {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "首次触发到期日期格式不正确")
	}
	if req.AmountCents != nil && *req.AmountCents < 0 {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "规则金额不能为负数")
	}

	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	ruleID := uuid.NewString()

	var titleVal sql.NullString
	if req.Title != nil {
		titleVal = sql.NullString{String: *req.Title, Valid: true}
	}
	var amountVal sql.NullInt64
	if req.AmountCents != nil {
		amountVal = sql.NullInt64{Int64: *req.AmountCents, Valid: true}
	}
	var catVal sql.NullString
	if req.CategoryID != nil {
		catVal = sql.NullString{String: *req.CategoryID, Valid: true}
	}
	var payerVal sql.NullString
	if req.PayerUserID != nil {
		payerVal = sql.NullString{String: *req.PayerUserID, Valid: true}
	}
	var splitVal sql.NullString
	if req.SplitMethod != nil {
		splitVal = sql.NullString{String: *req.SplitMethod, Valid: true}
	}
	var noteVal sql.NullString
	if req.Note != nil {
		noteVal = sql.NullString{String: *req.Note, Valid: true}
	}

	tagsStr := strings.Join(req.TagNames, ",")
	var tagsVal sql.NullString
	if tagsStr != "" {
		tagsVal = sql.NullString{String: tagsStr, Valid: true}
	}

	rule := &RecurringRule{
		ID:              ruleID,
		LedgerID:        ledgerID,
		Name:            req.Name,
		Type:            req.Type,
		Title:           titleVal,
		AmountCents:     amountVal,
		CategoryID:      catVal,
		PayerUserID:     payerVal,
		SplitMethod:     splitVal,
		TagNames:        tagsVal,
		Note:            noteVal,
		Frequency:       req.Frequency,
		NextDueDate:     req.NextDueDate,
		CreatedByUserID: currentUserID,
	}

	if err := s.repo.CreateRecurringRule(ctx, rule); err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "创建周期规则失败")
	}

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	return s.toRecurringRuleResponse(rule), nil
}

// ListRecurringRules 获取指定账本下的所有周期账单规则
func (s *Service) ListRecurringRules(ctx context.Context, currentUserID string) ([]*RecurringRuleResponse, error) {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	rules, err := s.repo.ListRecurringRules(ctx, ledgerID)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取周期规则列表失败")
	}

	var res []*RecurringRuleResponse
	for _, r := range rules {
		res = append(res, s.toRecurringRuleResponse(r))
	}
	return res, nil
}

// DeleteRecurringRule 删除指定的周期规则
func (s *Service) DeleteRecurringRule(ctx context.Context, currentUserID string, id string) error {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	rule, err := s.repo.GetRecurringRuleByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return appErrors.NewAppError(404, "NOT_FOUND", "欲删除的周期规则不存在")
		}
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "读取周期规则失败")
	}

	if rule.LedgerID != ledgerID {
		return appErrors.NewAppError(403, "FORBIDDEN", "无权删除该周期规则")
	}

	if err := s.repo.DeleteRecurringRule(ctx, id, ledgerID); err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "删除周期规则失败")
	}

	return nil
}

// ListRecurringReminders 获取账本下的待确认到期提醒列表（在获取前自动扫描并生成最新的 reminder 数据）
func (s *Service) ListRecurringReminders(ctx context.Context, currentUserID string) ([]*RecurringReminderResponse, error) {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 1. 懒加载检测：生成因时间流逝而过期的提醒实例
	if err := s.checkAndGenerateReminders(ctx, ledgerID); err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "生成到期提醒发生异常")
	}

	// 2. 从数据库拉取
	details, err := s.repo.ListRecurringRemindersWithDetails(ctx, ledgerID)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取周期提醒列表失败")
	}

	var res []*RecurringReminderResponse
	for _, d := range details {
		res = append(res, s.toRecurringReminderResponse(d))
	}
	return res, nil
}

// ConfirmReminder 确认周期提醒并转为真实交易
func (s *Service) ConfirmReminder(ctx context.Context, currentUserID string, reminderID string) error {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	reminder, err := s.repo.GetRecurringReminderByID(ctx, reminderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return appErrors.NewAppError(404, "NOT_FOUND", "欲确认的提醒不存在")
		}
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "读取到期提醒失败")
	}

	if reminder.LedgerID != ledgerID {
		return appErrors.NewAppError(403, "FORBIDDEN", "无权确认该到期提醒")
	}

	if reminder.Status != "pending" {
		return appErrors.NewAppError(400, "VALIDATION_ERROR", "该周期提醒已非 pending 待确认状态")
	}

	rule, err := s.repo.GetRecurringRuleByID(ctx, reminder.RuleID)
	if err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "读取关联周期规则失败")
	}

	var users []string
	if rule.Type == "shared_expense" {
		var err error
		users, err = s.getSystemUsers(ctx)
		if err != nil {
			return appErrors.NewAppError(500, "INTERNAL_ERROR", "拉取系统成员列表失败")
		}
	}

	// 开始事务
	dbConn := s.repo.GetDB()
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "启动事务失败")
	}
	defer tx.Rollback()

	occurredAt, err := time.Parse("2006-01-02", reminder.DueDate)
	if err != nil {
		occurredAt = time.Now()
	}

	txID := uuid.NewString()

	if rule.Type == "shared_expense" {
		amount := rule.AmountCents.Int64
		payer := rule.PayerUserID.String
		splitMethod := rule.SplitMethod.String
		if splitMethod == "" {
			splitMethod = "equal"
		}

		var otherUserID string
		for _, u := range users {
			if u != payer {
				otherUserID = u
				break
			}
		}

		var payerShare, otherShare int64
		if splitMethod == "equal" {
			base := amount / 2
			rem := amount % 2
			payerShare = base + rem
			otherShare = base
		} else {
			payerShare = amount
			otherShare = 0
		}

		splits := []TransactionSplit{
			{ID: uuid.NewString(), TransactionID: txID, UserID: payer, ShareAmount: payerShare},
			{ID: uuid.NewString(), TransactionID: txID, UserID: otherUserID, ShareAmount: otherShare},
		}

		realTx := &Transaction{
			ID:              txID,
			LedgerID:        ledgerID,
			Type:            "shared_expense",
			Title:           rule.Title.String,
			Amount:          amount,
			Currency:        "CNY",
			OccurredAt:      occurredAt,
			OwnerUserID:     currentUserID,
			CreatedByUserID: currentUserID,
			PayerUserID:     payer,
			CategoryID:      rule.CategoryID,
			Visibility:      "shared",
			SplitMethod:     sql.NullString{String: splitMethod, Valid: true},
			Note:            rule.Note,
		}

		var tags []string
		if rule.TagNames.String != "" {
			tags = strings.Split(rule.TagNames.String, ",")
		}

		if err = s.repo.CreateWithTx(ctx, tx, realTx, tags); err != nil {
			return appErrors.NewAppError(500, "INTERNAL_ERROR", "生成真实交易失败")
		}

		err = s.repo.CreateSplitsWithTx(ctx, tx, splits)
		if err != nil {
			return appErrors.NewAppError(500, "INTERNAL_ERROR", "生成账单分摊数据失败: " + err.Error())
		}
	} else {
		amount := rule.AmountCents.Int64
		payer := rule.PayerUserID.String
		if payer == "" {
			payer = currentUserID
		}

		realTx := &Transaction{
			ID:              txID,
			LedgerID:        ledgerID,
			Type:            rule.Type,
			Title:           rule.Title.String,
			Amount:          amount,
			Currency:        "CNY",
			OccurredAt:      occurredAt,
			OwnerUserID:     currentUserID,
			CreatedByUserID: currentUserID,
			PayerUserID:     payer,
			CategoryID:      rule.CategoryID,
			Visibility:      "partner_readable",
			Note:            rule.Note,
		}

		var tags []string
		if rule.TagNames.String != "" {
			tags = strings.Split(rule.TagNames.String, ",")
		}

		if err = s.repo.CreateWithTx(ctx, tx, realTx, tags); err != nil {
			return appErrors.NewAppError(500, "INTERNAL_ERROR", "生成真实交易失败")
		}
	}

	err = s.repo.UpdateRecurringReminderStatusWithTx(ctx, tx, reminderID, ledgerID, "confirmed", sql.NullString{String: txID, Valid: true})
	if err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "更新到期提醒状态失败")
	}

	if err := tx.Commit(); err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "确认提醒事务提交失败")
	}

	return nil
}

// IgnoreReminder 忽略到期提醒
func (s *Service) IgnoreReminder(ctx context.Context, currentUserID string, reminderID string) error {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	reminder, err := s.repo.GetRecurringReminderByID(ctx, reminderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return appErrors.NewAppError(404, "NOT_FOUND", "欲忽略的周期提醒不存在")
		}
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "读取到期提醒失败")
	}

	if reminder.LedgerID != ledgerID {
		return appErrors.NewAppError(403, "FORBIDDEN", "无权忽略该到期提醒")
	}

	if reminder.Status != "pending" {
		return appErrors.NewAppError(400, "VALIDATION_ERROR", "该到期提醒已被处理，无法再次忽略")
	}

	err = s.repo.UpdateRecurringReminderStatusWithTx(ctx, nil, reminderID, ledgerID, "ignored", sql.NullString{Valid: false})
	if err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "更新到期提醒状态为忽略失败")
	}

	return nil
}

// checkAndGenerateReminders 周期账单过期检测并自动生成待处理 Reminders
func (s *Service) checkAndGenerateReminders(ctx context.Context, ledgerID string) error {
	rules, err := s.repo.ListRecurringRules(ctx, ledgerID)
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")

	for _, rule := range rules {
		for rule.NextDueDate <= today {
			if err := s.generateSingleReminderTx(ctx, ledgerID, rule); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) generateSingleReminderTx(ctx context.Context, ledgerID string, rule *RecurringRule) error {
	dbConn := s.repo.GetDB()
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM recurring_reminders WHERE rule_id = ? AND due_date = ?", rule.ID, rule.NextDueDate).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		reminder := &RecurringReminder{
			ID:       uuid.NewString(),
			LedgerID: ledgerID,
			RuleID:   rule.ID,
			DueDate:  rule.NextDueDate,
			Status:   "pending",
		}
		if err := s.repo.CreateRecurringReminder(ctx, tx, reminder); err != nil {
			return err
		}
	}

	nextDueDate, err := s.calculateNextDueDate(rule.NextDueDate, rule.Frequency)
	if err != nil {
		return err
	}

	err = s.repo.UpdateRecurringRuleNextDueDateWithTx(ctx, tx, rule.ID, nextDueDate)
	if err != nil {
		return err
	}

	rule.NextDueDate = nextDueDate

	return tx.Commit()
}

func (s *Service) calculateNextDueDate(current string, frequency string) (string, error) {
	t, err := time.Parse("2006-01-02", current)
	if err != nil {
		return "", err
	}

	var next time.Time
	switch frequency {
	case "weekly":
		next = t.AddDate(0, 0, 7)
	case "monthly":
		next = t.AddDate(0, 1, 0)
	case "yearly":
		next = t.AddDate(1, 0, 0)
	default:
		return "", fmt.Errorf("invalid frequency: %s", frequency)
	}

	return next.Format("2006-01-02"), nil
}

// 辅助 DTO 映射方法
func (s *Service) toRecurringRuleResponse(rule *RecurringRule) *RecurringRuleResponse {
	var amount *int64
	if rule.AmountCents.Valid {
		val := rule.AmountCents.Int64
		amount = &val
	}

	var tags []string
	if rule.TagNames.Valid && rule.TagNames.String != "" {
		tags = strings.Split(rule.TagNames.String, ",")
	}

	return &RecurringRuleResponse{
		ID:              rule.ID,
		Name:            rule.Name,
		Type:            rule.Type,
		Title:           rule.Title.String,
		AmountCents:     amount,
		CategoryID:      rule.CategoryID.String,
		PayerUserID:     rule.PayerUserID.String,
		SplitMethod:     rule.SplitMethod.String,
		TagNames:        tags,
		Note:            rule.Note.String,
		Frequency:       rule.Frequency,
		NextDueDate:     rule.NextDueDate,
		CreatedByUserID: rule.CreatedByUserID,
		CreatedAt:       rule.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       rule.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *Service) toRecurringReminderResponse(d *RecurringReminderDetail) *RecurringReminderResponse {
	var amount *int64
	if d.AmountCents.Valid {
		val := d.AmountCents.Int64
		amount = &val
	}

	var tags []string
	if d.TagNames.Valid && d.TagNames.String != "" {
		tags = strings.Split(d.TagNames.String, ",")
	}

	return &RecurringReminderResponse{
		ID:            d.Reminder.ID,
		RuleID:        d.Reminder.RuleID,
		RuleName:      d.RuleName,
		Type:          d.Type,
		Title:         d.Title.String,
		AmountCents:   amount,
		CategoryID:    d.CategoryID.String,
		CategoryName:  d.CategoryName.String,
		PayerUserID:   d.PayerUserID.String,
		SplitMethod:   d.SplitMethod.String,
		TagNames:      tags,
		Note:          d.Note.String,
		Frequency:     d.Frequency,
		DueDate:       d.Reminder.DueDate,
		Status:        d.Reminder.Status,
		TransactionID: d.Reminder.TransactionID.String,
		CreatedAt:     d.Reminder.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     d.Reminder.UpdatedAt.Format(time.RFC3339),
	}
}

// BatchTag 批量为多条账单追加标签并写入审计记录
func (s *Service) BatchTag(ctx context.Context, currentUserID string, req BatchTagRequest) error {
	if len(req.TransactionIDs) == 0 {
		return appErrors.NewAppError(400, "VALIDATION_ERROR", "请选择至少一笔交易")
	}
	if len(req.TagNames) == 0 {
		return appErrors.NewAppError(400, "VALIDATION_ERROR", "标签名称不能为空")
	}

	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 开启事务
	dbConn := s.repo.GetDB()
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "启动事务失败")
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)

	for _, txID := range req.TransactionIDs {
		// 校验越权：查询这笔交易是否属于当前账本
		txModel, _, err := s.repo.GetByID(ctx, txID)
		if err != nil {
			return appErrors.NewAppError(404, "NOT_FOUND", "账单交易未找到: "+txID)
		}
		if txModel.LedgerID != ledgerID {
			return appErrors.NewAppError(403, "FORBIDDEN", "无权操作该账单交易")
		}
		if !s.CanEditTransaction(currentUserID, txModel) {
			return appErrors.NewAppError(403, "FORBIDDEN", "无权操作该账单交易")
		}

		// 追加标签 (associateTags 会 INSERT OR IGNORE 关联，起到追加且去重效果)
		if err = s.repo.associateTags(ctx, tx, txID, ledgerID, req.TagNames, now); err != nil {
			return appErrors.NewAppError(500, "INTERNAL_ERROR", "关联标签失败: "+err.Error())
		}

		// 写入审计日志记录
		auditLog := &AuditLog{
			LedgerID:    ledgerID,
			ActorUserID: currentUserID,
			Action:      "batch_tag",
			EntityType:  "transaction",
			EntityID:    txID,
			BeforeJSON:  sql.NullString{String: "{}", Valid: true},
			AfterJSON:   sql.NullString{String: fmt.Sprintf(`{"added_tags":%v}`, req.TagNames), Valid: true},
		}
		if err = s.repo.CreateAuditLogWithTx(ctx, tx, auditLog); err != nil {
			return appErrors.NewAppError(500, "INTERNAL_ERROR", "记录审计日志失败")
		}
	}

	if err = tx.Commit(); err != nil {
		return appErrors.NewAppError(500, "INTERNAL_ERROR", "提交事务失败")
	}

	return nil
}

