package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTask531DefaultProfileRoutesUseExplicitLedgerContextAndOwnerApply(t *testing.T) {
	database := setupRBACRouterDB(t)
	fixture := seedRBACLedger(t, database)
	router := New(database, rbacRouterConfig(t))

	getRequest := httptest.NewRequest(http.MethodGet, "/api/metadata/default-profile?profile=basic_cn_v1", nil)
	getRequest.Header.Set("X-Ledger-Id", fixture.LedgerID)
	getRequest.AddCookie(authCookie(t, fixture.UserAID))
	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("default profile get status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}

	previewRequest := httptest.NewRequest(http.MethodPost, "/api/metadata/default-profile/preview", strings.NewReader(`{"profile":"basic_cn_v1"}`))
	previewRequest.Header.Set("X-Ledger-Id", fixture.LedgerID)
	previewRequest.AddCookie(authCookie(t, fixture.UserAID))
	previewResponse := httptest.NewRecorder()
	router.ServeHTTP(previewResponse, previewRequest)
	if previewResponse.Code != http.StatusOK {
		t.Fatalf("default profile preview status=%d body=%s", previewResponse.Code, previewResponse.Body.String())
	}

	editorApplyRequest := httptest.NewRequest(http.MethodPost, "/api/metadata/default-profile/apply", strings.NewReader(`{"profile":"basic_cn_v1","resolutions":[]}`))
	editorApplyRequest.Header.Set("X-Ledger-Id", fixture.LedgerID)
	editorApplyRequest.AddCookie(authCookie(t, fixture.UserBID))
	editorApplyResponse := httptest.NewRecorder()
	router.ServeHTTP(editorApplyResponse, editorApplyRequest)
	if editorApplyResponse.Code != http.StatusForbidden {
		t.Fatalf("editor profile apply status=%d body=%s", editorApplyResponse.Code, editorApplyResponse.Body.String())
	}

	ownerApplyRequest := httptest.NewRequest(http.MethodPost, "/api/metadata/default-profile/apply", strings.NewReader(`{"profile":"basic_cn_v1","resolutions":[]}`))
	ownerApplyRequest.Header.Set("X-Ledger-Id", fixture.LedgerID)
	ownerApplyRequest.AddCookie(authCookie(t, fixture.UserAID))
	ownerApplyResponse := httptest.NewRecorder()
	router.ServeHTTP(ownerApplyResponse, ownerApplyRequest)
	if ownerApplyResponse.Code != http.StatusOK {
		t.Fatalf("owner profile apply status=%d body=%s", ownerApplyResponse.Code, ownerApplyResponse.Body.String())
	}
}
