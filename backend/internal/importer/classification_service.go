package importer

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"time"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/importer/classifier"
)

func (s *Service) applyClassifications(ctx context.Context, batch *PreviewBatch) error {
	classificationContext, err := s.repo.LoadClassificationContext(ctx, batch.LedgerID)
	if err != nil {
		return err
	}
	for index := range batch.Rows {
		result := classifier.Classify(classificationContext, classifierRow(batch, batch.Rows[index]))
		applyClassificationResult(&batch.Rows[index], result, s.classificationMode)
	}
	recountBatch(batch)
	return nil
}

func classifierRow(batch *PreviewBatch, row PreviewRow) classifier.Row {
	currentSource := classifier.ClassificationSource(row.Classification.Source)
	if currentSource == "" && row.RowStatus == RowStatusAdjusted {
		currentSource = classifier.SourceManual
	}
	return classifier.Row{
		LedgerID:              batch.LedgerID,
		SourceType:            batch.SourceType,
		Merchant:              row.Merchant,
		Title:                 row.Title,
		Description:           row.Description,
		SourceAccount:         row.SourceAccount,
		AmountCents:           row.AmountCents,
		Direction:             row.Direction,
		TargetTransactionType: row.TargetTransactionType,
		DuplicateStatus:       row.DuplicateStatus,
		RowStatus:             row.RowStatus,
		CurrentSource:         currentSource,
		SelectedCategoryID:    row.SelectedCategoryID,
		SelectedAccountID:     row.SelectedAccountID,
		SelectedTagIDs:        append([]string(nil), row.SelectedTagIDs...),
	}
}

func applyClassificationResult(row *PreviewRow, result classifier.Result, mode string) {
	if result.Protected {
		return
	}
	resetClassification(row)
	if !result.Evaluated {
		return
	}

	decision := result.Decision
	row.Classification = Classification{
		Status:              string(decision.Status),
		Confidence:          string(decision.Confidence),
		Source:              string(decision.Source),
		ReasonCode:          decision.ReasonCode,
		ReasonText:          decision.ReasonText,
		MatchedRuleIDs:      copyStrings(decision.MatchedRuleIDs),
		SuggestedCategoryID: decision.SuggestedCategoryID,
		SuggestedAccountID:  decision.SuggestedAccountID,
		SuggestedTagIDs:     copyStrings(decision.SuggestedTagIDs),
	}

	switch decision.Status {
	case classifier.StatusAutoSelected:
		row.SuggestedCategoryID = decision.SelectedCategoryID
		row.SuggestedAccountID = decision.SelectedAccountID
		row.SuggestedTagIDs = copyStrings(decision.SelectedTagIDs)
		if mode == ClassificationModeGraded {
			row.SelectedCategoryID = decision.SelectedCategoryID
			row.SelectedAccountID = decision.SelectedAccountID
			row.SelectedTagIDs = copyStrings(decision.SelectedTagIDs)
			row.Classification.SuggestedCategoryID = decision.SelectedCategoryID
			row.Classification.SuggestedAccountID = decision.SelectedAccountID
			row.Classification.SuggestedTagIDs = copyStrings(decision.SelectedTagIDs)
		} else {
			row.Classification.Status = ClassificationStatusSuggested
			row.Classification.SuggestedCategoryID = decision.SelectedCategoryID
			row.Classification.SuggestedAccountID = decision.SelectedAccountID
			row.Classification.SuggestedTagIDs = copyStrings(decision.SelectedTagIDs)
		}
	case classifier.StatusSuggested:
		row.SuggestedCategoryID = decision.SuggestedCategoryID
		row.SuggestedAccountID = decision.SuggestedAccountID
		row.SuggestedTagIDs = copyStrings(decision.SuggestedTagIDs)
	case classifier.StatusFallback:
		row.Classification.SuggestedCategoryID = decision.SelectedCategoryID
		row.Classification.SuggestedAccountID = decision.SelectedAccountID
		row.Classification.SuggestedTagIDs = copyStrings(decision.SelectedTagIDs)
		row.SuggestedCategoryID = decision.SelectedCategoryID
		row.SuggestedAccountID = decision.SelectedAccountID
		row.SuggestedTagIDs = copyStrings(decision.SelectedTagIDs)
		if mode == ClassificationModeGraded {
			row.SelectedCategoryID = decision.SelectedCategoryID
			row.SelectedAccountID = decision.SelectedAccountID
			row.SelectedTagIDs = copyStrings(decision.SelectedTagIDs)
		}
	}

	if len(row.Classification.MatchedRuleIDs) > 0 {
		row.SuggestedRuleID = row.Classification.MatchedRuleIDs[0]
	}
	row.SuggestionReason = row.Classification.ReasonText
}

