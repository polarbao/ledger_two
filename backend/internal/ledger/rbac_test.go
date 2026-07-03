package ledger

import "testing"

func TestRolePolicyOwnerPermissions(t *testing.T) {
	policy := NewRolePolicy()

	allowed := []Operation{
		OperationViewLedger,
		OperationViewMembers,
		OperationRenameLedger,
		OperationManageMembers,
		OperationCreateTransaction,
		OperationEditOwnTransaction,
		OperationDeleteOwnTransaction,
		OperationCreateSharedExpense,
		OperationCreateSettlement,
		OperationViewReports,
		OperationExportData,
		OperationManualBackup,
		OperationRestoreBackup,
		OperationManageMetadata,
	}
	for _, operation := range allowed {
		if !policy.Can(RoleOwner, operation) {
			t.Fatalf("owner should be allowed to %s", operation)
		}
	}

	if policy.Can(RoleOwner, OperationEditAnyTransaction) {
		t.Fatalf("owner should not edit others' transactions by default")
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
		OperationExportData,
		OperationManualBackup,
		OperationRestoreBackup,
		OperationManageMetadata,
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
		OperationManualBackup,
		OperationRestoreBackup,
		OperationManageMetadata,
	}
	for _, operation := range denied {
		if policy.Can(RoleViewer, operation) {
			t.Fatalf("viewer should not be allowed to %s", operation)
		}
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
