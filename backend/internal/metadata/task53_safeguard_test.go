package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	ledgerctx "ledger_two/internal/ledger"
)

func TestTask534CMetadataListCountsOnlyActiveCurrentLedgerRuleReferences(t *testing.T) {
	database := openMetadataTestDB(t)
	seedTask534CMetadataFixture(t, database)
	repo := NewRepository(database)

	assertTask534CReferenceCount(t, repo, KindCategory, "fallback-expense", 1)
	assertTask534CReferenceCount(t, repo, KindCategory, "replacement-expense", 0)
	assertTask534CReferenceCount(t, repo, KindTag, "tag-fixed", 1)
	assertTask534CReferenceCount(t, repo, KindAccount, "account-cash", 1)
}

func TestTask534CFallbackArchiveRequiresValidSameLedgerReplacement(t *testing.T) {
	t.Run("missing replacement", func(t *testing.T) {
		database := openMetadataTestDB(t)
		seedTask534CMetadataFixture(t, database)
		service := NewService(NewRepository(database))

		_, err := service.Archive(task534CMetadataContext(), "owner-profile", KindCategory, "fallback-expense", ArchiveRequest{})
		assertMetadataAppError(t, err, http.StatusConflict, appErrors.ErrCodeCategoryFallbackRequired)
		assertTask534CFallbackState(t, database, "fallback-expense", "expense_other", false)
	})

	invalid := []struct {
		name          string
		replacementID string
	}{
		{name: "same category", replacementID: "fallback-expense"},
		{name: "wrong type", replacementID: "replacement-income"},
		{name: "existing system key", replacementID: "replacement-keyed"},
		{name: "archived replacement", replacementID: "replacement-archived"},
		{name: "other ledger", replacementID: "foreign-expense"},
		{name: "missing category", replacementID: "missing-category"},
	}
	for _, testCase := range invalid {
		t.Run(testCase.name, func(t *testing.T) {
			database := openMetadataTestDB(t)
			seedTask534CMetadataFixture(t, database)
			service := NewService(NewRepository(database))

			_, err := service.Archive(task534CMetadataContext(), "owner-profile", KindCategory, "fallback-expense", ArchiveRequest{
				ReplacementCategoryID: testCase.replacementID,
			})
			assertMetadataAppError(t, err, http.StatusConflict, appErrors.ErrCodeCategoryFallbackReplacementInvalid)
			assertTask534CFallbackState(t, database, "fallback-expense", "expense_other", false)
		})
	}
}

