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
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	Icon       string `json:"icon,omitempty"`
	Color      string `json:"color,omitempty"`
	SortOrder  int    `json:"sort_order"`
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
