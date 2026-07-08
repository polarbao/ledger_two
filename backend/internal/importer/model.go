package importer

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
	DuplicateStatusSuspicious = "suspicious"
	DuplicateStatusInvalid    = "invalid"

	RowStatusPending = "pending"
	RowStatusSkipped = "skipped"
	RowStatusFailed  = "failed"

	ErrorCodeAmountInvalid = "IMPORT_ROW_AMOUNT_INVALID"
	ErrorCodeTimeInvalid   = "IMPORT_ROW_TIME_INVALID"
)

type Preview struct {
	SourceType string       `json:"source_type"`
	Rows       []PreviewRow `json:"rows"`
}

type PreviewRow struct {
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
	Error                 *RowError `json:"error,omitempty"`
}

type RowError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
