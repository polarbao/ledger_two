package settlement

import (
	"database/sql"
	"time"
)

// Settlement 结算物理实体
// @brief 记录双人账本中双方一次差额补款的物理表映射实体
type Settlement struct {
	ID              string
	LedgerID        string
	FromUserID      string
	ToUserID        string
	Amount          int64
	Currency        string
	OccurredAt      time.Time
	Note            sql.NullString
	CreatedByUserID string
	CreatedAt       time.Time
}

// CreateSettlementRequest 结算创建请求体
// @brief 接收补款结算提交参数的 DTO 对象
type CreateSettlementRequest struct {
	FromUserID  string `json:"from_user_id"`
	ToUserID    string `json:"to_user_id"`
	AmountCents int64  `json:"amount_cents"`
	OccurredAt  string `json:"occurred_at"`
	Note        string `json:"note"`
}

// SettlementResponse 结算数据标准响应体
// @brief 输出给 API 前端的结算明细 DTO
type SettlementResponse struct {
	ID              string `json:"id"`
	FromUserID      string `json:"from_user_id"`
	ToUserID        string `json:"to_user_id"`
	AmountCents     int64  `json:"amount_cents"`
	Currency        string `json:"currency"`
	OccurredAt      string `json:"occurred_at"`
	Note            string `json:"note"`
	CreatedByUserID string `json:"created_by_user_id"`
	CreatedAt       string `json:"created_at"`
}

// UserBalance 用户结算净额结构
// @brief 单个用户在账本中的付费、分摊与最终欠款/应收报表
type UserBalance struct {
	UserID          string `json:"user_id"`
	PaidCents       int64  `json:"paid_cents"`
	ShareCents      int64  `json:"share_cents"`
	SettledOutCents int64  `json:"settled_out_cents"`
	SettledInCents  int64  `json:"settled_in_cents"`
	NetCents        int64  `json:"net_cents"` // > 0 表示应收, < 0 表示应付
}

// SuggestedTransfer 建议转账路径
// @brief 通过贪心算法算出的债务人到债权人的直接转账建议
type SuggestedTransfer struct {
	FromUserID  string `json:"from_user_id"`
	ToUserID    string `json:"to_user_id"`
	AmountCents int64  `json:"amount_cents"`
}

// BalanceResponse 结算中心余额及净额响应体
// @brief 输出给前端展示的所有用户余额及转账建议
type BalanceResponse struct {
	UserBalances       []UserBalance       `json:"user_balances"`
	SuggestedTransfers []SuggestedTransfer `json:"suggested_transfers"`

	// 以下为向下兼容的老旧双人硬编码字段，未来版本可移除
	UserAPaidCents       int64  `json:"user_a_paid_cents,omitempty"`
	UserAShareCents      int64  `json:"user_a_share_cents,omitempty"`
	UserBPaidCents       int64  `json:"user_b_paid_cents,omitempty"`
	UserBShareCents      int64  `json:"user_b_share_cents,omitempty"`
	UserASettledToBCents int64  `json:"user_a_settled_to_b_cents,omitempty"`
	UserBSettledToACents int64  `json:"user_b_settled_to_a_cents,omitempty"`
	UserANetCents        int64  `json:"user_a_net_cents,omitempty"`
	UserBNetCents        int64  `json:"user_b_net_cents,omitempty"`
	FromUserID           string `json:"from_user_id,omitempty"`
	ToUserID             string `json:"to_user_id,omitempty"`
	AmountCents          int64  `json:"amount_cents,omitempty"`
}
