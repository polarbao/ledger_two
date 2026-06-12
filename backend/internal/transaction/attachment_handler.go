package transaction

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

// HandleUploadAttachment 处理附件上传请求
func (h *Handler) HandleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserIDFromContext(r.Context())
	if currentUserID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录系统")
		return
	}

	// 限制读取最大 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)
	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "文件大小不能超过 10MB")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "获取上传文件失败")
		return
	}
	defer file.Close()

	// 校验扩展名
	filename := header.Filename
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "仅支持上传 jpg, jpeg, png, webp 格式的图片")
		return
	}

	// 校验 MIME 类型
	head := make([]byte, 512)
	n, _ := file.Read(head)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "文件处理失败")
		return
	}

	mime := http.DetectContentType(head[:n])
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "非法文件类型，只允许图片格式")
		return
	}

	// 保存物理文件
	newFilename := uuid.NewString() + ext
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "创建上传目录失败")
		return
	}

	outPath := filepath.Join(uploadDir, newFilename)
	out, err := os.Create(outPath)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "创建物理文件失败")
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "写入物理文件失败")
		return
	}

	path := "/uploads/" + newFilename
	response.JSON(w, http.StatusOK, map[string]string{
		"path": path,
	})
}
