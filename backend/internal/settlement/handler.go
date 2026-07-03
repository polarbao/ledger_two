package settlement

import (
	"encoding/json"
	"net/http"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

// Handler 结算中心 HTTP 端点控制器
// @brief 提供结算余额查询、新建结算补款及结算历史列表拉取端点
type Handler struct {
	service *Service
}

// NewHandler 实例化 Handler
// @brief 创建 Settlement 的 Handler 控制器实例
// @param service *Service 业务服务句柄
// @return *Handler 控制器实例
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleGetBalance 查询双方结算余额及欠款净额接口
// @brief 处理 GET /api/settlements/balance 请求
// @param w http.ResponseWriter 响应句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	res, err := h.service.GetBalance(r.Context(), currentUserID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleCreate 提交并执行差额结算补款接口
// @brief 处理 POST /api/settlements 请求
// @param w http.ResponseWriter 响应句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	var req CreateSettlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "解析请求参数失败"))
		return
	}

	res, err := h.service.CreateSettlement(r.Context(), currentUserID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, res)
}

// HandleList 查询历史结算明细流水接口
// @brief 处理 GET /api/settlements 请求
// @param w http.ResponseWriter 响应句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	month := r.URL.Query().Get("month")
	res, err := h.service.List(r.Context(), currentUserID, month)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}
