package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	appErrors "ledger_two/internal/errors"
)

type fakeUnsettledBalanceProvider struct {
	snapshot UnsettledBalanceSnapshot
	err      error
}

func (f fakeUnsettledBalanceProvider) GetUnsettledBalance(_ context.Context, _ *sql.Tx, _ LedgerContext) (UnsettledBalanceSnapshot, error) {
	return f.snapshot, f.err
}

func TestTask503ACreateLedgerValidatesUnicodeNameAndHasNoArbitraryCountCap(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	service := NewService(NewRepository(database))

	for _, name := range []string{"", "   ", strings.Repeat("账", 61)} {
		_, err := service.CreateLedger(context.Background(), "user-a", CreateLedgerReq{Name: name})
		assertLedgerAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeValidationError)
	}

	validName := strings.Repeat("账", 60)
	created, err := service.CreateLedger(context.Background(), "user-a", CreateLedgerReq{Name: "  " + validName + "  "})
	if err != nil {
		t.Fatalf("create unicode ledger: %v", err)
	}
	if created.Name != validName || created.Role != string(RoleOwner) || created.Status != LedgerStatusActive || created.Version != 1 || created.MemberCount != 1 {
		t.Fatalf("unexpected created ledger: %+v", created)
	}

	for i := 0; i < 65; i++ {
		if _, err := service.CreateLedger(context.Background(), "user-a", CreateLedgerReq{Name: "同名账本"}); err != nil {
			t.Fatalf("create ledger %d without a frozen count cap: %v", i, err)
		}
	}

	var createAudits int
	if err := database.QueryRow(`
		SELECT COUNT(*)
		FROM audit_logs
		WHERE actor_user_id = 'user-a'
		  AND actor_role = 'owner'
		  AND action = 'ledger_create'
	`).Scan(&createAudits); err != nil {
		t.Fatalf("count create audits: %v", err)
	}
	if createAudits != 66 {
		t.Fatalf("expected 66 create audits, got %d", createAudits)
	}
}

func TestTask531CreateLedgerAppliesDefaultOrExplicitEmptyMetadataAtomically(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	service := NewService(NewRepository(database))

	created, err := service.CreateLedger(context.Background(), "user-a", CreateLedgerReq{Name: "默认基础包"})
	if err != nil {
		t.Fatalf("create ledger with default metadata: %v", err)
	}
	assertLedgerMetadataCounts(t, database, created.ID, 19, 8, 1)

	empty, err := service.CreateLedger(context.Background(), "user-a", CreateLedgerReq{Name: "空基础包", MetadataProfile: "empty"})
	if err != nil {
		t.Fatalf("create ledger with empty metadata: %v", err)
	}
	assertLedgerMetadataCounts(t, database, empty.ID, 0, 0, 0)

	_, err = service.CreateLedger(context.Background(), "user-a", CreateLedgerReq{Name: "无效基础包", MetadataProfile: "unknown"})
	assertLedgerAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeValidationError)
}

