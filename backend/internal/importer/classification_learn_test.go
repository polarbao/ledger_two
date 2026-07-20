package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/ledger"
)

func TestTask534BLearnMerchantRuleIsDeterministicAndUpdatesOnlyCategoryAndTags(t *testing.T) {
	database, service, batch, row := newTask534BLearnFixture(t)
	originalImportHash := row.ImportHash

	created, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	if err != nil {
		t.Fatalf("LearnMerchantRule returned error: %v", err)
	}
	parsedID, err := uuid.Parse(created.RuleID)
	if err != nil || parsedID.Version() != 5 {
		t.Fatalf("learned rule id must be UUIDv5: id=%q err=%v", created.RuleID, err)
	}
	if created.Action != LearnActionCreated || created.NormalizedMerchant != "园区餐厅" || created.SourceScope != LearnSourceScopeCurrent || created.SourceType == nil || *created.SourceType != SourceTypeGeneric {
		t.Fatalf("unexpected create result: %+v", created)
	}
	fixtureData, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "fixtures", "category-tag", "expected", "learn-created.json"))
	if err != nil {
		t.Fatalf("read learn expected fixture: %v", err)
	}
	var expected LearnMerchantResult
	if err := json.Unmarshal(fixtureData, &expected); err != nil {
		t.Fatalf("decode learn expected fixture: %v", err)
	}
	if created.RuleID != expected.RuleID || created.Action != expected.Action ||
		created.NormalizedMerchant != expected.NormalizedMerchant || created.SourceScope != expected.SourceScope ||
		created.SourceType == nil || expected.SourceType == nil || *created.SourceType != *expected.SourceType {
		t.Fatalf("learn response drifted from anonymous fixture: got=%+v want=%+v", created, expected)
	}
	rule, err := service.repo.GetImportRule(context.Background(), "ledger-one", created.RuleID)
	if err != nil {
		t.Fatalf("read learned rule: %v", err)
	}
	assertTask534BLearnedRule(t, rule, "cat-food", []string{"tag-breakfast"}, SourceTypeGeneric)
	if rule.Result.AccountID != "" || rule.Result.Visibility != "" {
		t.Fatalf("learned rule copied account or visibility: %+v", rule.Result)
	}

	rowStatus := RowStatusAdjusted
	categoryID := "cat-travel"
	accountID := "account-cash"
	emptyTags := []string{}
	visibility := "shared"
	if _, err := service.UpdatePreviewRow(context.Background(), UpdateRowCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Patch: UpdateRowRequest{
			RowStatus: &rowStatus, SelectedCategoryID: &categoryID, SelectedAccountID: &accountID,
			SelectedTagIDs: emptyTags, Visibility: &visibility,
		},
	}); err != nil {
		t.Fatalf("update saved row before relearn: %v", err)
	}

	updated, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	if err != nil {
		t.Fatalf("repeat LearnMerchantRule returned error: %v", err)
	}
	if updated.RuleID != created.RuleID || updated.Action != LearnActionUpdated {
		t.Fatalf("repeat learn was not idempotent: created=%+v updated=%+v", created, updated)
	}
	rule, err = service.repo.GetImportRule(context.Background(), "ledger-one", created.RuleID)
	if err != nil {
		t.Fatalf("read updated learned rule: %v", err)
	}
	assertTask534BLearnedRule(t, rule, "cat-travel", []string{}, SourceTypeGeneric)
	if rule.Result.AccountID != "" || rule.Result.Visibility != "" {
		t.Fatalf("relearn copied account or visibility: %+v", rule.Result)
	}
	if countWhere(t, database, "import_rules", "origin = 'learned'") != 1 {
		t.Fatalf("repeat learn created a duplicate learned rule")
	}
	if countWhere(t, database, "audit_logs", "action = 'import_rule_learn'") != 2 {
		t.Fatalf("each successful explicit learn must write one audit")
	}
	if countRows(t, database, "transactions") != 0 {
		t.Fatalf("learning must not create transactions")
	}
	if countRows(t, database, "transaction_splits") != 0 || countRows(t, database, "settlements") != 0 {
		t.Fatalf("learning must not create split or settlement records")
	}
	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("read batch after learning: %v", err)
	}
	if findPreviewRow(t, stored, row.ID).ImportHash != originalImportHash {
		t.Fatalf("learning changed the import hash")
	}
	rows, err := database.Query(`SELECT after_json FROM audit_logs WHERE action = 'import_rule_learn'`)
	if err != nil {
		t.Fatalf("query learn audits: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var auditJSON string
		if err := rows.Scan(&auditJSON); err != nil {
			t.Fatalf("scan learn audit: %v", err)
		}
		for _, forbidden := range []string{"园区餐厅", "merchant", "normalized_merchant", "通用模板支出"} {
			if strings.Contains(auditJSON, forbidden) {
				t.Fatalf("learn audit leaked merchant data %q: %s", forbidden, auditJSON)
			}
		}
	}
}

