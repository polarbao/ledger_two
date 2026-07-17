package metadata

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"testing"

	appErrors "ledger_two/internal/errors"
	ledgerctx "ledger_two/internal/ledger"
	"ledger_two/internal/metadata/defaults"
)

func TestTask531ProfilePreviewRequiresExplicitConflictResolutionAndApplyIsIdempotent(t *testing.T) {
	database := openMetadataTestDB(t)
	seedMetadataProfileLedger(t, database)
	if _, err := database.Exec(`
		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, icon, color, sort_order,
			is_system, is_archived, created_at, updated_at
		) VALUES (
			'user-food', 'ledger-profile', 'owner-profile', '餐饮', 'expense', 'user-icon', '#123456', 31,
			0, 0, '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z'
		);
		INSERT INTO tags (
			id, ledger_id, owner_user_id, name, color, sort_order, is_archived, created_at, updated_at
		) VALUES (
			'user-travel', 'ledger-profile', 'owner-profile', '旅行', '#654321', 32, 0,
			'2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z'
		);
	`); err != nil {
		t.Fatalf("seed profile conflicts: %v", err)
	}

	service := NewService(NewRepository(database))
	ctx := profileLedgerContext("owner-profile", ledgerctx.RoleOwner)
	preview, err := service.PreviewDefaultProfile(ctx, "owner-profile", ProfilePreviewRequest{Profile: defaults.ProfileBasicCNV1})
	if err != nil {
		t.Fatalf("preview default profile: %v", err)
	}
	if preview.CreateCount != 25 || preview.ReuseCount != 0 || preview.ConflictCount != 2 {
		t.Fatalf("unexpected profile preview counts: %+v", preview)
	}
	assertProfileAction(t, preview.Profile.Items, "expense_food", ProfileActionConflict, "user-food")
	assertProfileAction(t, preview.Profile.Items, "tag_travel", ProfileActionConflict, "user-travel")

	_, err = service.ApplyDefaultProfile(ctx, "owner-profile", ProfileApplyRequest{Profile: defaults.ProfileBasicCNV1})
	assertMetadataAppError(t, err, http.StatusConflict, appErrors.ErrCodeMetadataProfileConflict)

	result, err := service.ApplyDefaultProfile(ctx, "owner-profile", ProfileApplyRequest{
		Profile: defaults.ProfileBasicCNV1,
		Resolutions: []ProfileConflictResolution{
			{SystemKey: "expense_food", Action: ProfileResolutionReuse, ExistingID: "user-food"},
			{SystemKey: "tag_travel", Action: ProfileResolutionSkip},
		},
	})
	if err != nil {
		t.Fatalf("apply default profile: %v", err)
	}
	if result.CreatedCount != 25 || result.ReusedCount != 1 || result.SkippedCount != 1 || result.MetadataProfileVersion != 1 {
		t.Fatalf("unexpected profile apply result: %+v", result)
	}

	var systemKey, icon, color string
	var sortOrder int
	if err := database.QueryRow(`
		SELECT COALESCE(system_key, ''), COALESCE(icon, ''), COALESCE(color, ''), sort_order
		FROM categories WHERE id = 'user-food'
	`).Scan(&systemKey, &icon, &color, &sortOrder); err != nil {
		t.Fatalf("read reused category: %v", err)
	}
	if systemKey != "" || icon != "user-icon" || color != "#123456" || sortOrder != 31 {
		t.Fatalf("reuse changed user category: key=%q icon=%q color=%q order=%d", systemKey, icon, color, sortOrder)
	}

	repeated, err := service.ApplyDefaultProfile(ctx, "owner-profile", ProfileApplyRequest{Profile: defaults.ProfileBasicCNV1})
	if err != nil {
		t.Fatalf("repeat default profile apply: %v", err)
	}
	if repeated.CreatedCount != 0 || repeated.ReusedCount != 0 || repeated.SkippedCount != 0 || repeated.MetadataProfileVersion != 1 {
		t.Fatalf("repeat apply must be a no-op: %+v", repeated)
	}

	var profileVersion, auditCount int
	if err := database.QueryRow("SELECT metadata_profile_version FROM ledgers WHERE id = 'ledger-profile'").Scan(&profileVersion); err != nil {
		t.Fatalf("read ledger profile version: %v", err)
	}
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM audit_logs
		WHERE ledger_id = 'ledger-profile' AND action = 'metadata_profile_apply'
	`).Scan(&auditCount); err != nil {
		t.Fatalf("count profile audits: %v", err)
	}
	if profileVersion != 1 || auditCount != 2 {
		t.Fatalf("unexpected profile state: version=%d audits=%d", profileVersion, auditCount)
	}
}

func TestTask531ProfileApplyIsOwnerOnlyAndRollsBackInjectedFailure(t *testing.T) {
	database := openMetadataTestDB(t)
	seedMetadataProfileLedger(t, database)
	service := NewService(NewRepository(database))

	editorCtx := profileLedgerContext("editor-profile", ledgerctx.RoleEditor)
	_, err := service.ApplyDefaultProfile(editorCtx, "editor-profile", ProfileApplyRequest{Profile: defaults.ProfileBasicCNV1})
	assertMetadataAppError(t, err, http.StatusForbidden, appErrors.ErrCodeForbidden)

	if _, err := database.Exec(`
		CREATE TRIGGER task53_fail_profile_insert
		BEFORE INSERT ON categories
		FOR EACH ROW WHEN NEW.system_key = 'expense_health'
		BEGIN
			SELECT RAISE(ABORT, 'injected task53 profile failure');
		END;
	`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	ownerCtx := profileLedgerContext("owner-profile", ledgerctx.RoleOwner)
	if _, err := service.ApplyDefaultProfile(ownerCtx, "owner-profile", ProfileApplyRequest{Profile: defaults.ProfileBasicCNV1}); err == nil {
		t.Fatal("expected injected profile apply failure")
	}

	var metadataCount, profileVersion, auditCount int
	if err := database.QueryRow(`
		SELECT (SELECT COUNT(*) FROM categories WHERE ledger_id = 'ledger-profile') +
		       (SELECT COUNT(*) FROM tags WHERE ledger_id = 'ledger-profile')
	`).Scan(&metadataCount); err != nil {
		t.Fatalf("count rolled back metadata: %v", err)
	}
	if err := database.QueryRow("SELECT metadata_profile_version FROM ledgers WHERE id = 'ledger-profile'").Scan(&profileVersion); err != nil {
		t.Fatalf("read rolled back profile version: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE ledger_id = 'ledger-profile' AND action = 'metadata_profile_apply'").Scan(&auditCount); err != nil {
		t.Fatalf("count rolled back audit: %v", err)
	}
	if metadataCount != 0 || profileVersion != 0 || auditCount != 0 {
		t.Fatalf("profile failure left partial state: metadata=%d version=%d audits=%d", metadataCount, profileVersion, auditCount)
	}
}

func seedMetadataProfileLedger(t *testing.T, database *sql.DB) {
	t.Helper()
	_, err := database.Exec(`
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at) VALUES
			('owner-profile', 'owner_profile', 'Owner', 'hash', 'user', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z'),
			('editor-profile', 'editor_profile', 'Editor', 'hash', 'user', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z');
		INSERT INTO ledgers (id, name, default_currency, created_at, updated_at)
		VALUES ('ledger-profile', 'Profile Ledger', 'CNY', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z');
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at) VALUES
			('ledger-profile', 'owner-profile', 'owner', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z'),
			('ledger-profile', 'editor-profile', 'editor', '2026-07-17T00:00:00Z', '2026-07-17T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("seed profile ledger: %v", err)
	}
}

func profileLedgerContext(userID string, role ledgerctx.Role) context.Context {
	return ledgerctx.ContextWithLedgerContext(context.Background(), ledgerctx.LedgerContext{
		UserID:     userID,
		LedgerID:   "ledger-profile",
		Role:       role,
		Status:     ledgerctx.LedgerStatusActive,
		Version:    1,
		IsExplicit: true,
	})
}

func assertProfileAction(t *testing.T, items []ProfileItem, systemKey string, action ProfileAction, existingID string) {
	t.Helper()
	for _, item := range items {
		if item.SystemKey == systemKey {
			if item.Action != action || item.ExistingID != existingID {
				t.Fatalf("unexpected profile item %s: %+v", systemKey, item)
			}
			return
		}
	}
	t.Fatalf("profile item %s not found", systemKey)
}

func assertMetadataAppError(t *testing.T, err error, status int, code string) {
	t.Helper()
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.Status != status || appErr.Code != code {
		t.Fatalf("expected app error status=%d code=%s, got %v", status, code, err)
	}
}
