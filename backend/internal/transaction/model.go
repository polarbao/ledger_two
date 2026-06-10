package transaction

import (
	"database/sql"
	"time"
)

// Transaction 交易明细实体
// @brief 表示系统中的单条普通收支或结算流水
type Transaction struct {
	ID              string
	LedgerID        string
	Type            string // expense, income, settlement
	Title           string
	Amount          int64 // 整数分
	Currency        string
	OccurredAt      time.Time
	OwnerUserID     string
	CreatedByUserID string
	PayerUserID     string
	AccountID       sql.NullString
	CategoryID      sql.NullString
	Visibility      string // private, partner_readable, shared
	SplitMethod     sql.NullString
	Note            sql.NullString
	Status          string // normal, deleted
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       sql.NullTime
}

// CreateTransactionRequest 记账请求结构
// @brief 创建普通账单的传参对象
type CreateTransactionRequest struct {
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	AmountCents int64    `json:"amount_cents"`
	Currency    string   `json:"currency"`
	OccurredAt  string   `json:"occurred_at"`
	PayerUserID string   `json:"payer_user_id"`
	AccountID   *string  `json:"account_id"`
	CategoryID  *string  `json:"category_id"`
	Visibility  string   `json:"visibility"`
	TagNames    []string `json:"tag_names"`
	Note        string   `json:"note"`
}

// UpdateTransactionRequest 编辑账单请求结构
// @brief 局部编辑普通账单的传参对象
type UpdateTransactionRequest struct {
	Title       *string   `json:"title"`
	AmountCents *int64    `json:"amount_cents"`
	OccurredAt  *string   `json:"occurred_at"`
	PayerUserID *string   `json:"payer_user_id"`
	AccountID   **string  `json:"account_id"`  // 允许传空以置 null
	CategoryID  **string  `json:"category_id"` // 允许传空以置 null
	Visibility  *string   `json:"visibility"`
	TagNames    *[]string `json:"tag_names"`
	Note        *string   `json:"note"`
	SplitMethod *string   `json:"split_method"` // 新增，用于共同支出编辑
}

// TransactionResponse 统一输出的账单明细 DTO
// @brief 交易流水的标准 API 输出模型
type TransactionResponse struct {
	ID              string          `json:"id"`
	Type            string          `json:"type"`
	Title           string          `json:"title"`
	AmountCents     int64           `json:"amount_cents"`
	Currency        string          `json:"currency"`
	OccurredAt      string          `json:"occurred_at"`
	OwnerUserID     string          `json:"owner_user_id"`
	CreatedByUserID string          `json:"created_by_user_id"`
	PayerUserID     string          `json:"payer_user_id"`
	AccountID       *string         `json:"account_id"`
	CategoryID      *string         `json:"category_id"`
	Visibility      string          `json:"visibility"`
	Note            string          `json:"note"`
	Status          string          `json:"status"`
	Tags            []string        `json:"tags"`
	SplitMethod     *string         `json:"split_method,omitempty"`
	Participants    []SplitResponse `json:"participants,omitempty"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

// TransactionSplit 分摊明细实体
// @brief 记录共享账单的每位成员应分摊金额
type TransactionSplit struct {
	ID            string
	TransactionID string
	UserID        string
	ShareAmount   int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SplitResponse 分摊输出明细 DTO
// @brief 单个成员的分摊信息响应体
type SplitResponse struct {
	UserID           string `json:"user_id"`
	ShareAmountCents int64  `json:"share_amount_cents"`
}

// CreateSharedExpenseRequest 共同支出创建请求结构
// @brief 创建共同支出账单的传参对象
type CreateSharedExpenseRequest struct {
	Title       string   `json:"title"`
	AmountCents int64    `json:"amount_cents"`
	Currency    string   `json:"currency"`
	OccurredAt  string   `json:"occurred_at"`
	PayerUserID string   `json:"payer_user_id"`
	CategoryID  *string  `json:"category_id"`
	SplitMethod string   `json:"split_method"` // equal, payer_only
	TagNames    []string `json:"tag_names"`
	Note        string   `json:"note"`
}

// UpdateSharedExpenseRequest 共同支出编辑请求结构
// @brief 局部编辑共同支出的传参对象
type UpdateSharedExpenseRequest struct {
	Title       *string   `json:"title"`
	AmountCents *int64    `json:"amount_cents"`
	OccurredAt  *string   `json:"occurred_at"`
	PayerUserID *string   `json:"payer_user_id"`
	CategoryID  **string  `json:"category_id"`
	SplitMethod *string   `json:"split_method"`
	TagNames    *[]string `json:"tag_names"`
	Note        *string   `json:"note"`
}

// AuditLog 核心审计日志结构
// @brief 记录账务金额修改和删除动作的审计行
type AuditLog struct {
	ID         string
	LedgerID   string
	ActorUserID string
	Action     string // create, update, delete
	EntityType string // transaction
	EntityID   string
	BeforeJSON sql.NullString
	AfterJSON  sql.NullString
	CreatedAt  time.Time
}
