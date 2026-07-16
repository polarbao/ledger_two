package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTask503BMemberRoutesUsePatchETagTransferAndLeaveContract(t *testing.T) {
	database := setupRBACRouterDB(t)
	handler := New(database, rbacRouterConfig(t))
	fixture := seedRBACLedger(t, database)

	getReq := httptest.NewRequest(http.MethodGet, "/api/ledgers/"+fixture.LedgerID+"/members", nil)
	getReq.AddCookie(authCookie(t, fixture.UserAID))
	getRecorder := httptest.NewRecorder()
	handler.ServeHTTP(getRecorder, getReq)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get members: status=%d body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	if getRecorder.Header().Get("ETag") == "" {
		t.Fatal("get members must return ledger ETag")
	}
	var initial memberListEnvelope
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode initial member list: %v", err)
	}
	if initial.Data.Ledger.ID != fixture.LedgerID || len(initial.Data.Members) != 2 {
		t.Fatalf("unexpected initial member response: %+v", initial.Data)
	}

	patchRecorder := performMemberRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodPatch,
		"/api/ledgers/"+fixture.LedgerID+"/members/"+fixture.UserBID,
		fixture.LedgerID,
		1,
		`{"role":"viewer"}`,
	)
	if patchRecorder.Code != http.StatusOK || patchRecorder.Header().Get("ETag") != ledgerETag(fixture.LedgerID, 2) {
		t.Fatalf("patch member: status=%d etag=%s body=%s", patchRecorder.Code, patchRecorder.Header().Get("ETag"), patchRecorder.Body.String())
	}
	assertRBACRole(t, database, fixture.LedgerID, fixture.UserBID, "viewer")

	putRecorder := performMemberRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodPut,
		"/api/ledgers/"+fixture.LedgerID+"/members/"+fixture.UserBID,
		fixture.LedgerID,
		2,
		`{"role":"editor"}`,
	)
	if putRecorder.Code != http.StatusOK || putRecorder.Header().Get("ETag") != ledgerETag(fixture.LedgerID, 3) {
		t.Fatalf("compat put member: status=%d etag=%s body=%s", putRecorder.Code, putRecorder.Header().Get("ETag"), putRecorder.Body.String())
	}

	transferRecorder := performMemberRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodPost,
		"/api/ledgers/"+fixture.LedgerID+"/members/"+fixture.UserBID+"/transfer-owner",
		fixture.LedgerID,
		3,
		`{"acknowledge_permission_change":true}`,
	)
	if transferRecorder.Code != http.StatusOK || transferRecorder.Header().Get("ETag") != ledgerETag(fixture.LedgerID, 4) {
		t.Fatalf("transfer owner: status=%d etag=%s body=%s", transferRecorder.Code, transferRecorder.Header().Get("ETag"), transferRecorder.Body.String())
	}
	assertRBACRole(t, database, fixture.LedgerID, fixture.UserAID, "editor")
	assertRBACRole(t, database, fixture.LedgerID, fixture.UserBID, "owner")

	leaveRecorder := performMemberRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodPost,
		"/api/ledgers/"+fixture.LedgerID+"/leave",
		fixture.LedgerID,
		4,
		"",
	)
	if leaveRecorder.Code != http.StatusOK || leaveRecorder.Header().Get("ETag") != ledgerETag(fixture.LedgerID, 5) {
		t.Fatalf("leave ledger: status=%d etag=%s body=%s", leaveRecorder.Code, leaveRecorder.Header().Get("ETag"), leaveRecorder.Body.String())
	}
	var remaining int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM ledger_members
		WHERE ledger_id = ? AND user_id = ?
	`, fixture.LedgerID, fixture.UserAID).Scan(&remaining); err != nil {
		t.Fatalf("count leaving membership: %v", err)
	}
	if remaining != 0 {
		t.Fatal("leaving member relationship still exists")
	}
}

func TestTask503BAddMemberRequiresHistoryAcknowledgementAndMemberLimit(t *testing.T) {
	database := setupRBACRouterDB(t)
	handler := New(database, rbacRouterConfig(t))
	fixture := seedRBACLedger(t, database)
	singleLedgerID := insertTask50Ledger(t, database, "task50-single-member", fixture.UserAID)

	addRecorder := performMemberRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodPost,
		"/api/ledgers/"+singleLedgerID+"/members",
		singleLedgerID,
		1,
		`{"username":"userB","role":"editor","acknowledge_history_visibility":true}`,
	)
	if addRecorder.Code != http.StatusCreated || addRecorder.Header().Get("ETag") != ledgerETag(singleLedgerID, 2) {
		t.Fatalf("add member: status=%d etag=%s body=%s", addRecorder.Code, addRecorder.Header().Get("ETag"), addRecorder.Body.String())
	}

	insertRBACUser(t, database, "userC", "User C")
	limitRecorder := performMemberRequest(
		t,
		handler,
		fixture.UserAID,
		http.MethodPost,
		"/api/ledgers/"+singleLedgerID+"/members",
		singleLedgerID,
		2,
		`{"username":"userC","role":"viewer","acknowledge_history_visibility":true}`,
	)
	assertRouterError(t, limitRecorder, http.StatusConflict, "LEDGER_MEMBER_LIMIT_REACHED")

	var version int64
	if err := database.QueryRow("SELECT version FROM ledgers WHERE id = ?", singleLedgerID).Scan(&version); err != nil {
		t.Fatalf("read member limit version: %v", err)
	}
	if version != 2 {
		t.Fatalf("member limit failure changed version to %d", version)
	}
}

type memberListEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		Ledger struct {
			ID      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"ledger"`
		Members []struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
			Role     string `json:"role"`
			JoinedAt string `json:"joined_at"`
		} `json:"members"`
	} `json:"data"`
}

func performMemberRequest(
	t *testing.T,
	handler http.Handler,
	userID, method, path, ledgerID string,
	version int64,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	var requestBody *bytes.Reader
	if body == "" {
		requestBody = bytes.NewReader(nil)
	} else {
		requestBody = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, requestBody)
	req.Header.Set("X-Ledger-Id", ledgerID)
	req.Header.Set("If-Match", ledgerETag(ledgerID, version))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(authCookie(t, userID))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func ledgerETag(ledgerID string, version int64) string {
	return fmt.Sprintf(`"ledger:%s:v%d"`, ledgerID, version)
}
