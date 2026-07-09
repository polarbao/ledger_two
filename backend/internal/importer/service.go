package importer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
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