func TestTask534BLearnRejectsManualConflictWithoutRollingBackSavedRow(t *testing.T) {
	database, service, batch := newTask534ABulkFixture(t)
	row := findPreviewRowByNumber(t, batch, 1)
	rowStatus := RowStatusAdjusted
	categoryID := "cat-travel"
	if _, err := service.UpdatePreviewRow(context.Background(), UpdateRowCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Patch: UpdateRowRequest{RowStatus: &rowStatus, SelectedCategoryID: &categoryID, SelectedTagIDs: []string{}},
	}); err != nil {
		t.Fatalf("save row before conflicting learn: %v", err)
	}

	_, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeClassificationConflict)
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %v", err)
	}
	details, ok := appErr.Details.(map[string]string)
	if !ok || details["rule_id"] != "rule-breakfast" {
		t.Fatalf("manual conflict details must contain only the current-ledger rule id: %#v", appErr.Details)
	}
	stored, readErr := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if readErr != nil {
		t.Fatalf("read row after conflicting learn: %v", readErr)
	}
	saved := findPreviewRow(t, stored, row.ID)
	if saved.SelectedCategoryID != categoryID || saved.Classification.Status != ClassificationStatusManual {
		t.Fatalf("learn failure rolled back the earlier row save: %+v", saved)
	}
	if countWhere(t, database, "import_rules", "origin = 'learned'") != 0 || countWhere(t, database, "audit_logs", "action = 'import_rule_learn'") != 0 {
		t.Fatalf("conflicting learn wrote a rule or audit")
	}
}

func TestTask534BLearnRestoresArchivedRuleAndPreservesEditableSettings(t *testing.T) {
	_, service, batch, row := newTask534BLearnFixture(t)
	created, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeAll},
	})
	if err != nil {
		t.Fatalf("create all-source learned rule: %v", err)
	}
	if created.SourceType != nil {
		t.Fatalf("all_sources learned rule must store a null source_type: %+v", created)
	}
	if _, err := service.ArchiveImportRule(context.Background(), ownerLedgerContext(), created.RuleID); err != nil {
		t.Fatalf("archive learned rule: %v", err)
	}
	applyMode := "suggest"
	existing, err := service.repo.GetImportRule(context.Background(), "ledger-one", created.RuleID)
	if err != nil {
		t.Fatalf("read archived learned rule: %v", err)
	}
	if _, err := service.UpdateImportRule(context.Background(), ownerLedgerContext(), created.RuleID, ImportRuleUpsertRequest{
		Name: "保留的学习规则名称", MatchType: existing.MatchType, Pattern: existing.Pattern,
		Priority: intPointer(321), ApplyMode: &applyMode,
		Result: ImportRuleResult{CategoryID: "cat-food", TagIDs: []string{"tag-breakfast"}},
	}); err != nil {
		t.Fatalf("edit archived learned rule: %v", err)
	}

	restored, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeAll},
	})
	if err != nil {
		t.Fatalf("restore learned rule by explicit relearn: %v", err)
	}
	if restored.RuleID != created.RuleID || restored.Action != LearnActionRestored || restored.SourceType != nil {
		t.Fatalf("unexpected restored result: %+v", restored)
	}
	rule, err := service.repo.GetImportRule(context.Background(), "ledger-one", created.RuleID)
	if err != nil {
		t.Fatalf("read restored rule: %v", err)
	}
	if rule.Status != "active" || rule.Name != "保留的学习规则名称" || rule.Priority != 321 || rule.ApplyMode != "suggest" {
		t.Fatalf("relearn did not preserve editable settings: %+v", rule)
	}
}