func resetClassification(row *PreviewRow) {
	row.SuggestedCategoryID = ""
	row.SuggestedAccountID = ""
	row.SuggestedTagIDs = nil
	row.SuggestedRuleID = ""
	row.SuggestionReason = ""
	row.SelectedCategoryID = ""
	row.SelectedAccountID = ""
	row.SelectedTagIDs = nil
	row.Classification = defaultClassification()
}

func defaultClassification() Classification {
	return Classification{
		Status:          ClassificationStatusUnresolved,
		Confidence:      string(classifier.ConfidenceNone),
		MatchedRuleIDs:  []string{},
		SuggestedTagIDs: []string{},
	}
}

func markManualClassification(row *PreviewRow) {
	row.Classification = Classification{
		Status:              ClassificationStatusManual,
		Confidence:          string(classifier.ConfidenceHigh),
		Source:              string(classifier.SourceManual),
		ReasonCode:          "manual_adjustment",
		ReasonText:          "用户已手工调整此行",
		MatchedRuleIDs:      copyStrings(row.Classification.MatchedRuleIDs),
		SuggestedCategoryID: row.SuggestedCategoryID,
		SuggestedAccountID:  row.SuggestedAccountID,
		SuggestedTagIDs:     copyStrings(row.SuggestedTagIDs),
	}
}

func recountClassificationSummary(batch *PreviewBatch) {
	var summary ClassificationSummary
	for index := range batch.Rows {
		normalizeClassification(&batch.Rows[index].Classification)
		switch batch.Rows[index].Classification.Status {
		case ClassificationStatusAutoSelected:
			summary.AutoSelected++
		case ClassificationStatusSuggested:
			summary.Suggested++
		case ClassificationStatusFallback:
			summary.Fallback++
		case ClassificationStatusManual:
			summary.Manual++
		case ClassificationStatusBulk:
			summary.Bulk++
		case ClassificationStatusConflict:
			summary.Conflict++
		default:
			summary.Unresolved++
		}
	}
	batch.ClassificationSummary = summary
}

func normalizeClassification(value *Classification) {
	if value.Status == "" {
		value.Status = ClassificationStatusUnresolved
	}
	if value.Confidence == "" {
		value.Confidence = string(classifier.ConfidenceNone)
	}
	if value.MatchedRuleIDs == nil {
		value.MatchedRuleIDs = []string{}
	}
	if value.SuggestedTagIDs == nil {
		value.SuggestedTagIDs = []string{}
	}
}

