package classifier

import (
	"reflect"
	"testing"
)

func TestTask532AnonymousFixtureMatrix(t *testing.T) {
	contextA := fixtureContext("ledger-a")
	contextB := fixtureContext("ledger-b")

	tests := []struct {
		name              string
		ctx               Context
		row               Row
		wantEvaluated     bool
		wantStatus        ClassificationStatus
		wantCategoryID    string
		wantSuggestedTags []string
		wantSelectedTags  []string
	}{
		{
			name:          "CT-R01 user rule auto selects transport and commute",
			ctx:           contextA,
			row:           fixtureExpenseRow("滴滴出行", "行程"),
			wantEvaluated: true, wantStatus: StatusAutoSelected,
			wantCategoryID: "cat-a-transport", wantSelectedTags: []string{"tag-a-commute"},
		},
		{
			name:          "CT-R02 learned exact rule auto selects food",
			ctx:           contextA,
			row:           fixtureExpenseRow(" 星河咖啡　", "咖啡"),
			wantEvaluated: true, wantStatus: StatusAutoSelected, wantCategoryID: "cat-a-food",
		},
		{
			name:          "CT-R03 builtin takeout only suggests",
			ctx:           contextA,
			row:           fixtureExpenseRow("美团外卖订单", "晚餐"),
			wantEvaluated: true, wantStatus: StatusSuggested,
			wantCategoryID: "cat-a-food", wantSuggestedTags: []string{"tag-a-takeout"},
		},
		{
			name:          "CT-R04 unknown expense falls back",
			ctx:           contextA,
			row:           fixtureExpenseRow("未知商户甲", "普通消费"),
			wantEvaluated: true, wantStatus: StatusFallback, wantCategoryID: "cat-a-other-expense",
		},
		{
			name:          "CT-R05 salary builtin only suggests",
			ctx:           contextA,
			row:           Row{LedgerID: "ledger-a", SourceType: "alipay", Title: "工资发放", Direction: "income", TargetTransactionType: "income", DuplicateStatus: "new", RowStatus: "pending"},
			wantEvaluated: true, wantStatus: StatusSuggested, wantCategoryID: "cat-a-salary",
		},
		{
			name:          "CT-R06 refund builtin only suggests",
			ctx:           contextA,
			row:           Row{LedgerID: "ledger-a", SourceType: "wechat", Title: "原路退款", Direction: "refund", TargetTransactionType: "income", DuplicateStatus: "new", RowStatus: "pending"},
			wantEvaluated: true, wantStatus: StatusSuggested, wantCategoryID: "cat-a-refund",
		},
		{
			name:          "CT-R07 equal rank category rules conflict",
			ctx:           contextA,
			row:           fixtureExpenseRow("冲突商户", "冲突订单"),
			wantEvaluated: true, wantStatus: StatusConflict,
		},
		{
			name:          "CT-R08 archived tag invalidates the whole rule",
			ctx:           contextA,
			row:           fixtureExpenseRow("旧店", "旧订单"),
			wantEvaluated: true, wantStatus: StatusFallback, wantCategoryID: "cat-a-other-expense",
		},
		{
			name:          "CT-R09 empty merchant falls back",
			ctx:           contextA,
			row:           fixtureExpenseRow("", "普通消费"),
			wantEvaluated: true, wantStatus: StatusFallback, wantCategoryID: "cat-a-other-expense",
		},
		{
			name:          "CT-R10 duplicate does not enter classifier",
			ctx:           contextA,
			row:           Row{LedgerID: "ledger-a", TargetTransactionType: "expense", DuplicateStatus: "duplicate", RowStatus: "pending"},
			wantEvaluated: false,
		},
		{
			name:          "CT-R11 invalid row does not enter classifier",
			ctx:           contextA,
			row:           Row{LedgerID: "ledger-a", TargetTransactionType: "expense", DuplicateStatus: "invalid", RowStatus: "skipped"},
			wantEvaluated: false,
		},
		{
			name:          "CT-R12 ledger A rules do not leak to ledger B",
			ctx:           contextB,
			row:           fixtureExpenseRowForLedger("ledger-b", "滴滴出行", "行程"),
			wantEvaluated: true, wantStatus: StatusFallback, wantCategoryID: "cat-b-other-expense",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Classify(test.ctx, test.row)
			if result.Evaluated != test.wantEvaluated {
				t.Fatalf("evaluated = %v, want %v", result.Evaluated, test.wantEvaluated)
			}
			if !test.wantEvaluated {
				return
			}
			if result.Decision.Status != test.wantStatus {
				t.Fatalf("status = %q, want %q; decision=%+v", result.Decision.Status, test.wantStatus, result.Decision)
			}
			categoryID := result.Decision.SelectedCategoryID
			if categoryID == "" {
				categoryID = result.Decision.SuggestedCategoryID
			}
			if categoryID != test.wantCategoryID {
				t.Fatalf("category = %q, want %q; decision=%+v", categoryID, test.wantCategoryID, result.Decision)
			}
			if !reflect.DeepEqual(result.Decision.SuggestedTagIDs, test.wantSuggestedTags) {
				t.Fatalf("suggested tags = %v, want %v", result.Decision.SuggestedTagIDs, test.wantSuggestedTags)
			}
			if !reflect.DeepEqual(result.Decision.SelectedTagIDs, test.wantSelectedTags) {
				t.Fatalf("selected tags = %v, want %v", result.Decision.SelectedTagIDs, test.wantSelectedTags)
			}
		})
	}
}

