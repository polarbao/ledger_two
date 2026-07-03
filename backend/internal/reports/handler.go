package reports

import (
	"net/http"
	"time"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// getQueryMonth 提取并校验查询月份，默认当月
func (h *Handler) getQueryMonth(r *http.Request) (string, error) {
	month := r.URL.Query().Get("month")
	if month == "" {
		return time.Now().Format("2006-01"), nil
	}
	_, err := time.Parse("2006-01", month)
	if err != nil {
		return "", err
	}
	return month, nil
}

// HandleGetMonthlySummary 获取当月消费与收入总额汇总
func (h *Handler) HandleGetMonthlySummary(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	month, err := h.getQueryMonth(r)
	if err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "查询月份格式错误，应为 YYYY-MM"))
		return
	}

	res, err := h.service.GetMonthlySummary(r.Context(), currentUserID, month)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleGetCategorySummary 获取当月消费分类汇总
func (h *Handler) HandleGetCategorySummary(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	month, err := h.getQueryMonth(r)
	if err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "查询月份格式错误，应为 YYYY-MM"))
		return
	}

	res, err := h.service.GetCategorySummary(r.Context(), currentUserID, month)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleGetTagSummary 获取当月标签消费汇总
func (h *Handler) HandleGetTagSummary(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	month, err := h.getQueryMonth(r)
	if err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "查询月份格式错误，应为 YYYY-MM"))
		return
	}

	res, err := h.service.GetTagSummary(r.Context(), currentUserID, month)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleGetMemberSummary 获取当月成员支出与分摊结算明细
func (h *Handler) HandleGetMemberSummary(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	month, err := h.getQueryMonth(r)
	if err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "查询月份格式错误，应为 YYYY-MM"))
		return
	}

	res, err := h.service.GetMemberSummary(r.Context(), currentUserID, month)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}