func (s *Service) ReclassifyPreviewBatch(ctx context.Context, cmd ReclassifyCommand) (*ReclassifyResult, error) {
	if err := requireImportBatchRole(cmd.LedgerContext); err != nil {
		return nil, err
	}
	if cmd.BatchID == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入批次 ID 不能为空")
	}
	if s.classificationMode == ClassificationModeOff {
		return nil, reclassifyConflict("当前环境未开启 Task53 分类器")
	}
	if _, err := s.requireOwnedImportBatch(ctx, cmd.LedgerContext, cmd.BatchID); err != nil {
		return nil, err
	}

	batch, err := s.repo.GetPreviewBatch(ctx, cmd.LedgerContext.LedgerID, cmd.BatchID)
	if err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取导入批次失败")
	}
	if err := requireImportBatchAccess(cmd.LedgerContext, batch); err != nil {
		return nil, err
	}
	if !isReadyForReclassify(batch, time.Now()) {
		return nil, reclassifyConflict("仅有效的待确认导入批次可以重新分类")
	}

	classificationContext, err := s.repo.LoadClassificationContext(ctx, batch.LedgerID)
	if err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取分类规则失败")
	}

	projected := clonePreviewRows(batch.Rows)
	result := &ReclassifyResult{DryRun: cmd.DryRun, Changes: []ReclassifyRowChange{}}
	changedRows := make([]PreviewRow, 0)
	for index := range projected {
		row := &projected[index]
		if row.Classification.Source == string(classifier.SourceBulk) || row.Classification.Status == ClassificationStatusBulk {
			result.ProtectedBulkRows++
			continue
		}
		if row.Classification.Source == string(classifier.SourceManual) || row.Classification.Status == ClassificationStatusManual || row.RowStatus == RowStatusAdjusted {
			result.ProtectedManualRows++
			continue
		}

		classificationResult := classifier.Classify(classificationContext, classifierRow(batch, *row))
		if !classificationResult.Evaluated {
			continue
		}
		result.EligibleRows++
		before := *row
		applyClassificationResult(row, classificationResult, s.classificationMode)
		if classificationRowsEqual(before, *row) {
			result.UnchangedRows++
			continue
		}
		result.ChangedRows++
		result.Changes = append(result.Changes, classificationChange(before, *row))
		changedRows = append(changedRows, *row)
	}
	reclassified := &PreviewBatch{Rows: projected}
	recountClassificationSummary(reclassified)
	result.Summary = reclassified.ClassificationSummary
	result.ConflictRows = result.Summary.Conflict

	if !cmd.DryRun && len(changedRows) > 0 {
		if err := s.repo.ApplyReclassification(ctx, cmd.LedgerContext, batch.ID, batch.UpdatedAt, changedRows, result); err != nil {
			if errors.Is(err, errReclassifyBatchChanged) {
				return nil, reclassifyConflict("导入批次状态已变化，请刷新后重试")
			}
			return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "保存重新分类结果失败")
		}
	}
	return result, nil
}

func isReadyForReclassify(batch *PreviewBatch, now time.Time) bool {
	if batch.Status != batchStatusReady {
		return false
	}
	if batch.ExpiresAt == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339, batch.ExpiresAt)
	return err == nil && expiresAt.After(now)
}

func classificationRowsEqual(left PreviewRow, right PreviewRow) bool {
	return left.SuggestedCategoryID == right.SuggestedCategoryID &&
		left.SuggestedAccountID == right.SuggestedAccountID &&
		reflect.DeepEqual(normalizedStrings(left.SuggestedTagIDs), normalizedStrings(right.SuggestedTagIDs)) &&
		left.SuggestedRuleID == right.SuggestedRuleID &&
		left.SuggestionReason == right.SuggestionReason &&
		left.SelectedCategoryID == right.SelectedCategoryID &&
		left.SelectedAccountID == right.SelectedAccountID &&
		reflect.DeepEqual(normalizedStrings(left.SelectedTagIDs), normalizedStrings(right.SelectedTagIDs)) &&
		reflect.DeepEqual(left.Classification, right.Classification)
}

func classificationChange(before PreviewRow, after PreviewRow) ReclassifyRowChange {
	return ReclassifyRowChange{
		RowID:         before.ID,
		OldStatus:     before.Classification.Status,
		NewStatus:     after.Classification.Status,
		OldCategoryID: effectiveCategoryID(before),
		NewCategoryID: effectiveCategoryID(after),
		OldTagIDs:     effectiveTagIDs(before),
		NewTagIDs:     effectiveTagIDs(after),
	}
}

func effectiveCategoryID(row PreviewRow) string {
	if row.SelectedCategoryID != "" {
		return row.SelectedCategoryID
	}
	return row.SuggestedCategoryID
}

func effectiveTagIDs(row PreviewRow) []string {
	if len(row.SelectedTagIDs) > 0 {
		return copyStrings(row.SelectedTagIDs)
	}
	return copyStrings(row.SuggestedTagIDs)
}

func clonePreviewRows(rows []PreviewRow) []PreviewRow {
	result := make([]PreviewRow, len(rows))
	for index, row := range rows {
		result[index] = row
		result[index].SuggestedTagIDs = copyStrings(row.SuggestedTagIDs)
		result[index].SelectedTagIDs = copyStrings(row.SelectedTagIDs)
		result[index].Classification.MatchedRuleIDs = copyStrings(row.Classification.MatchedRuleIDs)
		result[index].Classification.SuggestedTagIDs = copyStrings(row.Classification.SuggestedTagIDs)
	}
	return result
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func normalizedStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func reclassifyConflict(message string) *appErrors.AppError {
	return appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeImportReclassifyConflict, message)
}
