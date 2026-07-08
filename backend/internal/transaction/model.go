package transaction

import (
	"database/sql"
	"time"
)

// Transaction 交易明细实体
// @brief 表示系统中的单条普通收支或结算流水
type Transaction struct {
	ID               string
	LedgerID         string
	Type             string // expense, income, settlement
	Title            string
	Amount           int64 // 整数分
	Currency         string
	OccurredAt       time.Time
	OwnerUserID      string
	CreatedByUserID  string
	PayerUserID      string
	AccountID        sql.NullString
	CategoryID       sql.NullString
	CategoryName     sql.NullString
	CategoryArchived sql.NullBool
	Visibility       string // private, partner_readable, shared
	SplitMethod      sql.NullString
	Note             sql.NullString
	AttachmentPaths  sql.NullString // 附件相对路径 JSON 数组或列表
	Status           string         // normal, deleted
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        sql.NullTime
}

// TransactionDefault 记录当前用户在当前账本下的快捷记账偏好
type TransactionDefault struct {
	LedgerID    string
	UserID      string
	Type        string
	CategoryID  sql.NullString
	AccountID   sql.NullString
	PayerUserID sql.NullString
	Visibility  string
	SplitMethod sql.NullString
	TagNames    sql.NullString
	UpdatedAt   time.Time
}

// TransactionDefaultResponse 快捷记账默认值响应
type TransactionDefaultResponse struct {
	Type        string   `json:"type"`
	CategoryID  *string  `json:"category_id,omitempty"`
	AccountID   *string  `json:"account_id,omitempty"`
	PayerUserID string   `json:"payer_user_id"`
	Visibility  string   `json:"visibility"`
	SplitMethod string   `json:"split_method"`
	TagNames    []string `json:"tag_names"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

// CreateTransactionRequest 记账请求结构
// @brief 创建普通账单的传参对象
type CreateTransactionRequest struct {
	Type            string   `json:"type"`
	Title           string   `json:"title"`
	AmountCents     int64    `json:"amount_cents"`
	Currency        string   `json:"currency"`
	OccurredAt      string   `json:"occurred_at"`
	PayerUserID     string   `json:"payer_user_id"`
	AccountID       *string  `json:"account_id"`
	CategoryID      *string  `json:"category_id"`
	Visibility      string   `json:"visibility"`
	TagNames        []string `json:"tag_names"`
	Note            string   `json:"note"`
	AttachmentPaths []string `json:"attachment_paths"`
}

// UpdateTransactionRequest 编辑账单请求结构
// @brief 局部编辑普通账单的传参对象
type UpdateTransactionRequest struct {
	Title           *string       `json:"title"`
	AmountCents     *int64        `json:"amount_cents"`
	OccurredAt      *string       `json:"occurred_at"`
	PayerUserID     *string       `json:"payer_user_id"`
	AccountID       **string      `json:"account_id"`  // 允许传空以置 null
	CategoryID      **string      `json:"category_id"` // 允许传空以置 null
	Visibility      *string       `json:"visibility"`
	TagNames        *[]string     `json:"tag_names"`
	Note            *string       `json:"note"`
	SplitMethod     *string       `json:"split_method"`     // 新增，用于共同支出编辑
	Splits          *[]SplitInput `json:"splits,omitempty"` // 用于多人的高级分摊编辑
	AttachmentPaths *[]string     `json:"attachment_paths"`
}

// TransactionResponse 统一输出的账单明细 DTO
// @brief 交易流水的标准 API 输出模型
type TransactionResponse struct {
	ID               string          `json:"id"`
	Type             string          `json:"type"`
	Title            string          `json:"title"`
	AmountCents      int64           `json:"amount_cents"`
	Currency         string          `json:"currency"`
	OccurredAt       string          `json:"occurred_at"`
	OwnerUserID      string          `json:"owner_user_id"`
	CreatedByUserID  string          `json:"created_by_user_id"`
	PayerUserID      string          `json:"payer_user_id"`
	AccountID        *string         `json:"account_id"`
	CategoryID       *string         `json:"category_id"`
	CategoryName     *string         `json:"category_name,omitempty"`
	CategoryArchived *bool           `json:"category_is_archived,omitempty"`
	Visibility       string          `json:"visibility"`
	Note             string          `json:"note"`
	Status           string          `json:"status"`
	Tags             []string        `json:"tags"`
	SplitMethod      *string         `json:"split_method,omitempty"`
	Participants     []SplitResponse `json:"participants,omitempty"`
	AttachmentPaths  []string        `json:"attachment_paths"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
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

// SplitInput 分摊输入参数
// @brief 接收高级分摊计算参数 (金额cents, 比例百分比, 或者份数)
type SplitInput struct {
	UserID string  `json:"user_id"`
	Value  float64 `json:"value"`
}

// CreateSharedExpenseRequest 共同支出创建请求结构
// @brief 创建共同支出账单的传参对象
type CreateSharedExpenseRequest struct {
	Title       string       `json:"title"`
	AmountCents int64        `json:"amount_cents"`
	Currency    string       `json:"currency"`
	OccurredAt  string       `json:"occurred_at"`
	PayerUserID string       `json:"payer_user_id"`
	CategoryID  *string      `json:"category_id"`
	SplitMethod string       `json:"split_method"` // equal, payer_only, amount, ratio, shares
	Splits      []SplitInput `json:"splits,omitempty"`
	TagNames    []string     `json:"tag_names"`
	Note        string       `json:"note"`
}

// UpdateSharedExpenseRequest 共同支出编辑请求结构
// @brief 局部编辑共同支出的传参对象
type UpdateSharedExpenseRequest struct {
	Title       *string       `json:"title"`
	AmountCents *int64        `json:"amount_cents"`
	OccurredAt  *string       `json:"occurred_at"`
	PayerUserID *string       `json:"payer_user_id"`
	CategoryID  **string      `json:"category_id"`
	SplitMethod *string       `json:"split_method"`
	Splits      *[]SplitInput `json:"splits,omitempty"`
	TagNames    *[]string     `json:"tag_names"`
	Note        *string       `json:"note"`
}

// AuditLog 核心审计日志结构
// @brief 记录账务金额修改和删除动作的审计行
type AuditLog struct {
	ID          string
	LedgerID    string
	ActorUserID string
	Action      string // create, update, delete
	EntityType  string // transaction
	EntityID    string
	BeforeJSON  sql.NullString
	AfterJSON   sql.NullString
	CreatedAt   time.Time
}

// TransactionTemplate 账单模板实体
type TransactionTemplate struct {
	ID              string
	LedgerID        string
	Name            string
	Type            string // expense, income, shared_expense
	Title           sql.NullString
	AmountCents     sql.NullInt64
	CategoryID      sql.NullString
	AccountID       sql.NullString
	PayerUserID     sql.NullString
	SplitMethod     sql.NullString
	TagNames        sql.NullString // 逗号分隔的标签名列表
	Note            sql.NullString
	CreatedByUserID string
	IsArchived      bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ArchivedAt      sql.NullTime
}

// CreateTemplateRequest 创建与更新模板的请求结构
type CreateTemplateRequest struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // expense, income, shared_expense
	Title       *string  `json:"title"`
	AmountCents *int64   `json:"amount_cents"`
	CategoryID  *string  `json:"category_id"`
	AccountID   *string  `json:"account_id"`
	PayerUserID *string  `json:"payer_user_id"`
	SplitMethod *string  `json:"split_method"`
	TagNames    []string `json:"tag_names"`
	Note        *string  `json:"note"`
}

