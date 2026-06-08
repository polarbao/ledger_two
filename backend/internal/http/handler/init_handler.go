package handler

import (
	"encoding/json"
	"net/http"

	"ledger_two/internal/http/response"
	"ledger_two/internal/service"
)

type InitHandler struct {
	svc *service.InitService
}

func NewInitHandler(svc *service.InitService) *InitHandler {
	return &InitHandler{svc: svc}
}

func (h *InitHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	isInit, err := h.svc.CheckStatus(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"initialized": isInit,
	})
}

func (h *InitHandler) HandleSetup(w http.ResponseWriter, r *http.Request) {
	var req service.SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	err := h.svc.RunSetup(r.Context(), req)
	if err != nil {
		// 拦截二次初始化异常，准确返回状态码 409
		if err == service.ErrAlreadyInitialized {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "setup completed successfully",
	})
}
