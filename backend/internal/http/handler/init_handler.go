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
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "检查初始化状态失败")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"initialized": isInit,
	})
}

func (h *InitHandler) HandleSetup(w http.ResponseWriter, r *http.Request) {
	var req service.SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "请求参数解析失败")
		return
	}

	err := h.svc.RunSetup(r.Context(), req)
	if err != nil {
		// 拦截二次初始化异常，准确返回状态码 409
		if err == service.ErrAlreadyInitialized {
			response.Error(w, http.StatusConflict, "ALREADY_INITIALIZED", "账本已经初始化，无法重复设置")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "初始化失败，请重试")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "setup completed successfully",
	})
}
