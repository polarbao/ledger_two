package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ledger_two/internal/http/middleware"
	ledgerctx "ledger_two/internal/ledger"
)

func TestTask531DefaultProfileHandlersExposeDefinitionPreviewAndApply(t *testing.T) {
	database := openMetadataTestDB(t)
	seedMetadataProfileLedger(t, database)
	handler := NewHandler(NewService(NewRepository(database)))

	getRequest := httptest.NewRequest(http.MethodGet, "/api/metadata/default-profile?profile=basic_cn_v1", nil)
	getRequest = getRequest.WithContext(profileHandlerContext(getRequest.Context()))
	getResponse := httptest.NewRecorder()
	handler.GetDefaultProfile(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get profile status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}

	previewRequest := httptest.NewRequest(http.MethodPost, "/api/metadata/default-profile/preview", bytes.NewBufferString(`{"profile":"basic_cn_v1"}`))
	previewRequest = previewRequest.WithContext(profileHandlerContext(previewRequest.Context()))
	previewResponse := httptest.NewRecorder()
	handler.PreviewDefaultProfile(previewResponse, previewRequest)
	if previewResponse.Code != http.StatusOK {
		t.Fatalf("preview profile status=%d body=%s", previewResponse.Code, previewResponse.Body.String())
	}
	var previewEnvelope struct {
		Success bool                 `json:"success"`
		Data    ProfilePreviewResult `json:"data"`
	}
	if err := json.NewDecoder(previewResponse.Body).Decode(&previewEnvelope); err != nil {
		t.Fatalf("decode preview response: %v", err)
	}
	if !previewEnvelope.Success || previewEnvelope.Data.CreateCount != 27 {
		t.Fatalf("unexpected preview response: %+v", previewEnvelope)
	}

	applyRequest := httptest.NewRequest(http.MethodPost, "/api/metadata/default-profile/apply", bytes.NewBufferString(`{"profile":"basic_cn_v1","resolutions":[]}`))
	applyRequest = applyRequest.WithContext(profileHandlerContext(applyRequest.Context()))
	applyResponse := httptest.NewRecorder()
	handler.ApplyDefaultProfile(applyResponse, applyRequest)
	if applyResponse.Code != http.StatusOK {
		t.Fatalf("apply profile status=%d body=%s", applyResponse.Code, applyResponse.Body.String())
	}
	var applyEnvelope struct {
		Success bool               `json:"success"`
		Data    ProfileApplyResult `json:"data"`
	}
	if err := json.NewDecoder(applyResponse.Body).Decode(&applyEnvelope); err != nil {
		t.Fatalf("decode apply response: %v", err)
	}
	if !applyEnvelope.Success || applyEnvelope.Data.CreatedCount != 27 || applyEnvelope.Data.MetadataProfileVersion != 1 {
		t.Fatalf("unexpected apply response: %+v", applyEnvelope)
	}
}

func profileHandlerContext(parent context.Context) context.Context {
	ctx := context.WithValue(parent, middleware.UserIDKey, "owner-profile")
	return ledgerctx.ContextWithLedgerContext(ctx, ledgerctx.LedgerContext{
		UserID:     "owner-profile",
		LedgerID:   "ledger-profile",
		Role:       ledgerctx.RoleOwner,
		Status:     ledgerctx.LedgerStatusActive,
		Version:    1,
		IsExplicit: true,
	})
}
