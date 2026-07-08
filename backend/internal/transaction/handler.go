package transaction

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

// Handler 交易模块 HTTP 端点控制器
type Handler struct {
	service   *Service
	uploadDir string
}

// NewHandler 实例化 Handler
// @brief 创建 Transaction 的 Handler 控制器实例
// @param service *Service 业务服务句柄
// @return *Handler 控制器实例
func NewHandler(service *Service, uploadDir ...string) *Handler {
	dir := "./uploads"
	if len(uploadDir) > 0 && uploadDir[0] != "" {
		dir = uploadDir[0]
	}
	return &Handler{service: service, uploadDir: dir}
}

// HandleCreate 记账流水接口
// @brief 处理 POST /api/transactions 记一笔普通账单的请求
// @param w http.ResponseWriter 响应写入句柄
// @param r *http.Request 携带请求数据的 HTTP Request
func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "解析请求参数失败")
		return
	}

	res, err := h.service.Create(r.Context(), currentUserID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, res)
}

// HandleGetByID 查询流水详情接口
// @brief 处理 GET /api/transactions/{id} 请求
// @param w http.ResponseWriter 响应句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleGetByID(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "账单 ID 不能为空")
		return
	}

	res, err := h.service.GetByID(r.Context(), currentUserID, id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleUpdate 局部更新账单接口
// @brief 处理 PATCH /api/transactions/{id} 编辑普通账单属性的请求
// @param w http.ResponseWriter 响应句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "账单 ID 不能为空")
		return
	}

	var req UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "解析请求参数失败")
		return
	}

	res, err := h.service.Update(r.Context(), currentUserID, id, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleDelete 软删除账单接口
// @brief 处理 DELETE /api/transactions/{id} 软删除普通账单的请求
// @param w http.ResponseWriter 响应句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "账单 ID 不能为空")
		return
	}

	err := h.service.Delete(r.Context(), currentUserID, id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleList 查询流水列表接口
// @brief 处理 GET /api/transactions 列表过滤查询请求
// @param w http.ResponseWriter 响应句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	var minAmount *int64
	if minStr := q.Get("min_amount"); minStr != "" {
		if val, err := strconv.ParseInt(minStr, 10, 64); err == nil {
			minAmount = &val
		}
	}

	var maxAmount *int64
	if maxStr := q.Get("max_amount"); maxStr != "" {
		if val, err := strconv.ParseInt(maxStr, 10, 64); err == nil {
			maxAmount = &val
		}
	}

	filter := TransactionFilter{
		Month:       q.Get("month"),
		Type:        q.Get("type"),
		CategoryID:  q.Get("category_id"),
		Keyword:     q.Get("keyword"),
		MinAmount:   minAmount,
		MaxAmount:   maxAmount,
		PayerUserID: q.Get("payer_user_id"),
		Visibility:  q.Get("visibility"),
		Tag:         q.Get("tag"),
		Page:        page,
		PageSize:    pageSize,
	}

	res, err := h.service.List(r.Context(), currentUserID, filter)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleCreateSharedExpense 共同支出记账接口
// @brief 处理 POST /api/shared-expenses 记一笔共同支出的请求
// @param w http.ResponseWriter 响应写入句柄
// @param r *http.Request 携带请求数据的 HTTP Request
func (h *Handler) HandleCreateSharedExpense(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	var req CreateSharedExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "解析请求参数失败")
		return
	}

	res, err := h.service.CreateSharedExpense(r.Context(), currentUserID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, res)
}

// HandleGetSharedExpenseByID 获取共同支出详情接口
// @brief 处理 GET /api/shared-expenses/{id} 共同账单详情拉取请求
// @param w http.ResponseWriter 响应写入句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleGetSharedExpenseByID(w http.ResponseWriter, r *http.Request) {
	// 鉴权以及获取逻辑和普通流水详情完全一致，可以直接复用
	h.HandleGetByID(w, r)
}

