package dashboard

import (
	"ledger_two/internal/settlement"
	"ledger_two/internal/transaction"
)

// DashboardResponse 首页 Dashboard 聚合返回 DTO
// @brief 包含月度总收支、两端垫付消费总额、最近流水以及分类、标签和成员统计数据
type DashboardResponse struct {
	Month              string                           `json:"month"`
	TotalExpenseCents  int64                            `json:"total_expense_cents"`
	TotalIncomeCents   int64                            `json:"total_income_cents"`
	MyPaidCents        int64                            `json:"my_paid_cents"`
	PartnerPaidCents   int64                            `json:"partner_paid_cents"`
	SharedBalance      settlement.BalanceResponse       `json:"shared_balance"`
	RecentTransactions []transaction.TransactionResponse `json:"recent_transactions"`
	CategorySummary    []SummaryItem                    `json:"category_summary"`
	TagSummary         []SummaryItem                    `json:"tag_summary"`
	UserStats          []UserStatItem                   `json:"user_stats"`
}

// SummaryItem 分类或标签支出汇总 DTO
// @brief 记录某一维度消费支出的金额及在当月消费总支出的百分占比
type SummaryItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	AmountCents int64   `json:"amount_cents"`
	Percent     float64 `json:"percent"`
}

// UserStatItem 成员账务消费统计 DTO
// @brief 记录每个成员本月实际支付的消费金额与实际应该分摊承担的消费金额
type UserStatItem struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	PaidCents   int64  `json:"paid_cents"`
	ShareCents  int64  `json:"share_cents"`
}
