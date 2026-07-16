package ledger

import (
	"time"
)

type LedgerStatus string

const (
	LedgerStatusActive   LedgerStatus = "active"
	LedgerStatusArchived LedgerStatus = "archived"
)

type LedgerListStatus string

const (
	LedgerListActive   LedgerListStatus = "active"
	LedgerListArchived LedgerListStatus = "archived"
	LedgerListAll      LedgerListStatus = "all"
)

type Ledger struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Status           LedgerStatus `json:"status"`
	ArchivedAt       *time.Time   `json:"archived_at"`
	ArchivedByUserID *string      `json:"archived_by_user_id"`
	Version          int64        `json:"version"`
	MemberCount      int          `json:"member_count"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

type LedgerMember struct {
	LedgerID  string    `json:"ledger_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"` // owner, editor, viewer
	JoinedAt  time.Time `json:"joined_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LedgerWithRole struct {
	Ledger
	Role string `json:"role"`
}

type UnsettledBalanceSnapshot struct {
	FromUserID  *string `json:"from_user_id"`
	ToUserID    *string `json:"to_user_id"`
	AmountCents int64   `json:"amount_cents"`
}

type ArchivePreflight struct {
	Ledger                           LedgerWithRole           `json:"ledger"`
	UnsettledBalance                 UnsettledBalanceSnapshot `json:"unsettled_balance"`
	ReadyImportBatchCount            int                      `json:"ready_import_batch_count"`
	CanArchive                       bool                     `json:"can_archive"`
	RequiresUnsettledAcknowledgement bool                     `json:"requires_unsettled_acknowledgement"`
}

type MemberDetail struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type MemberListData struct {
	Ledger  LedgerWithRole `json:"ledger"`
	Members []MemberDetail `json:"members"`
}

type LeaveLedgerResult struct {
	LedgerID string `json:"ledger_id"`
	Version  int64  `json:"version"`
}
