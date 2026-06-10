package settlement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	appErrors "ledger_two/internal/errors"
)

// Service 结算模块业务逻辑服务
// @brief 实现双方共同支出差额统计及事务级结算记账
type Service struct {
	repo *Repository
}

// NewService 实例化业务服务
// @brief 创建 Settlement 的 Service 实例
// @param repo *Repository 数据库仓库句柄
// @return *Service 服务实例
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetBalance 计算并获取双方最新的余额与欠款净额报表
// @brief 抓取全部共同支出与结算明细，通过差额公式算好谁欠谁多少钱
// @param ctx context.Context 上下文
// @return *BalanceResponse 结算净额报表 DTO
// @return error 错误信息
func (s *Service) GetBalance(ctx context.Context) (*BalanceResponse, error) {
	// 1. 获取全局唯一 LedgerID
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 2. 获取系统内仅有的两个用户 ID
	users, err := s.getSystemUsers(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统用户失败")
	}
	if len(users) != 2 {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "系统初始化异常：用户数不等于2")
	}
	userAID := users[0]
	userBID := users[1]

	// 3. 拉取底层汇总数据
	paidMap, shareMap, settledMap, err := s.repo.GetSharedExpensesNetStats(ctx, ledgerID)
	if err != nil {
		return nil, err
	}

	userAPaid := paidMap[userAID]
	userAShare := shareMap[userAID]
	userASettledOut := settledMap[userAID]
	userBSettledOut := settledMap[userBID]

	userBPaid := paidMap[userBID]
	userBShare := shareMap[userBID]

	// 4. 应用差额计算公式：net = paid - share + settled_out - settled_in
	userANet := userAPaid - userAShare + userASettledOut - userBSettledOut
	userBNet := userBPaid - userBShare + userBSettledOut - userASettledOut

	var fromUser, toUser string
	var amountCents int64

	if userANet > 0 {
		// A 垫付多，B 欠 A
		fromUser = userBID
		toUser = userAID
		amountCents = userANet
	} else if userANet < 0 {
		// B 垫付多，A 欠 B
		fromUser = userAID
		toUser = userBID
		amountCents = -userANet
	}

	return &BalanceResponse{
		UserAPaidCents:       userAPaid,
		UserAShareCents:      userAShare,
		UserBPaidCents:       userBPaid,
		UserBShareCents:      userBShare,
		UserASettledToBCents: userASettledOut,
		UserBSettledToACents: userBSettledOut,
		UserANetCents:        userANet,
		UserBNetCents:        userBNet,
		FromUserID:           fromUser,
		ToUserID:             toUser,
		AmountCents:          amountCents,
	}, nil
}

