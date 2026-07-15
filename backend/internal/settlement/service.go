package settlement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	ledgerctx "ledger_two/internal/ledger"
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

// GetBalance 计算并获取各方最新的余额与欠款净额报表
// @brief 抓取全部共同支出与结算明细，通过差额公式算好各自净额，并给出建议转账路径
// @param ctx context.Context 上下文
// @return *BalanceResponse 结算净额报表 DTO
// @return error 错误信息
func (s *Service) GetBalance(ctx context.Context, currentUserID string) (*BalanceResponse, error) {
	return s.getBalance(ctx, currentUserID, "")
}

// GetBalanceForMonth 按可选月份计算各方余额；空月份等同全部账期。
func (s *Service) GetBalanceForMonth(ctx context.Context, currentUserID, month string) (*BalanceResponse, error) {
	if month != "" {
		if _, err := time.Parse("2006-01", month); err != nil {
			return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "查询月份格式错误，应为 YYYY-MM")
		}
	}

	return s.getBalance(ctx, currentUserID, month)
}

func (s *Service) getBalance(ctx context.Context, currentUserID, month string) (*BalanceResponse, error) {
	// 1. 获取全局唯一 LedgerID
	ledgerID, err := s.getUserLedgerID(ctx, currentUserID)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 2. 获取账单内所有的用户 ID
	users, err := s.getLedgerUsers(ctx, ledgerID)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取账本成员失败")
	}

	// 3. 拉取底层汇总数据
	paidMap, shareMap, settledOutMap, settledInMap, err := s.repo.GetSharedExpensesNetStats(ctx, ledgerID, month)
	if err != nil {
		return nil, err
	}

	var userBalances []UserBalance
	// 记录债务人与债权人用于贪心算法
	type userNet struct {
		userID   string
		netCents int64
	}
	var debtors []userNet   // net < 0
	var creditors []userNet // net > 0

	// 4. 应用差额计算公式：raw_net = paid - share，final_net = raw_net + settled_out - settled_in
	for _, u := range users {
		paid := paidMap[u]
		share := shareMap[u]
		settledOut := settledOutMap[u]
		settledIn := settledInMap[u]

		rawNet := paid - share
		settlementNet := settledOut - settledIn
		finalNet := rawNet + settlementNet

		userBalances = append(userBalances, UserBalance{
			UserID:             u,
			PaidCents:          paid,
			ShareCents:         share,
			RawNetCents:        rawNet,
			SettledOutCents:    settledOut,
			SettledInCents:     settledIn,
			SettlementNetCents: settlementNet,
			FinalNetCents:      finalNet,
			NetCents:           finalNet,
		})

		if finalNet > 0 {
			creditors = append(creditors, userNet{userID: u, netCents: finalNet})
		} else if finalNet < 0 {
			debtors = append(debtors, userNet{userID: u, netCents: finalNet})
		}
	}

	// 5. 贪心算法消债
	var suggestedTransfers []SuggestedTransfer
	// 简单贪心匹配：不一定是最优（最小交易次数），但能平账。
	// 这里不再强制要求图的最优解，优先匹配第一个可抵消的债务
	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		debt := -debtors[i].netCents
		credit := creditors[j].netCents

		amount := debt
		if credit < amount {
			amount = credit
		}

		if amount > 0 {
			suggestedTransfers = append(suggestedTransfers, SuggestedTransfer{
				FromUserID:  debtors[i].userID,
				ToUserID:    creditors[j].userID,
				AmountCents: amount,
			})
		}

		debtors[i].netCents += amount
		creditors[j].netCents -= amount

		if debtors[i].netCents == 0 {
			i++
		}
		if creditors[j].netCents == 0 {
			j++
		}
	}

	resp := &BalanceResponse{
		UserBalances:       userBalances,
		SuggestedTransfers: suggestedTransfers,
	}

	// 兼容老代码双人模式的数据绑定
	if len(users) == 2 {
		userAID := users[0]
		userBID := users[1]

		resp.UserAPaidCents = paidMap[userAID]
		resp.UserAShareCents = shareMap[userAID]
		resp.UserBPaidCents = paidMap[userBID]
		resp.UserBShareCents = shareMap[userBID]
		resp.UserASettledToBCents = settledOutMap[userAID]
		resp.UserBSettledToACents = settledOutMap[userBID]

		for _, ub := range userBalances {
			if ub.UserID == userAID {
				resp.UserANetCents = ub.NetCents
			} else if ub.UserID == userBID {
				resp.UserBNetCents = ub.NetCents
			}
		}

		if resp.UserANetCents > 0 {
			resp.FromUserID = userBID
			resp.ToUserID = userAID
			resp.AmountCents = resp.UserANetCents
		} else if resp.UserANetCents < 0 {
			resp.FromUserID = userAID
			resp.ToUserID = userBID
			resp.AmountCents = -resp.UserANetCents
		}
	} else if len(suggestedTransfers) == 1 {
		// 为了向下兼容单条建议的UI展示
		resp.FromUserID = suggestedTransfers[0].FromUserID
		resp.ToUserID = suggestedTransfers[0].ToUserID
		resp.AmountCents = suggestedTransfers[0].AmountCents
	}

	return resp, nil
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

	ledgerID, err := s.getUserLedgerID(ctx, currentUserID)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 验证用户在系统中是否存在
	users, err := s.getLedgerUsers(ctx, ledgerID)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取账本用户失败")
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

	// 4. 获取 LedgerID (已在上文获取)

	if err := s.checkRole(ctx, ledgerID, currentUserID, "owner", "editor"); err != nil {
		return nil, err
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
func (s *Service) List(ctx context.Context, currentUserID string, month string) ([]*SettlementResponse, error) {
	ledgerID, err := s.getUserLedgerID(ctx, currentUserID)
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
func (s *Service) getUserLedgerID(ctx context.Context, userID string) (string, error) {
	if lc, ok := ledgerctx.LedgerContextFromContext(ctx); ok && lc.UserID == userID {
		return lc.LedgerID, nil
	}

	var id string
	dbConn := s.repo.GetDB()

	headerLedgerID := middleware.GetHeaderLedgerIDFromContext(ctx)
	if headerLedgerID != "" {
		err := dbConn.QueryRowContext(ctx, "SELECT ledger_id FROM ledger_members WHERE ledger_id = ? AND user_id = ?", headerLedgerID, userID).Scan(&id)
		return id, err
	}

	err := dbConn.QueryRowContext(ctx, "SELECT ledger_id FROM ledger_members WHERE user_id = ? LIMIT 1", userID).Scan(&id)
	return id, err
}

// 辅助方法：校验用户在账本中的角色
func (s *Service) checkRole(ctx context.Context, ledgerID string, userID string, allowedRoles ...string) error {
	if lc, ok := ledgerctx.LedgerContextFromContext(ctx); ok && lc.UserID == userID && lc.LedgerID == ledgerID {
		for _, r := range allowedRoles {
			if lc.Role == ledgerctx.Role(r) {
				return nil
			}
		}
		return appErrors.NewAppError(403, "FORBIDDEN", "当前角色无权执行此操作")
	}

	var role string
	err := s.repo.GetDB().QueryRowContext(ctx, "SELECT role FROM ledger_members WHERE ledger_id = ? AND user_id = ?", ledgerID, userID).Scan(&role)
	if err != nil {
		return appErrors.NewAppError(403, "FORBIDDEN", "您不是该账本的成员")
	}

	for _, r := range allowedRoles {
		if role == r {
			return nil
		}
	}
	return appErrors.NewAppError(403, "FORBIDDEN", "当前角色无权执行此操作")
}

// 辅助方法：查询所有账本成员 ID
func (s *Service) getLedgerUsers(ctx context.Context, ledgerID string) ([]string, error) {
	dbConn := s.repo.GetDB()
	rows, err := dbConn.QueryContext(ctx, "SELECT lm.user_id FROM ledger_members lm JOIN users u ON lm.user_id = u.id WHERE lm.ledger_id = ? ORDER BY u.username ASC", ledgerID)
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
