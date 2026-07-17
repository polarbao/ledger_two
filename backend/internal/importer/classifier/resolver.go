package classifier

import "sort"

const maxTags = 8

func Classify(ctx Context, row Row) Result {
	if ctx.LedgerID == "" || row.LedgerID != ctx.LedgerID {
		return Result{}
	}
	if row.CurrentSource == SourceManual || row.CurrentSource == SourceBulk {
		status := StatusManual
		if row.CurrentSource == SourceBulk {
			status = StatusBulk
		}
		return Result{
			Protected: true,
			Decision: Decision{
				Status: status, Confidence: ConfidenceHigh, Source: row.CurrentSource,
				SelectedCategoryID: row.SelectedCategoryID,
				SelectedAccountID:  row.SelectedAccountID,
				SelectedTagIDs:     append([]string(nil), row.SelectedTagIDs...),
			},
		}
	}
	if !eligible(row) {
		return Result{}
	}

	index := newMetadataIndex(ctx)
	candidates := ruleCandidates(ctx, row, index)
	candidates = append(candidates, builtinCandidates(ctx, row, index)...)
	sortCandidates(candidates)
	result := Result{Evaluated: true, Candidates: cloneCandidates(candidates)}
	if len(candidates) == 0 {
		result.Decision = fallbackDecision(row, index)
		return result
	}

	auto := filterCandidates(candidates, func(candidate Candidate) bool {
		return candidate.ApplyMode == ApplyModeAuto && candidate.Confidence == ConfidenceHigh && candidate.Source != SourceBuiltin
	})
	if len(auto) > 0 {
		result.Decision = resolveCandidates(auto, true)
		return result
	}
	explicit := filterCandidates(candidates, func(candidate Candidate) bool {
		return candidate.Source == SourceUserRule || candidate.Source == SourceLearned
	})
	if len(explicit) > 0 {
		result.Decision = resolveCandidates(explicit, false)
		return result
	}
	result.Decision = resolveCandidates(candidates, false)
	return result
}

func eligible(row Row) bool {
	if row.DuplicateStatus == "invalid" || row.DuplicateStatus == "duplicate" {
		return false
	}
	if row.RowStatus == "skipped" || row.TargetTransactionType == "skipped" {
		return false
	}
	return row.TargetTransactionType == "expense" || row.TargetTransactionType == "income"
}

func resolveCandidates(candidates []Candidate, selected bool) Decision {
	decision := Decision{Status: StatusSuggested}
	if selected {
		decision.Status = StatusAutoSelected
	}
	top := candidates[0]
	decision.Confidence = top.Confidence
	decision.Source = top.Source
	decision.ReasonCode = top.ReasonCode
	decision.ReasonText = top.ReasonText
	decision.MatchedRuleIDs = matchedRuleIDs(candidates)

	categoryCandidates := filterCandidates(candidates, func(candidate Candidate) bool { return candidate.CategoryID != "" })
	categoryID, conflict, conflictIDs := resolveCategory(categoryCandidates)
	if conflict {
		return Decision{
			Status: StatusConflict, Confidence: top.Confidence, Source: top.Source,
			ReasonCode: ReasonCategoryConflict, ReasonText: "同级规则给出了不同分类",
			MatchedRuleIDs: conflictIDs,
		}
	}
	accountID := firstAccountID(candidates)
	tagIDs := stableTagUnion(candidates)
	if len(tagIDs) > maxTags {
		return Decision{
			Status: StatusConflict, Confidence: top.Confidence, Source: top.Source,
			ReasonCode: ReasonTagLimitExceeded, ReasonText: "分类规则给出的标签超过 8 个",
			MatchedRuleIDs: decision.MatchedRuleIDs,
		}
	}
	if selected {
		decision.SelectedCategoryID = categoryID
		decision.SelectedAccountID = accountID
		decision.SelectedTagIDs = tagIDs
	} else {
		decision.SuggestedCategoryID = categoryID
		decision.SuggestedAccountID = accountID
		decision.SuggestedTagIDs = tagIDs
	}
	return decision
}

func resolveCategory(candidates []Candidate) (string, bool, []string) {
	if len(candidates) == 0 {
		return "", false, nil
	}
	top := candidates[0]
	ids := map[string]bool{}
	var matched []string
	for _, candidate := range candidates {
		if candidate.providerRank != top.providerRank || candidate.Priority != top.Priority || candidate.sourceRank != top.sourceRank || candidate.Specificity != top.Specificity {
			break
		}
		ids[candidate.CategoryID] = true
		if candidate.RuleID != "" {
			matched = append(matched, candidate.RuleID)
		}
	}
	if len(ids) > 1 {
		sort.Strings(matched)
		return "", true, matched
	}
	return top.CategoryID, false, nil
}

func stableTagUnion(candidates []Candidate) []string {
	seen := map[string]bool{}
	var result []string
	for _, candidate := range candidates {
		for _, tagID := range candidate.TagIDs {
			if tagID == "" || seen[tagID] {
				continue
			}
			seen[tagID] = true
			result = append(result, tagID)
		}
	}
	return result
}

func firstAccountID(candidates []Candidate) string {
	for _, candidate := range candidates {
		if candidate.AccountID != "" {
			return candidate.AccountID
		}
	}
	return ""
}

func matchedRuleIDs(candidates []Candidate) []string {
	var result []string
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate.RuleID == "" || seen[candidate.RuleID] {
			continue
		}
		seen[candidate.RuleID] = true
		result = append(result, candidate.RuleID)
	}
	return result
}

func fallbackDecision(row Row, index metadataIndex) Decision {
	systemKey := "expense_other"
	if row.TargetTransactionType == "income" {
		systemKey = "income_other"
	}
	category, ok := index.activeBySystemKey(systemKey)
	if !ok || !categoryMatchesTarget(category.Kind, row.TargetTransactionType) {
		return Decision{Status: StatusUnresolved, Confidence: ConfidenceNone}
	}
	return Decision{
		Status: StatusFallback, Confidence: ConfidenceNone, Source: SourceFallback,
		ReasonCode: ReasonFallback, ReasonText: "没有可靠的分类候选，使用兜底分类",
		SelectedCategoryID: category.ID,
	}
}

func filterCandidates(candidates []Candidate, keep func(Candidate) bool) []Candidate {
	result := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if keep(candidate) {
			result = append(result, candidate)
		}
	}
	return result
}

func cloneCandidates(candidates []Candidate) []Candidate {
	cloned := append([]Candidate(nil), candidates...)
	for index := range cloned {
		cloned[index].TagIDs = append([]string(nil), cloned[index].TagIDs...)
	}
	return cloned
}
