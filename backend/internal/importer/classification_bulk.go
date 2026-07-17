package importer

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/importer/classifier"
	"ledger_two/internal/ledger"
)

const maxBulkAdjustRows = 500

type bulkRowUpdate struct {
	Before PreviewRow
	After  PreviewRow
}

func (s *Service) BulkAdjustPreviewRows(ctx context.Context, cmd BulkAdjustCommand) (*BulkClassificationResult, error) {
	if cmd.LedgerContext.Role != ledger.RoleOwner {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅账本 Owner 可批量调整导入预览")
	}
	request, err := validateBulkClassificationRequest(cmd.BatchID, cmd.Request)
	if err != nil {
		return nil, err
	}

	batch, err := s.repo.GetPreviewBatch(ctx, cmd.LedgerContext.LedgerID, cmd.BatchID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "导入批次不存在或不属于当前账本")
		}
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取导入批次失败")
	}
	if !isReadyForReclassify(batch, time.Now()) {
		return nil, bulkAdjustConflict("仅有效的待确认导入批次可以批量调整")
	}

	classificationMetadata, err := s.repo.LoadClassificationMetadata(ctx, batch.LedgerID)
	if err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取分类元数据失败")
	}
	activeMetadata := activeClassificationMetadata(classificationMetadata)
	if request.Action == BulkAdjustActionApplyValues {
		if err := validateBulkApplyMetadata(request, activeMetadata); err != nil {
			return nil, err
		}
	}

	rowsByID := make(map[string]PreviewRow, len(batch.Rows))
	projected := clonePreviewRows(batch.Rows)
	projectedIndex := make(map[string]int, len(projected))
	for index, row := range batch.Rows {
		rowsByID[row.ID] = row
		projectedIndex[row.ID] = index
	}

	result := &BulkClassificationResult{
		UpdatedRowIDs:  []string{},
		SkippedRowIDs:  []string{},
		ConflictRowIDs: []string{},
		Errors:         []ClassificationRowError{},
	}
	updates := make([]bulkRowUpdate, 0, len(request.RowIDs))
	for _, rowID := range request.RowIDs {
		row, exists := rowsByID[rowID]
		if !exists {
			result.Errors = append(result.Errors, classificationRowError(rowID, appErrors.ErrCodeLedgerObjectNotFound, "导入预览行不存在或不属于当前账本"))
			continue
		}
		if isBulkAdjustSkipped(row) {
			result.SkippedRowIDs = append(result.SkippedRowIDs, rowID)
			continue
		}

		adjusted := row
		switch request.Action {
		case BulkAdjustActionAcceptSuggestions:
			if row.Classification.Status == ClassificationStatusConflict {
				result.ConflictRowIDs = append(result.ConflictRowIDs, rowID)
				continue
			}
			if row.Classification.Status == ClassificationStatusManual || row.Classification.Status == ClassificationStatusBulk {
				result.SkippedRowIDs = append(result.SkippedRowIDs, rowID)
				continue
			}
			if !hasPersistedSuggestion(row) {
				result.SkippedRowIDs = append(result.SkippedRowIDs, rowID)
				continue
			}
			if code := validateSuggestedMetadata(row, activeMetadata); code != "" {
				result.Errors = append(result.Errors, classificationRowError(rowID, code, bulkRowErrorMessage(code)))
				continue
			}
			adjusted.SelectedCategoryID = row.SuggestedCategoryID
			adjusted.SelectedAccountID = row.SuggestedAccountID
			adjusted.SelectedTagIDs = copyStrings(row.SuggestedTagIDs)
			markBulkClassification(&adjusted, "bulk_accept_suggestions", "已批量接受持久化建议")
		case BulkAdjustActionApplyValues:
			categoryID := strings.TrimSpace(*request.CategoryID.Value)
			if !categoryMatchesBulkRow(activeMetadata[categoryID], row.TargetTransactionType) {
				result.Errors = append(result.Errors, classificationRowError(rowID, appErrors.ErrCodeCategoryTypeMismatch, bulkRowErrorMessage(appErrors.ErrCodeCategoryTypeMismatch)))
				continue
			}
			adjusted.SelectedCategoryID = categoryID
			adjusted.SelectedAccountID = nullableStringValue(request.AccountID)
			adjusted.SelectedTagIDs = copyStrings(*request.TagIDs)
			markBulkClassification(&adjusted, "bulk_apply_values", "已批量应用指定分类和标签")
		}

		updates = append(updates, bulkRowUpdate{Before: row, After: adjusted})
		projected[projectedIndex[rowID]] = adjusted
		result.UpdatedRowIDs = append(result.UpdatedRowIDs, rowID)
	}

	result.AffectedRows = len(result.UpdatedRowIDs)
	result.SkippedRows = len(result.SkippedRowIDs)
	result.ConflictRows = len(result.ConflictRowIDs)
	projectedBatch := &PreviewBatch{Rows: projected}
	recountClassificationSummary(projectedBatch)
	result.Summary = projectedBatch.ClassificationSummary

	if err := s.repo.ApplyBulkAdjustment(ctx, cmd.LedgerContext, batch.ID, batch.UpdatedAt, updates, result, request.Action); err != nil {
		if errors.Is(err, errBulkAdjustBatchChanged) || errors.Is(err, errBulkAdjustMetadataChanged) {
			return nil, bulkAdjustConflict("导入批次或分类元数据已变化，请刷新后重试")
		}
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "保存批量调整结果失败")
	}
	return result, nil
}