func TestTask534BLearnEnforcesLedgerEligibilityMerchantAndMetadata(t *testing.T) {
	database, service, batch, row := newTask534BLearnFixture(t)

	foreign := ownerLedgerContext()
	foreign.LedgerID = "ledger-two"
	_, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: foreign, BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	assertAppError(t, err, http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound)

	unsaved := findPreviewRowByNumber(t, batch, 3)
	_, err = service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: unsaved.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeClassificationRuleStale)

	if _, err := database.Exec(`UPDATE import_items SET merchant = '   ' WHERE id = ?`, row.ID); err != nil {
		t.Fatalf("blank learned merchant: %v", err)
	}
	_, err = service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	assertAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeClassificationMerchantRequired)

	if _, err := database.Exec(`UPDATE import_items SET merchant = '园区餐厅' WHERE id = ?; UPDATE categories SET is_archived = 1 WHERE id = 'cat-food'`, row.ID); err != nil {
		t.Fatalf("archive learned category: %v", err)
	}
	_, err = service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeClassificationRuleStale)
	if countWhere(t, database, "import_rules", "origin = 'learned'") != 0 {
		t.Fatalf("rejected learn created a rule")
	}
}

func TestTask534BLearnRejectsAnotherMembersBatch(t *testing.T) {
	database, service, batch, row := newTask534BLearnFixture(t)
	if _, err := database.Exec(`UPDATE import_batches SET created_by_user_id = 'editor-user' WHERE id = ?`, batch.ID); err != nil {
		t.Fatalf("change batch creator: %v", err)
	}

	_, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	assertAppError(t, err, http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound)
	if countWhere(t, database, "import_rules", "origin = 'learned'") != 0 {
		t.Fatalf("cross-member learn created a rule")
	}
}

func TestTask534BLearnRejectsCategoryMismatchAndTagOverflow(t *testing.T) {
	t.Run("category mismatch", func(t *testing.T) {
		database, service, batch, row := newTask534BLearnFixture(t)
		if _, err := database.Exec(`UPDATE import_items SET selected_category_id = 'cat-income-other' WHERE id = ?`, row.ID); err != nil {
			t.Fatalf("set mismatched category: %v", err)
		}
		_, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
			LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
			Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
		})
		assertAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeCategoryTypeMismatch)
	})

	t.Run("tag overflow", func(t *testing.T) {
		database, service, batch, row := newTask534BLearnFixture(t)
		if _, err := database.Exec(`UPDATE import_items SET selected_tag_ids_json = '["1","2","3","4","5","6","7","8","9"]' WHERE id = ?`, row.ID); err != nil {
			t.Fatalf("set overflowing tags: %v", err)
		}
		_, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
			LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
			Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
		})
		assertAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeTagLimitExceeded)
	})
}

func TestTask534BLearnScopeAndRestoreRespectManualRules(t *testing.T) {
	database, service, batch := newTask534ABulkFixture(t)
	row := findPreviewRowByNumber(t, batch, 1)
	rowStatus := RowStatusAdjusted
	categoryID := "cat-travel"
	if _, err := service.UpdatePreviewRow(context.Background(), UpdateRowCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Patch: UpdateRowRequest{RowStatus: &rowStatus, SelectedCategoryID: &categoryID, SelectedTagIDs: []string{}},
	}); err != nil {
		t.Fatalf("save row for scope test: %v", err)
	}
	learned, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeAll},
	})
	if err != nil {
		t.Fatalf("current-source manual rule must not block all-source learning: %v", err)
	}
	if _, err := service.ArchiveImportRule(context.Background(), ownerLedgerContext(), learned.RuleID); err != nil {
		t.Fatalf("archive all-source learned rule: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO import_rules (
			id, ledger_id, keyword, created_by_user_id, name, match_type, pattern,
			priority, result_json, status, origin, source_type, apply_mode, confidence,
			created_at, updated_at
		) VALUES (
			'rule-breakfast-all', 'ledger-one', '早餐店', 'owner-user', '全来源早餐规则',
			'merchant_equals', '早餐店', 5, '{"category_id":"cat-food"}',
			'active', 'manual', NULL, 'auto', 'high',
			'2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed all-source manual conflict: %v", err)
	}
	_, err = service.RestoreImportRule(context.Background(), ownerLedgerContext(), learned.RuleID)
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeClassificationConflict)
	rule, readErr := service.repo.GetImportRule(context.Background(), "ledger-one", learned.RuleID)
	if readErr != nil {
		t.Fatalf("read blocked restored rule: %v", readErr)
	}
	if rule.Status != "archived" {
		t.Fatalf("manual conflict restored learned rule: %+v", rule)
	}
}

