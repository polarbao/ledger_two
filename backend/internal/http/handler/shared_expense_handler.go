package handler

import (
	"encoding/json"
	"net/http"

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
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req service.CreateSharedExpenseReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	ledgerID, err := h.svc.GetUserLedgerID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get ledger", http.StatusInternalServerError)
		return
	}

	id, err := h.svc.Create(r.Context(), ledgerID, userID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
