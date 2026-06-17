package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	// 获取路由器实例（healthz 测试无需真实 DB 与 Cfg）
	r := New(nil, nil)

	// 构造测试请求
	req, err := http.NewRequest("GET", "/api/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 初始化 ResponseRecorder
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// 检查状态码
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// 检查返回体结构是否符合文档约定
	var resp struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if !resp.Success || resp.Data["status"] != "ok" {
		t.Errorf("handler returned unexpected body: %v", rr.Body.String())
	}
}
