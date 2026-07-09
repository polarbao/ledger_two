package importer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/ledger"
)

const MaxPreviewRows = 500

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type PreviewFileRequest struct {
	LedgerContext ledger.LedgerContext
	Filename      string
	SourceType    string
	Content       []byte
}

func (s *Service) PreviewCSV(ctx context.Context, req PreviewFileRequest) (*PreviewBatch, error) {
	if req.LedgerContext.Role != ledger.RoleOwner {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅账本 Owner 可导入账单")
	}
	if !isSupportedSourceType(req.SourceType) {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "不支持的导入来源")
	}
	if len(req.Content) == 0 {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeImportFileInvalid, "导入文件不能为空")
	}

	preview, err := ParseCSV(req.SourceType, bytes.NewReader(req.Content))
	if err != nil {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeImportFileInvalid, "解析 CSV 失败")
	}
	if len(preview.Rows) > MaxPreviewRows {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, fmt.Sprintf("单批导入最多支持 %d 行", MaxPreviewRows))
	}
	if err := s.applyImportRules(ctx, req.LedgerContext.LedgerID, preview.Rows); err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "应用导入规则失败")
	}

	batch := buildPreviewBatch(req, preview.Rows)
	if err := s.applyExistingDuplicates(ctx, batch); err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "分析导入重复数据失败")
	}
	recountBatch(batch)

	if err := s.repo.CreatePreviewBatch(ctx, batch); err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "保存导入预览失败")
	}

	return s.repo.GetPreviewBatch(ctx, req.LedgerContext.LedgerID, batch.ID)
}

func (s *Service) GetPreviewBatch(ctx context.Context, lc ledger.LedgerContext, batchID string) (*PreviewBatch, error) {
	if batchID == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入批次 ID 不能为空")
	}
	batch, err := s.repo.GetPreviewBatch(ctx, lc.LedgerID, batchID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeNotFound, "导入批次不存在")
		}
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取导入批次失败")
	}
	return batch, nil
}

type UpdateRowCommand struct {
	LedgerContext ledger.LedgerContext
	BatchID       string
	RowID         string
	Patch         UpdateRowRequest
}

func (s *Service) UpdatePreviewRow(ctx context.Context, cmd UpdateRowCommand) (*PreviewBatch, error) {
	if cmd.LedgerContext.Role != ledger.RoleOwner {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅账本 Owner 可调整导入预览")
	}
	if cmd.BatchID == "" || cmd.RowID == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入批次 ID 和行 ID 不能为空")
	}

	batch, row, err := s.repo.GetPreviewRow(ctx, cmd.LedgerContext.LedgerID, cmd.BatchID, cmd.RowID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeNotFound, "导入预览行不存在")
		}
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取导入预览行失败")
	}
	if batch.Status == "committed" {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeConflict, "已提交的导入批次不可调整")
	}

	updated := *row
	adjustment := RowAdjustment{}
	if cmd.Patch.TargetTransactionType != nil {
		if !isValidTargetTransactionType(*cmd.Patch.TargetTransactionType) {
			return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入目标类型无效")
		}
		updated.TargetTransactionType = *cmd.Patch.TargetTransactionType
		adjustment.TargetTransactionType = *cmd.Patch.TargetTransactionType
	}
	if cmd.Patch.RowStatus != nil {
		if !isValidMutableRowStatus(*cmd.Patch.RowStatus) {
			return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入行状态无效")
		}
		updated.RowStatus = *cmd.Patch.RowStatus
		adjustment.RowStatus = *cmd.Patch.RowStatus
	}
	if cmd.Patch.SelectedCategoryID != nil {
		updated.SelectedCategoryID = *cmd.Patch.SelectedCategoryID
		adjustment.SelectedCategoryID = *cmd.Patch.SelectedCategoryID
	}
	if cmd.Patch.SelectedAccountID != nil {
		updated.SelectedAccountID = *cmd.Patch.SelectedAccountID
		adjustment.SelectedAccountID = *cmd.Patch.SelectedAccountID
	}
	if cmd.Patch.SelectedTagIDs != nil {
		updated.SelectedTagIDs = cmd.Patch.SelectedTagIDs
		adjustment.SelectedTagIDs = cmd.Patch.SelectedTagIDs
	}
	if cmd.Patch.Visibility != nil {
		if !isValidVisibility(*cmd.Patch.Visibility) {
			return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入行可见性无效")
		}
		updated.Visibility = *cmd.Patch.Visibility
		adjustment.Visibility = *cmd.Patch.Visibility
	}

	if updated.DuplicateStatus == DuplicateStatusInvalid && updated.RowStatus != RowStatusSkipped {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "无效导入行只能跳过，需修正字段后才能导入")
	}
	if updated.RowStatus == RowStatusSkipped {
		updated.TargetTransactionType = TargetTransactionSkipped
		adjustment.TargetTransactionType = TargetTransactionSkipped
	}
	if updated.RowStatus == RowStatusAdjusted && updated.TargetTransactionType == TargetTransactionSkipped {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "调整为待导入时必须选择有效的目标类型")
	}

	for i := range batch.Rows {
		if batch.Rows[i].ID == updated.ID {
			batch.Rows[i] = updated
			break
		}
	}

	updatedBatch, err := s.repo.UpdatePreviewRow(ctx, batch, updated, adjustment)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeNotFound, "导入预览行不存在")
		}
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "保存导入预览行失败")
	}
	return updatedBatch, nil
}

