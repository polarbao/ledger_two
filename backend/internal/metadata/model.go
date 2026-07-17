package metadata

type Kind string

const (
	KindCategory Kind = "categories"
	KindTag      Kind = "tags"
	KindAccount  Kind = "accounts"
)

type Item struct {
	ID         string `json:"id"`
	LedgerID   string `json:"ledger_id"`
	SystemKey  string `json:"system_key,omitempty"`
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	Icon       string `json:"icon,omitempty"`
	Color      string `json:"color,omitempty"`
	SortOrder  int    `json:"sort_order"`
	UsageCount int    `json:"usage_count"`
	IsArchived bool   `json:"is_archived"`
}

type UpsertRequest struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

type ReorderRequest struct {
	OrderedIDs []string `json:"ordered_ids"`
}

type ProfileAction string

const (
	ProfileActionCreate   ProfileAction = "create"
	ProfileActionReuse    ProfileAction = "reuse"
	ProfileActionSkip     ProfileAction = "skip"
	ProfileActionConflict ProfileAction = "conflict"
	ProfileActionExisting ProfileAction = "existing"
)

const (
	ProfileResolutionReuse = "reuse"
	ProfileResolutionSkip  = "skip"
)

type ProfileItem struct {
	SystemKey  string        `json:"system_key"`
	Kind       string        `json:"kind"`
	Name       string        `json:"name"`
	Icon       string        `json:"icon,omitempty"`
	Color      string        `json:"color,omitempty"`
	Action     ProfileAction `json:"action"`
	ExistingID string        `json:"existing_id,omitempty"`
}

type DefaultProfile struct {
	Key     string        `json:"key"`
	Version int           `json:"version"`
	Items   []ProfileItem `json:"items"`
}

type ProfilePreviewRequest struct {
	Profile string `json:"profile"`
}

type ProfilePreviewResult struct {
	Profile       DefaultProfile `json:"profile"`
	CreateCount   int            `json:"create_count"`
	ReuseCount    int            `json:"reuse_count"`
	ConflictCount int            `json:"conflict_count"`
}

type ProfileConflictResolution struct {
	SystemKey  string `json:"system_key"`
	Action     string `json:"action"`
	ExistingID string `json:"existing_id,omitempty"`
}

type ProfileApplyRequest struct {
	Profile     string                      `json:"profile"`
	Resolutions []ProfileConflictResolution `json:"resolutions"`
}

type ProfileApplyResult struct {
	Profile                string `json:"profile"`
	CreatedCount           int    `json:"created_count"`
	ReusedCount            int    `json:"reused_count"`
	SkippedCount           int    `json:"skipped_count"`
	MetadataProfileVersion int    `json:"metadata_profile_version"`
}
