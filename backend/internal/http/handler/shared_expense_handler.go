package handler

import (
	"encoding/json"
	"net/http"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/service"
)

type SharedExpenseHandler struct {
	svc *service.SharedExpenseService
}

func NewSharedExpenseHandler(svc *service.SharedExpenseService) *SharedExpenseHandler {
	return &SharedExpenseHandler{svc: svc}
}

func (h *SharedExpenseHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	var req service.CreateSharedExpenseReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}

	ledgerID, err := h.svc.GetUserLedgerID(r.Context(), userID)
	if err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "获取系统账本失败"))
		return
	}

	id, err := h.svc.Create(r.Context(), ledgerID, userID, req)
	if err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, err.Error()))
		return
	}

	response.JSON(w, http.StatusCreated, map[string]string{
		"id": id,
	})
}

// 占位符接口，为了满足后续列表等规范要求
func (h *SharedExpenseHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, []interface{}{})
}

func (h *SharedExpenseHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]interface{}{})
}
