package transaction

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AppError 全局统一的业务/参数校验错误
type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

// NewAppError 构造 AppError 实例
// @brief 创建带状态码和错误码的业务异常
// @param status int HTTP 状态码
// @param code string 错误码
// @param message string 描述语
// @return *AppError 异常实例
func NewAppError(status int, code string, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

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
		return nil, NewAppError(400, "VALIDATION_ERROR", "金额必须大于 0")
	}

	// 2. 类型校验
	if req.Type != "expense" && req.Type != "income" {
		return nil, NewAppError(400, "VALIDATION_ERROR", "记账类型必须为 expense 或 income")
	}

	// 3. 时间校验
	occurredAt, err := time.Parse(time.RFC3339, req.OccurredAt)
	if err != nil {
		return nil, NewAppError(400, "VALIDATION_ERROR", "交易时间格式必须符合 ISO8601 标准")
	}

	// 4. 用户存在校验
	if req.PayerUserID == "" {
		return nil, NewAppError(400, "VALIDATION_ERROR", "付款人用户 ID 不能为空")
	}

	// 5. 获取全局唯一 LedgerID
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 6. 可见性处理
	visibility := req.Visibility
	if visibility == "" {
		visibility = "private"
	}
	if visibility != "private" && visibility != "partner_readable" && visibility != "shared" {
		return nil, NewAppError(400, "VALIDATION_ERROR", "无效的可见性属性值")
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
		return nil, NewAppError(404, "NOT_FOUND", "账单未找到或已删除")
	}

	// 校验查看权限
	if !s.CanViewTransaction(currentUserID, tx) {
		return nil, NewAppError(403, "FORBIDDEN", "无权查看此账单")
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
		return nil, NewAppError(404, "NOT_FOUND", "账单未找到")
	}

	// 校验编辑权限：谁创建谁编辑，被删除的拒绝编辑
	if !s.CanEditTransaction(currentUserID, tx) {
		return nil, NewAppError(403, "FORBIDDEN", "无权编辑此账单")
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
			return nil, NewAppError(400, "VALIDATION_ERROR", "金额必须大于 0")
		}
		tx.Amount = *req.AmountCents
	}
	if req.OccurredAt != nil {
		occurredAt, err := time.Parse(time.RFC3339, *req.OccurredAt)
		if err != nil {
			return nil, NewAppError(400, "VALIDATION_ERROR", "交易时间格式错误")
		}
		tx.OccurredAt = occurredAt
	}
	if req.PayerUserID != nil {
		if *req.PayerUserID == "" {
			return nil, NewAppError(400, "VALIDATION_ERROR", "付款人用户 ID 不能为空")
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
			return nil, NewAppError(400, "VALIDATION_ERROR", "无效的可见性属性值")
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
				return nil, NewAppError(400, "VALIDATION_ERROR", "无效的分摊方式")
			}
			tx.SplitMethod = sql.NullString{String: splitMethodVal, Valid: true}
		} else if tx.SplitMethod.Valid {
			splitMethodVal = tx.SplitMethod.String
		}

		// 重新计算分摊金额
		users, err := s.getSystemUsers(ctx)
		if err != nil {
			return nil, NewAppError(500, "INTERNAL_ERROR", "获取系统用户失败")
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
			return nil, NewAppError(400, "VALIDATION_ERROR", "无效的付款人")
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
		return NewAppError(404, "NOT_FOUND", "账单未找到")
	}

	// 校验编辑/删除权限：谁创建谁删除
	if !s.CanEditTransaction(currentUserID, tx) {
		return NewAppError(403, "FORBIDDEN", "无权删除此账单")
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
		return nil, NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
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
		return nil, NewAppError(400, "VALIDATION_ERROR", "金额必须大于 0")
	}

	// 2. 时间校验
	occurredAt, err := time.Parse(time.RFC3339, req.OccurredAt)
	if err != nil {
		return nil, NewAppError(400, "VALIDATION_ERROR", "交易时间格式必须符合 ISO8601 标准")
	}

	// 3. 分摊方式校验
	if req.SplitMethod != "equal" && req.SplitMethod != "payer_only" {
		return nil, NewAppError(400, "VALIDATION_ERROR", "分摊类型必须为 equal 或 payer_only")
	}

	// 4. 用户校验与分摊计算
	users, err := s.getSystemUsers(ctx)
	if err != nil {
		return nil, NewAppError(500, "INTERNAL_ERROR", "获取系统用户失败")
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
		return nil, NewAppError(400, "VALIDATION_ERROR", "付款人用户不在当前账本成员中")
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
		return nil, NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
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
		return nil, NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}
	return s.repo.ListCategories(ctx, ledgerID)
}


