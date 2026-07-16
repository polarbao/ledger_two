package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"ledger_two/internal/db/repo"
	"ledger_two/internal/http/handler"
	"ledger_two/internal/http/response"
	"ledger_two/internal/service"
	"ledger_two/internal/transaction"
)

func TestTransactionUpdateContractPreservesArchivedTagsAndUnchangedSplits(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	const jwtSecret = "test-secret-update-contract"
	initHandler := handler.NewInitHandler(service.NewInitService(repo.NewInitRepo(database)))
	authHandler := handler.NewAuthHandler(service.NewAuthService(repo.NewAuthRepo(database), jwtSecret))
	txHandler := transaction.NewHandler(transaction.NewService(transaction.NewRepository(database)))

	router := chi.NewRouter()
	router.Post("/api/init/setup", initHandler.HandleSetup)
	router.Post("/api/auth/login", authHandler.HandleLogin)
	router.Group(func(r chi.Router) {
		r.Use(testAuthenticatedLedgerContext(database, jwtSecret))
		r.Route("/api/transactions", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreate)
			r.Patch("/{id}", txHandler.HandleUpdate)
		})
		r.Route("/api/shared-expenses", func(r chi.Router) {
			r.Post("/", txHandler.HandleCreateSharedExpense)
			r.Patch("/{id}", txHandler.HandleUpdateSharedExpense)
		})
	})

	setupBody, _ := json.Marshal(map[string]string{
		"ledger_name":         "Update Contract Ledger",
		"user_a_username":     "userA",
		"user_a_display_name": "User A",
		"user_a_password":     "pass123",
		"user_b_username":     "userB",
		"user_b_display_name": "User B",
		"user_b_password":     "pass456",
	})
	setupRequest, _ := http.NewRequest(http.MethodPost, "/api/init/setup", bytes.NewReader(setupBody))
	setupRecorder := httptest.NewRecorder()
	router.ServeHTTP(setupRecorder, setupRequest)
	if setupRecorder.Code != http.StatusOK {
		t.Fatalf("setup failed: %s", setupRecorder.Body.String())
	}

	cookie := getLoginCookie(t, router, "userA", "pass123")
	var userAID, categoryID string
	if err := database.QueryRow("SELECT id FROM users WHERE username = 'userA'").Scan(&userAID); err != nil {
		t.Fatalf("query user A: %v", err)
	}
	if err := database.QueryRow("SELECT id FROM categories LIMIT 1").Scan(&categoryID); err != nil {
		t.Fatalf("query category: %v", err)
	}

	createTransaction := func(path string, payload map[string]interface{}) string {
		t.Helper()
		body, _ := json.Marshal(payload)
		request, _ := http.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		request.AddCookie(cookie)
		setTestLedgerHeader(t, database, request, "Update Contract Ledger")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusCreated {
			t.Fatalf("create transaction at %s failed: %d %s", path, recorder.Code, recorder.Body.String())
		}
		var result response.SuccessResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode create response: %v", err)
		}
		return result.Data.(map[string]interface{})["id"].(string)
	}

	ordinaryID := createTransaction("/api/transactions/", map[string]interface{}{
		"type":             "expense",
		"title":            "午餐",
		"amount_cents":     int64(3580),
		"currency":         "CNY",
		"occurred_at":      time.Now().Format(time.RFC3339),
		"payer_user_id":    userAID,
		"category_id":      categoryID,
		"visibility":       "partner_readable",
		"tag_names":        []string{"历史标签"},
		"attachment_paths": []string{},
	})
	if _, err := database.Exec("UPDATE tags SET is_archived = 1 WHERE name = '历史标签'"); err != nil {
		t.Fatalf("archive tag: %v", err)
	}

	patch := func(path string, payload map[string]interface{}) *httptest.ResponseRecorder {
		t.Helper()
		body, _ := json.Marshal(payload)
		request, _ := http.NewRequest(http.MethodPatch, path, bytes.NewReader(body))
		request.AddCookie(cookie)
		setTestLedgerHeader(t, database, request, "Update Contract Ledger")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		return recorder
	}

	ordinaryUpdate := patch("/api/transactions/"+ordinaryID, map[string]interface{}{"title": "午餐报销"})
	if ordinaryUpdate.Code != http.StatusOK {
		t.Fatalf("ordinary update failed: %d %s", ordinaryUpdate.Code, ordinaryUpdate.Body.String())
	}
	var archived int
	if err := database.QueryRow("SELECT is_archived FROM tags WHERE name = '历史标签'").Scan(&archived); err != nil {
		t.Fatalf("query archived tag: %v", err)
	}
	if archived != 1 {
		t.Fatalf("title-only edit must not restore archived tag, got is_archived=%d", archived)
	}

	invalidCategory := patch("/api/transactions/"+ordinaryID, map[string]interface{}{"category_id": "foreign-category"})
	if invalidCategory.Code != http.StatusBadRequest {
		t.Fatalf("invalid category should be rejected, got %d %s", invalidCategory.Code, invalidCategory.Body.String())
	}
	invalidPayer := patch("/api/transactions/"+ordinaryID, map[string]interface{}{"payer_user_id": "foreign-user"})
	if invalidPayer.Code != http.StatusBadRequest {
		t.Fatalf("invalid payer should be rejected, got %d %s", invalidPayer.Code, invalidPayer.Body.String())
	}

	sharedID := createTransaction("/api/shared-expenses/", map[string]interface{}{
		"title":         "家庭采购",
		"amount_cents":  int64(10001),
		"currency":      "CNY",
		"occurred_at":   time.Now().Format(time.RFC3339),
		"payer_user_id": userAID,
		"category_id":   categoryID,
		"split_method":  "equal",
	})
	rows, err := database.Query("SELECT id, share_amount FROM transaction_splits WHERE transaction_id = ? ORDER BY user_id", sharedID)
	if err != nil {
		t.Fatalf("query original splits: %v", err)
	}
	type splitSnapshot struct {
		id     string
		amount int64
	}
	var before []splitSnapshot
	for rows.Next() {
		var item splitSnapshot
		if err := rows.Scan(&item.id, &item.amount); err != nil {
			rows.Close()
			t.Fatalf("scan original split: %v", err)
		}
		before = append(before, item)
	}
	rows.Close()

	sharedUpdate := patch("/api/shared-expenses/"+sharedID, map[string]interface{}{"note": "只修改备注"})
	if sharedUpdate.Code != http.StatusOK {
		t.Fatalf("shared metadata update failed: %d %s", sharedUpdate.Code, sharedUpdate.Body.String())
	}
	rows, err = database.Query("SELECT id, share_amount FROM transaction_splits WHERE transaction_id = ? ORDER BY user_id", sharedID)
	if err != nil {
		t.Fatalf("query updated splits: %v", err)
	}
	var after []splitSnapshot
	for rows.Next() {
		var item splitSnapshot
		if err := rows.Scan(&item.id, &item.amount); err != nil {
			rows.Close()
			t.Fatalf("scan updated split: %v", err)
		}
		after = append(after, item)
	}
	rows.Close()
	if len(before) != len(after) {
		t.Fatalf("metadata-only edit changed participant count: before=%d after=%d", len(before), len(after))
	}
	for index := range before {
		if before[index] != after[index] {
			t.Fatalf("metadata-only edit rewrote split %d: before=%+v after=%+v", index, before[index], after[index])
		}
	}
}
