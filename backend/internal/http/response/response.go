package response

import (
	"encoding/json"
	"net/http"
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

// JSON 统一的成功 HTTP JSON 返回体格式化工具
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
	}); err != nil {
		// 这里理论上只有在 response 被破坏时才会报错，作为保障处理
		http.Error(w, "internal encoding error", http.StatusInternalServerError)
	}
}

type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Success bool         `json:"success"`
	Error   ErrorDetails `json:"error"`
}

// Error 统一的错误 HTTP JSON 返回体格式化工具
// @brief 统一输出失败的 HTTP JSON 响应
// @param w http.ResponseWriter 响应写入器
// @param status int HTTP 状态码
// @param code string 错误标识码
// @param message string 错误解释信息
func Error(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Error: ErrorDetails{
			Code:    code,
			Message: message,
		},
	})
}

