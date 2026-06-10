package dashboard

import (
	"net/http"

	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

// Handler 首页 Dashboard 统计 HTTP 端点控制器
// @brief 提供当月账务数据多维聚合与统计图表查询端点
type Handler struct {
	service *Service
}

// NewHandler 实例化 Handler
// @brief 创建 Dashboard Handler 控制器实例
// @param service *Service 业务服务句柄
// @return *Handler 控制器实例
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleGetDashboard 查询当月 Dashboard 汇总统计数据接口
// @brief 处理 GET /api/dashboard 聚合统计请求
// @param w http.ResponseWriter 响应句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleGetDashboard(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
		return
	}

	month := r.URL.Query().Get("month")
	res, err := h.service.GetDashboardData(r.Context(), currentUserID, month)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

