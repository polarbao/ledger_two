package reports

import (
	"context"
	"database/sql"
	"net/http"
	"sort"

	"ledger_two/internal/dashboard"
	"ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/settlement"
)

type Service struct {
	db            *sql.DB
	dashboardRepo *dashboard.Repository
	settleSvc     *settlement.Service
}

func NewService(db *sql.DB, dashboardRepo *dashboard.Repository, settleSvc *settlement.Service) *Service {
	return &Service{
		db:            db,
		dashboardRepo: dashboardRepo,
		settleSvc:     settleSvc,
	}
}

type MonthlySummaryResponse struct {
	Month            string `json:"month"`
	TotalExpense     int64  `json:"total_expense"`
	TotalIncome      int64  `json:"total_income"`
	SharedExpense    int64  `json:"shared_expense"`
	PersonalExpense  int64  `json:"personal_expense"`
	SettlementAmount int64  `json:"settlement_amount"`
}

type CategorySummaryItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	AmountCents int64   `json:"amount_cents"`
	Percent     float64 `json:"percent"`
}

type TagSummaryItem struct {
	Name        string  `json:"name"`
	AmountCents int64   `json:"amount_cents"`
	Percent     float64 `json:"percent"`
}

type MemberStatItem struct {
	UserID             string `json:"user_id"`
	DisplayName        string `json:"display_name"`
	PaidAmount         int64  `json:"paid_amount"`
	ShareAmount        int64  `json:"share_amount"`
	RawNet             int64  `json:"raw_net"`
	SettlementPaid     int64  `json:"settlement_paid"`
	SettlementReceived int64  `json:"settlement_received"`
	FinalNet           int64  `json:"final_net"`
}

// getUserLedgerID 辅助获取 Ledger ID
func (s *Service) getUserLedgerID(ctx context.Context, userID string) (string, error) {
	var id string

	headerLedgerID := middleware.GetHeaderLedgerIDFromContext(ctx)
	if headerLedgerID != "" {
		err := s.db.QueryRowContext(ctx, "SELECT ledger_id FROM ledger_members WHERE ledger_id = ? AND user_id = ?", headerLedgerID, userID).Scan(&id)
		return id, err
	}

	err := s.db.QueryRowContext(ctx, "SELECT ledger_id FROM ledger_members WHERE user_id = ? LIMIT 1", userID).Scan(&id)
	return id, err
}

// GetMonthlySummary 获取月度汇总
func (s *Service) GetMonthlySummary(ctx context.Context, currentUserID string, month string) (*MonthlySummaryResponse, error) {
	ledgerID, err := s.getUserLedgerID(ctx, currentUserID)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "获取系统账本失败")
	}

	// 拉取结算中心轧差 (全局)
	sharedBalance, err := s.settleSvc.GetBalance(ctx, currentUserID)
	if err != nil {
		return nil, err
	}

	// 获取本月数据
	list, _, _, _, _, err := s.dashboardRepo.GetDashboardRawData(ctx, ledgerID, currentUserID, month)
	if err != nil {
		return nil, err
	}

	var totalExpense int64
	var totalIncome int64
	var sharedExpense int64
	var personalExpense int64

	for _, tx := range list {
		if tx.Type == "expense" {
			totalExpense += tx.Amount
			personalExpense += tx.Amount
		} else if tx.Type == "shared_expense" {
			totalExpense += tx.Amount
			sharedExpense += tx.Amount
		} else if tx.Type == "income" {
			totalIncome += tx.Amount
		}
	}

	return &MonthlySummaryResponse{
		Month:            month,
		TotalExpense:     totalExpense,
		TotalIncome:      totalIncome,
		SharedExpense:    sharedExpense,
		PersonalExpense:  personalExpense,
		SettlementAmount: sharedBalance.AmountCents,
	}, nil
}

// GetCategorySummary 获取分类汇总
func (s *Service) GetCategorySummary(ctx context.Context, currentUserID string, month string) ([]CategorySummaryItem, error) {
	ledgerID, err := s.getUserLedgerID(ctx, currentUserID)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "获取系统账本失败")
	}

	list, _, _, categoryMap, _, err := s.dashboardRepo.GetDashboardRawData(ctx, ledgerID, currentUserID, month)
	if err != nil {
		return nil, err
	}

	var totalExpense int64
	catAmountMap := make(map[string]int64)

	for _, tx := range list {
		if tx.Type == "expense" || tx.Type == "shared_expense" {
			totalExpense += tx.Amount
			catID := ""
			if tx.CategoryID.Valid {
				catID = tx.CategoryID.String
			}
			catAmountMap[catID] += tx.Amount
		}
	}

	var items []CategorySummaryItem
	for catID, amount := range catAmountMap {
		name, exists := categoryMap[catID]
		if !exists || name == "" {
			name = "未分类消费"
		}
		var percent float64
		if totalExpense > 0 {
			percent = float64(amount) / float64(totalExpense) * 100
		}
		items = append(items, CategorySummaryItem{
			ID:          catID,
			Name:        name,
			AmountCents: amount,
			Percent:     percent,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].AmountCents > items[j].AmountCents
	})

	return items, nil
}

