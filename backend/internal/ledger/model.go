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

type MemberDetail struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
