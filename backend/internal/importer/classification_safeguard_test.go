package importer

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/importer/classifier"
)

func TestTask534CRuleDTOReportsStaleReferencesAndCommittedHitMetrics(t *testing.T) {
	database, service := newTask534CRuleFixture(t)

	rules, err := service.ListImportRules(context.Background(), ownerLedgerContext(), "all")
	if err != nil {
		t.Fatalf("list Task53.4C rules: %v", err)
	}
	valid := findTask534CRule(t, rules, "rule-valid")
	if valid.IsStale || len(valid.StaleReferenceIDs) != 0 {
		t.Fatalf("valid rule reported stale references: %+v", valid)
	}
	if valid.CommittedHitCount != 2 || valid.LastCommittedHitAt == nil || *valid.LastCommittedHitAt != "2026-07-20T02:00:00Z" {
		t.Fatalf("committed hit metrics included preview/skipped/cross-ledger rows or lost the latest time: %+v", valid)
	}

	stale := findTask534CRule(t, rules, "rule-stale")
	if !stale.IsStale {
		t.Fatalf("stale rule was not marked stale: %+v", stale)
	}
	assertStringList(t, stale.StaleReferenceIDs, []string{"account-missing", "cat-archived", "tag-missing"})
	if stale.CommittedHitCount != 0 || stale.LastCommittedHitAt != nil {
		t.Fatalf("stale rule received uncommitted metrics: %+v", stale)
	}

	loaded, err := service.repo.GetImportRule(context.Background(), "ledger-one", "rule-valid")
	if err != nil {
		t.Fatalf("get Task53.4C rule: %v", err)
	}
	if loaded.CommittedHitCount != valid.CommittedHitCount || loaded.LastCommittedHitAt == nil || *loaded.LastCommittedHitAt != *valid.LastCommittedHitAt {
		t.Fatalf("get/list rule metrics drifted: get=%+v list=%+v", loaded, valid)
	}

	if countWhere(t, database, "import_items", "status = 'imported'") != 5 {
		t.Fatalf("metrics read changed import item state")
	}
}

func TestTask534CStaleRuleDoesNotProduceCandidateAndRestoreRequiresRepair(t *testing.T) {
	_, service := newTask534CRuleFixture(t)
	classificationContext, err := service.repo.LoadClassificationContext(context.Background(), "ledger-one")
	if err != nil {
		t.Fatalf("load stale classification context: %v", err)
	}
	result := classifier.Classify(classificationContext, classifier.Row{
		LedgerID: "ledger-one", SourceType: SourceTypeGeneric, Merchant: "坏商户",
		Direction: "expense", TargetTransactionType: TargetTransactionExpense,
		DuplicateStatus: DuplicateStatusNew, RowStatus: RowStatusPending,
	})
	if result.Decision.Status != classifier.StatusFallback || result.Decision.SelectedCategoryID != "cat-expense-other" {
		t.Fatalf("stale rule produced a candidate instead of fallback: %+v", result.Decision)
	}
	for _, ruleID := range result.Decision.MatchedRuleIDs {
		if ruleID == "rule-stale" {
			t.Fatalf("stale rule leaked into matched_rule_ids: %+v", result.Decision)
		}
	}

	if _, err := service.ArchiveImportRule(context.Background(), ownerLedgerContext(), "rule-stale"); err != nil {
		t.Fatalf("archive stale rule: %v", err)
	}
	_, err = service.RestoreImportRule(context.Background(), ownerLedgerContext(), "rule-stale")
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeClassificationRuleStale)

	autoMode := string(classifier.ApplyModeAuto)
	repaired, err := service.UpdateImportRule(context.Background(), ownerLedgerContext(), "rule-stale", ImportRuleUpsertRequest{
		Name: "已修复规则", MatchType: string(classifier.MatchMerchantEquals), Pattern: "坏商户",
		ApplyMode: &autoMode,
		Result:    ImportRuleResult{CategoryID: "cat-food", TagIDs: []string{"tag-breakfast"}},
	})
	if err != nil {
		t.Fatalf("repair stale archived rule: %v", err)
	}
	if repaired.IsStale || len(repaired.StaleReferenceIDs) != 0 {
		t.Fatalf("repaired rule remained stale: %+v", repaired)
	}
	restored, err := service.RestoreImportRule(context.Background(), ownerLedgerContext(), "rule-stale")
	if err != nil {
		t.Fatalf("restore repaired rule: %v", err)
	}
	if restored.Status != "active" || restored.IsStale {
		t.Fatalf("repaired rule was not restored: %+v", restored)
	}
}

func TestTask534CRestoreRechecksReferencesInsideStatusTransaction(t *testing.T) {
	database, service := newTask534CRuleFixture(t)
	if _, err := database.Exec(`
		UPDATE import_rules SET status = 'archived', archived_at = '2026-07-20T04:00:00Z'
		WHERE id = 'rule-valid';
		CREATE TRIGGER task53_archive_reference_during_restore
		BEFORE UPDATE OF status ON import_rules
		WHEN OLD.id = 'rule-valid' AND NEW.status = 'active'
		BEGIN
			UPDATE categories SET is_archived = 1 WHERE id = 'cat-food';
		END;
	`); err != nil {
		t.Fatalf("prepare restore race fixture: %v", err)
	}

	_, err := service.RestoreImportRule(context.Background(), ownerLedgerContext(), "rule-valid")
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeClassificationRuleStale)

	var ruleStatus string
	var categoryArchived, restoreAuditCount int
	if err := database.QueryRow("SELECT status FROM import_rules WHERE id = 'rule-valid'").Scan(&ruleStatus); err != nil {
		t.Fatalf("read rule status after rejected restore: %v", err)
	}
	if err := database.QueryRow("SELECT is_archived FROM categories WHERE id = 'cat-food'").Scan(&categoryArchived); err != nil {
		t.Fatalf("read category status after rejected restore: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE action = 'import_rule_restore' AND entity_id = 'rule-valid'").Scan(&restoreAuditCount); err != nil {
		t.Fatalf("count restore audits: %v", err)
	}
	if ruleStatus != "archived" || categoryArchived != 0 || restoreAuditCount != 0 {
		t.Fatalf("rejected restore was not atomic: rule=%q category_archived=%d audits=%d", ruleStatus, categoryArchived, restoreAuditCount)
	}
}

