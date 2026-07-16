package ledger

import "context"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

type Operation string

const (
	OperationViewLedger             Operation = "view_ledger"
	OperationViewMembers            Operation = "view_members"
	OperationRenameLedger           Operation = "rename_ledger"
	OperationArchiveLedger          Operation = "archive_ledger"
	OperationRestoreLedger          Operation = "restore_ledger"
	OperationManageMembers          Operation = "manage_members"
	OperationLeaveLedger            Operation = "leave_ledger"
	OperationTransferLedgerOwner    Operation = "transfer_ledger_owner"
	OperationCreateTransaction      Operation = "create_transaction"
	OperationEditOwnTransaction     Operation = "edit_own_transaction"
	OperationEditAnyTransaction     Operation = "edit_any_transaction"
	OperationDeleteOwnTransaction   Operation = "delete_own_transaction"
	OperationCreateSharedExpense    Operation = "create_shared_expense"
	OperationCreateSettlement       Operation = "create_settlement"
	OperationViewReports            Operation = "view_reports"
	OperationExportData             Operation = "export_data"
	OperationManageMetadata         Operation = "manage_metadata"
	OperationManageImports          Operation = "manage_imports"
	OperationDiscardImportBatch     Operation = "discard_import_batch"
	OperationManualDatabaseBackup   Operation = "manual_database_backup"
	OperationPrepareDatabaseRestore Operation = "prepare_database_restore"
	OperationSystemDiagnostics      Operation = "system_diagnostics"
)

type LedgerContext struct {
	UserID     string
	LedgerID   string
	Role       Role
	Status     LedgerStatus
	Version    int64
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
				OperationArchiveLedger:        true,
				OperationRestoreLedger:        true,
				OperationManageMembers:        true,
				OperationLeaveLedger:          true,
				OperationTransferLedgerOwner:  true,
				OperationCreateTransaction:    true,
				OperationEditOwnTransaction:   true,
				OperationDeleteOwnTransaction: true,
				OperationCreateSharedExpense:  true,
				OperationCreateSettlement:     true,
				OperationViewReports:          true,
				OperationExportData:           true,
				OperationManageMetadata:       true,
				OperationManageImports:        true,
				OperationDiscardImportBatch:   true,
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
				OperationExportData:           true,
				OperationLeaveLedger:          true,
			},
			RoleViewer: {
				OperationViewLedger:  true,
				OperationViewMembers: true,
				OperationViewReports: true,
			},
		},
	}
}

type LifecycleOperation string

const (
	LifecycleRead    LifecycleOperation = "read"
	LifecycleWrite   LifecycleOperation = "write"
	LifecycleExport  LifecycleOperation = "export"
	LifecycleRestore LifecycleOperation = "restore"
)

type LifecyclePolicy struct{}

func NewLifecyclePolicy() LifecyclePolicy {
	return LifecyclePolicy{}
}

func (LifecyclePolicy) Can(status LedgerStatus, operation LifecycleOperation) bool {
	switch status {
	case LedgerStatusActive:
		return operation == LifecycleRead || operation == LifecycleWrite || operation == LifecycleExport
	case LedgerStatusArchived:
		return operation == LifecycleRead || operation == LifecycleExport || operation == LifecycleRestore
	default:
		return false
	}
}

type InstanceAdminLookup interface {
	IsInstanceAdmin(ctx context.Context, userID string) (bool, error)
}

type InstancePolicy struct {
	lookup InstanceAdminLookup
}

func NewInstancePolicy(lookup InstanceAdminLookup) InstancePolicy {
	return InstancePolicy{lookup: lookup}
}

func (p InstancePolicy) Can(ctx context.Context, userID string) (bool, error) {
	if userID == "" || p.lookup == nil {
		return false, nil
	}
	return p.lookup.IsInstanceAdmin(ctx, userID)
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
