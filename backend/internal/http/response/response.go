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
