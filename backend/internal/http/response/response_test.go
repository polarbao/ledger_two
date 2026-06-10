package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/response"
)

func TestErrorResponseFormat(t *testing.T) {
	// Test Case 1: Standard Error
	rec1 := httptest.NewRecorder()
	response.Error(rec1, http.StatusBadRequest, "VALIDATION_ERROR", "参数错误")

	if rec1.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec1.Code)
	}

	var resp1 response.ErrorResponse
	if err := json.Unmarshal(rec1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to unmarshal resp1: %v", err)
	}

	if resp1.Success != false {
		t.Errorf("expected success false, got true")
	}
	if resp1.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %s", resp1.Error.Code)
	}
	if resp1.Error.Message != "参数错误" {
		t.Errorf("expected message '参数错误', got '%s'", resp1.Error.Message)
	}
	if resp1.Error.Details != nil {
		t.Errorf("expected details nil, got %+v", resp1.Error.Details)
	}

	// Test Case 2: WriteError with AppError
	rec2 := httptest.NewRecorder()
	appErr := appErrors.NewAppError(http.StatusConflict, "APP_ALREADY_INITIALIZED", "已经初始化")
	response.WriteError(rec2, appErr)

	if rec2.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rec2.Code)
	}

	var resp2 response.ErrorResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("failed to unmarshal resp2: %v", err)
	}
	if resp2.Error.Code != "APP_ALREADY_INITIALIZED" {
		t.Errorf("expected code APP_ALREADY_INITIALIZED, got %s", resp2.Error.Code)
	}

	// Test Case 3: WriteError with generic internal error
	rec3 := httptest.NewRecorder()
	genericErr := errors.New("something went wrong in db")
	response.WriteError(rec3, genericErr)

	if rec3.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec3.Code)
	}

	var resp3 response.ErrorResponse
	if err := json.Unmarshal(rec3.Body.Bytes(), &resp3); err != nil {
		t.Fatalf("failed to unmarshal resp3: %v", err)
	}
	if resp3.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code INTERNAL_ERROR, got %s", resp3.Error.Code)
	}
	// 确保内部堆栈没有泄露给前端
	if resp3.Error.Message == "something went wrong in db" {
		t.Errorf("generic internal error leaked message to user")
	}
}
