package importer

import "database/sql"

const (
	SourceTypeWechat  = "wechat"
	SourceTypeAlipay  = "alipay"
	SourceTypeGeneric = "generic"

	DirectionExpense  = "expense"
	DirectionIncome   = "income"
	DirectionRefund   = "refund"
	DirectionTransfer = "transfer"
	DirectionUnknown  = "unknown"

	TargetTransactionExpense = "expense"
	TargetTransactionIncome  = "income"
	TargetTransactionSkipped = "skipped"

	DuplicateStatusNew        = "new"
	DuplicateStatusDuplicate  = "duplicate"
	DuplicateStatusSuspicious = "suspicious"
	DuplicateStatusInvalid    = "invalid"

	RowStatusPending  = "pending"
	RowStatusAdjusted = "adjusted"
	RowStatusSkipped  = "skipped"
	RowStatusImported = "imported"
	RowStatusFailed   = "failed"

	ErrorCodeAmountInvalid = "IMPORT_ROW_AMOUNT_INVALID"
	ErrorCodeTimeInvalid   = "IMPORT_ROW_TIME_INVALID"
)

type Preview struct {
	SourceType string       `json:"source_type"`
	Rows       []PreviewRow `json:"rows"`
}

type PreviewBatch struct {
	ID              string       `json:"id"`
	LedgerID        string       `json:"ledger_id"`
	SourceType      string       `json:"source_type"`
	Filename        string       `json:"filename"`
	FileSHA256      string       `json:"file_sha256"`
	Status          string       `json:"status"`
	TotalRows       int          `json:"total_rows"`
	NewRows         int          `json:"new_rows"`
	DuplicateRows   int          `json:"duplicate_rows"`
	SuspiciousRows  int          `json:"suspicious_rows"`
	InvalidRows     int          `json:"invalid_rows"`
	ImportedRows    int          `json:"imported_rows"`
	SkippedRows     int          `json:"skipped_rows"`
	FailedRows      int          `json:"failed_rows"`
	CreatedByUserID string       `json:"created_by_user_id"`
	CreatedAt       string       `json:"created_at"`
	UpdatedAt       string       `json:"updated_at"`
	CommittedAt     string       `json:"committed_at,omitempty"`
	ExpiresAt       string       `json:"expires_at,omitempty"`
	Rows            []PreviewRow `json:"rows"`
}

type PreviewRow struct {
	ID                    string    `json:"id,omitempty"`
	BatchID               string    `json:"batch_id,omitempty"`
	RowNumber             int       `json:"row_number"`
	OccurredAt            string    `json:"occurred_at,omitempty"`
	Title                 string    `json:"title"`
	Merchant              string    `json:"merchant"`
	Description           string    `json:"description,omitempty"`
	AmountCents           int64     `json:"amount_cents"`
	Direction             string    `json:"direction"`
	TargetTransactionType string    `json:"target_transaction_type"`
	DuplicateStatus       string    `json:"duplicate_status"`
	RowStatus             string    `json:"row_status"`
	SourceAccount         string    `json:"source_account,omitempty"`
	ExternalOrderID       string    `json:"external_order_id,omitempty"`
	SuspiciousReason      string    `json:"suspicious_reason,omitempty"`
	SuggestedCategoryID   string    `json:"suggested_category_id,omitempty"`
	SuggestedAccountID    string    `json:"suggested_account_id,omitempty"`
	SuggestedTagIDs       []string  `json:"suggested_tag_ids,omitempty"`
	SelectedCategoryID    string    `json:"selected_category_id,omitempty"`
	SelectedAccountID     string    `json:"selected_account_id,omitempty"`
	SelectedTagIDs        []string  `json:"selected_tag_ids,omitempty"`
	Visibility            string    `json:"visibility,omitempty"`
	ImportHash            string    `json:"-"`
	Error                 *RowError `json:"error,omitempty"`
}

type RowError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type UpdateRowRequest struct {
	TargetTransactionType *string  `json:"target_transaction_type,omitempty"`
	RowStatus             *string  `json:"row_status,omitempty"`
	SelectedCategoryID    *string  `json:"selected_category_id,omitempty"`
	SelectedAccountID     *string  `json:"selected_account_id,omitempty"`
	SelectedTagIDs        []string `json:"selected_tag_ids,omitempty"`
	Visibility            *string  `json:"visibility,omitempty"`
}

type RowAdjustment struct {
	TargetTransactionType string   `json:"target_transaction_type,omitempty"`
	RowStatus             string   `json:"row_status,omitempty"`
	SelectedCategoryID    string   `json:"selected_category_id,omitempty"`
	SelectedAccountID     string   `json:"selected_account_id,omitempty"`
	SelectedTagIDs        []string `json:"selected_tag_ids,omitempty"`
	Visibility            string   `json:"visibility,omitempty"`
}

type CommitResult struct {
	BatchID                 string   `json:"batch_id"`
	Status                  string   `json:"status"`
	ImportedRows            int      `json:"imported_rows"`
	SkippedRows             int      `json:"skipped_rows"`
	FailedRows              int      `json:"failed_rows"`
	GeneratedTransactionIDs []string `json:"generated_transaction_ids"`
}

type ImportRuleResult struct {
	CategoryID string   `json:"category_id,omitempty"`
	AccountID  string   `json:"account_id,omitempty"`
	TagIDs     []string `json:"tag_ids,omitempty"`
	Visibility string   `json:"visibility,omitempty"`
}

type ImportRuleUpsertRequest struct {
	Name           string           `json:"name,omitempty"`
	MatchType      string           `json:"match_type"`
	Pattern        string           `json:"pattern"`
	AmountMinCents *int64           `json:"amount_min_cents,omitempty"`
	AmountMaxCents *int64           `json:"amount_max_cents,omitempty"`
	Priority       *int             `json:"priority,omitempty"`
	Result         ImportRuleResult `json:"result"`
}

type ImportRuleResponse struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	MatchType       string           `json:"match_type"`
	Pattern         string           `json:"pattern"`
	AmountMinCents  *int64           `json:"amount_min_cents,omitempty"`
	AmountMaxCents  *int64           `json:"amount_max_cents,omitempty"`
	Priority        int              `json:"priority"`
	Status          string           `json:"status"`
	Result          ImportRuleResult `json:"result"`
	CreatedByUserID string           `json:"created_by_user_id"`
	CreatedAt       string           `json:"created_at"`
	UpdatedAt       string           `json:"updated_at"`
	ArchivedAt      string           `json:"archived_at,omitempty"`
}

type importRuleRecord struct {
	ID              string
	LedgerID        string
	Name            string
	MatchType       string
	Pattern         string
	AmountMinCents  sql.NullInt64
	AmountMaxCents  sql.NullInt64
	Priority        int
	ResultJSON      string
	Status          string
	CreatedByUserID string
	CreatedAt       string
	UpdatedAt       string
	ArchivedAt      string
}