func TestTask534BRepositoryRejectsConcurrentRowChange(t *testing.T) {
	database, service, batch, row := newTask534BLearnFixture(t)
	snapshot, err := service.repo.LoadLearnMerchantSnapshot(context.Background(), "ledger-one", batch.ID, row.ID)
	if err != nil {
		t.Fatalf("load expected learn snapshot: %v", err)
	}
	normalized := "园区餐厅"
	sourceType := SourceTypeGeneric
	spec := learnedRuleSpec{
		RuleID:             learnedMerchantRuleID("ledger-one", &sourceType, normalized),
		NormalizedMerchant: normalized, SourceScope: LearnSourceScopeCurrent, SourceType: &sourceType,
		Result: ImportRuleResult{CategoryID: snapshot.SelectedCategoryID, TagIDs: copyStrings(snapshot.SelectedTagIDs)},
	}
	if _, err := database.Exec(`UPDATE import_items SET selected_category_id = 'cat-travel' WHERE id = ?`, row.ID); err != nil {
		t.Fatalf("inject concurrent row change: %v", err)
	}
	_, err = service.repo.UpsertLearnedMerchantRule(context.Background(), ownerLedgerContext(), snapshot, spec)
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeClassificationRuleStale)
	if countWhere(t, database, "import_rules", "origin = 'learned'") != 0 || countWhere(t, database, "audit_logs", "action = 'import_rule_learn'") != 0 {
		t.Fatalf("concurrent row change left a learned rule or audit")
	}
}

func TestTask534BLearnRejectsInvalidScopeCategoryTypeAndTagLimit(t *testing.T) {
	database, service, batch, row := newTask534BLearnFixture(t)

	_, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: "other"},
	})
	assertAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeValidationError)

	if _, err := database.Exec(`
		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, color, is_archived, created_at, updated_at
		) VALUES (
			'cat-income-learn', 'ledger-one', 'owner-user', '测试收入', 'income', '#64748b', 0,
			'2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		);
		UPDATE import_items SET selected_category_id = 'cat-income-learn' WHERE id = ?;
	`, row.ID); err != nil {
		t.Fatalf("prepare category mismatch: %v", err)
	}
	_, err = service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	assertAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeCategoryTypeMismatch)

	if _, err := database.Exec(`
		UPDATE import_items
		SET selected_category_id = 'cat-food',
		    selected_tag_ids_json = '["tag-1","tag-2","tag-3","tag-4","tag-5","tag-6","tag-7","tag-8","tag-9"]'
		WHERE id = ?
	`, row.ID); err != nil {
		t.Fatalf("prepare tag limit violation: %v", err)
	}
	_, err = service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	assertAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeTagLimitExceeded)
	if countWhere(t, database, "import_rules", "origin = 'learned'") != 0 {
		t.Fatalf("invalid learned results created a rule")
	}
}

