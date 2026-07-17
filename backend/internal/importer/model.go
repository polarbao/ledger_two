package importer

import (
	"database/sql"
	"encoding/json"

	"ledger_two/internal/importer/tabular"
	"ledger_two/internal/ledger"
)

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
	ErrorCodeTitleInvalid  = "IMPORT_ROW_TITLE_INVALID"

	ClassificationModeOff     = "off"
	ClassificationModeSuggest = "suggest"
	ClassificationModeGraded  = "graded"

	ClassificationStatusAutoSelected = "auto_selected"
	ClassificationStatusSuggested    = "suggested"
	ClassificationStatusFallback     = "fallback"
	ClassificationStatusManual       = "manual"
	ClassificationStatusBulk         = "bulk"
	ClassificationStatusConflict     = "conflict"
	ClassificationStatusUnresolved   = "unresolved"

	BulkAdjustActionAcceptSuggestions = "accept_suggestions"
	BulkAdjustActionApplyValues       = "apply_values"
)

type Classification struct {
	Status              string   `json:"status"`
	Confidence          string   `json:"confidence"`
	Source              string   `json:"source,omitempty"`
	ReasonCode          string   `json:"reason_code,omitempty"`
	ReasonText          string   `json:"reason_text,omitempty"`
	MatchedRuleIDs      []string `json:"matched_rule_ids"`
	SuggestedCategoryID string   `json:"suggested_category_id,omitempty"`
	SuggestedAccountID  string   `json:"suggested_account_id,omitempty"`
	SuggestedTagIDs     []string `json:"suggested_tag_ids"`
}

type ClassificationSummary struct {
	AutoSelected int `json:"auto_selected"`
	Suggested    int `json:"suggested"`
	Fallback     int `json:"fallback"`
	Manual       int `json:"manual"`
	Bulk         int `json:"bulk"`
	Conflict     int `json:"conflict"`
	Unresolved   int `json:"unresolved"`
}

type Preview struct {
	SourceType string       `json:"source_type"`
	Rows       []PreviewRow `json:"rows"`
}

type PreviewBatch struct {
	ID                    string                `json:"id"`
	LedgerID              string                `json:"ledger_id"`
	SourceType            string                `json:"source_type"`
	FileFormat            string                `json:"file_format"`
	ParserMetadata        tabular.Metadata      `json:"parser_metadata"`
	Filename              string                `json:"filename"`
	FileSHA256            string                `json:"file_sha256"`
	Status                string                `json:"status"`
	TotalRows             int                   `json:"total_rows"`
	NewRows               int                   `json:"new_rows"`
	DuplicateRows         int                   `json:"duplicate_rows"`
	SuspiciousRows        int                   `json:"suspicious_rows"`
	InvalidRows           int                   `json:"invalid_rows"`
	ImportedRows          int                   `json:"imported_rows"`
	SkippedRows           int                   `json:"skipped_rows"`
	FailedRows            int                   `json:"failed_rows"`
	CreatedByUserID       string                `json:"created_by_user_id"`
	CreatedAt             string                `json:"created_at"`
	UpdatedAt             string                `json:"updated_at"`
	CommittedAt           string                `json:"committed_at,omitempty"`
	ExpiresAt             string                `json:"expires_at,omitempty"`
	ClassificationSummary ClassificationSummary `json:"classification_summary"`
	Rows                  []PreviewRow          `json:"rows"`
}

