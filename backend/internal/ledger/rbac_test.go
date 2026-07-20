package ledger

import (
	"context"
	"testing"
)

func TestRolePolicyOwnerPermissions(t *testing.T) {
	policy := NewRolePolicy()

	allowed := []Operation{
		OperationViewLedger,
		OperationViewMembers,
		OperationRenameLedger,
		OperationArchiveLedger,
		OperationRestoreLedger,
		OperationManageMembers,
		OperationLeaveLedger,
		OperationTransferLedgerOwner,
		OperationCreateTransaction,
		OperationEditOwnTransaction,
		OperationDeleteOwnTransaction,
		OperationCreateSharedExpense,
		OperationCreateSettlement,
		OperationViewReports,
		OperationExportData,
		OperationManageMetadata,
		OperationUseImports,
		OperationManageImports,
		OperationDiscardImportBatch,
	}
	for _, operation := range allowed {
		if !policy.Can(RoleOwner, operation) {
			t.Fatalf("owner should be allowed to %s", operation)
		}
	}

	if policy.Can(RoleOwner, OperationEditAnyTransaction) {
		t.Fatalf("owner should not edit others' transactions by default")
	}
	for _, operation := range []Operation{OperationManualDatabaseBackup, OperationPrepareDatabaseRestore, OperationSystemDiagnostics} {
		if policy.Can(RoleOwner, operation) {
			t.Fatalf("ledger owner must not receive instance operation %s", operation)
		}
	}
}

func TestRolePolicyEditorPermissions(t *testing.T) {
	policy := NewRolePolicy()

	allowed := []Operation{
		OperationViewLedger,
		OperationViewMembers,
		OperationCreateTransaction,
		OperationEditOwnTransaction,
		OperationDeleteOwnTransaction,
		OperationCreateSharedExpense,
		OperationCreateSettlement,
		OperationViewReports,
		OperationExportData,
		OperationLeaveLedger,
		OperationUseImports,
		OperationDiscardImportBatch,
	}
	for _, operation := range allowed {
		if !policy.Can(RoleEditor, operation) {
			t.Fatalf("editor should be allowed to %s", operation)
		}
	}

	denied := []Operation{
		OperationRenameLedger,
		OperationManageMembers,
		OperationEditAnyTransaction,
		OperationManageMetadata,
		OperationManageImports,
		OperationArchiveLedger,
		OperationRestoreLedger,
		OperationTransferLedgerOwner,
	}
	for _, operation := range denied {
		if policy.Can(RoleEditor, operation) {
			t.Fatalf("editor should not be allowed to %s", operation)
		}
	}
}

func TestRolePolicyViewerPermissions(t *testing.T) {
	policy := NewRolePolicy()

	allowed := []Operation{
		OperationViewLedger,
		OperationViewMembers,
		OperationViewReports,
		OperationLeaveLedger,
	}
	for _, operation := range allowed {
		if !policy.Can(RoleViewer, operation) {
			t.Fatalf("viewer should be allowed to %s", operation)
		}
	}

	denied := []Operation{
		OperationCreateTransaction,
		OperationEditOwnTransaction,
		OperationDeleteOwnTransaction,
		OperationCreateSharedExpense,
		OperationCreateSettlement,
		OperationExportData,
		OperationUseImports,
		OperationManageImports,
		OperationDiscardImportBatch,
		OperationManageMetadata,
	}
	for _, operation := range denied {
		if policy.Can(RoleViewer, operation) {
			t.Fatalf("viewer should not be allowed to %s", operation)
		}
	}
}

func TestLifecyclePolicyAllowsArchivedReadsButRejectsWrites(t *testing.T) {
	policy := NewLifecyclePolicy()
	if !policy.Can(LedgerStatusArchived, LifecycleRead) || !policy.Can(LedgerStatusArchived, LifecycleExport) || !policy.Can(LedgerStatusArchived, LifecycleRestore) {
		t.Fatal("archived ledger should remain readable, exportable, and restorable")
	}
	if policy.Can(LedgerStatusArchived, LifecycleWrite) {
		t.Fatal("archived ledger must reject business writes")
	}
	if !policy.Can(LedgerStatusActive, LifecycleWrite) {
		t.Fatal("active ledger should allow lifecycle-guarded writes")
	}
}

type fakeInstanceAdminLookup struct {
	allowed bool
	err     error
}

func (f fakeInstanceAdminLookup) IsInstanceAdmin(_ context.Context, _ string) (bool, error) {
	return f.allowed, f.err
}

func TestInstancePolicyUsesIndependentAdministratorFact(t *testing.T) {
	allowed, err := NewInstancePolicy(fakeInstanceAdminLookup{allowed: true}).Can(context.Background(), "user-a")
	if err != nil || !allowed {
		t.Fatalf("expected instance admin permission, allowed=%t err=%v", allowed, err)
	}
	allowed, err = NewInstancePolicy(fakeInstanceAdminLookup{allowed: false}).Can(context.Background(), "ledger-owner")
	if err != nil || allowed {
		t.Fatalf("ledger owner without instance grant must be denied, allowed=%t err=%v", allowed, err)
	}
}

func TestIsValidRole(t *testing.T) {
	if !IsValidRole(RoleOwner) || !IsValidRole(RoleEditor) || !IsValidRole(RoleViewer) {
		t.Fatalf("expected built-in roles to be valid")
	}
	if IsValidRole(Role("admin")) || IsValidRole(Role("")) {
		t.Fatalf("unexpected role should be invalid")
	}
}
