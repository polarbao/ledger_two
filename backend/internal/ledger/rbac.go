package ledger

type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

type Operation string

const (
	OperationViewLedger           Operation = "view_ledger"
	OperationViewMembers          Operation = "view_members"
	OperationRenameLedger         Operation = "rename_ledger"
	OperationManageMembers        Operation = "manage_members"
	OperationCreateTransaction    Operation = "create_transaction"
	OperationEditOwnTransaction   Operation = "edit_own_transaction"
	OperationEditAnyTransaction   Operation = "edit_any_transaction"
	OperationDeleteOwnTransaction Operation = "delete_own_transaction"
	OperationCreateSharedExpense  Operation = "create_shared_expense"
	OperationCreateSettlement     Operation = "create_settlement"
	OperationViewReports          Operation = "view_reports"
	OperationExportData           Operation = "export_data"
	OperationManualBackup         Operation = "manual_backup"
	OperationRestoreBackup        Operation = "restore_backup"
	OperationManageMetadata       Operation = "manage_metadata"
)

type LedgerContext struct {
	UserID     string
	LedgerID   string
	Role       Role
	IsExplicit bool
}

type RolePolicy struct {
	allowed map[Role]map[Operation]bool
}

func NewRolePolicy() RolePolicy {
	return RolePolicy{
		allowed: map[Role]map[Operation]bool{
			RoleOwner: {
				OperationViewLedger:           true,
				OperationViewMembers:          true,
				OperationRenameLedger:         true,
				OperationManageMembers:        true,
				OperationCreateTransaction:    true,
				OperationEditOwnTransaction:   true,
				OperationDeleteOwnTransaction: true,
				OperationCreateSharedExpense:  true,
				OperationCreateSettlement:     true,
				OperationViewReports:          true,
				OperationExportData:           true,
				OperationManualBackup:         true,
				OperationRestoreBackup:        true,
				OperationManageMetadata:       true,
			},
			RoleEditor: {
				OperationViewLedger:           true,
				OperationViewMembers:          true,
				OperationCreateTransaction:    true,
				OperationEditOwnTransaction:   true,
				OperationDeleteOwnTransaction: true,
				OperationCreateSharedExpense:  true,
				OperationCreateSettlement:     true,
				OperationViewReports:          true,
			},
			RoleViewer: {
				OperationViewLedger:  true,
				OperationViewMembers: true,
				OperationViewReports: true,
			},
		},
	}
}

func (p RolePolicy) Can(role Role, operation Operation) bool {
	operations, ok := p.allowed[role]
	if !ok {
		return false
	}
	return operations[operation]
}

func IsValidRole(role Role) bool {
	switch role {
	case RoleOwner, RoleEditor, RoleViewer:
		return true
	default:
		return false
	}
}