func TestTask534CRestoreRollsBackStatusWhenAuditFails(t *testing.T) {
	database, service := newTask534CRuleFixture(t)
	if _, err := database.Exec(`
		UPDATE import_rules SET status = 'archived', archived_at = '2026-07-20T04:00:00Z'
		WHERE id = 'rule-valid';
		CREATE TRIGGER task53_fail_restore_audit
		BEFORE INSERT ON audit_logs
		WHEN NEW.action = 'import_rule_restore' AND NEW.entity_id = 'rule-valid'
		BEGIN
			SELECT RAISE(ABORT, 'injected restore audit failure');
		END;
	`); err != nil {
		t.Fatalf("prepare restore audit failure: %v", err)
	}

	if _, err := service.RestoreImportRule(context.Background(), ownerLedgerContext(), "rule-valid"); err == nil {
		t.Fatal("expected restore audit failure")
	}
	var status string
	if err := database.QueryRow("SELECT status FROM import_rules WHERE id = 'rule-valid'").Scan(&status); err != nil {
		t.Fatalf("read rule status after audit failure: %v", err)
	}
	if status != "archived" {
		t.Fatalf("restore status survived audit rollback: %q", status)
	}
}

func newTask534CRuleFixture(t *testing.T) (*sql.DB, *Service) {
	t.Helper()
	database := openImporterTestDB(t)
	seedTask533ClassificationData(t, database)
	if _, err := database.Exec(`
		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, color, is_archived, created_at, updated_at
		) VALUES (
			'cat-archived', 'ledger-one', 'owner-user', '归档分类', 'expense', '#94a3b8', 1,
			'2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z'
		);
		INSERT INTO import_rules (
			id, ledger_id, keyword, created_by_user_id, name, match_type, pattern,
			priority, result_json, status, origin, source_type, apply_mode, confidence,
			created_at, updated_at
		) VALUES
			('rule-valid', 'ledger-one', '好商户', 'owner-user', '有效规则', 'merchant_equals', '好商户',
			 20, '{"category_id":"cat-food","tag_ids":["tag-breakfast"]}', 'active', 'manual', 'generic', 'auto', 'high',
			 '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z'),
			('rule-stale', 'ledger-one', '坏商户', 'owner-user', '失效规则', 'merchant_equals', '坏商户',
			 21, '{"category_id":"cat-archived","account_id":"account-missing","tag_ids":["tag-missing"]}', 'active', 'manual', 'generic', 'auto', 'high',
			 '2026-07-20T00:00:00Z', '2026-07-20T00:00:00Z');
		INSERT INTO import_batches (
			id, ledger_id, filename, created_by_user_id, status, committed_at, created_at
		) VALUES
			('batch-committed-1', 'ledger-one', 'one.csv', 'owner-user', 'committed', '2026-07-20T01:00:00Z', '2026-07-20T00:00:00Z'),
			('batch-committed-2', 'ledger-one', 'two.csv', 'owner-user', 'committed', '2026-07-20T02:00:00Z', '2026-07-20T00:00:00Z'),
			('batch-ready', 'ledger-one', 'ready.csv', 'owner-user', 'ready', NULL, '2026-07-20T00:00:00Z'),
			('batch-foreign', 'ledger-two', 'foreign.csv', 'owner-user', 'committed', '2026-07-20T03:00:00Z', '2026-07-20T00:00:00Z');
		INSERT INTO import_items (
			id, batch_id, import_hash, status, matched_rule_ids_json, created_at
		) VALUES
			('item-hit-1', 'batch-committed-1', 'hash-hit-1', 'imported', '["rule-valid"]', '2026-07-20T01:00:00Z'),
			('item-hit-2', 'batch-committed-2', 'hash-hit-2', 'imported', '["rule-valid","rule-valid"]', '2026-07-20T02:00:00Z'),
			('item-ready', 'batch-ready', 'hash-ready', 'imported', '["rule-valid"]', '2026-07-20T00:00:00Z'),
			('item-skipped', 'batch-committed-2', 'hash-skipped', 'skipped', '["rule-valid"]', '2026-07-20T02:00:00Z'),
			('item-foreign', 'batch-foreign', 'hash-foreign', 'imported', '["rule-valid"]', '2026-07-20T03:00:00Z'),
			('item-no-hit', 'batch-committed-2', 'hash-no-hit', 'imported', '[]', '2026-07-20T02:00:00Z');
	`); err != nil {
		t.Fatalf("seed Task53.4C rule fixture: %v", err)
	}
	return database, NewService(NewRepository(database), WithClassificationMode(ClassificationModeGraded))
}

func findTask534CRule(t *testing.T, rules []ImportRuleResponse, id string) ImportRuleResponse {
	t.Helper()
	for _, rule := range rules {
		if rule.ID == id {
			return rule
		}
	}
	t.Fatalf("rule %s not found", id)
	return ImportRuleResponse{}
}
