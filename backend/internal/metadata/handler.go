package metadata

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	kind, ok := ParseKind(chi.URLParam(r, "kind"))
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "不支持的元数据类型"))
		return
	}
	includeArchived := r.URL.Query().Get("include_archived") == "true"

	items, err := h.service.List(r.Context(), userID, kind, includeArchived)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) GetDefaultProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	profile, err := h.service.GetDefaultProfile(r.Context(), userID, strings.TrimSpace(r.URL.Query().Get("profile")))
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, profile)
}

func (h *Handler) PreviewDefaultProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	var req ProfilePreviewRequest
	if err := decodeProfileJSON(r, &req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}
	if strings.TrimSpace(req.Profile) == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "默认元数据模板不能为空"))
		return
	}
	result, err := h.service.PreviewDefaultProfile(r.Context(), userID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) ApplyDefaultProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	var req ProfileApplyRequest
	if err := decodeProfileJSON(r, &req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}
	if strings.TrimSpace(req.Profile) == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "默认元数据模板不能为空"))
		return
	}
	result, err := h.service.ApplyDefaultProfile(r.Context(), userID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func decodeProfileJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体只能包含一个 JSON 对象")
	}
	return nil
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	kind, ok := ParseKind(chi.URLParam(r, "kind"))
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "不支持的元数据类型"))
		return
	}
	var req UpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}

	item, err := h.service.Create(r.Context(), userID, kind, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	kind, ok := ParseKind(chi.URLParam(r, "kind"))
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "不支持的元数据类型"))
		return
	}
	var req UpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}

	if err := h.service.Update(r.Context(), userID, kind, chi.URLParam(r, "id"), req); err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	h.setArchived(w, r, true)
}

func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	h.setArchived(w, r, false)
}

func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	kind, ok := ParseKind(chi.URLParam(r, "kind"))
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "不支持的元数据类型"))
		return
	}
	var req ReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}
	if err := h.service.Reorder(r.Context(), userID, kind, req); err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) setArchived(w http.ResponseWriter, r *http.Request, archived bool) {
	userID := middleware.GetUserIDFromContext(r.Context())
	kind, ok := ParseKind(chi.URLParam(r, "kind"))
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "不支持的元数据类型"))
		return
	}
	if err := h.service.SetArchived(r.Context(), userID, kind, chi.URLParam(r, "id"), archived); err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}
