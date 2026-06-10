package response

import (
	"encoding/json"
	"errors"
	"net/http"

	appErrors "ledger_two/internal/errors"
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

type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}

type ErrorResponse struct {
	Success bool     `json:"success"`
	Error   APIError `json:"error"`
}

// Error 统一的错误 HTTP JSON 返回体格式化工具
// @brief 统一输出失败 of HTTP JSON 响应
// @param w http.ResponseWriter 响应写入器
// @param status int HTTP 状态码
// @param code string 错误标识码
// @param message string 错误解释信息
func Error(w http.ResponseWriter, status int, code string, message string) {
	ErrorWithDetails(w, status, code, message, nil)
}

// ErrorWithDetails 支持 Details 字段的错误输出
func ErrorWithDetails(w http.ResponseWriter, status int, code string, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// WriteError 将任意 Go 错误写入 http 响应中。
// 如果是 AppError，则自动解包成对应的状态码和错误响应；
// 如果是普通的错误，则作为 INTERNAL_ERROR 500 处理，且不把敏感底层细节返回给前端。
func WriteError(w http.ResponseWriter, err error) {
	var appErr *appErrors.AppError
	if errors.As(err, &appErr) {
		ErrorWithDetails(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
		return
	}
	Error(w, http.StatusInternalServerError, appErrors.ErrCodeInternalError, "内部服务错误，请重试")
}



