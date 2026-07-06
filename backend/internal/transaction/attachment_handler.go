package transaction

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

// HandleUploadAttachment 处理附件上传请求
func (h *Handler) HandleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	// 限制读取最大 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)
	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		writeValidationError(w, "文件大小不能超过 10MB")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeValidationError(w, "获取上传文件失败")
		return
	}
	defer file.Close()

	// 校验扩展名
	filename := header.Filename
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		writeValidationError(w, "仅支持上传 jpg, jpeg, png, webp 格式的图片")
		return
	}

	// 校验 MIME 类型
	head := make([]byte, 512)
	n, _ := file.Read(head)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeInternalError(w, "文件处理失败")
		return
	}

	mime := http.DetectContentType(head[:n])
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		writeValidationError(w, "非法文件类型，只允许图片格式")
		return
	}

	// 保存物理文件
	newFilename := uuid.NewString() + ext
	uploadDir := h.uploadDir
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		writeInternalError(w, "创建上传目录失败")
		return
	}

	outPath := filepath.Join(uploadDir, newFilename)
	out, err := os.Create(outPath)
	if err != nil {
		writeInternalError(w, "创建物理文件失败")
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		writeInternalError(w, "写入物理文件失败")
		return
	}

	path := "/uploads/" + newFilename
	response.JSON(w, http.StatusOK, map[string]string{
		"path": path,
	})
}

func (h *Handler) HandleGetAttachment(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		writeUnauthorized(w)
		return
	}

	filename := strings.TrimSpace(chi.URLParam(r, "filename"))
	if filename == "" || filename != filepath.Base(filename) || strings.Contains(filename, "..") {
		writeValidationError(w, "附件文件名不合法")
		return
	}

	attachmentPath := "/uploads/" + filename
	if err := h.service.CanViewAttachment(r.Context(), currentUserID, attachmentPath); err != nil {
		response.WriteError(w, err)
		return
	}

	http.ServeFile(w, r, filepath.Join(h.uploadDir, filename))
}