func validateBulkClassificationRequest(batchID string, request BulkClassificationRequest) (BulkClassificationRequest, error) {
	if strings.TrimSpace(batchID) == "" {
		return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入批次 ID 不能为空")
	}
	if len(request.RowIDs) == 0 || len(request.RowIDs) > maxBulkAdjustRows {
		return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "批量调整行数必须为 1 到 500")
	}
	seenRows := make(map[string]struct{}, len(request.RowIDs))
	for _, rowID := range request.RowIDs {
		if rowID == "" || rowID != strings.TrimSpace(rowID) {
			return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入预览行 ID 无效")
		}
		if _, exists := seenRows[rowID]; exists {
			return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "批量调整行 ID 不可重复")
		}
		seenRows[rowID] = struct{}{}
	}

	switch request.Action {
	case BulkAdjustActionAcceptSuggestions:
		if request.CategoryID.Set || request.AccountID.Set || request.TagIDs != nil {
			return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "接受建议时不得提交临时分类、账户或标签")
		}
	case BulkAdjustActionApplyValues:
		if !request.CategoryID.Set || request.CategoryID.Value == nil || strings.TrimSpace(*request.CategoryID.Value) == "" ||
			!request.AccountID.Set || request.TagIDs == nil {
			return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "应用指定值时必须完整提交分类、可空账户和标签")
		}
		if request.AccountID.Value != nil && strings.TrimSpace(*request.AccountID.Value) == "" {
			return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "账户 ID 必须为有效 ID 或 null")
		}
		if len(*request.TagIDs) > 8 {
			return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeTagLimitExceeded, "单条账单最多选择 8 个标签")
		}
		seenTags := make(map[string]struct{}, len(*request.TagIDs))
		for index, tagID := range *request.TagIDs {
			trimmed := strings.TrimSpace(tagID)
			if trimmed == "" || trimmed != tagID {
				return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "标签 ID 无效")
			}
			if _, exists := seenTags[tagID]; exists {
				return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "标签 ID 不可重复")
			}
			seenTags[tagID] = struct{}{}
			(*request.TagIDs)[index] = trimmed
		}
	default:
		return request, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "批量调整动作无效")
	}
	return request, nil
}

func activeClassificationMetadata(items []classifier.MetadataItem) map[string]classifier.MetadataItem {
	result := make(map[string]classifier.MetadataItem, len(items))
	for _, item := range items {
		if !item.IsArchived {
			result[item.ID] = item
		}
	}
	return result
}

