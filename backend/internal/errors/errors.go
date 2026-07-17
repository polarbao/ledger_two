package errors

import "fmt"

// AppError 全局统一的业务/参数校验错误
type AppError struct {
	Status  int    `json:"-"` // HTTP 状态码，不直接序列化在 JSON 中
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewAppError 构造 AppError 实例
func NewAppError(status int, code string, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// NewAppErrorWithDetails 构造带 Details 的 AppError 实例
func NewAppErrorWithDetails(status int, code string, message string, details any) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Details: details}
}

const (
	// 通用错误码
	ErrCodeBadRequest                    = "BAD_REQUEST"
	ErrCodeValidationError               = "VALIDATION_ERROR"
	ErrCodeUnauthorized                  = "UNAUTHORIZED"
	ErrCodeForbidden                     = "FORBIDDEN"
	ErrCodeNotFound                      = "NOT_FOUND"
	ErrCodeConflict                      = "CONFLICT"
	ErrCodeInternalError                 = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable            = "SERVICE_UNAVAILABLE"
	ErrCodeLedgerRequired                = "LEDGER_REQUIRED"
	ErrCodeLedgerContextMismatch         = "LEDGER_CONTEXT_MISMATCH"
	ErrCodeLedgerAccessDenied            = "LEDGER_ACCESS_DENIED"
	ErrCodeLedgerObjectNotFound          = "LEDGER_OBJECT_NOT_FOUND"
	ErrCodeLedgerArchived                = "LEDGER_ARCHIVED"
	ErrCodeLedgerInvalidState            = "LEDGER_INVALID_STATE"
	ErrCodeLedgerVersionConflict         = "LEDGER_VERSION_CONFLICT"
	ErrCodeLedgerMemberLimitReached      = "LEDGER_MEMBER_LIMIT_REACHED"
	ErrCodeLedgerOwnerTransferRequired   = "LEDGER_OWNER_TRANSFER_REQUIRED"
	ErrCodeLedgerOwnerInvariantViolation = "LEDGER_OWNER_INVARIANT_VIOLATION"
	ErrCodeLedgerReadyImportExists       = "LEDGER_READY_IMPORT_EXISTS"
	ErrCodeInstanceAdminRequired         = "INSTANCE_ADMIN_REQUIRED"

	// 初始化与认证
	ErrCodeAppAlreadyInitialized = "APP_ALREADY_INITIALIZED"
	ErrCodeAppNotInitialized     = "APP_NOT_INITIALIZED"
	ErrCodeInvalidCredentials    = "INVALID_CREDENTIALS"
	ErrCodeSessionExpired        = "SESSION_EXPIRED"
	ErrCodePasswordTooWeak       = "PASSWORD_TOO_WEAK"

	// 账单
	ErrCodeTransactionNotFound            = "TRANSACTION_NOT_FOUND"
	ErrCodeTransactionAmountInvalid       = "TRANSACTION_AMOUNT_INVALID"
	ErrCodeTransactionTypeInvalid         = "TRANSACTION_TYPE_INVALID"
	ErrCodeTransactionVisibilityInvalid   = "TRANSACTION_VISIBILITY_INVALID"
	ErrCodeTransactionNotEditable         = "TRANSACTION_NOT_EDITABLE"
	ErrCodeTransactionAlreadyDeleted      = "TRANSACTION_ALREADY_DELETED"
	ErrCodeTransactionHasSettlementEffect = "TRANSACTION_HAS_SETTLEMENT_EFFECT"

	// 共同支出与分摊
	ErrCodeSplitMethodInvalid       = "SPLIT_METHOD_INVALID"
	ErrCodeSplitParticipantsInvalid = "SPLIT_PARTICIPANTS_INVALID"
	ErrCodeSplitAmountMismatch      = "SPLIT_AMOUNT_MISMATCH"
	ErrCodeSplitRatioMismatch       = "SPLIT_RATIO_MISMATCH"
	ErrCodePayerNotFound            = "PAYER_NOT_FOUND"

	// 结算
	ErrCodeSettlementAmountInvalid    = "SETTLEMENT_AMOUNT_INVALID"
	ErrCodeSettlementNotRequired      = "SETTLEMENT_NOT_REQUIRED"
	ErrCodeSettlementDirectionInvalid = "SETTLEMENT_DIRECTION_INVALID"
	ErrCodeSettlementNotFound         = "SETTLEMENT_NOT_FOUND"

	// 分类、标签、账户
	ErrCodeCategoryNotFound        = "CATEGORY_NOT_FOUND"
	ErrCodeCategoryArchived        = "CATEGORY_ARCHIVED"
	ErrCodeTagNotFound             = "TAG_NOT_FOUND"
	ErrCodeAccountNotFound         = "ACCOUNT_NOT_FOUND"
	ErrCodeDuplicateName           = "DUPLICATE_NAME"
	ErrCodeMetadataProfileConflict = "METADATA_PROFILE_CONFLICT"

	// 导入、导出、备份
	ErrCodeExportFailed                  = "EXPORT_FAILED"
	ErrCodeBackupFailed                  = "BACKUP_FAILED"
	ErrCodeBackupNotFound                = "BACKUP_NOT_FOUND"
	ErrCodeBackupPathInvalid             = "BACKUP_PATH_INVALID"
	ErrCodeImportFileInvalid             = "IMPORT_FILE_INVALID"
	ErrCodeImportFileFormatMismatch      = "IMPORT_FILE_FORMAT_MISMATCH"
	ErrCodeImportFileUnsupported         = "IMPORT_FILE_UNSUPPORTED"
	ErrCodeImportSourceMismatch          = "IMPORT_SOURCE_MISMATCH"
	ErrCodeImportWorkbookAmbiguous       = "IMPORT_WORKBOOK_AMBIGUOUS"
	ErrCodeImportWorkbookStructure       = "IMPORT_FILE_STRUCTURE_UNSUPPORTED"
	ErrCodeImportBatchTooLarge           = "IMPORT_BATCH_TOO_LARGE"
	ErrCodeImportDuplicateItem           = "IMPORT_DUPLICATE_ITEM"
	ErrCodeImportPreviewExpired          = "IMPORT_PREVIEW_EXPIRED"
	ErrCodeImportBatchNotFound           = "IMPORT_BATCH_NOT_FOUND"
	ErrCodeImportRowInvalid              = "IMPORT_ROW_INVALID"
	ErrCodeImportRowRequiresConfirmation = "IMPORT_ROW_REQUIRES_CONFIRMATION"
	ErrCodeImportCommitConflict          = "IMPORT_COMMIT_CONFLICT"
	ErrCodeImportReclassifyConflict      = "IMPORT_RECLASSIFY_CONFLICT"
	ErrCodeImportBulkAdjustConflict      = "IMPORT_BULK_ADJUST_CONFLICT"
	ErrCodeCategoryTypeMismatch          = "CATEGORY_TYPE_MISMATCH"
	ErrCodeTagLimitExceeded              = "TAG_LIMIT_EXCEEDED"
	ErrCodeClassificationConflict        = "CLASSIFICATION_CONFLICT"
	ErrCodeClassificationRuleStale       = "CLASSIFICATION_RULE_STALE"
)
