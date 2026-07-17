package classifier

type ClassificationSource string

const (
	SourceManual   ClassificationSource = "manual"
	SourceBulk     ClassificationSource = "bulk"
	SourceUserRule ClassificationSource = "user_rule"
	SourceLearned  ClassificationSource = "learned_rule"
	SourceBuiltin  ClassificationSource = "builtin"
	SourceFallback ClassificationSource = "fallback"
)

type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
	ConfidenceNone   Confidence = "none"
)

type ClassificationStatus string

const (
	StatusAutoSelected ClassificationStatus = "auto_selected"
	StatusSuggested    ClassificationStatus = "suggested"
	StatusFallback     ClassificationStatus = "fallback"
	StatusManual       ClassificationStatus = "manual"
	StatusBulk         ClassificationStatus = "bulk"
	StatusConflict     ClassificationStatus = "conflict"
	StatusUnresolved   ClassificationStatus = "unresolved"
)

type RuleOrigin string

const (
	OriginManual  RuleOrigin = "manual"
	OriginLearned RuleOrigin = "learned"
)

type ApplyMode string

const (
	ApplyModeAuto    ApplyMode = "auto"
	ApplyModeSuggest ApplyMode = "suggest"
)

type MatchType string

const (
	MatchMerchantEquals      MatchType = "merchant_equals"
	MatchMerchantContains    MatchType = "merchant_contains"
	MatchDescriptionContains MatchType = "description_contains"
	MatchSourceAccount       MatchType = "source_account"
	MatchAmountRange         MatchType = "amount_range"
)

const RuleStatusActive = "active"

type MetadataKind string

const (
	MetadataExpenseCategory MetadataKind = "expense_category"
	MetadataIncomeCategory  MetadataKind = "income_category"
	MetadataTag             MetadataKind = "tag"
	MetadataAccount         MetadataKind = "account"
)

const (
	ReasonCategoryConflict = "CLASSIFICATION_CONFLICT"
	ReasonTagLimitExceeded = "TAG_LIMIT_EXCEEDED"
	ReasonFallback         = "fallback_other"
)

type Row struct {
	LedgerID              string
	SourceType            string
	Merchant              string
	Title                 string
	Description           string
	SourceAccount         string
	AmountCents           int64
	Direction             string
	TargetTransactionType string
	DuplicateStatus       string
	RowStatus             string
	CurrentSource         ClassificationSource
	SelectedCategoryID    string
	SelectedAccountID     string
	SelectedTagIDs        []string
}

type RuleResult struct {
	CategoryID string   `json:"category_id"`
	AccountID  string   `json:"account_id"`
	TagIDs     []string `json:"tag_ids"`
}

type Rule struct {
	ID              string
	LedgerID        string
	Origin          RuleOrigin
	SourceType      string
	ApplyMode       ApplyMode
	Confidence      Confidence
	MatchType       MatchType
	Pattern         string
	AmountMinCents  *int64
	AmountMaxCents  *int64
	Priority        int
	Result          RuleResult
	Status          string
	CreatedAt       string
	CreatedByUserID string
}

type MetadataItem struct {
	ID         string
	LedgerID   string
	SystemKey  string
	Kind       MetadataKind
	IsArchived bool
}

type BuiltinRule struct {
	Key               string
	Directions        []string
	TargetType        string
	MerchantContains  []string
	TitleContains     []string
	CategorySystemKey string
	TagSystemKeys     []string
	Confidence        Confidence
	ReasonCode        string
	ReasonText        string
}

type Context struct {
	LedgerID string
	Rules    []Rule
	Metadata []MetadataItem
	Builtins []BuiltinRule
}

type Candidate struct {
	CategoryID   string
	AccountID    string
	TagIDs       []string
	Source       ClassificationSource
	Confidence   Confidence
	ApplyMode    ApplyMode
	Priority     int
	Specificity  int
	RuleID       string
	CandidateID  string
	ReasonCode   string
	ReasonText   string
	CreatedAt    string
	providerRank int
	sourceRank   int
}

type Decision struct {
	Status              ClassificationStatus
	Confidence          Confidence
	Source              ClassificationSource
	ReasonCode          string
	ReasonText          string
	MatchedRuleIDs      []string
	SuggestedCategoryID string
	SuggestedAccountID  string
	SuggestedTagIDs     []string
	SelectedCategoryID  string
	SelectedAccountID   string
	SelectedTagIDs      []string
}

type Result struct {
	Evaluated  bool
	Protected  bool
	Decision   Decision
	Candidates []Candidate
}