func (s *Service) CommitPreviewBatch(ctx context.Context, lc ledger.LedgerContext, batchID string) (*CommitResult, error) {
	if lc.Role != ledger.RoleOwner {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅账本 Owner 可提交导入批次")
	}
	if batchID == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入批次 ID 不能为空")
	}

	batch, err := s.repo.GetPreviewBatch(ctx, lc.LedgerID, batchID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeImportBatchNotFound, "导入批次不存在")
		}
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取导入批次失败")
	}
	if batch.Status != "ready" {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeImportCommitConflict, "当前导入批次状态不允许提交")
	}
	if err := validateRowsForCommit(batch.Rows); err != nil {
		return nil, err
	}

	result, err := s.repo.CommitPreviewBatch(ctx, lc, batch)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeImportBatchNotFound, "导入批次不存在")
		}
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeImportCommitConflict, "导入提交失败，可能存在重复数据或批次状态变化")
	}
	return result, nil
}

func (s *Service) CreateImportRule(ctx context.Context, lc ledger.LedgerContext, req ImportRuleUpsertRequest) (*ImportRuleResponse, error) {
	if err := s.validateRuleWrite(ctx, lc, &req); err != nil {
		return nil, err
	}
	ruleID := uuid.NewString()
	rule, err := s.repo.CreateImportRule(ctx, lc.LedgerID, lc.UserID, ruleID, req)
	if err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "创建导入规则失败")
	}
	if err := s.repo.CreateImportRuleAudit(ctx, lc.LedgerID, lc.UserID, "import_rule_create", rule.ID, rule); err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "记录导入规则审计失败")
	}
	return rule, nil
}

func (s *Service) UpdateImportRule(ctx context.Context, lc ledger.LedgerContext, ruleID string, req ImportRuleUpsertRequest) (*ImportRuleResponse, error) {
	if ruleID == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则 ID 不能为空")
	}
	if err := s.validateRuleWrite(ctx, lc, &req); err != nil {
		return nil, err
	}
	rule, err := s.repo.UpdateImportRule(ctx, lc.LedgerID, ruleID, req)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeNotFound, "导入规则不存在")
		}
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "更新导入规则失败")
	}
	if err := s.repo.CreateImportRuleAudit(ctx, lc.LedgerID, lc.UserID, "import_rule_update", rule.ID, rule); err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "记录导入规则审计失败")
	}
	return rule, nil
}

func (s *Service) ListImportRules(ctx context.Context, lc ledger.LedgerContext, status string) ([]ImportRuleResponse, error) {
	if !isValidImportRuleStatusFilter(status) {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则状态过滤无效")
	}
	list, err := s.repo.ListImportRules(ctx, lc.LedgerID, status)
	if err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取导入规则失败")
	}
	return list, nil
}

