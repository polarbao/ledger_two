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

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Status        string `json:"status"`
			DB            string `json:"db"`
			Version       string `json:"version"`
			SchemaVersion int64  `json:"schema_version"`
		} `json:"data"`
	}

	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if !resp.Success {
		t.Errorf("handler returned unexpected body: %v", rr.Body.String())
	}
	if resp.Data.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Data.Status)
	}
	if resp.Data.DB != "none" {
		t.Errorf("db = %q, want none", resp.Data.DB)
	}
	if resp.Data.Version != appVersion {
		t.Errorf("version = %q, want %q", resp.Data.Version, appVersion)
	}
	if resp.Data.SchemaVersion != 0 {
		t.Errorf("schema_version = %d, want 0", resp.Data.SchemaVersion)
	}
}
