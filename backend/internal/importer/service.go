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
	}
}

func isSupportedSourceType(sourceType string) bool {
	switch sourceType {
	case SourceTypeWechat, SourceTypeAlipay, SourceTypeGeneric:
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
