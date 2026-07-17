package classifier

func BuiltinV1() []BuiltinRule {
	return []BuiltinRule{
		{
			Key: "builtin_takeout_v1", Directions: []string{"expense"}, TargetType: "expense",
			MerchantContains: []string{"外卖"}, TitleContains: []string{"外卖"},
			CategorySystemKey: "expense_food", TagSystemKeys: []string{"tag_takeout"},
			Confidence: ConfidenceMedium, ReasonCode: "builtin_takeout_merchant", ReasonText: "包含通用外卖关键词",
		},
		{
			Key: "builtin_salary_v1", Directions: []string{"income"}, TargetType: "income",
			TitleContains: []string{"工资"}, CategorySystemKey: "income_salary",
			Confidence: ConfidenceMedium, ReasonCode: "builtin_salary_title", ReasonText: "包含通用工资关键词",
		},
		{
			Key: "builtin_refund_v1", Directions: []string{"income", "refund"}, TargetType: "income",
			TitleContains: []string{"退款", "退回"}, CategorySystemKey: "income_refund",
			Confidence: ConfidenceMedium, ReasonCode: "builtin_refund_title", ReasonText: "包含通用退款关键词",
		},
	}
}

func builtinCandidates(ctx Context, row Row, index metadataIndex) []Candidate {
	var candidates []Candidate
	for _, rule := range ctx.Builtins {
		if rule.TargetType != "" && rule.TargetType != row.TargetTransactionType {
			continue
		}
		if !containsString(rule.Directions, row.Direction) {
			continue
		}
		if !builtinMatches(rule, row) {
			continue
		}
		category, ok := index.activeBySystemKey(rule.CategorySystemKey)
		if !ok || !categoryMatchesTarget(category.Kind, row.TargetTransactionType) {
			continue
		}
		tagIDs := make([]string, 0, len(rule.TagSystemKeys))
		valid := true
		for _, systemKey := range rule.TagSystemKeys {
			tag, found := index.activeBySystemKey(systemKey)
			if !found || tag.Kind != MetadataTag {
				valid = false
				break
			}
			tagIDs = append(tagIDs, tag.ID)
		}
		if !valid {
			continue
		}
		candidates = append(candidates, Candidate{
			CategoryID: category.ID,
			TagIDs:     append([]string(nil), tagIDs...),
			Source:     SourceBuiltin, Confidence: rule.Confidence, ApplyMode: ApplyModeSuggest,
			Priority: 1000, Specificity: 3, CandidateID: rule.Key,
			ReasonCode: rule.ReasonCode, ReasonText: rule.ReasonText, providerRank: 1, sourceRank: 2,
		})
	}
	return candidates
}

func builtinMatches(rule BuiltinRule, row Row) bool {
	merchant := NormalizeText(row.Merchant)
	title := NormalizeText(row.Title)
	for _, value := range rule.MerchantContains {
		if term := NormalizeText(value); term != "" && stringsContains(merchant, term) {
			return true
		}
	}
	for _, value := range rule.TitleContains {
		if term := NormalizeText(value); term != "" && stringsContains(title, term) {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