// TemplateResponse 模板输出统一 DTO
type TemplateResponse struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Title           string   `json:"title"`
	AmountCents     *int64   `json:"amount_cents,omitempty"`
	CategoryID      string   `json:"category_id"`
	AccountID       string   `json:"account_id"`
	PayerUserID     string   `json:"payer_user_id"`
	SplitMethod     string   `json:"split_method"`
	TagNames        []string `json:"tag_names"`
	Note            string   `json:"note"`
	CreatedByUserID string   `json:"created_by_user_id"`
	IsArchived      bool     `json:"is_archived"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	ArchivedAt      string   `json:"archived_at,omitempty"`
}

// BatchTagRequest 批量打标签请求结构
type BatchTagRequest struct {
	TransactionIDs []string `json:"transaction_ids"`
	TagNames       []string `json:"tag_names"`
}

// ImportItemRequest 单条导入交易请求结构
type ImportItemRequest struct {
	OccurredAt  string   `json:"occurred_at"`
	AmountCents int64    `json:"amount_cents"`
	Title       string   `json:"title"`
	Merchant    string   `json:"merchant"`
	CategoryID  string   `json:"category_id"`
	AccountID   string   `json:"account_id"`
	PayerUserID string   `json:"payer_user_id"`
	Type        string   `json:"type"` // expense, shared_expense
	TagNames    []string `json:"tag_names"`
	Note        string   `json:"note"`
}

// AnalyzeImportRequest 导入去重分析请求结构
type AnalyzeImportRequest struct {
	Items []ImportItemRequest `json:"items"`
}

// AnalyzeImportResponse 导入去重分析响应结构
type AnalyzeImportResponse struct {
	TotalCount  int `json:"total_count"`
	ImportCount int `json:"import_count"`
	SkipCount   int `json:"skip_count"`
}

// CommitImportRequest 提交导入请求结构
type CommitImportRequest struct {
	Filename string              `json:"filename"`
	Items    []ImportItemRequest `json:"items"`
}

// Account 账户实体
type Account struct {
	ID             string `json:"id"`
	LedgerID       string `json:"ledger_id"`
	OwnerUserID    string `json:"owner_user_id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Currency       string `json:"currency"`
	InitialBalance int64  `json:"initial_balance"`
	IsArchived     bool   `json:"is_archived"`
}

// ImportRule 导入去重与分类规则实体
type ImportRule struct {
	ID              string    `json:"id"`
	LedgerID        string    `json:"ledger_id"`
	Keyword         string    `json:"keyword"`
	CategoryID      string    `json:"category_id"`
	TagNames        string    `json:"tag_names"` // 逗号分隔的标签
	AccountID       string    `json:"account_id"`
	CreatedByUserID string    `json:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateImportRuleRequest 创建规则请求
type CreateImportRuleRequest struct {
	Keyword    string   `json:"keyword"`
	CategoryID string   `json:"category_id"`
	TagNames   []string `json:"tag_names"`
	AccountID  string   `json:"account_id"`
}

// ImportRuleResponse 导入规则响应 DTO
type ImportRuleResponse struct {
	ID         string   `json:"id"`
	Keyword    string   `json:"keyword"`
	CategoryID string   `json:"category_id"`
	TagNames   []string `json:"tag_names"`
	AccountID  string   `json:"account_id"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}