func TestHistoricalSuggestRuleNeverWritesSelectedMetadata(t *testing.T) {
	ctx := fixtureContext("ledger-a")
	ctx.Rules = []Rule{{
		ID: "history", LedgerID: "ledger-a", Origin: OriginManual, ApplyMode: ApplyModeSuggest,
		Confidence: ConfidenceHigh, MatchType: MatchMerchantContains, Pattern: "咖啡", Priority: 1,
		Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-food", TagIDs: []string{"tag-a-takeout"}},
	}}
	result := Classify(ctx, fixtureExpenseRow("咖啡店", "咖啡"))
	if result.Decision.Status != StatusSuggested || result.Decision.SelectedCategoryID != "" || len(result.Decision.SelectedTagIDs) != 0 {
		t.Fatalf("historical suggest rule changed behavior: %+v", result.Decision)
	}
	if result.Decision.SuggestedCategoryID != "cat-a-food" || !reflect.DeepEqual(result.Decision.SuggestedTagIDs, []string{"tag-a-takeout"}) {
		t.Fatalf("historical suggestion missing: %+v", result.Decision)
	}
}

func TestUserSuggestionAlwaysPrecedesBuiltinRegardlessOfRulePriority(t *testing.T) {
	ctx := fixtureContext("ledger-a")
	ctx.Rules = []Rule{{
		ID: "low-priority-user-rule", LedgerID: "ledger-a", Origin: OriginManual, ApplyMode: ApplyModeSuggest,
		Confidence: ConfidenceHigh, MatchType: MatchMerchantContains, Pattern: "外卖", Priority: 9999,
		Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-transport"},
	}}
	result := Classify(ctx, fixtureExpenseRow("外卖商户", "外卖订单"))
	if result.Decision.Status != StatusSuggested || result.Decision.SuggestedCategoryID != "cat-a-transport" || result.Decision.Source != SourceUserRule {
		t.Fatalf("builtin outranked explicit user rule: %+v", result.Decision)
	}
	if len(result.Decision.SuggestedTagIDs) != 0 {
		t.Fatalf("builtin tags leaked into an explicit user suggestion: %+v", result.Decision)
	}
}