type PreviewRow struct {
	ID                    string         `json:"id,omitempty"`
	BatchID               string         `json:"batch_id,omitempty"`
	RowNumber             int            `json:"row_number"`
	OccurredAt            string         `json:"occurred_at,omitempty"`
	Title                 string         `json:"title"`
	Merchant              string         `json:"merchant"`
	Description           string         `json:"description,omitempty"`
	AmountCents           int64          `json:"amount_cents"`
	Direction             string         `json:"direction"`
	TargetTransactionType string         `json:"target_transaction_type"`
	DuplicateStatus       string         `json:"duplicate_status"`
	RowStatus             string         `json:"row_status"`
	SourceAccount         string         `json:"source_account,omitempty"`
	ExternalOrderID       string         `json:"external_order_id,omitempty"`
	SuspiciousReason      string         `json:"suspicious_reason,omitempty"`
	SuggestedCategoryID   string         `json:"suggested_category_id,omitempty"`
	SuggestedAccountID    string         `json:"suggested_account_id,omitempty"`
	SuggestedTagIDs       []string       `json:"suggested_tag_ids,omitempty"`
	SuggestedRuleID       string         `json:"suggested_rule_id,omitempty"`
	SuggestionReason      string         `json:"suggestion_reason,omitempty"`
	SelectedCategoryID    string         `json:"selected_category_id,omitempty"`
	SelectedAccountID     string         `json:"selected_account_id,omitempty"`
	SelectedTagIDs        []string       `json:"selected_tag_ids,omitempty"`
	Visibility            string         `json:"visibility,omitempty"`
	Classification        Classification `json:"classification"`
	ImportHash            string         `json:"-"`
	Error                 *RowError      `json:"error,omitempty"`
}

type ReclassifyCommand struct {
	LedgerContext ledger.LedgerContext
	BatchID       string
	DryRun        bool
}

type ReclassifyRequest struct {
	DryRun *bool `json:"dry_run,omitempty"`
}

type ReclassifyRowChange struct {
	RowID         string   `json:"row_id"`
	OldStatus     string   `json:"old_status"`
	NewStatus     string   `json:"new_status"`
	OldCategoryID string   `json:"old_category_id,omitempty"`
	NewCategoryID string   `json:"new_category_id,omitempty"`
	OldTagIDs     []string `json:"old_tag_ids"`
	NewTagIDs     []string `json:"new_tag_ids"`
}

type ReclassifyResult struct {
	DryRun              bool                  `json:"dry_run"`
	EligibleRows        int                   `json:"eligible_rows"`
	ChangedRows         int                   `json:"changed_rows"`
	UnchangedRows       int                   `json:"unchanged_rows"`
	ProtectedManualRows int                   `json:"protected_manual_rows"`
	ProtectedBulkRows   int                   `json:"protected_bulk_rows"`
	ConflictRows        int                   `json:"conflict_rows"`
	Summary             ClassificationSummary `json:"summary"`
	Changes             []ReclassifyRowChange `json:"changes"`
}

// NullableString records whether a nullable JSON field was present. Task53.4A
// distinguishes omitted fields from explicit null values in its action union.
type NullableString struct {
	Set   bool
	Value *string
}

func (value *NullableString) UnmarshalJSON(data []byte) error {
	value.Set = true
	if string(data) == "null" {
		value.Value = nil
		return nil
	}
	var decoded string
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	value.Value = &decoded
	return nil
}

type BulkClassificationRequest struct {
	RowIDs     []string       `json:"row_ids"`
	Action     string         `json:"action"`
	CategoryID NullableString `json:"category_id"`
	AccountID  NullableString `json:"account_id"`
	TagIDs     *[]string      `json:"tag_ids,omitempty"`
}

type BulkAdjustCommand struct {
	LedgerContext ledger.LedgerContext
	BatchID       string
	Request       BulkClassificationRequest
}

type ClassificationRowError struct {
	RowID   string `json:"row_id"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type BulkClassificationResult struct {
	AffectedRows   int                      `json:"affected_rows"`
	SkippedRows    int                      `json:"skipped_rows"`
	ConflictRows   int                      `json:"conflict_rows"`
	UpdatedRowIDs  []string                 `json:"updated_row_ids"`
	SkippedRowIDs  []string                 `json:"skipped_row_ids"`
	ConflictRowIDs []string                 `json:"conflict_row_ids"`
	Errors         []ClassificationRowError `json:"errors"`
	Summary        ClassificationSummary    `json:"summary"`
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

type DiscardImportBatchRequest struct {
	Reason string `json:"reason"`
}

type DiscardImportBatchResult struct {
	BatchID       string `json:"batch_id"`
	Status        string `json:"status"`
	DiscardReason string `json:"discard_reason"`
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
