package ledger

import (
	"encoding/json"
	"net/http"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateLedger(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	var req CreateLedgerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}

	l, err := h.svc.CreateLedger(r.Context(), userID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, l)
}

func (h *Handler) ListUserLedgers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	list, err := h.svc.ListUserLedgers(r.Context(), userID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, list)
}

func (h *Handler) GetLedgerMembers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	ledgerID := r.PathValue("id")
	if ledgerID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "缺少账本 ID"))
		return
	}

	members, err := h.svc.GetLedgerMembers(r.Context(), userID, ledgerID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, members)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	ledgerID := r.PathValue("id")
	if userID == "" || ledgerID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "参数不完整"))
		return
	}

	var req AddMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}

	if err := h.svc.AddMember(r.Context(), userID, ledgerID, req); err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, nil)
}

func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	ledgerID := r.PathValue("id")
	targetUserID := r.PathValue("userId")
	if userID == "" || ledgerID == "" || targetUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "参数不完整"))
		return
	}

	var req UpdateMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}

	if err := h.svc.UpdateMemberRole(r.Context(), userID, ledgerID, targetUserID, req); err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, nil)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	ledgerID := r.PathValue("id")
	targetUserID := r.PathValue("userId")
	if userID == "" || ledgerID == "" || targetUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "参数不完整"))
		return
	}

	if err := h.svc.RemoveMember(r.Context(), userID, ledgerID, targetUserID); err != nil {
		response.WriteError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, nil)
}