func validateBulkApplyMetadata(request BulkClassificationRequest, metadata map[string]classifier.MetadataItem) error {
	categoryID := strings.TrimSpace(*request.CategoryID.Value)
	category, exists := metadata[categoryID]
	if !exists || (category.Kind != classifier.MetadataExpenseCategory && category.Kind != classifier.MetadataIncomeCategory) {
		return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "分类不存在、已归档或不属于当前账本")
	}
	if accountID := nullableStringValue(request.AccountID); accountID != "" {
		account, exists := metadata[accountID]
		if !exists || account.Kind != classifier.MetadataAccount {
			return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "账户不存在、已归档或不属于当前账本")
		}
	}
	for _, tagID := range *request.TagIDs {
		tag, exists := metadata[tagID]
		if !exists || tag.Kind != classifier.MetadataTag {
			return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "标签不存在、已归档或不属于当前账本")
		}
	}
	return nil
}

func validateSuggestedMetadata(row PreviewRow, metadata map[string]classifier.MetadataItem) string {
	if len(row.SuggestedTagIDs) > 8 || hasDuplicateStrings(row.SuggestedTagIDs) {
		return appErrors.ErrCodeTagLimitExceeded
	}
	if row.SuggestedCategoryID != "" {
		category, exists := metadata[row.SuggestedCategoryID]
		if !exists {
			return appErrors.ErrCodeClassificationRuleStale
		}
		if !categoryMatchesBulkRow(category, row.TargetTransactionType) {
			return appErrors.ErrCodeCategoryTypeMismatch
		}
	}
	if row.SuggestedAccountID != "" {
		account, exists := metadata[row.SuggestedAccountID]
		if !exists || account.Kind != classifier.MetadataAccount {
			return appErrors.ErrCodeClassificationRuleStale
		}
	}
	for _, tagID := range row.SuggestedTagIDs {
		tag, exists := metadata[tagID]
		if tagID == "" || !exists || tag.Kind != classifier.MetadataTag {
			return appErrors.ErrCodeClassificationRuleStale
		}
	}
	return ""
}

func categoryMatchesBulkRow(category classifier.MetadataItem, targetType string) bool {
	return (targetType == TargetTransactionExpense && category.Kind == classifier.MetadataExpenseCategory) ||
		(targetType == TargetTransactionIncome && category.Kind == classifier.MetadataIncomeCategory)
}

func isBulkAdjustSkipped(row PreviewRow) bool {
	return row.DuplicateStatus == DuplicateStatusDuplicate || row.DuplicateStatus == DuplicateStatusInvalid ||
		row.RowStatus == RowStatusSkipped || row.RowStatus == RowStatusImported || row.RowStatus == RowStatusFailed ||
		row.TargetTransactionType == TargetTransactionSkipped
}

func hasPersistedSuggestion(row PreviewRow) bool {
	return row.SuggestedCategoryID != "" || row.SuggestedAccountID != "" || len(row.SuggestedTagIDs) > 0
}

func hasDuplicateStrings(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func markBulkClassification(row *PreviewRow, reasonCode string, reasonText string) {
	row.RowStatus = RowStatusAdjusted
	row.Classification.Status = ClassificationStatusBulk
	row.Classification.Confidence = string(classifier.ConfidenceHigh)
	row.Classification.Source = string(classifier.SourceBulk)
	row.Classification.ReasonCode = reasonCode
	row.Classification.ReasonText = reasonText
	row.Classification.MatchedRuleIDs = copyStrings(row.Classification.MatchedRuleIDs)
	row.Classification.SuggestedCategoryID = row.SuggestedCategoryID
	row.Classification.SuggestedAccountID = row.SuggestedAccountID
	row.Classification.SuggestedTagIDs = copyStrings(row.SuggestedTagIDs)
}

func nullableStringValue(value NullableString) string {
	if value.Value == nil {
		return ""
	}
	return strings.TrimSpace(*value.Value)
}

func classificationRowError(rowID string, code string, message string) ClassificationRowError {
	return ClassificationRowError{RowID: rowID, Code: code, Message: message}
}

func bulkRowErrorMessage(code string) string {
	switch code {
	case appErrors.ErrCodeCategoryTypeMismatch:
		return "分类与账单收支类型不匹配"
	case appErrors.ErrCodeTagLimitExceeded:
		return "最终标签数量或内容无效"
	default:
		return "持久化建议引用的分类、账户或标签已不可用"
	}
}

func bulkAdjustConflict(message string) *appErrors.AppError {
	return appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeImportBulkAdjustConflict, message)
}
