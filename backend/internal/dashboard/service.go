package dashboard

import (
	"context"
	"sort"
	"time"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/settlement"
	"ledger_two/internal/transaction"
)

// Service Dashboard 与统计核心业务服务
// @brief 串联结算轧差及本月多表汇总进行内存级高性能统计分析
type Service struct {
	repo      *Repository
	settleSvc *settlement.Service
}

// NewService 实例化业务服务
// @brief 创建 Dashboard Service 实例
// @param repo *Repository 数据仓储
// @param settleSvc *settlement.Service 结算业务服务
// @return *Service 服务实例
func NewService(repo *Repository, settleSvc *settlement.Service) *Service {
	return &Service{repo: repo, settleSvc: settleSvc}
}

// GetDashboardData 获取当月首页 Dashboard 的完整统计报表
// @brief 校验月份参数，拉取结算中心轧差，调用 repo 获取本月原始流并做内存指标聚合
// @param ctx context.Context 上下文
// @param currentUserID string 登录用户 ID
// @param month string 查询月份 (如 '2026-06')
// @return *DashboardResponse 首页聚合数据报表
// @return error 错误信息
func (s *Service) GetDashboardData(ctx context.Context, currentUserID string, month string) (*DashboardResponse, error) {
	// 1. 月份处理
	if month == "" {
		month = time.Now().Format("2006-01")
	} else {
		// 校验格式为 YYYY-MM
		_, err := time.Parse("2006-01", month)
		if err != nil {
			return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "查询月份格式错误，应为 YYYY-MM")
		}
	}

	// 2. 获取全局唯一 LedgerID
	ledgerID, err := s.getUserLedgerID(ctx, currentUserID)
	if err != nil {
		return nil, appErrors.NewAppError(500, "INTERNAL_ERROR", "获取系统账本失败")
	}

	// 3. 获取全局结算净额轧差（跨月份）
	sharedBalance, err := s.settleSvc.GetBalance(ctx, currentUserID)
	if err != nil {
		return nil, err
	}

	// 4. 获取本月所有的可见交易、关联 tags、关联 splits、分类映射与用户名映射
	list, tagsMap, splitsMap, categoryMap, userMap, err := s.repo.GetDashboardRawData(ctx, ledgerID, currentUserID, month)
	if err != nil {
		return nil, err
	}

	// 5. 内存计算月度总指标
	var totalExpenseCents int64 = 0
	var totalIncomeCents int64 = 0
	var myPaidCents int64 = 0
	var partnerPaidCents int64 = 0

	// 内存临时归类汇总
	catAmountMap := make(map[string]int64)
	tagAmountMap := make(map[string]int64)
	userPaidMap := make(map[string]int64)
	userShareMap := make(map[string]int64)

	// 初始化各用户 ID 统计结构（保证哪怕当月无消费，两个用户也都会出现在 user_stats 里）
	for userID := range userMap {
		userPaidMap[userID] = 0
		userShareMap[userID] = 0
	}

	for _, tx := range list {
		// 消费统计范围：仅包括 expense 和 shared_expense，排除收入 (income) 及结算流 (settlement)
		isExpenseType := tx.Type == "expense" || tx.Type == "shared_expense"

		if isExpenseType {
			totalExpenseCents += tx.Amount

			// 统计我方与对方支付
			if tx.PayerUserID == currentUserID {
				myPaidCents += tx.Amount
			} else {
				partnerPaidCents += tx.Amount
			}

			// 统计分类消费
			catID := ""
			if tx.CategoryID.Valid {
				catID = tx.CategoryID.String
			}
			catAmountMap[catID] += tx.Amount

			// 统计标签消费
			if tags, ok := tagsMap[tx.ID]; ok {
				for _, name := range tags {
					tagAmountMap[name] += tx.Amount
				}
			}

			// 统计成员实际支付与分摊承担金额
			userPaidMap[tx.PayerUserID] += tx.Amount

			if tx.Type == "expense" {
				// 普通消费归记账人个人承担
				userShareMap[tx.CreatedByUserID] += tx.Amount
			} else if tx.Type == "shared_expense" {
				// 共同支出按照 splits 表的分摊承担
				if splits, ok := splitsMap[tx.ID]; ok {
					for _, split := range splits {
						userShareMap[split.UserID] += split.ShareAmountCents
					}
				}
			}
		} else if tx.Type == "income" {
			totalIncomeCents += tx.Amount
		}
	}

	// 6. 构造分类消费汇总并排序
	var categorySummary []SummaryItem
	for catID, amount := range catAmountMap {
		name, exists := categoryMap[catID]
		if !exists || name == "" {
			name = "未分类消费"
		}
		var percent float64 = 0
		if totalExpenseCents > 0 {
			percent = float64(amount) / float64(totalExpenseCents) * 100
		}
		categorySummary = append(categorySummary, SummaryItem{
			ID:          catID,
			Name:        name,
			AmountCents: amount,
			Percent:     percent,
		})
	}
	// 按消费金额倒序排列
	sort.Slice(categorySummary, func(i, j int) bool {
		return categorySummary[i].AmountCents > categorySummary[j].AmountCents
	})

	// 7. 构造标签消费汇总并排序
	var tagSummary []SummaryItem
	for name, amount := range tagAmountMap {
		var percent float64 = 0
		if totalExpenseCents > 0 {
			percent = float64(amount) / float64(totalExpenseCents) * 100
		}
		tagSummary = append(tagSummary, SummaryItem{
			Name:        name,
			AmountCents: amount,
			Percent:     percent,
		})
	}
	// 按消费金额倒序排列
	sort.Slice(tagSummary, func(i, j int) bool {
		return tagSummary[i].AmountCents > tagSummary[j].AmountCents
	})

	// 8. 构造成员消费统计
	var userStats []UserStatItem
	for userID, dispName := range userMap {
		userStats = append(userStats, UserStatItem{
			UserID:      userID,
			DisplayName: dispName,
			PaidCents:   userPaidMap[userID],
			ShareCents:  userShareMap[userID],
		})
	}
	// 按用户名称排序以稳定输出顺序
	sort.Slice(userStats, func(i, j int) bool {
		return userStats[i].DisplayName < userStats[j].DisplayName
	})

	// 9. 构造最近 10 条流水
	recentCount := len(list)
	if recentCount > 10 {
		recentCount = 10
	}
	var recentTransactions []transaction.TransactionResponse
	for i := 0; i < recentCount; i++ {
		tx := list[i]
		tags := tagsMap[tx.ID]
		dto := s.toTransactionDTO(tx, tags)

		// 如果是共同支出，注入分摊详情
		if tx.Type == "shared_expense" {
			if splits, ok := splitsMap[tx.ID]; ok {
				dto.SplitMethod = &tx.SplitMethod.String
				dto.Participants = splits
			}
		}
		recentTransactions = append(recentTransactions, *dto)
	}

	return &DashboardResponse{
		Month:              month,
		TotalExpenseCents:  totalExpenseCents,
		TotalIncomeCents:   totalIncomeCents,
		MyPaidCents:        myPaidCents,
		PartnerPaidCents:   partnerPaidCents,
		SharedBalance:      *sharedBalance,
		RecentTransactions: recentTransactions,
		CategorySummary:    categorySummary,
		TagSummary:         tagSummary,
		UserStats:          userStats,
	}, nil
}

// 辅助方法：查询唯一 LedgerID
func (s *Service) getUserLedgerID(ctx context.Context, userID string) (string, error) {
	lc := middleware.GetLedgerContext(ctx)
	if lc != nil {
		return lc.LedgerID, nil
	}

	var id string

	headerLedgerID := middleware.GetHeaderLedgerIDFromContext(ctx)
	if headerLedgerID != "" {
		err := s.repo.db.QueryRowContext(ctx, "SELECT ledger_id FROM ledger_members WHERE ledger_id = ? AND user_id = ?", headerLedgerID, userID).Scan(&id)
		return id, err
	}

	err := s.repo.db.QueryRowContext(ctx, "SELECT ledger_id FROM ledger_members WHERE user_id = ? LIMIT 1", userID).Scan(&id)
	return id, err
}

// 辅助转换
func (s *Service) toTransactionDTO(tx *transaction.Transaction, tags []string) *transaction.TransactionResponse {
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

	return &transaction.TransactionResponse{
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