func TestTask531CreateLedgerRollsBackLedgerOwnerMetadataAndAuditOnProfileFailure(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	service := NewService(NewRepository(database))

	if _, err := database.Exec(`
		CREATE TRIGGER task53_fail_new_ledger_profile
		BEFORE INSERT ON categories
		FOR EACH ROW WHEN NEW.system_key = 'expense_health'
		BEGIN
			SELECT RAISE(ABORT, 'injected new ledger profile failure');
		END;
	`); err != nil {
		t.Fatalf("create new ledger failure trigger: %v", err)
	}

	var ledgerCountBefore, memberCountBefore, auditCountBefore int
	if err := database.QueryRow("SELECT COUNT(*) FROM ledgers").Scan(&ledgerCountBefore); err != nil {
		t.Fatalf("count ledgers before failure: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM ledger_members").Scan(&memberCountBefore); err != nil {
		t.Fatalf("count members before failure: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&auditCountBefore); err != nil {
		t.Fatalf("count audits before failure: %v", err)
	}

	if _, err := service.CreateLedger(context.Background(), "user-a", CreateLedgerReq{Name: "应整体回滚"}); err == nil {
		t.Fatal("expected injected new ledger profile failure")
	}

	var ledgerCountAfter, memberCountAfter, metadataCountAfter, auditCountAfter int
	if err := database.QueryRow("SELECT COUNT(*) FROM ledgers").Scan(&ledgerCountAfter); err != nil {
		t.Fatalf("count ledgers after failure: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM ledger_members").Scan(&memberCountAfter); err != nil {
		t.Fatalf("count members after failure: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM categories WHERE system_key IS NOT NULL").Scan(&metadataCountAfter); err != nil {
		t.Fatalf("count metadata after failure: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&auditCountAfter); err != nil {
		t.Fatalf("count audits after failure: %v", err)
	}
	if ledgerCountAfter != ledgerCountBefore || memberCountAfter != memberCountBefore || metadataCountAfter != 0 || auditCountAfter != auditCountBefore {
		t.Fatalf(
			"new ledger failure left partial state: ledgers %d->%d members %d->%d metadata=%d audits %d->%d",
			ledgerCountBefore,
			ledgerCountAfter,
			memberCountBefore,
			memberCountAfter,
			metadataCountAfter,
			auditCountBefore,
			auditCountAfter,
		)
	}
}

func assertLedgerMetadataCounts(t *testing.T, database *sql.DB, ledgerID string, wantCategories, wantTags, wantVersion int) {
	t.Helper()
	var categoryCount, tagCount, profileVersion int
	if err := database.QueryRow("SELECT COUNT(*) FROM categories WHERE ledger_id = ?", ledgerID).Scan(&categoryCount); err != nil {
		t.Fatalf("count ledger categories: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM tags WHERE ledger_id = ?", ledgerID).Scan(&tagCount); err != nil {
		t.Fatalf("count ledger tags: %v", err)
	}
	if err := database.QueryRow("SELECT metadata_profile_version FROM ledgers WHERE id = ?", ledgerID).Scan(&profileVersion); err != nil {
		t.Fatalf("read ledger profile version: %v", err)
	}
	if categoryCount != wantCategories || tagCount != wantTags || profileVersion != wantVersion {
		t.Fatalf(
			"unexpected ledger metadata counts: categories=%d/%d tags=%d/%d version=%d/%d",
			categoryCount,
			wantCategories,
			tagCount,
			wantTags,
			profileVersion,
			wantVersion,
		)
	}
}

func TestTask503AListDetailAndRenameUseFrozenStatusAndVersionContract(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	service := NewService(NewRepository(database))
	ctx := context.Background()

	active, err := service.ListUserLedgers(ctx, "user-a", LedgerListActive)
	if err != nil || len(active) != 1 || active[0].ID != "ledger-active" {
		t.Fatalf("unexpected active list: %+v err=%v", active, err)
	}
	archived, err := service.ListUserLedgers(ctx, "user-a", LedgerListArchived)
	if err != nil || len(archived) != 1 || archived[0].ID != "ledger-archived" {
		t.Fatalf("unexpected archived list: %+v err=%v", archived, err)
	}

	lc := LedgerContext{UserID: "user-a", LedgerID: "ledger-active", Role: RoleOwner, Status: LedgerStatusActive, Version: 3, IsExplicit: true}
	detail, err := service.GetLedger(ctx, lc)
	if err != nil {
		t.Fatalf("get ledger detail: %v", err)
	}
	if detail.ID != lc.LedgerID || detail.Role != string(RoleOwner) || detail.Version != 3 {
		t.Fatalf("unexpected detail: %+v", detail)
	}

	renamed, err := service.RenameLedger(ctx, lc, 3, RenameLedgerReq{Name: "  新名称  "})
	if err != nil {
		t.Fatalf("rename ledger: %v", err)
	}
	if renamed.Name != "新名称" || renamed.Version != 4 || renamed.Role != string(RoleOwner) {
		t.Fatalf("unexpected renamed ledger: %+v", renamed)
	}

	_, err = service.RenameLedger(ctx, lc, 3, RenameLedgerReq{Name: "旧版本覆盖"})
	assertLedgerAppError(t, err, http.StatusConflict, appErrors.ErrCodeLedgerVersionConflict)

	var name string
	var version int64
	if err := database.QueryRow("SELECT name, version FROM ledgers WHERE id = 'ledger-active'").Scan(&name, &version); err != nil {
		t.Fatalf("read renamed ledger: %v", err)
	}
	if name != "新名称" || version != 4 {
		t.Fatalf("stale rename changed ledger: name=%s version=%d", name, version)
	}

	var auditRole string
	if err := database.QueryRow(`
		SELECT actor_role
		FROM audit_logs
		WHERE ledger_id = 'ledger-active' AND action = 'ledger_rename'
	`).Scan(&auditRole); err != nil {
		t.Fatalf("read rename audit: %v", err)
	}
	if auditRole != string(RoleOwner) {
		t.Fatalf("unexpected audit role %q", auditRole)
	}
}

func TestTask503AArchivePreflightIsReadOnlyAndUsesServerSnapshot(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	insertLifecycleImportBatch(t, database, "ready-future", "ledger-active", "user-a", "ready", "2099-01-01T00:00:00Z")
	provider := fakeUnsettledBalanceProvider{snapshot: UnsettledBalanceSnapshot{
		FromUserID:  lifecycleStringPtr("user-b"),
		ToUserID:    lifecycleStringPtr("user-a"),
		AmountCents: 1250,
	}}
	service := NewService(NewRepository(database), provider)
	lc := LedgerContext{UserID: "user-a", LedgerID: "ledger-active", Role: RoleOwner, Status: LedgerStatusActive, Version: 3, IsExplicit: true}

	preflight, err := service.GetArchivePreflight(context.Background(), lc)
	if err != nil {
		t.Fatalf("archive preflight: %v", err)
	}
	if preflight.Ledger.Version != 3 || preflight.ReadyImportBatchCount != 1 || preflight.CanArchive || !preflight.RequiresUnsettledAcknowledgement {
		t.Fatalf("unexpected preflight: %+v", preflight)
	}
	if preflight.UnsettledBalance != provider.snapshot {
		t.Fatalf("unexpected balance snapshot: %+v", preflight.UnsettledBalance)
	}

	var version int64
	var auditCount int
	if err := database.QueryRow("SELECT version FROM ledgers WHERE id = 'ledger-active'").Scan(&version); err != nil {
		t.Fatalf("read preflight version: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE ledger_id = 'ledger-active'").Scan(&auditCount); err != nil {
		t.Fatalf("count preflight audits: %v", err)
	}
	if version != 3 || auditCount != 0 {
		t.Fatalf("preflight wrote data: version=%d audits=%d", version, auditCount)
	}
}

func TestTask503AArchiveRollsBackVersionAndExpiryCleanupWhenReadyBatchBlocks(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	insertLifecycleImportBatch(t, database, "ready-expired", "ledger-active", "user-a", "ready", "2020-01-01T00:00:00Z")
	insertLifecycleImportBatch(t, database, "ready-future", "ledger-active", "user-a", "ready", "2099-01-01T00:00:00Z")
	service := NewService(NewRepository(database), fakeUnsettledBalanceProvider{})
	lc := LedgerContext{UserID: "user-a", LedgerID: "ledger-active", Role: RoleOwner, Status: LedgerStatusActive, Version: 3, IsExplicit: true}

	_, err := service.ArchiveLedger(context.Background(), lc, 3, ArchiveLedgerReq{AcknowledgeUnsettledBalance: lifecycleBoolPtr(false)})
	assertLedgerAppError(t, err, http.StatusConflict, appErrors.ErrCodeLedgerReadyImportExists)

	var status string
	var version int64
	if err := database.QueryRow("SELECT status, version FROM ledgers WHERE id = 'ledger-active'").Scan(&status, &version); err != nil {
		t.Fatalf("read blocked ledger: %v", err)
	}
	if status != string(LedgerStatusActive) || version != 3 {
		t.Fatalf("blocked archive changed ledger: status=%s version=%d", status, version)
	}
	for _, batchID := range []string{"ready-expired", "ready-future"} {
		var batchStatus string
		if err := database.QueryRow("SELECT status FROM import_batches WHERE id = ?", batchID).Scan(&batchStatus); err != nil {
			t.Fatalf("read batch %s: %v", batchID, err)
		}
		if batchStatus != "ready" {
			t.Fatalf("blocked archive changed batch %s to %s", batchID, batchStatus)
		}
	}
}

func TestTask503AArchiveAndRestoreAreAtomicAuditedLifecycleMutations(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	insertLifecycleImportBatch(t, database, "ready-expired", "ledger-active", "user-a", "ready", "2020-01-01T00:00:00Z")
	provider := fakeUnsettledBalanceProvider{snapshot: UnsettledBalanceSnapshot{
		FromUserID:  lifecycleStringPtr("user-b"),
		ToUserID:    lifecycleStringPtr("user-a"),
		AmountCents: 2500,
	}}
	service := NewService(NewRepository(database), provider)
	lc := LedgerContext{UserID: "user-a", LedgerID: "ledger-active", Role: RoleOwner, Status: LedgerStatusActive, Version: 3, IsExplicit: true}

	_, err := service.ArchiveLedger(context.Background(), lc, 3, ArchiveLedgerReq{})
	assertLedgerAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeValidationError)

	_, err = service.ArchiveLedger(context.Background(), lc, 3, ArchiveLedgerReq{AcknowledgeUnsettledBalance: lifecycleBoolPtr(false)})
	assertLedgerAppError(t, err, http.StatusBadRequest, appErrors.ErrCodeValidationError)

	archived, err := service.ArchiveLedger(context.Background(), lc, 3, ArchiveLedgerReq{AcknowledgeUnsettledBalance: lifecycleBoolPtr(true)})
	if err != nil {
		t.Fatalf("archive ledger: %v", err)
	}
	if archived.Status != LedgerStatusArchived || archived.Version != 4 || archived.ArchivedAt == nil || archived.ArchivedByUserID == nil || *archived.ArchivedByUserID != "user-a" {
		t.Fatalf("unexpected archived ledger: %+v", archived)
	}

	var batchStatus string
	if err := database.QueryRow("SELECT status FROM import_batches WHERE id = 'ready-expired'").Scan(&batchStatus); err != nil {
		t.Fatalf("read expired batch: %v", err)
	}
	if batchStatus != "expired" {
		t.Fatalf("expected expired batch cleanup, got %s", batchStatus)
	}

	archivedContext := LedgerContext{UserID: "user-a", LedgerID: "ledger-active", Role: RoleOwner, Status: LedgerStatusArchived, Version: 4, IsExplicit: true}
	restored, err := service.RestoreLedger(context.Background(), archivedContext, 4)
	if err != nil {
		t.Fatalf("restore ledger: %v", err)
	}
	if restored.Status != LedgerStatusActive || restored.Version != 5 || restored.ArchivedAt != nil || restored.ArchivedByUserID != nil {
		t.Fatalf("unexpected restored ledger: %+v", restored)
	}

	rows, err := database.Query(`
		SELECT action, actor_role
		FROM audit_logs
		WHERE ledger_id = 'ledger-active'
		  AND action IN ('ledger_archive', 'ledger_restore')
		ORDER BY created_at, action
	`)
	if err != nil {
		t.Fatalf("query lifecycle audits: %v", err)
	}
	defer rows.Close()
	seen := map[string]string{}
	for rows.Next() {
		var action, role string
		if err := rows.Scan(&action, &role); err != nil {
			t.Fatalf("scan lifecycle audit: %v", err)
		}
		seen[action] = role
	}
	if seen["ledger_archive"] != string(RoleOwner) || seen["ledger_restore"] != string(RoleOwner) {
		t.Fatalf("unexpected lifecycle audits: %+v", seen)
	}

	var archiveAfterJSON string
	if err := database.QueryRow(`
		SELECT after_json
		FROM audit_logs
		WHERE ledger_id = 'ledger-active' AND action = 'ledger_archive'
	`).Scan(&archiveAfterJSON); err != nil {
		t.Fatalf("read archive audit payload: %v", err)
	}
	var archiveAfter struct {
		Status                LedgerStatus             `json:"status"`
		Version               int64                    `json:"version"`
		UnsettledBalance      UnsettledBalanceSnapshot `json:"unsettled_balance"`
		ReadyImportBatchCount int                      `json:"ready_import_batch_count"`
	}
	if err := json.Unmarshal([]byte(archiveAfterJSON), &archiveAfter); err != nil {
		t.Fatalf("decode archive audit payload: %v", err)
	}
	if archiveAfter.Status != LedgerStatusArchived || archiveAfter.Version != 4 || archiveAfter.UnsettledBalance.AmountCents != 2500 || archiveAfter.ReadyImportBatchCount != 0 {
		t.Fatalf("unexpected archive audit payload: %+v", archiveAfter)
	}
}

func TestTask503ALifecycleServiceRechecksOwnerAndStateInsideTransaction(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	service := NewService(NewRepository(database), fakeUnsettledBalanceProvider{})

	viewerContext := LedgerContext{UserID: "user-b", LedgerID: "ledger-active", Role: RoleOwner, Status: LedgerStatusActive, Version: 3, IsExplicit: true}
	_, err := service.RenameLedger(context.Background(), viewerContext, 3, RenameLedgerReq{Name: "伪造 Owner"})
	assertLedgerAppError(t, err, http.StatusForbidden, appErrors.ErrCodeLedgerAccessDenied)

	activeContext := LedgerContext{UserID: "user-a", LedgerID: "ledger-active", Role: RoleOwner, Status: LedgerStatusArchived, Version: 3, IsExplicit: true}
	_, err = service.RestoreLedger(context.Background(), activeContext, 3)
	assertLedgerAppError(t, err, http.StatusConflict, appErrors.ErrCodeLedgerInvalidState)

	archivedContext := LedgerContext{UserID: "user-a", LedgerID: "ledger-archived", Role: RoleOwner, Status: LedgerStatusActive, Version: 8, IsExplicit: true}
	_, err = service.RenameLedger(context.Background(), archivedContext, 8, RenameLedgerReq{Name: "归档改名"})
	assertLedgerAppError(t, err, http.StatusConflict, appErrors.ErrCodeLedgerArchived)
}

func insertLifecycleImportBatch(t *testing.T, database *sql.DB, batchID, ledgerID, userID, status, expiresAt string) {
	t.Helper()
	if _, err := database.Exec(`
		INSERT INTO import_batches (
			id, ledger_id, filename, created_by_user_id, status, created_at, expires_at
		) VALUES (?, ?, 'fixture.csv', ?, ?, '2026-07-01T00:00:00Z', ?)
	`, batchID, ledgerID, userID, status, expiresAt); err != nil {
		t.Fatalf("insert import batch %s: %v", batchID, err)
	}
}

func assertLedgerAppError(t *testing.T, err error, status int, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s", code)
	}
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.Status != status || appErr.Code != code {
		t.Fatalf("expected %d/%s, got %d/%s", status, code, appErr.Status, appErr.Code)
	}
}

func lifecycleStringPtr(value string) *string {
	return &value
}

func lifecycleBoolPtr(value bool) *bool {
	return &value
}
