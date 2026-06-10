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

// BalanceResponse 结算中心余额及净额响应体
// @brief 输出给前端展示的双方已付、应摊、已结及最终谁欠谁的净额报表 DTO
type BalanceResponse struct {
	UserAPaidCents       int64  `json:"user_a_paid_cents"`         // A 垫付的共同支出总额
	UserAShareCents      int64  `json:"user_a_share_cents"`        // A 实际应承担的共同支出总额
	UserBPaidCents       int64  `json:"user_b_paid_cents"`         // B 垫付的共同支出总额
	UserBShareCents      int64  `json:"user_b_share_cents"`        // B 实际应承担的共同支出总额
	UserASettledToBCents int64  `json:"user_a_settled_to_b_cents"` // A 已向 B 结算补款总额
	UserBSettledToACents int64  `json:"user_b_settled_to_a_cents"` // B 已向 A 结算补款总额
	UserANetCents        int64  `json:"user_a_net_cents"`          // A 的最终未结净额
	UserBNetCents        int64  `json:"user_b_net_cents"`          // B 的最终未结净额
	FromUserID           string `json:"from_user_id"`              // 最终债务人 (谁欠款，结清时为空)
	ToUserID             string `json:"to_user_id"`                // 最终债权人 (欠谁款，结清时为空)
	AmountCents          int64  `json:"amount_cents"`              // 欠款总额 (结清时为 0)
}
