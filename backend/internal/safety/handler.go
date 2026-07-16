package safety

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	backup, err := h.service.ManualBackup(r.Context(), currentUserID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, backup)
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
	if err := decodeSafetyJSON(r, &req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "无效的请求格式"))
		return
	}
	if strings.TrimSpace(req.Filename) == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "备份文件名不能为空"))
		return
	}

	preparation, err := h.service.RestoreBackup(r.Context(), currentUserID, req.Filename)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, preparation)
}

// HandleGetBackups 处理获取备份列表请求 GET /api/admin/backups
func (h *Handler) HandleGetBackups(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}
	list, err := h.service.ListBackups(r.Context(), currentUserID)
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

	download, err := h.service.OpenBackupDownload(r.Context(), currentUserID, filename)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	defer download.File.Close()

	// 格式化输出文件流
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(download.Filename)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", download.SizeBytes))

	http.ServeContent(w, r, filepath.Base(download.Filename), download.ModifiedAt, download.File)
}

func decodeSafetyJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体只能包含一个 JSON 对象")
	}
	return nil
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
