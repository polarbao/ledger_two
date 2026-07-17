package defaults

import "testing"

func TestBasicCNV1ProfileUsesFrozenSystemKeysAndOrder(t *testing.T) {
	profile, ok := Get(ProfileBasicCNV1)
	if !ok {
		t.Fatal("expected basic_cn_v1 profile")
	}
	if profile.Key != ProfileBasicCNV1 || profile.Version != 1 {
		t.Fatalf("unexpected profile identity: %+v", profile)
	}

	wantKeys := []string{
		"expense_food",
		"expense_shopping",
		"expense_transport",
		"expense_housing",
		"expense_utilities",
		"expense_health",
		"expense_education",
		"expense_leisure",
		"expense_travel",
		"expense_social",
		"expense_work",
		"expense_other",
		"income_salary",
		"income_bonus",
		"income_reimbursement",
		"income_refund",
		"income_gift",
		"income_business",
		"income_other",
		"tag_takeout",
		"tag_supermarket",
		"tag_commute",
		"tag_fixed",
		"tag_subscription",
		"tag_work",
		"tag_travel",
		"tag_reimbursement",
	}
	if len(profile.Items) != len(wantKeys) {
		t.Fatalf("profile item count = %d, want %d", len(profile.Items), len(wantKeys))
	}
	for index, key := range wantKeys {
		if profile.Items[index].SystemKey != key {
			t.Fatalf("profile key at %d = %q, want %q", index, profile.Items[index].SystemKey, key)
		}
	}

	if profile.Items[11].Name != "其他支出" || profile.Items[11].Kind != KindExpenseCategory {
		t.Fatalf("unexpected expense fallback: %+v", profile.Items[11])
	}
	if profile.Items[18].Name != "其他收入" || profile.Items[18].Kind != KindIncomeCategory {
		t.Fatalf("unexpected income fallback: %+v", profile.Items[18])
	}
}

func TestEmptyProfileIsExplicitAndContainsNoMetadata(t *testing.T) {
	profile, ok := Get(ProfileEmpty)
	if !ok {
		t.Fatal("expected empty profile")
	}
	if profile.Version != 0 || len(profile.Items) != 0 {
		t.Fatalf("unexpected empty profile: %+v", profile)
	}
	if _, ok := Get("unknown"); ok {
		t.Fatal("unknown profile must not resolve")
	}
}
