package defaults

type Kind string

const (
	KindExpenseCategory Kind = "expense_category"
	KindIncomeCategory  Kind = "income_category"
	KindTag             Kind = "tag"
)

const (
	ProfileBasicCNV1 = "basic_cn_v1"
	ProfileEmpty     = "empty"
)

type Item struct {
	SystemKey string
	Kind      Kind
	Name      string
	Icon      string
	Color     string
	SortOrder int
}

type Profile struct {
	Key     string
	Version int
	Items   []Item
}

var basicCNV1 = Profile{
	Key:     ProfileBasicCNV1,
	Version: 1,
	Items: []Item{
		{SystemKey: "expense_food", Kind: KindExpenseCategory, Name: "餐饮", Icon: "utensils", Color: "#f97316", SortOrder: 0},
		{SystemKey: "expense_shopping", Kind: KindExpenseCategory, Name: "购物", Icon: "shopping-bag", Color: "#ec4899", SortOrder: 1},
		{SystemKey: "expense_transport", Kind: KindExpenseCategory, Name: "交通", Icon: "car", Color: "#3b82f6", SortOrder: 2},
		{SystemKey: "expense_housing", Kind: KindExpenseCategory, Name: "居住", Icon: "house", Color: "#8b5cf6", SortOrder: 3},
		{SystemKey: "expense_utilities", Kind: KindExpenseCategory, Name: "生活缴费", Icon: "receipt", Color: "#06b6d4", SortOrder: 4},
		{SystemKey: "expense_health", Kind: KindExpenseCategory, Name: "医疗健康", Icon: "heart-pulse", Color: "#ef4444", SortOrder: 5},
		{SystemKey: "expense_education", Kind: KindExpenseCategory, Name: "教育成长", Icon: "book-open", Color: "#6366f1", SortOrder: 6},
		{SystemKey: "expense_leisure", Kind: KindExpenseCategory, Name: "休闲娱乐", Icon: "gamepad-2", Color: "#a855f7", SortOrder: 7},
		{SystemKey: "expense_travel", Kind: KindExpenseCategory, Name: "旅行", Icon: "plane", Color: "#0ea5e9", SortOrder: 8},
		{SystemKey: "expense_social", Kind: KindExpenseCategory, Name: "人情往来", Icon: "gift", Color: "#f43f5e", SortOrder: 9},
		{SystemKey: "expense_work", Kind: KindExpenseCategory, Name: "工作经营", Icon: "briefcase-business", Color: "#64748b", SortOrder: 10},
		{SystemKey: "expense_other", Kind: KindExpenseCategory, Name: "其他支出", Icon: "circle-ellipsis", Color: "#94a3b8", SortOrder: 11},
		{SystemKey: "income_salary", Kind: KindIncomeCategory, Name: "工资", Icon: "wallet-cards", Color: "#10b981", SortOrder: 0},
		{SystemKey: "income_bonus", Kind: KindIncomeCategory, Name: "奖金", Icon: "badge-dollar-sign", Color: "#22c55e", SortOrder: 1},
		{SystemKey: "income_reimbursement", Kind: KindIncomeCategory, Name: "报销", Icon: "receipt-text", Color: "#14b8a6", SortOrder: 2},
		{SystemKey: "income_refund", Kind: KindIncomeCategory, Name: "退款", Icon: "rotate-ccw", Color: "#06b6d4", SortOrder: 3},
		{SystemKey: "income_gift", Kind: KindIncomeCategory, Name: "红包礼金", Icon: "gift", Color: "#f43f5e", SortOrder: 4},
		{SystemKey: "income_business", Kind: KindIncomeCategory, Name: "经营收入", Icon: "briefcase-business", Color: "#3b82f6", SortOrder: 5},
		{SystemKey: "income_other", Kind: KindIncomeCategory, Name: "其他收入", Icon: "circle-ellipsis", Color: "#94a3b8", SortOrder: 6},
		{SystemKey: "tag_takeout", Kind: KindTag, Name: "外卖", Color: "#f97316", SortOrder: 0},
		{SystemKey: "tag_supermarket", Kind: KindTag, Name: "超市", Color: "#ec4899", SortOrder: 1},
		{SystemKey: "tag_commute", Kind: KindTag, Name: "通勤", Color: "#3b82f6", SortOrder: 2},
		{SystemKey: "tag_fixed", Kind: KindTag, Name: "固定支出", Color: "#8b5cf6", SortOrder: 3},
		{SystemKey: "tag_subscription", Kind: KindTag, Name: "订阅服务", Color: "#6366f1", SortOrder: 4},
		{SystemKey: "tag_work", Kind: KindTag, Name: "工作相关", Color: "#64748b", SortOrder: 5},
		{SystemKey: "tag_travel", Kind: KindTag, Name: "旅行", Color: "#0ea5e9", SortOrder: 6},
		{SystemKey: "tag_reimbursement", Kind: KindTag, Name: "报销相关", Color: "#14b8a6", SortOrder: 7},
	},
}

func Get(key string) (Profile, bool) {
	switch key {
	case ProfileBasicCNV1:
		return cloneProfile(basicCNV1), true
	case ProfileEmpty:
		return Profile{Key: ProfileEmpty, Version: 0, Items: []Item{}}, true
	default:
		return Profile{}, false
	}
}

func cloneProfile(profile Profile) Profile {
	cloned := profile
	cloned.Items = append([]Item(nil), profile.Items...)
	return cloned
}