func TestTask534CFallbackArchiveTransfersSystemKeyWithoutRewritingHistory(t *testing.T) {
	database := openMetadataTestDB(t)
	seedTask534CMetadataFixture(t, database)
	service := NewService(NewRepository(database))

	result, err := service.Archive(task534CMetadataContext(), "owner-profile", KindCategory, "fallback-expense", ArchiveRequest{
		ReplacementCategoryID: "replacement-expense",
	})
	if err != nil {
		t.Fatalf("archive fallback category: %v", err)
	}
	if result.ArchivedID != "fallback-expense" || !result.FallbackReplaced || result.TransferredSystemKey != "expense_other" || result.ReplacementCategoryID != "replacement-expense" {
		t.Fatalf("unexpected fallback archive result: %+v", result)
	}
	assertTask534CFallbackState(t, database, "fallback-expense", "", true)
	assertTask534CFallbackState(t, database, "replacement-expense", "expense_other", false)

	var transactionCategoryID, ruleResult string
	var profileVersion, transactionCount, auditCount int
	if err := database.QueryRow("SELECT category_id FROM transactions WHERE id = 'tx-fallback-history'").Scan(&transactionCategoryID); err != nil {
		t.Fatalf("read historical transaction: %v", err)
	}
	if err := database.QueryRow("SELECT result_json FROM import_rules WHERE id = 'rule-current-active'").Scan(&ruleResult); err != nil {
		t.Fatalf("read historical rule: %v", err)
	}
	if err := database.QueryRow("SELECT metadata_profile_version FROM ledgers WHERE id = 'ledger-profile'").Scan(&profileVersion); err != nil {
		t.Fatalf("read metadata profile version: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM transactions WHERE ledger_id = 'ledger-profile'").Scan(&transactionCount); err != nil {
		t.Fatalf("count historical transactions: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action = 'metadata_fallback_replace'").Scan(&auditCount); err != nil {
		t.Fatalf("count fallback audits: %v", err)
	}
	if transactionCategoryID != "fallback-expense" || !strings.Contains(ruleResult, `"category_id":"fallback-expense"`) || profileVersion != 1 || transactionCount != 1 || auditCount != 1 {
		t.Fatalf("fallback archive rewrote historical state: category=%q rule=%q profile=%d tx=%d audits=%d", transactionCategoryID, ruleResult, profileVersion, transactionCount, auditCount)
	}
}

func TestTask534CFallbackArchiveRollsBackSystemKeyWhenAuditFails(t *testing.T) {
	database := openMetadataTestDB(t)
	seedTask534CMetadataFixture(t, database)
	if _, err := database.Exec(`
		CREATE TRIGGER task53_fail_fallback_audit
		BEFORE INSERT ON audit_logs
		WHEN NEW.action = 'metadata_fallback_replace'
		BEGIN
			SELECT RAISE(ABORT, 'injected fallback audit failure');
		END;
	`); err != nil {
		t.Fatalf("create fallback audit failure trigger: %v", err)
	}

	service := NewService(NewRepository(database))
	if _, err := service.Archive(task534CMetadataContext(), "owner-profile", KindCategory, "fallback-expense", ArchiveRequest{
		ReplacementCategoryID: "replacement-expense",
	}); err == nil {
		t.Fatal("expected injected fallback audit failure")
	}
	assertTask534CFallbackState(t, database, "fallback-expense", "expense_other", false)
	assertTask534CFallbackState(t, database, "replacement-expense", "", false)
}

func TestTask534CArchiveHandlerAcceptsFallbackPayloadAndRejectsUnknownFields(t *testing.T) {
	database := openMetadataTestDB(t)
	seedTask534CMetadataFixture(t, database)
	handler := NewHandler(NewService(NewRepository(database)))

	request := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("kind", string(KindCategory))
		routeContext.URLParams.Add("id", "fallback-expense")
		req := httptest.NewRequest(http.MethodPost, "/api/metadata/categories/fallback-expense/archive", strings.NewReader(body))
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeContext)
		ctx = context.WithValue(ctx, middleware.UserIDKey, "owner-profile")
		ctx = ledgerctx.ContextWithLedgerContext(ctx, ledgerctx.LedgerContext{
			UserID: "owner-profile", LedgerID: "ledger-profile", Role: ledgerctx.RoleOwner, IsExplicit: true,
		})
		recorder := httptest.NewRecorder()
		handler.Archive(recorder, req.WithContext(ctx))
		return recorder
	}

	rejected := request(`{"replacement_category_id":"replacement-expense","unexpected":true}`)
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("archive accepted unknown field: status=%d body=%s", rejected.Code, rejected.Body.String())
	}
	nullBody := request(`null`)
	if nullBody.Code != http.StatusBadRequest {
		t.Fatalf("archive accepted null body: status=%d body=%s", nullBody.Code, nullBody.Body.String())
	}
	trailingBody := request(`{"replacement_category_id":"replacement-expense"} {}`)
	if trailingBody.Code != http.StatusBadRequest {
		t.Fatalf("archive accepted trailing JSON: status=%d body=%s", trailingBody.Code, trailingBody.Body.String())
	}
	accepted := request(`{"replacement_category_id":"replacement-expense"}`)
	if accepted.Code != http.StatusOK {
		t.Fatalf("fallback archive status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	var response struct {
		Success bool          `json:"success"`
		Data    ArchiveResult `json:"data"`
	}
	if err := json.Unmarshal(accepted.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode fallback archive response: %v", err)
	}
	if !response.Success || !response.Data.FallbackReplaced || response.Data.ReplacementCategoryID != "replacement-expense" {
		t.Fatalf("unexpected fallback archive response: %+v", response)
	}
}

