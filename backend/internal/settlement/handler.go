package settlement

import (
	"encoding/json"
	"errors"
	"net/http"

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
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
		return
	}

	res, err := h.service.GetBalance(r.Context())
	if err != nil {
		h.handleError(w, err)
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
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
		return
	}

	var req CreateSettlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "解析请求参数失败")
		return
	}

	res, err := h.service.CreateSettlement(r.Context(), currentUserID, req)
	if err != nil {
		h.handleError(w, err)
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
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
		return
	}

	month := r.URL.Query().Get("month")
	res, err := h.service.List(r.Context(), month)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// 统一解析结算模块 AppError 和系统级内部报错的转换器
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		response.Error(w, appErr.Status, appErr.Code, appErr.Message)
		return
	}

	// 兜底记录内部报错并返回 500
	response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "内部服务错误，请重试")
}