// GetTagSummary 获取标签汇总 (每个标签计入全额)
func (s *Service) GetTagSummary(ctx context.Context, currentUserID string, month string) ([]TagSummaryItem, error) {
	ledgerID, err := s.getUserLedgerID(ctx, currentUserID)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "获取系统账本失败")
	}

	list, tagsMap, _, _, _, err := s.dashboardRepo.GetDashboardRawData(ctx, ledgerID, currentUserID, month)
	if err != nil {
		return nil, err
	}

	var totalExpense int64
	tagAmountMap := make(map[string]int64)

	for _, tx := range list {
		if tx.Type == "expense" || tx.Type == "shared_expense" {
			totalExpense += tx.Amount
			if tags, ok := tagsMap[tx.ID]; ok {
				for _, name := range tags {
					tagAmountMap[name] += tx.Amount
				}
			}
		}
	}

	var items []TagSummaryItem
	for name, amount := range tagAmountMap {
		var percent float64
		if totalExpense > 0 {
			percent = float64(amount) / float64(totalExpense) * 100
		}
		items = append(items, TagSummaryItem{
			Name:        name,
			AmountCents: amount,
			Percent:     percent,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].AmountCents > items[j].AmountCents
	})

	return items, nil
}

// GetMemberSummary 获取成员消费及结算汇总
func (s *Service) GetMemberSummary(ctx context.Context, currentUserID string, month string) ([]MemberStatItem, error) {
	ledgerID, err := s.getUserLedgerID(ctx, currentUserID)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "获取系统账本失败")
	}

	list, _, splitsMap, _, userMap, err := s.dashboardRepo.GetDashboardRawData(ctx, ledgerID, currentUserID, month)
	if err != nil {
		return nil, err
	}

	paidMap := make(map[string]int64)
	shareMap := make(map[string]int64)

	for userID := range userMap {
		paidMap[userID] = 0
		shareMap[userID] = 0
	}

	for _, tx := range list {
		if tx.Type == "expense" || tx.Type == "shared_expense" {
			paidMap[tx.PayerUserID] += tx.Amount

			if tx.Type == "expense" {
				// 个人账单由创建者承担
				shareMap[tx.CreatedByUserID] += tx.Amount
			} else if tx.Type == "shared_expense" {
				// 共享账单按 split 分摊
				if splits, ok := splitsMap[tx.ID]; ok {
					for _, split := range splits {
						shareMap[split.UserID] += split.ShareAmountCents
					}
				}
			}
		}
	}

	// 查出该月份结算记录以求 settlement_paid / settlement_received
	settlePaidMap := make(map[string]int64)
	settleRecvMap := make(map[string]int64)
	for userID := range userMap {
		settlePaidMap[userID] = 0
		settleRecvMap[userID] = 0
	}

	sRows, err := s.db.QueryContext(ctx, "SELECT from_user_id, to_user_id, amount FROM settlements WHERE ledger_id = ? AND occurred_at LIKE ?", ledgerID, month+"%")
	if err == nil {
		defer sRows.Close()
		for sRows.Next() {
			var fromUser, toUser string
			var amount int64
			if err := sRows.Scan(&fromUser, &toUser, &amount); err == nil {
				settlePaidMap[fromUser] += amount
				settleRecvMap[toUser] += amount
			}
		}
	}

	var items []MemberStatItem
	for userID, dispName := range userMap {
		paid := paidMap[userID]
		share := shareMap[userID]
		rawNet := paid - share

		sPaid := settlePaidMap[userID]
		sRecv := settleRecvMap[userID]

		// final_net = raw_net - (received - paid)
		finalNet := rawNet - (sRecv - sPaid)

		items = append(items, MemberStatItem{
			UserID:             userID,
			DisplayName:        dispName,
			PaidAmount:         paid,
			ShareAmount:        share,
			RawNet:             rawNet,
			SettlementPaid:     sPaid,
			SettlementReceived: sRecv,
			FinalNet:           finalNet,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].DisplayName < items[j].DisplayName
	})

	return items, nil
}