func seedTask534CMetadataFixture(t *testing.T, database *sql.DB) {
	t.Helper()
	seedMetadataProfileLedger(t, database)
	if _, err := database.Exec(`
		UPDATE ledgers SET metadata_profile_version = 1 WHERE id = 'ledger-profile';
		INSERT INTO ledgers (id, name, default_currency, created_at, updated_at)
		VALUES ('ledger-foreign', 'Foreign Ledger', 'CNY', '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z');
		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, system_key, color, is_archived, created_at, updated_at
		) VALUES
			('fallback-expense', 'ledger-profile', 'owner-profile', '其他支出', 'expense', 'expense_other', '#64748b', 0, '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z'),
			('replacement-expense', 'ledger-profile', 'owner-profile', '临时支出', 'expense', NULL, '#22c55e', 0, '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z'),
			('replacement-income', 'ledger-profile', 'owner-profile', '临时收入', 'income', NULL, '#3b82f6', 0, '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z'),
			('replacement-keyed', 'ledger-profile', 'owner-profile', '餐饮', 'expense', 'expense_food', '#f59e0b', 0, '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z'),
			('replacement-archived', 'ledger-profile', 'owner-profile', '旧支出', 'expense', NULL, '#94a3b8', 1, '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z'),
			('foreign-expense', 'ledger-foreign', 'owner-profile', '外部支出', 'expense', NULL, '#ef4444', 0, '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z');
		INSERT INTO tags (
			id, ledger_id, owner_user_id, name, system_key, color, is_archived, created_at, updated_at
		) VALUES ('tag-fixed', 'ledger-profile', 'owner-profile', '固定支出', 'tag_fixed', '#0f766e', 0, '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z');
		INSERT INTO accounts (
			id, ledger_id, owner_user_id, name, type, currency, initial_balance, is_archived, created_at, updated_at
		) VALUES ('account-cash', 'ledger-profile', 'owner-profile', '现金', 'cash', 'CNY', 0, 0, '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z');
		INSERT INTO import_rules (
			id, ledger_id, keyword, created_by_user_id, name, match_type, pattern, priority,
			result_json, status, origin, apply_mode, confidence, created_at, updated_at
		) VALUES
			('rule-current-active', 'ledger-profile', '测试商户', 'owner-profile', '当前规则', 'merchant_equals', '测试商户', 1,
			 '{"category_id":"fallback-expense","account_id":"account-cash","tag_ids":["tag-fixed"]}',
			 'active', 'manual', 'auto', 'high', '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z'),
			('rule-current-archived', 'ledger-profile', '归档商户', 'owner-profile', '归档规则', 'merchant_equals', '归档商户', 2,
			 '{"category_id":"fallback-expense","account_id":"account-cash","tag_ids":["tag-fixed"]}',
			 'archived', 'manual', 'auto', 'high', '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z'),
			('rule-foreign', 'ledger-foreign', '外部商户', 'owner-profile', '外部规则', 'merchant_equals', '外部商户', 1,
			 '{"category_id":"fallback-expense","account_id":"account-cash","tag_ids":["tag-fixed"]}',
			 'active', 'manual', 'auto', 'high', '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z');
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id,
			visibility, status, created_at, updated_at
		) VALUES (
			'tx-fallback-history', 'ledger-profile', 'expense', '历史支出', 8800, 'CNY', '2026-07-20T08:00:00Z',
			'owner-profile', 'owner-profile', 'owner-profile', 'account-cash', 'fallback-expense',
			'private', 'normal', '2026-07-20T08:00:00Z', '2026-07-20T08:00:00Z'
		);
	`); err != nil {
		t.Fatalf("seed Task53.4C metadata fixture: %v", err)
	}
}

func task534CMetadataContext() context.Context {
	return ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID: "owner-profile", LedgerID: "ledger-profile", Role: ledgerctx.RoleOwner, IsExplicit: true,
	})
}

func assertTask534CReferenceCount(t *testing.T, repo *Repository, kind Kind, id string, expected int) {
	t.Helper()
	items, err := repo.List(context.Background(), kind, "ledger-profile", true)
	if err != nil {
		t.Fatalf("list %s metadata: %v", kind, err)
	}
	for _, item := range items {
		if item.ID == id {
			if item.RuleReferenceCount != expected {
				t.Fatalf("%s %s rule_reference_count=%d, want %d", kind, id, item.RuleReferenceCount, expected)
			}
			return
		}
	}
	t.Fatalf("metadata item %s not found", id)
}

func assertTask534CFallbackState(t *testing.T, database *sql.DB, id string, systemKey string, archived bool) {
	t.Helper()
	var storedSystemKey sql.NullString
	var storedArchived int
	if err := database.QueryRow("SELECT system_key, is_archived FROM categories WHERE id = ?", id).Scan(&storedSystemKey, &storedArchived); err != nil {
		t.Fatalf("read fallback state for %s: %v", id, err)
	}
	actualKey := ""
	if storedSystemKey.Valid {
		actualKey = storedSystemKey.String
	}
	if actualKey != systemKey || (storedArchived == 1) != archived {
		t.Fatalf("category %s state key=%q archived=%t, want key=%q archived=%t", id, actualKey, storedArchived == 1, systemKey, archived)
	}
}
