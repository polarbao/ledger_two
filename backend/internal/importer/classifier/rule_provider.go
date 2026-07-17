package classifier

import (
	"sort"
	"strings"
)

type metadataIndex struct {
	byID        map[string]MetadataItem
	bySystemKey map[string]MetadataItem
}

func newMetadataIndex(ctx Context) metadataIndex {
	index := metadataIndex{byID: map[string]MetadataItem{}, bySystemKey: map[string]MetadataItem{}}
	for _, item := range ctx.Metadata {
		if item.LedgerID != ctx.LedgerID {
			continue
		}
		index.byID[item.ID] = item
		if item.SystemKey != "" {
			index.bySystemKey[item.SystemKey] = item
		}
	}
	return index
}

func (m metadataIndex) activeBySystemKey(systemKey string) (MetadataItem, bool) {
	item, ok := m.bySystemKey[systemKey]
	return item, ok && !item.IsArchived
}

func ruleCandidates(ctx Context, row Row, index metadataIndex) []Candidate {
	candidates := make([]Candidate, 0, len(ctx.Rules))
	for _, rule := range ctx.Rules {
		if rule.LedgerID != ctx.LedgerID || rule.Status != RuleStatusActive {
			continue
		}
		if rule.SourceType != "" && rule.SourceType != row.SourceType {
			continue
		}
		if !ruleMatches(rule, row) {
			continue
		}
		candidate, ok := candidateFromRule(rule, row, index)
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	sortCandidates(candidates)
	return candidates
}

func candidateFromRule(rule Rule, row Row, index metadataIndex) (Candidate, bool) {
	if rule.Origin != OriginManual && rule.Origin != OriginLearned {
		return Candidate{}, false
	}
	if rule.ApplyMode != ApplyModeAuto && rule.ApplyMode != ApplyModeSuggest {
		return Candidate{}, false
	}
	if rule.Confidence != ConfidenceHigh && rule.Confidence != ConfidenceMedium && rule.Confidence != ConfidenceLow {
		return Candidate{}, false
	}
	if rule.Result.CategoryID == "" && rule.Result.AccountID == "" && len(rule.Result.TagIDs) == 0 {
		return Candidate{}, false
	}
	if rule.Result.CategoryID != "" {
		category, ok := index.byID[rule.Result.CategoryID]
		if !ok || category.IsArchived || !categoryMatchesTarget(category.Kind, row.TargetTransactionType) {
			return Candidate{}, false
		}
	}
	if rule.Result.AccountID != "" {
		account, ok := index.byID[rule.Result.AccountID]
		if !ok || account.IsArchived || account.Kind != MetadataAccount {
			return Candidate{}, false
		}
	}
	for _, tagID := range rule.Result.TagIDs {
		tag, ok := index.byID[tagID]
		if !ok || tag.IsArchived || tag.Kind != MetadataTag {
			return Candidate{}, false
		}
	}

	source := SourceUserRule
	sourceRank := 0
	if rule.Origin == OriginLearned {
		source = SourceLearned
		sourceRank = 1
	}
	return Candidate{
		CategoryID: rule.Result.CategoryID,
		AccountID:  rule.Result.AccountID,
		TagIDs:     append([]string(nil), rule.Result.TagIDs...),
		Source:     source, Confidence: rule.Confidence, ApplyMode: rule.ApplyMode,
		Priority: rule.Priority, Specificity: matchSpecificity(rule.MatchType),
		RuleID: rule.ID, CandidateID: rule.ID, ReasonCode: string(rule.MatchType),
		ReasonText: ruleReasonText(rule), CreatedAt: rule.CreatedAt, sourceRank: sourceRank,
	}, true
}

func ruleMatches(rule Rule, row Row) bool {
	pattern := NormalizeText(rule.Pattern)
	if pattern == "" {
		return false
	}
	merchant := NormalizeText(row.Merchant)
	description := NormalizeText(row.Description)
	sourceAccount := NormalizeText(row.SourceAccount)
	title := NormalizeText(row.Title)
	switch rule.MatchType {
	case MatchMerchantEquals:
		return merchant != "" && merchant == pattern
	case MatchMerchantContains:
		return stringsContains(merchant, pattern)
	case MatchDescriptionContains:
		return stringsContains(description, pattern)
	case MatchSourceAccount:
		return sourceAccount != "" && sourceAccount == pattern
	case MatchAmountRange:
		if rule.AmountMinCents != nil && row.AmountCents < *rule.AmountMinCents {
			return false
		}
		if rule.AmountMaxCents != nil && row.AmountCents > *rule.AmountMaxCents {
			return false
		}
		return stringsContains(title, pattern) || stringsContains(merchant, pattern) || stringsContains(description, pattern)
	default:
		return false
	}
}

func sortCandidates(candidates []Candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.providerRank != right.providerRank {
			return left.providerRank < right.providerRank
		}
		if left.Priority != right.Priority {
			return left.Priority < right.Priority
		}
		if left.sourceRank != right.sourceRank {
			return left.sourceRank < right.sourceRank
		}
		if left.Specificity != right.Specificity {
			return left.Specificity < right.Specificity
		}
		if left.CreatedAt != right.CreatedAt {
			return left.CreatedAt > right.CreatedAt
		}
		return left.CandidateID < right.CandidateID
	})
}

func matchSpecificity(matchType MatchType) int {
	switch matchType {
	case MatchMerchantEquals, MatchSourceAccount:
		return 0
	case MatchMerchantContains, MatchDescriptionContains:
		return 1
	case MatchAmountRange:
		return 2
	default:
		return 99
	}
}

func categoryMatchesTarget(kind MetadataKind, targetType string) bool {
	return (targetType == "expense" && kind == MetadataExpenseCategory) ||
		(targetType == "income" && kind == MetadataIncomeCategory)
}

func ruleReasonText(rule Rule) string {
	switch rule.MatchType {
	case MatchMerchantEquals:
		return "商户精确匹配「" + rule.Pattern + "」"
	case MatchMerchantContains:
		return "商户包含「" + rule.Pattern + "」"
	case MatchDescriptionContains:
		return "描述包含「" + rule.Pattern + "」"
	case MatchSourceAccount:
		return "来源账户匹配「" + rule.Pattern + "」"
	case MatchAmountRange:
		return "金额区间与文本匹配「" + rule.Pattern + "」"
	default:
		return "命中分类规则"
	}
}

func stringsContains(value string, term string) bool {
	return value != "" && term != "" && strings.Contains(value, term)
}