func TestTask534BLearnRepositoryRejectsConcurrentRowChange(t *testing.T) {
	database, service, batch, row := newTask534BLearnFixture(t)
	snapshot, err := service.repo.LoadLearnMerchantSnapshot(context.Background(), "ledger-one", batch.ID, row.ID)
	if err != nil {
		t.Fatalf("load expected learn snapshot: %v", err)
	}
	normalizedMerchant := "园区餐厅"
	sourceType := SourceTypeGeneric
	spec := learnedRuleSpec{
		RuleID:             learnedMerchantRuleID("ledger-one", &sourceType, normalizedMerchant),
		NormalizedMerchant: normalizedMerchant,
		SourceScope:        LearnSourceScopeCurrent,
		SourceType:         &sourceType,
		Result:             ImportRuleResult{CategoryID: snapshot.SelectedCategoryID, TagIDs: copyStrings(snapshot.SelectedTagIDs)},
	}
	if _, err := database.Exec(`UPDATE import_items SET selected_category_id = 'cat-travel' WHERE id = ?`, row.ID); err != nil {
		t.Fatalf("inject concurrent row change: %v", err)
	}
	_, err = service.repo.UpsertLearnedMerchantRule(context.Background(), ownerLedgerContext(), snapshot, spec)
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeClassificationRuleStale)
	if countWhere(t, database, "import_rules", "origin = 'learned'") != 0 ||
		countWhere(t, database, "audit_logs", "action = 'import_rule_learn'") != 0 {
		t.Fatalf("concurrent row change left a rule or audit")
	}
}

func TestTask534BLearnRollsBackRuleWhenAuditFailsButKeepsPriorRowSave(t *testing.T) {
	database, service, batch, row := newTask534BLearnFixture(t)
	if _, err := database.Exec(`
		CREATE TRIGGER fail_learn_audit BEFORE INSERT ON audit_logs
		WHEN NEW.action = 'import_rule_learn'
		BEGIN
			SELECT RAISE(ABORT, 'injected learn audit failure');
		END;
	`); err != nil {
		t.Fatalf("create learn audit failure trigger: %v", err)
	}

	_, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	assertAppError(t, err, http.StatusInternalServerError, appErrors.ErrCodeInternalError)
	if countWhere(t, database, "import_rules", "origin = 'learned'") != 0 || countWhere(t, database, "audit_logs", "action = 'import_rule_learn'") != 0 {
		t.Fatalf("audit failure left a partial learned rule or audit")
	}
	stored, readErr := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if readErr != nil {
		t.Fatalf("read saved row after learn rollback: %v", readErr)
	}
	if findPreviewRow(t, stored, row.ID).Classification.Status != ClassificationStatusManual {
		t.Fatalf("learn rollback changed the separately saved row")
	}
}

func TestTask534BLearnedRuleLifecycleAndManualRuleDTO(t *testing.T) {
	_, service, batch, row := newTask534BLearnFixture(t)
	learned, err := service.LearnMerchantRule(context.Background(), LearnMerchantCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Request: LearnMerchantRequest{SourceScope: LearnSourceScopeCurrent},
	})
	if err != nil {
		t.Fatalf("create learned rule: %v", err)
	}
	applyMode := "suggest"
	updated, err := service.UpdateImportRule(context.Background(), ownerLedgerContext(), learned.RuleID, ImportRuleUpsertRequest{
		Name: "园区规则", MatchType: "merchant_equals", Pattern: "  园区餐厅  ",
		Priority: intPointer(220), ApplyMode: &applyMode,
		Result: ImportRuleResult{CategoryID: "cat-travel", TagIDs: []string{}},
	})
	if err != nil {
		t.Fatalf("update learned editable fields: %v", err)
	}
	if updated.Origin != "learned" || updated.SourceType == nil || *updated.SourceType != SourceTypeGeneric || updated.ApplyMode != "suggest" || updated.Confidence != "high" || updated.Pattern != "园区餐厅" {
		t.Fatalf("learned immutable fields changed unexpectedly: %+v", updated)
	}
	_, err = service.UpdateImportRule(context.Background(), ownerLedgerContext(), learned.RuleID, ImportRuleUpsertRequest{
		MatchType: "merchant_contains", Pattern: "园区", Result: ImportRuleResult{CategoryID: "cat-travel"},
	})
	assertAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeValidationError)

	sourceType := SourceTypeAlipay
	autoMode := "auto"
	manual, err := service.CreateImportRule(context.Background(), ownerLedgerContext(), ImportRuleUpsertRequest{
		Name: "支付宝手工规则", MatchType: "merchant_contains", Pattern: "示例",
		SourceType: NullableString{Set: true, Value: &sourceType}, ApplyMode: &autoMode,
		Result: ImportRuleResult{CategoryID: "cat-food"},
	})
	if err != nil {
		t.Fatalf("create manual rule with Task53 fields: %v", err)
	}
	if manual.Origin != "manual" || manual.SourceType == nil || *manual.SourceType != SourceTypeAlipay || manual.ApplyMode != "auto" || manual.Confidence != "high" {
		t.Fatalf("manual rule DTO did not expose Task53 lifecycle fields: %+v", manual)
	}
}

