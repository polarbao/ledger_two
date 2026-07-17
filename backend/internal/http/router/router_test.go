package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ledger_two/internal/config"
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
			Status                   string `json:"status"`
			DB                       string `json:"db"`
			Version                  string `json:"version"`
			SchemaVersion            int64  `json:"schema_version"`
			DeploymentChannel        string `json:"deployment_channel"`
			ImportClassificationMode string `json:"import_classification_mode"`
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
	if resp.Data.DeploymentChannel != "unknown" {
		t.Errorf("deployment_channel = %q, want unknown", resp.Data.DeploymentChannel)
	}
	if resp.Data.ImportClassificationMode != "off" {
		t.Errorf("import_classification_mode = %q, want off", resp.Data.ImportClassificationMode)
	}
}

func TestHealthzReturnsDeploymentChannel(t *testing.T) {
	r := New(nil, &config.Config{
		DeploymentChannel:        config.DeploymentChannelStaging,
		ImportXLSXEnabled:        true,
		ImportClassificationMode: "suggest",
	})
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	var resp struct {
		Data struct {
			DeploymentChannel        string `json:"deployment_channel"`
			ImportXLSXEnabled        bool   `json:"import_xlsx_enabled"`
			ImportClassificationMode string `json:"import_classification_mode"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if resp.Data.DeploymentChannel != config.DeploymentChannelStaging {
		t.Fatalf("deployment_channel = %q, want staging", resp.Data.DeploymentChannel)
	}
	if !resp.Data.ImportXLSXEnabled {
		t.Fatalf("expected import_xlsx_enabled in health response")
	}
	if resp.Data.ImportClassificationMode != "suggest" {
		t.Fatalf("import_classification_mode = %q, want suggest", resp.Data.ImportClassificationMode)
	}
}
