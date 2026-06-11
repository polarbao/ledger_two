package transaction

import (
	"database/sql"
	"time"
)

// RecurringRule 周期账单规则实体
type RecurringRule struct {
	ID              string
	LedgerID        string
	Name            string
	Type            string // expense, income, shared_expense
	Title           sql.NullString
	AmountCents     sql.NullInt64
	CategoryID      sql.NullString
	PayerUserID     sql.NullString
	SplitMethod     sql.NullString // equal, payer_only
	TagNames        sql.NullString // 逗号分隔的标签名列表
	Note            sql.NullString
	Frequency       string // weekly, monthly, yearly
	NextDueDate     string // YYYY-MM-DD
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// RecurringReminder 待确认提醒实体
type RecurringReminder struct {
	ID            string
	LedgerID      string
	RuleID        string
	DueDate       string // YYYY-MM-DD
	Status        string // pending, confirmed, ignored
	TransactionID sql.NullString
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RecurringReminderDetail 包含规则细节的提醒实体，用于 Repository 多表联查
type RecurringReminderDetail struct {
	Reminder     *RecurringReminder
	RuleName     string
	Type         string
	Title        sql.NullString
	AmountCents  sql.NullInt64
	CategoryID   sql.NullString
	CategoryName sql.NullString
	PayerUserID  sql.NullString
	SplitMethod  sql.NullString
	TagNames     sql.NullString
	Note         sql.NullString
	Frequency    string
}

// CreateRecurringRuleRequest 创建与更新周期规则的请求结构
type CreateRecurringRuleRequest struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // expense, income, shared_expense
	Title       *string  `json:"title"`
	AmountCents *int64   `json:"amount_cents"`
	CategoryID  *string  `json:"category_id"`
	PayerUserID *string  `json:"payer_user_id"`
	SplitMethod *string  `json:"split_method"`
	TagNames    []string `json:"tag_names"`
	Note        *string  `json:"note"`
	Frequency   string   `json:"frequency"`     // weekly, monthly, yearly
	NextDueDate string   `json:"next_due_date"` // YYYY-MM-DD
}

// RecurringRuleResponse 周期规则输出统一 DTO
type RecurringRuleResponse struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Title           string   `json:"title"`
	AmountCents     *int64   `json:"amount_cents,omitempty"`
	CategoryID      string   `json:"category_id"`
	PayerUserID     string   `json:"payer_user_id"`
	SplitMethod     string   `json:"split_method"`
	TagNames        []string `json:"tag_names"`
	Note            string   `json:"note"`
	Frequency       string   `json:"frequency"`
	NextDueDate     string   `json:"next_due_date"`
	CreatedByUserID string   `json:"created_by_user_id"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// RecurringReminderResponse 周期提醒输出统一 DTO
type RecurringReminderResponse struct {
	ID            string   `json:"id"`
	RuleID        string   `json:"rule_id"`
	RuleName      string   `json:"rule_name"`
	Type          string   `json:"type"`
	Title         string   `json:"title"`
	AmountCents   *int64   `json:"amount_cents,omitempty"`
	CategoryID    string   `json:"category_id"`
	CategoryName  string   `json:"category_name"`
	PayerUserID   string   `json:"payer_user_id"`
	SplitMethod   string   `json:"split_method"`
	TagNames      []string `json:"tag_names"`
	Note          string   `json:"note"`
	Frequency     string   `json:"frequency"`
	DueDate       string   `json:"due_date"`
	Status        string   `json:"status"`
	TransactionID string   `json:"transaction_id,omitempty"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}
