package importer

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/ledger"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) HandlePreview(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}
	lc, ok := ledger.LedgerContextFromContext(r.Context())
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "缺少账本上下文"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeImportFileInvalid, "上传的文件大小超过 2MB 限制"))
		return
	}

	sourceType := strings.TrimSpace(r.FormValue("source_type"))
	file, header, err := r.FormFile("file")
	if err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeImportFileInvalid, "获取上传文件失败，请提供 file 字段"))
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeImportFileInvalid, "读取文件数据失败"))
		return
	}

	batch, err := h.service.PreviewFile(r.Context(), PreviewFileRequest{
		LedgerContext: lc,
		Filename:      header.Filename,
		SourceType:    sourceType,
		Content:       content,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, batch)
}

func (h *Handler) HandleGetBatch(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}
	lc, ok := ledger.LedgerContextFromContext(r.Context())
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "缺少账本上下文"))
		return
	}

	batch, err := h.service.GetPreviewBatch(r.Context(), lc, chi.URLParam(r, "batchID"))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, batch)
}

func (h *Handler) HandleUpdateRow(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}
	lc, ok := ledger.LedgerContextFromContext(r.Context())
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "缺少账本上下文"))
		return
	}

	var patch UpdateRowRequest
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体格式无效"))
		return
	}

	batch, err := h.service.UpdatePreviewRow(r.Context(), UpdateRowCommand{
		LedgerContext: lc,
		BatchID:       chi.URLParam(r, "batchID"),
		RowID:         chi.URLParam(r, "rowID"),
		Patch:         patch,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, batch)
}

func (h *Handler) HandleCommit(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}
	lc, ok := ledger.LedgerContextFromContext(r.Context())
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "缺少账本上下文"))
		return
	}

	result, err := h.service.CommitPreviewBatch(r.Context(), lc, chi.URLParam(r, "batchID"))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) HandleReclassify(w http.ResponseWriter, r *http.Request) {
	lc, ok := h.requireLedgerContext(w, r)
	if !ok {
		return
	}

	req := ReclassifyRequest{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil && err != io.EOF {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体格式无效"))
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体只能包含一个 JSON 对象"))
		return
	}
	dryRun := true
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}

	result, err := h.service.ReclassifyPreviewBatch(r.Context(), ReclassifyCommand{
		LedgerContext: lc,
		BatchID:       chi.URLParam(r, "batchID"),
		DryRun:        dryRun,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) HandleBulkAdjust(w http.ResponseWriter, r *http.Request) {
	lc, ok := h.requireLedgerContext(w, r)
	if !ok {
		return
	}

	var req BulkClassificationRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体格式无效"))
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体只能包含一个 JSON 对象"))
		return
	}

	result, err := h.service.BulkAdjustPreviewRows(r.Context(), BulkAdjustCommand{
		LedgerContext: lc,
		BatchID:       chi.URLParam(r, "batchID"),
		Request:       req,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) HandleDiscardBatch(w http.ResponseWriter, r *http.Request) {
	lc, ok := h.requireLedgerContext(w, r)
	if !ok {
		return
	}
	var req DiscardImportBatchRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体格式无效"))
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体只能包含一个 JSON 对象"))
		return
	}

	result, err := h.service.DiscardPreviewBatch(r.Context(), lc, chi.URLParam(r, "batchID"), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) HandleCreateRule(w http.ResponseWriter, r *http.Request) {
	lc, ok := h.requireLedgerContext(w, r)
	if !ok {
		return
	}

	var req ImportRuleUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体格式无效"))
		return
	}
	rule, err := h.service.CreateImportRule(r.Context(), lc, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, rule)
}

func (h *Handler) HandleUpdateRule(w http.ResponseWriter, r *http.Request) {
	lc, ok := h.requireLedgerContext(w, r)
	if !ok {
		return
	}

	var req ImportRuleUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体格式无效"))
		return
	}
	rule, err := h.service.UpdateImportRule(r.Context(), lc, chi.URLParam(r, "ruleID"), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, rule)
}

func (h *Handler) HandleListRules(w http.ResponseWriter, r *http.Request) {
	lc, ok := h.requireLedgerContext(w, r)
	if !ok {
		return
	}

	rules, err := h.service.ListImportRules(r.Context(), lc, strings.TrimSpace(r.URL.Query().Get("status")))
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, rules)
}

func (h *Handler) HandleArchiveRule(w http.ResponseWriter, r *http.Request) {
	lc, ok := h.requireLedgerContext(w, r)
	if !ok {
		return
	}
	rule, err := h.service.ArchiveImportRule(r.Context(), lc, chi.URLParam(r, "ruleID"))
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, rule)
}

func (h *Handler) HandleRestoreRule(w http.ResponseWriter, r *http.Request) {
	lc, ok := h.requireLedgerContext(w, r)
	if !ok {
		return
	}
	rule, err := h.service.RestoreImportRule(r.Context(), lc, chi.URLParam(r, "ruleID"))
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, rule)
}

func (h *Handler) requireLedgerContext(w http.ResponseWriter, r *http.Request) (ledger.LedgerContext, bool) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return ledger.LedgerContext{}, false
	}
	lc, ok := ledger.LedgerContextFromContext(r.Context())
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "缺少账本上下文"))
		return ledger.LedgerContext{}, false
	}
	return lc, true
}