func (s *Service) ArchiveImportRule(ctx context.Context, lc ledger.LedgerContext, ruleID string) (*ImportRuleResponse, error) {
	return s.setImportRuleStatus(ctx, lc, ruleID, "archived", "import_rule_archive")
}

func (s *Service) RestoreImportRule(ctx context.Context, lc ledger.LedgerContext, ruleID string) (*ImportRuleResponse, error) {
	return s.setImportRuleStatus(ctx, lc, ruleID, "active", "import_rule_restore")
}

func (s *Service) setImportRuleStatus(ctx context.Context, lc ledger.LedgerContext, ruleID string, status string, action string) (*ImportRuleResponse, error) {
	if lc.Role != ledger.RoleOwner {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅账本 Owner 可管理导入规则")
	}
	if ruleID == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则 ID 不能为空")
	}
	rule, err := s.repo.SetImportRuleStatus(ctx, lc.LedgerID, ruleID, status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeNotFound, "导入规则不存在")
		}
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "更新导入规则状态失败")
	}
	if err := s.repo.CreateImportRuleAudit(ctx, lc.LedgerID, lc.UserID, action, rule.ID, rule); err != nil {
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "记录导入规则审计失败")
	}
	return rule, nil
}

func (s *Service) validateRuleWrite(ctx context.Context, lc ledger.LedgerContext, req *ImportRuleUpsertRequest) error {
	if lc.Role != ledger.RoleOwner {
		return appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅账本 Owner 可管理导入规则")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.MatchType = strings.TrimSpace(req.MatchType)
	req.Pattern = strings.TrimSpace(req.Pattern)
	req.Result.Visibility = defaultVisibility(req.Result.Visibility)
	if !isValidImportRuleMatchType(req.MatchType) {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则匹配类型无效")
	}
	if req.Pattern == "" {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则匹配内容不能为空")
	}
	if req.Priority != nil && *req.Priority < 0 {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则优先级不能为负数")
	}
	if req.AmountMinCents != nil && *req.AmountMinCents < 0 {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "金额下限不能为负数")
	}
	if req.AmountMaxCents != nil && *req.AmountMaxCents < 0 {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "金额上限不能为负数")
	}
	if req.AmountMinCents != nil && req.AmountMaxCents != nil && *req.AmountMinCents > *req.AmountMaxCents {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "金额下限不能大于上限")
	}
	if !isValidVisibility(req.Result.Visibility) {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则可见性无效")
	}
	if req.Result.CategoryID == "" && req.Result.AccountID == "" && len(req.Result.TagIDs) == 0 {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则至少需要配置分类、账户或标签")
	}
	if err := s.validateRuleMetadata(ctx, lc.LedgerID, req.Result); err != nil {
		return err
	}
	return nil
}

func (s *Service) validateRuleMetadata(ctx context.Context, ledgerID string, result ImportRuleResult) error {
	if result.CategoryID != "" {
		ok, err := s.repo.ActiveMetadataExists(ctx, ledgerID, "categories", result.CategoryID)
		if err != nil {
			return appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "校验导入规则分类失败")
		}
		if !ok {
			return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则分类不存在或已归档")
		}
	}
	if result.AccountID != "" {
		ok, err := s.repo.ActiveMetadataExists(ctx, ledgerID, "accounts", result.AccountID)
		if err != nil {
			return appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "校验导入规则账户失败")
		}
		if !ok {
			return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则账户不存在或已归档")
		}
	}
	for _, tagID := range result.TagIDs {
		tagID = strings.TrimSpace(tagID)
		if tagID == "" {
			continue
		}
		ok, err := s.repo.ActiveMetadataExists(ctx, ledgerID, "tags", tagID)
		if err != nil {
			return appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "校验导入规则标签失败")
		}
		if !ok {
			return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入规则标签不存在或已归档")
		}
	}
	return nil
}

