package safety

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

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

// HandleManualBackup 处理手动备份请求 POST /api/admin/backup
func (h *Handler) HandleManualBackup(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	filename, err := h.service.ManualBackup(r.Context(), currentUserID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"filename": filename,
	})
}

// HandleRestoreBackup 处理恢复备份请求 POST /api/admin/restore
func (h *Handler) HandleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "无效的请求格式"))
		return
	}
	if req.Filename == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "备份文件名不能为空"))
		return
	}

	instructions, err := h.service.RestoreBackup(r.Context(), currentUserID, req.Filename)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"instructions": instructions,
	})
}

// HandleGetBackups 处理获取备份列表请求 GET /api/admin/backups
func (h *Handler) HandleGetBackups(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}
	if err := h.service.requireLedgerOwner(r.Context(), currentUserID); err != nil {
		response.WriteError(w, err)
		return
	}

	list, err := h.service.GetBackups(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, list)
}

// HandleDiagnostics 处理系统诊断请求 GET /api/admin/diagnostics
func (h *Handler) HandleDiagnostics(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	diagnostics, err := h.service.Diagnostics(r.Context(), currentUserID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, diagnostics)
}

// HandleDownloadBackup 处理备份文件下载 GET /api/admin/backups/* (或含 filename 路由参数)
func (h *Handler) HandleDownloadBackup(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}
	if err := h.service.requireLedgerOwner(r.Context(), currentUserID); err != nil {
		response.WriteError(w, err)
		return
	}

	// 兼容通配符或者 URLParam 获取 filename 路由参数
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		// 容错：如果 filename 是从通配符中拿，例如 chi 路由中用 backups/*，可以用 chi.URLParam(r, "*")
		filename = chi.URLParam(r, "*")
	}
	// 去除首尾的斜杠
	filename = strings.Trim(filename, "/")

	if filename == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "备份文件名不能为空"))
		return
	}

	backupDir := filepath.Clean(h.service.cfg.BackupDir)
	targetPath := filepath.Clean(filepath.Join(backupDir, filename))

	// 安全加固 1: 路径穿越攻击防御
	if !strings.HasPrefix(targetPath, backupDir) {
		response.WriteError(w, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "无权访问该路径下的物理文件"))
		return
	}

	// 安全加固 2: 仅允许下载 .db 文件
	if !strings.HasSuffix(strings.ToLower(targetPath), ".db") {
		response.WriteError(w, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅允许下载数据库备份文件"))
		return
	}

	// 检查物理文件是否存在
	fi, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			response.WriteError(w, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeBackupNotFound, "备份文件不存在"))
			return
		}
		response.WriteError(w, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取备份文件元数据失败"))
		return
	}

	file, err := os.Open(targetPath)
	if err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "打开备份文件失败"))
		return
	}
	defer file.Close()

	// 格式化输出文件流
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(filename)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))

	http.ServeContent(w, r, filepath.Base(filename), fi.ModTime(), file)
}

// HandleExportCSV 处理交易流水导出 CSV 请求 GET /api/export/transactions.csv
func (h *Handler) HandleExportCSV(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	month := r.URL.Query().Get("month")
	csvBytes, err := h.service.ExportCSV(r.Context(), currentUserID, month)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=transactions.csv")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(csvBytes)))

	_, _ = w.Write(csvBytes)
}

// HandleExportJSON 处理全量脱敏数据导出 JSON 请求 GET /api/export/full.json
func (h *Handler) HandleExportJSON(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	jsonBytes, err := h.service.ExportJSON(r.Context(), currentUserID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=full.json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))

	_, _ = w.Write(jsonBytes)
}