func TestTask534BLearnHTTPUsesOnlySavedRowAndRejectsUnknownFields(t *testing.T) {
	_, service, batch, row := newTask534BLearnFixture(t)

	request := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("batchID", batch.ID)
		routeContext.URLParams.Add("rowID", row.ID)
		req := httptest.NewRequest(http.MethodPost, "/api/imports/"+batch.ID+"/rows/"+row.ID+"/learn", strings.NewReader(body))
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeContext)
		ctx = context.WithValue(ctx, middleware.UserIDKey, "owner-user")
		ctx = ledger.ContextWithLedgerContext(ctx, ownerLedgerContext())
		recorder := httptest.NewRecorder()
		NewHandler(service).HandleLearnMerchant(recorder, req.WithContext(ctx))
		return recorder
	}

	rejected := request(`{"source_scope":"current_source","category_id":"cat-travel"}`)
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("learn accepted client classification fields: status=%d body=%s", rejected.Code, rejected.Body.String())
	}
	accepted := request(`{"source_scope":"current_source"}`)
	if accepted.Code != http.StatusOK {
		t.Fatalf("learn HTTP status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	var body struct {
		Success bool                `json:"success"`
		Data    LearnMerchantResult `json:"data"`
	}
	if err := json.Unmarshal(accepted.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode learn response: %v", err)
	}
	if !body.Success || body.Data.Action != LearnActionCreated || body.Data.RuleID == "" {
		t.Fatalf("unexpected learn response: %+v", body)
	}
}

func newTask534BLearnFixture(t *testing.T) (*sql.DB, *Service, *PreviewBatch, PreviewRow) {
	t.Helper()
	database, service, batch := newTask534ABulkFixture(t)
	row := findPreviewRowByNumber(t, batch, 2)
	rowStatus := RowStatusAdjusted
	categoryID := "cat-food"
	accountID := "account-cash"
	visibility := "partner_readable"
	if _, err := service.UpdatePreviewRow(context.Background(), UpdateRowCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Patch: UpdateRowRequest{
			RowStatus: &rowStatus, SelectedCategoryID: &categoryID, SelectedAccountID: &accountID,
			SelectedTagIDs: []string{"tag-breakfast"}, Visibility: &visibility,
		},
	}); err != nil {
		t.Fatalf("prepare saved learn row: %v", err)
	}
	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("read saved learn fixture: %v", err)
	}
	return database, service, stored, findPreviewRow(t, stored, row.ID)
}

func assertTask534BLearnedRule(t *testing.T, rule *ImportRuleResponse, categoryID string, tagIDs []string, sourceType string) {
	t.Helper()
	if rule.Origin != "learned" || rule.MatchType != "merchant_equals" || rule.Pattern != "园区餐厅" || rule.ApplyMode != "auto" || rule.Confidence != "high" || rule.Status != "active" {
		t.Fatalf("unexpected learned rule fields: %+v", rule)
	}
	if sourceType == "" {
		if rule.SourceType != nil {
			t.Fatalf("expected all-source learned rule: %+v", rule)
		}
	} else if rule.SourceType == nil || *rule.SourceType != sourceType {
		t.Fatalf("unexpected learned source: %+v", rule)
	}
	if rule.Result.CategoryID != categoryID {
		t.Fatalf("learned category=%q want=%q", rule.Result.CategoryID, categoryID)
	}
	assertStringList(t, rule.Result.TagIDs, tagIDs)
}

func intPointer(value int) *int {
	return &value
}