// HandleUpdateSharedExpense 更新共同支出接口
// @brief 处理 PATCH /api/shared-expenses/{id} 共同账单编辑请求
// @param w http.ResponseWriter 响应写入句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleUpdateSharedExpense(w http.ResponseWriter, r *http.Request) {
	// 编辑的底层逻辑也已经在 service.Update 中完成，可以直接复用
	h.HandleUpdate(w, r)
}

// HandleListCategories 拉取系统分类列表接口
// @brief 处理 GET /api/categories 请求，从 categories 表拉取本账本对应的系统分类
// @param w http.ResponseWriter 响应句柄
// @param r *http.Request 请求句柄
func (h *Handler) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	includeArchived := r.URL.Query().Get("include_archived") == "true"
	res, err := h.service.ListCategories(r.Context(), currentUserID, includeArchived)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleGetTransactionDefault 拉取当前用户快捷记账默认值
// @brief 处理 GET /api/transaction-defaults 请求
func (h *Handler) HandleGetTransactionDefault(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	res, err := h.service.GetTransactionDefault(r.Context(), currentUserID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleCreateTemplate 创建账单模板接口
func (h *Handler) HandleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "解析请求参数失败")
		return
	}

	res, err := h.service.CreateTemplate(r.Context(), currentUserID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, res)
}

// HandleListTemplates 获取模板列表接口
func (h *Handler) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	includeArchived := r.URL.Query().Get("include_archived") == "true"
	res, err := h.service.ListTemplates(r.Context(), currentUserID, includeArchived)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleGetTemplate 获取单个模板详情接口
func (h *Handler) HandleGetTemplate(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "模板 ID 不能为空")
		return
	}

	res, err := h.service.GetTemplate(r.Context(), currentUserID, id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleUpdateTemplate 更新模板接口
func (h *Handler) HandleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "模板 ID 不能为空")
		return
	}

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "解析请求参数失败")
		return
	}

	res, err := h.service.UpdateTemplate(r.Context(), currentUserID, id, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleArchiveTemplate 归档账单模板接口
func (h *Handler) HandleArchiveTemplate(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "模板 ID 不能为空")
		return
	}

	err := h.service.ArchiveTemplate(r.Context(), currentUserID, id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleRestoreTemplate 恢复账单模板接口
func (h *Handler) HandleRestoreTemplate(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "模板 ID 不能为空")
		return
	}

	err := h.service.RestoreTemplate(r.Context(), currentUserID, id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleDeleteTemplate 删除模板接口
func (h *Handler) HandleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "模板 ID 不能为空")
		return
	}

	err := h.service.DeleteTemplate(r.Context(), currentUserID, id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleCreateRecurringRule 创建周期账单规则接口
func (h *Handler) HandleCreateRecurringRule(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	var req CreateRecurringRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "解析请求参数失败")
		return
	}

	res, err := h.service.CreateRecurringRule(r.Context(), currentUserID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, res)
}

// HandleListRecurringRules 获取周期账单规则列表接口
func (h *Handler) HandleListRecurringRules(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	res, err := h.service.ListRecurringRules(r.Context(), currentUserID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleDeleteRecurringRule 删除周期账单规则接口
func (h *Handler) HandleDeleteRecurringRule(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "规则 ID 不能为空")
		return
	}

	err := h.service.DeleteRecurringRule(r.Context(), currentUserID, id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleListRecurringReminders 获取周期提醒列表接口（拉取时底层会自动触发过期提醒生成）
func (h *Handler) HandleListRecurringReminders(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	res, err := h.service.ListRecurringReminders(r.Context(), currentUserID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// HandleConfirmReminder 确认到期提醒接口
func (h *Handler) HandleConfirmReminder(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "提醒 ID 不能为空")
		return
	}

	err := h.service.ConfirmReminder(r.Context(), currentUserID, id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleIgnoreReminder 忽略到期提醒接口
func (h *Handler) HandleIgnoreReminder(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeValidationError(w, "提醒 ID 不能为空")
		return
	}

	err := h.service.IgnoreReminder(r.Context(), currentUserID, id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleBatchTag 批量打标签接口
func (h *Handler) HandleBatchTag(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	var req BatchTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "解析请求参数失败")
		return
	}

	err := h.service.BatchTag(r.Context(), currentUserID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}