// CreateSettlement 执行补款结算记账
// @brief 校验并在一个事务中写入 settlements 记录与 transactions 结算交易流水
// @param ctx context.Context 上下文
// @param currentUserID string 执行操作者 ID
// @param req CreateSettlementRequest 结算创建请求参数
// @return *SettlementResponse 结算明细 DTO
// @return error 错误信息
func (s *Service) CreateSettlement(ctx context.Context, currentUserID string, req CreateSettlementRequest) (*SettlementResponse, error) {
	// 1. 金额校验
	if req.AmountCents <= 0 {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "结算金额必须大于 0")
	}

	// 2. 参与人校验
	if req.FromUserID == "" || req.ToUserID == "" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "结算收付款人不能为空")
	}
	if req.FromUserID == req.ToUserID {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "结算双方不能为同一个人")
	}

	// 验证用户在系统中是否存在
	users, err := s.getSystemUsers(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统用户失败")
	}
	foundFrom := false
	foundTo := false
	for _, u := range users {
		if u == req.FromUserID {
			foundFrom = true
		}
		if u == req.ToUserID {
			foundTo = true
		}
	}
	if !foundFrom || !foundTo {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "结算用户不属于账本成员")
	}

	// 3. 时间校验
	occurredAt, err := time.Parse(time.RFC3339, req.OccurredAt)
	if err != nil {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "交易时间格式必须符合 ISO8601 标准")
	}

	// 4. 获取 LedgerID
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 5. 构造结算与流水记录
	settlementID := uuid.NewString()
	settleModel := &Settlement{
		ID:              settlementID,
		LedgerID:        ledgerID,
		FromUserID:      req.FromUserID,
		ToUserID:        req.ToUserID,
		Amount:          req.AmountCents,
		Currency:        "CNY",
		OccurredAt:      occurredAt,
		Note:            sql.NullString{String: req.Note, Valid: req.Note != ""},
		CreatedByUserID: currentUserID,
	}

	txID := uuid.NewString()
	nowStr := time.Now().Format(time.RFC3339)
	title := "结算往来"

	dbConn := s.repo.GetDB()
	dbTx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer dbTx.Rollback()

	// 写入 settlements 表
	err = s.repo.CreateWithTx(ctx, dbTx, settleModel)
	if err != nil {
		return nil, err
	}

	// 写入 transactions 表 (Type = 'settlement', Visibility = 'shared')
	_, err = dbTx.ExecContext(ctx, `
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id,
			visibility, note, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'normal', ?, ?)
	`,
		txID, ledgerID, "settlement", title, req.AmountCents, "CNY", occurredAt.Format(time.RFC3339),
		currentUserID, currentUserID, req.FromUserID,
		"shared", sql.NullString{String: req.Note, Valid: req.Note != ""}, nowStr, nowStr,
	)
	if err != nil {
		return nil, fmt.Errorf("insert settlement transaction stream failed: %w", err)
	}

	// 写入审计日志
	afterDTO := s.toDTO(settleModel)
	afterJSONBytes, _ := json.Marshal(afterDTO)
	err = s.repo.CreateAuditLogWithTx(ctx, dbTx, ledgerID, currentUserID, "create", "settlement", settlementID, "", string(afterJSONBytes))
	if err != nil {
		return nil, fmt.Errorf("create audit log for settlement failed: %w", err)
	}

	err = dbTx.Commit()
	if err != nil {
		return nil, err
	}

	return afterDTO, nil
}

// List 拉取历史结算明细列表
// @brief 获取按月份过滤的历史结算明细列表
// @param ctx context.Context 上下文
// @param month string 月份过滤参数 (如 '2026-06')
// @return []*SettlementResponse 结算明细 DTO 列表
// @return error 错误信息
func (s *Service) List(ctx context.Context, month string) ([]*SettlementResponse, error) {
	ledgerID, err := s.getLedgerID(ctx)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	list, err := s.repo.List(ctx, ledgerID, month)
	if err != nil {
		return nil, err
	}

	var res []*SettlementResponse
	for _, item := range list {
		res = append(res, s.toDTO(item))
	}
	return res, nil
}

// 辅助 DTO 映射
func (s *Service) toDTO(m *Settlement) *SettlementResponse {
	note := ""
	if m.Note.Valid {
		note = m.Note.String
	}
	return &SettlementResponse{
		ID:              m.ID,
		FromUserID:      m.FromUserID,
		ToUserID:        m.ToUserID,
		AmountCents:     m.Amount,
		Currency:        m.Currency,
		OccurredAt:      m.OccurredAt.Format(time.RFC3339),
		Note:            note,
		CreatedByUserID: m.CreatedByUserID,
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
	}
}

// 辅助方法：查询唯一 LedgerID
func (s *Service) getLedgerID(ctx context.Context) (string, error) {
	var id string
	dbConn := s.repo.GetDB()
	err := dbConn.QueryRowContext(ctx, "SELECT id FROM ledgers LIMIT 1").Scan(&id)
	return id, err
}

// 辅助方法：查询所有用户 ID
func (s *Service) getSystemUsers(ctx context.Context) ([]string, error) {
	dbConn := s.repo.GetDB()
	rows, err := dbConn.QueryContext(ctx, "SELECT id FROM users ORDER BY username ASC")
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