func buildPreviewBatch(req PreviewFileRequest, rows []PreviewRow) *PreviewBatch {
	now := time.Now().Format(time.RFC3339)
	fileHash := sha256.Sum256(req.Content)
	batchID := uuid.NewString()

	normalizedRows := make([]PreviewRow, len(rows))
	for i, row := range rows {
		row.ID = uuid.NewString()
		row.BatchID = batchID
		normalizedRows[i] = row
	}

	batch := &PreviewBatch{
		ID:              batchID,
		LedgerID:        req.LedgerContext.LedgerID,
		SourceType:      req.SourceType,
		Filename:        req.Filename,
		FileSHA256:      hex.EncodeToString(fileHash[:]),
		Status:          "ready",
		CreatedByUserID: req.LedgerContext.UserID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Rows:            normalizedRows,
	}
	recountBatch(batch)
	return batch
}

func (s *Service) applyExistingDuplicates(ctx context.Context, batch *PreviewBatch) error {
	hashes := make([]string, 0, len(batch.Rows))
	for _, row := range batch.Rows {
		if row.DuplicateStatus == DuplicateStatusInvalid {
			continue
		}
		hashes = append(hashes, calculateImportHash(batch.LedgerID, batch.SourceType, row))
	}
	existing, err := s.repo.ExistingImportedHashes(ctx, batch.LedgerID, hashes)
	if err != nil {
		return err
	}
	for i := range batch.Rows {
		hash := calculateImportHash(batch.LedgerID, batch.SourceType, batch.Rows[i])
		if existing[hash] && batch.Rows[i].DuplicateStatus != DuplicateStatusInvalid {
			batch.Rows[i].DuplicateStatus = DuplicateStatusDuplicate
			batch.Rows[i].RowStatus = RowStatusSkipped
			batch.Rows[i].TargetTransactionType = TargetTransactionSkipped
		}
	}
	return nil
}

func (s *Service) applyImportRules(ctx context.Context, ledgerID string, rows []PreviewRow) error {
	rules, err := s.repo.ListImportRules(ctx, ledgerID, "active")
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}

	for i := range rows {
		if rows[i].DuplicateStatus == DuplicateStatusInvalid || rows[i].TargetTransactionType == TargetTransactionSkipped {
			continue
		}
		for _, rule := range rules {
			if !importRuleMatches(rule, rows[i]) {
				continue
			}
			rows[i].SuggestedCategoryID = rule.Result.CategoryID
			rows[i].SuggestedAccountID = rule.Result.AccountID
			rows[i].SuggestedTagIDs = append([]string(nil), rule.Result.TagIDs...)
			rows[i].SuggestedRuleID = rule.ID
			rows[i].SuggestionReason = buildRuleSuggestionReason(rule)
			break
		}
	}
	return nil
}

func importRuleMatches(rule ImportRuleResponse, row PreviewRow) bool {
	pattern := strings.TrimSpace(rule.Pattern)
	if pattern == "" {
		return false
	}
	switch rule.MatchType {
	case "merchant_contains":
		return strings.Contains(row.Merchant, pattern)
	case "description_contains":
		return strings.Contains(row.Description, pattern)
	case "source_account":
		return strings.EqualFold(strings.TrimSpace(row.SourceAccount), pattern)
	case "amount_range":
		if rule.AmountMinCents != nil && row.AmountCents < *rule.AmountMinCents {
			return false
		}
		if rule.AmountMaxCents != nil && row.AmountCents > *rule.AmountMaxCents {
			return false
		}
		return strings.Contains(row.Title, pattern) || strings.Contains(row.Merchant, pattern) || strings.Contains(row.Description, pattern)
	default:
		return false
	}
}

func buildRuleSuggestionReason(rule ImportRuleResponse) string {
	switch rule.MatchType {
	case "merchant_contains":
		return "商户包含「" + rule.Pattern + "」"
	case "description_contains":
		return "描述包含「" + rule.Pattern + "」"
	case "source_account":
		return "来源账户匹配「" + rule.Pattern + "」"
	case "amount_range":
		return "金额区间与文本匹配「" + rule.Pattern + "」"
	default:
		return "命中导入规则「" + rule.Name + "」"
	}
}