func TestEqualsPrecedesContainsAtTheSameRuleRank(t *testing.T) {
	ctx := fixtureContext("ledger-a")
	ctx.Rules = []Rule{
		{ID: "contains", LedgerID: "ledger-a", Origin: OriginManual, ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantContains, Pattern: "咖啡", Priority: 10, Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-transport"}},
		{ID: "equals", LedgerID: "ledger-a", Origin: OriginManual, ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantEquals, Pattern: "星河咖啡", Priority: 10, Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-food"}},
	}
	result := Classify(ctx, fixtureExpenseRow("星河咖啡", "咖啡"))
	if result.Decision.Status != StatusAutoSelected || result.Decision.SelectedCategoryID != "cat-a-food" {
		t.Fatalf("merchant_equals did not precede contains: %+v", result.Decision)
	}
}

func TestRuleSourceScopeAndCrossLedgerMetadataAreRejected(t *testing.T) {
	ctx := fixtureContext("ledger-a")
	ctx.Metadata = append(ctx.Metadata, MetadataItem{ID: "cat-b-private", LedgerID: "ledger-b", Kind: MetadataExpenseCategory})
	ctx.Rules = []Rule{
		{ID: "wrong-source", LedgerID: "ledger-a", Origin: OriginManual, SourceType: "alipay", ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantContains, Pattern: "测试", Priority: 1, Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-food"}},
		{ID: "cross-ledger", LedgerID: "ledger-a", Origin: OriginManual, SourceType: "wechat", ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantContains, Pattern: "测试", Priority: 2, Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-b-private"}},
	}
	result := Classify(ctx, fixtureExpenseRow("测试商户", "测试"))
	if result.Decision.Status != StatusFallback || result.Decision.SelectedCategoryID != "cat-a-other-expense" {
		t.Fatalf("scoped or cross-ledger rule was applied: %+v", result.Decision)
	}
}

func TestBuiltinNegativeCasesRespectDirectionAndKeepSubjectsDistinct(t *testing.T) {
	ctx := fixtureContext("ledger-a")
	tests := []Row{
		fixtureExpenseRow("退款失败", "退款失败"),
		fixtureExpenseRow("工资卡还款", "工资卡还款"),
	}
	for _, row := range tests {
		result := Classify(ctx, row)
		if result.Decision.Status != StatusFallback {
			t.Fatalf("negative builtin case matched: row=%+v decision=%+v", row, result.Decision)
		}
	}

	income := Row{LedgerID: "ledger-a", SourceType: "wechat", Merchant: "外卖平台", Title: "外卖平台退款", Direction: "refund", TargetTransactionType: "income", DuplicateStatus: "new", RowStatus: "pending"}
	result := Classify(ctx, income)
	if result.Decision.Status != StatusSuggested || result.Decision.SuggestedCategoryID != "cat-a-refund" || len(result.Decision.SuggestedTagIDs) != 0 {
		t.Fatalf("refund should not inherit takeout classification: %+v", result.Decision)
	}
}

func TestResolverUsesHighestRuleGroupAndStableHighConfidenceTagUnion(t *testing.T) {
	ctx := fixtureContext("ledger-a")
	ctx.Metadata = append(ctx.Metadata,
		MetadataItem{ID: "tag-a-one", LedgerID: "ledger-a", Kind: MetadataTag},
		MetadataItem{ID: "tag-a-two", LedgerID: "ledger-a", Kind: MetadataTag},
	)
	ctx.Rules = []Rule{
		{ID: "lower", LedgerID: "ledger-a", Origin: OriginManual, ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantContains, Pattern: "商户", Priority: 20, CreatedAt: "2026-01-03T00:00:00Z", Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-transport", TagIDs: []string{"tag-a-two", "tag-a-one"}}},
		{ID: "top", LedgerID: "ledger-a", Origin: OriginManual, ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantEquals, Pattern: "测试商户", Priority: 10, CreatedAt: "2026-01-02T00:00:00Z", Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-food", TagIDs: []string{"tag-a-one"}}},
	}

	result := Classify(ctx, fixtureExpenseRow("测试商户", "订单"))
	if result.Decision.Status != StatusAutoSelected || result.Decision.SelectedCategoryID != "cat-a-food" {
		t.Fatalf("unexpected top rule decision: %+v", result.Decision)
	}
	if !reflect.DeepEqual(result.Decision.SelectedTagIDs, []string{"tag-a-one", "tag-a-two"}) {
		t.Fatalf("stable tag union = %v", result.Decision.SelectedTagIDs)
	}

	reversed := ctx
	reversed.Rules = []Rule{ctx.Rules[1], ctx.Rules[0]}
	if other := Classify(reversed, fixtureExpenseRow("测试商户", "订单")); !reflect.DeepEqual(result.Decision, other.Decision) {
		t.Fatalf("rule input order changed decision:\nfirst=%+v\nother=%+v", result.Decision, other.Decision)
	}
}

func TestResolverReturnsConflictInsteadOfTruncatingMoreThanEightTags(t *testing.T) {
	ctx := fixtureContext("ledger-a")
	ctx.Rules = nil
	for index := 0; index < 10; index++ {
		id := "tag-limit-" + string(rune('a'+index))
		ctx.Metadata = append(ctx.Metadata, MetadataItem{ID: id, LedgerID: "ledger-a", Kind: MetadataTag})
		ctx.Rules = append(ctx.Rules, Rule{
			ID: id, LedgerID: "ledger-a", Origin: OriginManual, ApplyMode: ApplyModeAuto,
			Confidence: ConfidenceHigh, MatchType: MatchMerchantContains, Pattern: "标签商户", Priority: index,
			Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-food", TagIDs: []string{id}},
		})
	}
	result := Classify(ctx, fixtureExpenseRow("标签商户", "订单"))
	if result.Decision.Status != StatusConflict || result.Decision.ReasonCode != ReasonTagLimitExceeded {
		t.Fatalf("expected tag limit conflict, got %+v", result.Decision)
	}
	if len(result.Decision.SelectedTagIDs) != 0 {
		t.Fatalf("tag limit conflict must not truncate selected tags: %v", result.Decision.SelectedTagIDs)
	}
}

func TestManualAndBulkSelectionsAreProtected(t *testing.T) {
	for _, source := range []ClassificationSource{SourceManual, SourceBulk} {
		t.Run(string(source), func(t *testing.T) {
			row := fixtureExpenseRow("滴滴出行", "行程")
			row.CurrentSource = source
			row.SelectedCategoryID = "manual-category"
			row.SelectedTagIDs = []string{"manual-tag"}
			result := Classify(fixtureContext("ledger-a"), row)
			if result.Evaluated || !result.Protected || result.Decision.SelectedCategoryID != "manual-category" {
				t.Fatalf("manual/bulk selection was not protected: %+v", result)
			}
		})
	}
}

func TestClassifierIsDeterministicAcrossRepeatedRuns(t *testing.T) {
	ctx := fixtureContext("ledger-a")
	row := fixtureExpenseRow("冲突商户", "冲突订单")
	want := Classify(ctx, row)
	for index := 0; index < 100; index++ {
		if got := Classify(ctx, row); !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d changed result:\nwant=%+v\ngot=%+v", index, want, got)
		}
	}
}

func fixtureContext(ledgerID string) Context {
	prefix := "a"
	if ledgerID == "ledger-b" {
		prefix = "b"
	}
	metadata := []MetadataItem{
		{ID: "cat-" + prefix + "-food", LedgerID: ledgerID, SystemKey: "expense_food", Kind: MetadataExpenseCategory},
		{ID: "cat-" + prefix + "-transport", LedgerID: ledgerID, SystemKey: "expense_transport", Kind: MetadataExpenseCategory},
		{ID: "cat-" + prefix + "-other-expense", LedgerID: ledgerID, SystemKey: "expense_other", Kind: MetadataExpenseCategory},
		{ID: "cat-" + prefix + "-salary", LedgerID: ledgerID, SystemKey: "income_salary", Kind: MetadataIncomeCategory},
		{ID: "cat-" + prefix + "-refund", LedgerID: ledgerID, SystemKey: "income_refund", Kind: MetadataIncomeCategory},
		{ID: "cat-" + prefix + "-other-income", LedgerID: ledgerID, SystemKey: "income_other", Kind: MetadataIncomeCategory},
		{ID: "tag-" + prefix + "-takeout", LedgerID: ledgerID, SystemKey: "tag_takeout", Kind: MetadataTag},
		{ID: "tag-" + prefix + "-commute", LedgerID: ledgerID, SystemKey: "tag_commute", Kind: MetadataTag},
		{ID: "tag-" + prefix + "-archived", LedgerID: ledgerID, Kind: MetadataTag, IsArchived: true},
	}
	ctx := Context{LedgerID: ledgerID, Metadata: metadata, Builtins: BuiltinV1()}
	if ledgerID != "ledger-a" {
		return ctx
	}
	ctx.Rules = []Rule{
		{ID: "rule-a-didi", LedgerID: ledgerID, Origin: OriginManual, ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantContains, Pattern: "滴滴", Priority: 100, CreatedAt: "2026-01-01T00:00:00Z", Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-transport", TagIDs: []string{"tag-a-commute"}}},
		{ID: "rule-a-coffee", LedgerID: ledgerID, Origin: OriginLearned, ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantEquals, Pattern: "星河咖啡", Priority: 500, CreatedAt: "2026-01-02T00:00:00Z", Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-food"}},
		{ID: "rule-a-archived-tag", LedgerID: ledgerID, Origin: OriginManual, ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantContains, Pattern: "旧店", Priority: 100, CreatedAt: "2026-01-03T00:00:00Z", Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-food", TagIDs: []string{"tag-a-archived"}}},
		{ID: "rule-a-conflict-1", LedgerID: ledgerID, Origin: OriginManual, ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantEquals, Pattern: "冲突商户", Priority: 10, CreatedAt: "2026-01-04T00:00:00Z", Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-food"}},
		{ID: "rule-a-conflict-2", LedgerID: ledgerID, Origin: OriginManual, ApplyMode: ApplyModeAuto, Confidence: ConfidenceHigh, MatchType: MatchMerchantEquals, Pattern: "冲突商户", Priority: 10, CreatedAt: "2026-01-05T00:00:00Z", Status: RuleStatusActive, Result: RuleResult{CategoryID: "cat-a-transport"}},
	}
	return ctx
}

func fixtureExpenseRow(merchant string, title string) Row {
	return fixtureExpenseRowForLedger("ledger-a", merchant, title)
}

func fixtureExpenseRowForLedger(ledgerID string, merchant string, title string) Row {
	return Row{
		LedgerID: ledgerID, SourceType: "wechat", Merchant: merchant, Title: title,
		Direction: "expense", TargetTransactionType: "expense", DuplicateStatus: "new", RowStatus: "pending",
	}
}
