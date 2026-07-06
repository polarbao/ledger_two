package metadata

import (
	"encoding/json"
	"net/http"

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