func recountBatch(batch *PreviewBatch) {
	batch.TotalRows = len(batch.Rows)
	batch.NewRows = 0
	batch.DuplicateRows = 0
	batch.SuspiciousRows = 0
	batch.InvalidRows = 0
	batch.ImportedRows = 0
	batch.SkippedRows = 0
	batch.FailedRows = 0

	for _, row := range batch.Rows {
		switch row.DuplicateStatus {
		case DuplicateStatusDuplicate:
			batch.DuplicateRows++
		case DuplicateStatusSuspicious:
			batch.SuspiciousRows++
		case DuplicateStatusInvalid:
			batch.InvalidRows++
		default:
			batch.NewRows++
		}
		if row.RowStatus == RowStatusSkipped {
			batch.SkippedRows++
		}
		if row.RowStatus == RowStatusImported {
			batch.ImportedRows++
		}
		if row.RowStatus == RowStatusFailed {
			batch.FailedRows++
		}
	}
}

func validateRowsForCommit(rows []PreviewRow) error {
	for _, row := range rows {
		if row.DuplicateStatus == DuplicateStatusInvalid || row.RowStatus == RowStatusFailed {
			if row.RowStatus == RowStatusSkipped {
				continue
			}
			return appErrors.NewAppErrorWithDetails(http.StatusBadRequest, appErrors.ErrCodeImportRowInvalid, "存在未跳过的无效导入行", map[string]any{
				"row_id":     row.ID,
				"row_number": row.RowNumber,
			})
		}
		if row.RowStatus == RowStatusSkipped || row.TargetTransactionType == TargetTransactionSkipped {
			continue
		}
		if row.DuplicateStatus == DuplicateStatusDuplicate {
			return appErrors.NewAppErrorWithDetails(http.StatusConflict, appErrors.ErrCodeImportDuplicateItem, "重复导入行必须跳过", map[string]any{
				"row_id":     row.ID,
				"row_number": row.RowNumber,
			})
		}
		if row.DuplicateStatus == DuplicateStatusSuspicious && row.RowStatus != RowStatusAdjusted {
			return appErrors.NewAppErrorWithDetails(http.StatusConflict, appErrors.ErrCodeImportRowRequiresConfirmation, "疑似重复导入行必须人工确认导入或跳过", map[string]any{
				"row_id":     row.ID,
				"row_number": row.RowNumber,
			})
		}
		if row.TargetTransactionType != TargetTransactionExpense && row.TargetTransactionType != TargetTransactionIncome {
			return appErrors.NewAppErrorWithDetails(http.StatusBadRequest, appErrors.ErrCodeImportRowInvalid, "导入目标类型暂不支持自动落库", map[string]any{
				"row_id":     row.ID,
				"row_number": row.RowNumber,
			})
		}
		if row.AmountCents <= 0 || row.OccurredAt == "" || row.Title == "" {
			return appErrors.NewAppErrorWithDetails(http.StatusBadRequest, appErrors.ErrCodeImportRowInvalid, "导入行缺少必需字段", map[string]any{
				"row_id":     row.ID,
				"row_number": row.RowNumber,
			})
		}
	}
	return nil
}

func isSupportedSourceType(sourceType string) bool {
	switch sourceType {
	case SourceTypeWechat, SourceTypeAlipay, SourceTypeGeneric:
		return true
	default:
		return false
	}
}

func isValidTargetTransactionType(value string) bool {
	switch value {
	case TargetTransactionExpense, TargetTransactionIncome, TargetTransactionSkipped:
		return true
	default:
		return false
	}
}

func isValidMutableRowStatus(value string) bool {
	switch value {
	case RowStatusPending, RowStatusAdjusted, RowStatusSkipped:
		return true
	default:
		return false
	}
}

func isValidVisibility(value string) bool {
	switch value {
	case "private", "shared", "partner_readable":
		return true
	default:
		return false
	}
}

func isValidImportRuleMatchType(value string) bool {
	switch value {
	case "merchant_contains", "description_contains", "source_account", "amount_range":
		return true
	default:
		return false
	}
}

func isValidImportRuleStatusFilter(value string) bool {
	switch value {
	case "", "active", "archived", "all":
		return true
	default:
		return false
	}
}

func calculateImportHash(ledgerID string, sourceType string, row PreviewRow) string {
	raw := fmt.Sprintf("%s|%s|%s|%d|%s|%s|%s",
		ledgerID,
		sourceType,
		row.OccurredAt,
		row.AmountCents,
		row.Merchant,
		row.Title,
		row.ExternalOrderID,
	)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
