package importer

import (
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

	batch, err := h.service.PreviewCSV(r.Context(), PreviewFileRequest{
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
