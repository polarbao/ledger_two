package importer

import (
	"context"
	"strings"
	"testing"

	"ledger_two/internal/importer/classifier"
)

func TestTask532LoadClassificationContextIsLedgerScopedAndPreservesHistoricalRuleMode(t *testing.T) {
	database := openImporterTestDB(t)
	_, err := database.Exec(`
		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, system_key, is_archived, created_at, updated_at
		) VALUES
			('cat-a-food', 'ledger-one', 'owner-user', '餐饮', 'expense', 'expense_food', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('cat-a-other', 'ledger-one', 'owner-user', '其他支出', 'expense', 'expense_other', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('cat-b-food', 'ledger-two', 'owner-user', '餐饮', 'expense', 'expense_food', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO tags (
			id, ledger_id, owner_user_id, name, system_key, is_archived, created_at, updated_at
		) VALUES
			('tag-a-takeout', 'ledger-one', 'owner-user', '外卖', 'tag_takeout', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('tag-a-old', 'ledger-one', 'owner-user', '旧标签', NULL, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('tag-b-takeout', 'ledger-two', 'owner-user', '外卖', 'tag_takeout', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO accounts (
			id, ledger_id, owner_user_id, name, type, currency, is_archived, created_at, updated_at
		) VALUES ('account-a', 'ledger-one', 'owner-user', '零钱', 'cash', 'CNY', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO import_rules (
			id, ledger_id, keyword, created_by_user_id, name, match_type, pattern,
			priority, result_json, status, origin, source_type, apply_mode, confidence,
			created_at, updated_at
		) VALUES
			('rule-history', 'ledger-one', '咖啡', 'owner-user', '历史规则', 'merchant_contains', '咖啡',
			 100, '{"category_id":"cat-a-food","account_id":"account-a","tag_ids":["tag-a-takeout"]}',
			 'active', 'manual', NULL, 'suggest', 'high', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z'),
			('rule-learned', 'ledger-one', '星河咖啡', 'owner-user', '学习规则', 'merchant_equals', '星河咖啡',
			 500, '{"category_id":"cat-a-food","tag_ids":[]}',
			 'active', 'learned', 'wechat', 'auto', 'high', '2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z'),
			('rule-archived', 'ledger-one', '旧店', 'owner-user', '归档规则', 'merchant_contains', '旧店',
			 1, '{"category_id":"cat-a-food","tag_ids":[]}',
			 'archived', 'manual', NULL, 'auto', 'high', '2026-01-04T00:00:00Z', '2026-01-04T00:00:00Z'),
			('rule-b', 'ledger-two', '咖啡', 'owner-user', 'B规则', 'merchant_contains', '咖啡',
			 1, '{"category_id":"cat-b-food","tag_ids":["tag-b-takeout"]}',
			 'active', 'manual', NULL, 'auto', 'high', '2026-01-05T00:00:00Z', '2026-01-05T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("seed classification context: %v", err)
	}

	ctx, err := NewRepository(database).LoadClassificationContext(context.Background(), "ledger-one")
	if err != nil {
		t.Fatalf("LoadClassificationContext returned error: %v", err)
	}
	if ctx.LedgerID != "ledger-one" || len(ctx.Rules) != 2 || len(ctx.Metadata) != 5 {
		t.Fatalf("unexpected classification context: ledger=%s rules=%d metadata=%d", ctx.LedgerID, len(ctx.Rules), len(ctx.Metadata))
	}
	if len(ctx.Builtins) != 3 {
		t.Fatalf("builtins = %d, want 3", len(ctx.Builtins))
	}
	for _, item := range ctx.Metadata {
		if item.LedgerID != "ledger-one" {
			t.Fatalf("cross-ledger metadata leaked: %+v", item)
		}
	}
	history := ctx.Rules[0]
	if history.ID != "rule-history" || history.Origin != classifier.OriginManual || history.ApplyMode != classifier.ApplyModeSuggest || history.Confidence != classifier.ConfidenceHigh {
		t.Fatalf("historical rule compatibility lost: %+v", history)
	}
	if history.Result.CategoryID != "cat-a-food" || history.Result.AccountID != "account-a" || len(history.Result.TagIDs) != 1 {
		t.Fatalf("historical rule result mismatch: %+v", history.Result)
	}
	learned := ctx.Rules[1]
	if learned.MatchType != classifier.MatchMerchantEquals || learned.Origin != classifier.OriginLearned || learned.SourceType != "wechat" || learned.ApplyMode != classifier.ApplyModeAuto {
		t.Fatalf("learned rule fields mismatch: %+v", learned)
	}
}

func TestTask532LoadClassificationContextRejectsInvalidRuleJSON(t *testing.T) {
	database := openImporterTestDB(t)
	_, err := database.Exec(`
		INSERT INTO import_rules (
			id, ledger_id, keyword, created_by_user_id, name, match_type, pattern,
			priority, result_json, status, origin, apply_mode, confidence, created_at, updated_at
		) VALUES (
			'rule-invalid', 'ledger-one', '坏规则', 'owner-user', '坏规则', 'merchant_contains', '坏规则',
			100, '{invalid', 'active', 'manual', 'suggest', 'high', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		)
	`)
	if err != nil {
		t.Fatalf("seed invalid classification rule: %v", err)
	}

	_, err = NewRepository(database).LoadClassificationContext(context.Background(), "ledger-one")
	if err == nil || !strings.Contains(err.Error(), "rule-invalid") {
		t.Fatalf("expected rule id in invalid JSON error, got %v", err)
	}
}

func TestTask532UnknownCategoryTypeCannotBecomeAnIncomeCandidate(t *testing.T) {
	database := openImporterTestDB(t)
	_, err := database.Exec(`
		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, system_key, is_archived, created_at, updated_at
		) VALUES
			('cat-invalid', 'ledger-one', 'owner-user', '损坏分类', 'unknown', NULL, 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('cat-income-other', 'ledger-one', 'owner-user', '其他收入', 'income', 'income_other', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO import_rules (
			id, ledger_id, keyword, created_by_user_id, name, match_type, pattern,
			priority, result_json, status, origin, apply_mode, confidence, created_at, updated_at
		) VALUES (
			'rule-invalid-type', 'ledger-one', '测试收入', 'owner-user', '损坏分类规则', 'merchant_contains', '测试收入',
			1, '{"category_id":"cat-invalid","tag_ids":[]}', 'active', 'manual', 'auto', 'high',
			'2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		)
	`)
	if err != nil {
		t.Fatalf("seed unknown category type: %v", err)
	}

	classificationContext, err := NewRepository(database).LoadClassificationContext(context.Background(), "ledger-one")
	if err != nil {
		t.Fatalf("LoadClassificationContext returned error: %v", err)
	}
	result := classifier.Classify(classificationContext, classifier.Row{
		LedgerID: "ledger-one", SourceType: "wechat", Merchant: "测试收入商户",
		Direction: "income", TargetTransactionType: "income", DuplicateStatus: "new", RowStatus: "pending",
	})
	if result.Decision.Status != classifier.StatusFallback || result.Decision.SelectedCategoryID != "cat-income-other" {
		t.Fatalf("unknown category type became an income candidate: %+v", result.Decision)
	}
}
