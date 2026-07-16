package ledger

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestTask503BAddMemberIsVersionedAtomicAndAudited(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	insertMemberTestLedger(t, database, "ledger-single", "user-a", 1)

	service := NewService(NewRepository(database))
	lc := LedgerContext{
		UserID:     "user-a",
		LedgerID:   "ledger-single",
		Role:       RoleOwner,
		Status:     LedgerStatusActive,
		Version:    1,
		IsExplicit: true,
	}

	result, err := service.AddMemberVersioned(context.Background(), lc, 1, AddMemberReq{
		Username:                     "cara",
		Role:                         string(RoleEditor),
		AcknowledgeHistoryVisibility: true,
	})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if result.Ledger.Version != 2 || result.Ledger.MemberCount != 2 {
		t.Fatalf("unexpected ledger after add: %+v", result.Ledger)
	}
	if len(result.Members) != 2 || result.Members[1].UserID != "user-c" || result.Members[1].Role != string(RoleEditor) {
		t.Fatalf("unexpected members after add: %+v", result.Members)
	}
	if result.Members[1].JoinedAt.IsZero() {
		t.Fatal("expected joined_at for added member")
	}

	var auditCount int
	if err := database.QueryRow(`
		SELECT COUNT(*)
		FROM audit_logs
		WHERE ledger_id = 'ledger-single'
		  AND actor_user_id = 'user-a'
		  AND actor_role = 'owner'
		  AND action = 'ledger_member_add'
	`).Scan(&auditCount); err != nil {
		t.Fatalf("count member add audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one member add audit, got %d", auditCount)
	}
}

func TestTask503BMemberFailuresRollBackVersionAndMembership(t *testing.T) {
	t.Run("history acknowledgement required", func(t *testing.T) {
		database := openLedgerRepositoryTestDB(t)
		seedLedgerRepositoryFixtures(t, database)
		insertMemberTestLedger(t, database, "ledger-ack", "user-a", 1)
		service := NewService(NewRepository(database))
		lc := LedgerContext{
			UserID:     "user-a",
			LedgerID:   "ledger-ack",
			Role:       RoleOwner,
			Status:     LedgerStatusActive,
			Version:    1,
			IsExplicit: true,
		}
		_, err := service.AddMemberVersioned(context.Background(), lc, 1, AddMemberReq{
			Username: "cara",
			Role:     string(RoleEditor),
		})
		assertLedgerAppError(t, err, 400, "VALIDATION_ERROR")
		assertMemberLedgerVersionAndCount(t, database, "ledger-ack", 1, 1)
	})

	t.Run("third member rejected", func(t *testing.T) {
		database := openLedgerRepositoryTestDB(t)
		seedLedgerRepositoryFixtures(t, database)
		service := NewService(NewRepository(database))
		lc := LedgerContext{
			UserID:     "user-a",
			LedgerID:   "ledger-active",
			Role:       RoleOwner,
			Status:     LedgerStatusActive,
			Version:    3,
			IsExplicit: true,
		}
		_, err := service.AddMemberVersioned(context.Background(), lc, 3, AddMemberReq{
			Username:                     "cara",
			Role:                         string(RoleViewer),
			AcknowledgeHistoryVisibility: true,
		})
		assertLedgerAppError(t, err, 409, "LEDGER_MEMBER_LIMIT_REACHED")
		assertMemberLedgerVersionAndCount(t, database, "ledger-active", 3, 2)
	})

	t.Run("inactive user rejected", func(t *testing.T) {
		database := openLedgerRepositoryTestDB(t)
		seedLedgerRepositoryFixtures(t, database)
		insertMemberTestLedger(t, database, "ledger-inactive", "user-a", 1)
		if _, err := database.Exec("UPDATE users SET is_active = 0 WHERE id = 'user-c'"); err != nil {
			t.Fatalf("deactivate user: %v", err)
		}
		service := NewService(NewRepository(database))
		lc := LedgerContext{
			UserID:     "user-a",
			LedgerID:   "ledger-inactive",
			Role:       RoleOwner,
			Status:     LedgerStatusActive,
			Version:    1,
			IsExplicit: true,
		}
		_, err := service.AddMemberVersioned(context.Background(), lc, 1, AddMemberReq{
			Username:                     "cara",
			Role:                         string(RoleViewer),
			AcknowledgeHistoryVisibility: true,
		})
		assertLedgerAppError(t, err, 404, "NOT_FOUND")
		assertMemberLedgerVersionAndCount(t, database, "ledger-inactive", 1, 1)
	})

	t.Run("stale version rejected", func(t *testing.T) {
		database := openLedgerRepositoryTestDB(t)
		seedLedgerRepositoryFixtures(t, database)
		insertMemberTestLedger(t, database, "ledger-stale-member", "user-a", 1)
		service := NewService(NewRepository(database))
		lc := LedgerContext{
			UserID:     "user-a",
			LedgerID:   "ledger-stale-member",
			Role:       RoleOwner,
			Status:     LedgerStatusActive,
			Version:    1,
			IsExplicit: true,
		}
		_, err := service.AddMemberVersioned(context.Background(), lc, 2, AddMemberReq{
			Username:                     "cara",
			Role:                         string(RoleViewer),
			AcknowledgeHistoryVisibility: true,
		})
		assertLedgerAppError(t, err, 409, "LEDGER_VERSION_CONFLICT")
		assertMemberLedgerVersionAndCount(t, database, "ledger-stale-member", 1, 1)
	})

	t.Run("audit failure rolls back member and version", func(t *testing.T) {
		database := openLedgerRepositoryTestDB(t)
		seedLedgerRepositoryFixtures(t, database)
		insertMemberTestLedger(t, database, "ledger-audit-failure", "user-a", 1)
		if _, err := database.Exec(`
			CREATE TRIGGER fail_member_add_audit
			BEFORE INSERT ON audit_logs
			FOR EACH ROW
			WHEN NEW.action = 'ledger_member_add'
			BEGIN
				SELECT RAISE(ABORT, 'injected member audit failure');
			END;
		`); err != nil {
			t.Fatalf("create member audit failure trigger: %v", err)
		}
		service := NewService(NewRepository(database))
		lc := LedgerContext{
			UserID:     "user-a",
			LedgerID:   "ledger-audit-failure",
			Role:       RoleOwner,
			Status:     LedgerStatusActive,
			Version:    1,
			IsExplicit: true,
		}
		_, err := service.AddMemberVersioned(context.Background(), lc, 1, AddMemberReq{
			Username:                     "cara",
			Role:                         string(RoleViewer),
			AcknowledgeHistoryVisibility: true,
		})
		if err == nil {
			t.Fatal("expected injected member audit failure")
		}
		assertMemberLedgerVersionAndCount(t, database, "ledger-audit-failure", 1, 1)
	})
}

func TestTask503BRoleUpdateRejectsOwnerAndUsesVersionedTransaction(t *testing.T) {
	database := openLedgerRepositoryTestDB(t)
	seedLedgerRepositoryFixtures(t, database)
	service := NewService(NewRepository(database))
	lc := LedgerContext{
		UserID:     "user-a",
		LedgerID:   "ledger-active",
		Role:       RoleOwner,
		Status:     LedgerStatusActive,
		Version:    3,
		IsExplicit: true,
	}

	result, err := service.UpdateMemberRoleVersioned(
		context.Background(),
		lc,
		3,
		"user-b",
		UpdateMemberReq{Role: string(RoleViewer)},
	)
	if err != nil {
		t.Fatalf("update member role: %v", err)
	}
	if result.Ledger.Version != 4 || result.Members[1].Role != string(RoleViewer) {
		t.Fatalf("unexpected role update result: %+v", result)
	}

	_, err = service.UpdateMemberRoleVersioned(
		context.Background(),
		LedgerContext{
			UserID:     "user-b",
			LedgerID:   "ledger-active",
			Role:       RoleViewer,
			Status:     LedgerStatusActive,
			Version:    4,
			IsExplicit: true,
		},
		4,
		"user-a",
		UpdateMemberReq{Role: string(RoleEditor)},
	)
	assertLedgerAppError(t, err, 403, "LEDGER_ACCESS_DENIED")

	_, err = service.UpdateMemberRoleVersioned(
		context.Background(),
		LedgerContext{
			UserID:     "user-a",
			LedgerID:   "ledger-active",
			Role:       RoleOwner,
			Status:     LedgerStatusActive,
			Version:    4,
			IsExplicit: true,
		},
		4,
		"user-a",
		UpdateMemberReq{Role: string(RoleEditor)},
	)
	assertLedgerAppError(t, err, 409, "LEDGER_OWNER_TRANSFER_REQUIRED")

	var version int64
	if err := database.QueryRow("SELECT version FROM ledgers WHERE id = 'ledger-active'").Scan(&version); err != nil {
		t.Fatalf("read version after rejected owner update: %v", err)
	}
	if version != 4 {
		t.Fatalf("rejected owner update changed version to %d", version)
	}
}

func TestTask503BTransferOwnerIsAtomicAndRollsBackInjectedFailure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		database := openLedgerRepositoryTestDB(t)
		seedLedgerRepositoryFixtures(t, database)
		service := NewService(NewRepository(database))
		lc := LedgerContext{
			UserID:     "user-a",
			LedgerID:   "ledger-active",
			Role:       RoleOwner,
			Status:     LedgerStatusActive,
			Version:    3,
			IsExplicit: true,
		}

		result, err := service.TransferOwnerVersioned(
			context.Background(),
			lc,
			3,
			"user-b",
			TransferOwnerReq{AcknowledgePermissionChange: true},
		)
		if err != nil {
			t.Fatalf("transfer owner: %v", err)
		}
		if result.Ledger.Version != 4 || result.Ledger.Role != string(RoleEditor) {
			t.Fatalf("unexpected ledger after transfer: %+v", result.Ledger)
		}
		roles := memberRoles(result.Members)
		if roles["user-a"] != string(RoleEditor) || roles["user-b"] != string(RoleOwner) {
			t.Fatalf("unexpected roles after transfer: %+v", roles)
		}

		var ownerCount, auditCount int
		if err := database.QueryRow(`
			SELECT COUNT(*) FROM ledger_members
			WHERE ledger_id = 'ledger-active' AND role = 'owner'
		`).Scan(&ownerCount); err != nil {
			t.Fatalf("count owners after transfer: %v", err)
		}
		if err := database.QueryRow(`
			SELECT COUNT(*) FROM audit_logs
			WHERE ledger_id = 'ledger-active' AND action = 'ledger_owner_transfer'
		`).Scan(&auditCount); err != nil {
			t.Fatalf("count owner transfer audits: %v", err)
		}
		if ownerCount != 1 || auditCount != 1 {
			t.Fatalf("unexpected transfer invariants: owners=%d audits=%d", ownerCount, auditCount)
		}
	})

	t.Run("rollback", func(t *testing.T) {
		database := openLedgerRepositoryTestDB(t)
		seedLedgerRepositoryFixtures(t, database)
		if _, err := database.Exec(`
			CREATE TRIGGER fail_owner_transfer_target
			BEFORE UPDATE OF role ON ledger_members
			FOR EACH ROW
			WHEN NEW.ledger_id = 'ledger-active'
			  AND NEW.user_id = 'user-b'
			  AND NEW.role = 'owner'
			BEGIN
				SELECT RAISE(ABORT, 'injected owner transfer failure');
			END;
		`); err != nil {
			t.Fatalf("create failure trigger: %v", err)
		}
		service := NewService(NewRepository(database))
		lc := LedgerContext{
			UserID:     "user-a",
			LedgerID:   "ledger-active",
			Role:       RoleOwner,
			Status:     LedgerStatusActive,
			Version:    3,
			IsExplicit: true,
		}

		_, err := service.TransferOwnerVersioned(
			context.Background(),
			lc,
			3,
			"user-b",
			TransferOwnerReq{AcknowledgePermissionChange: true},
		)
		if err == nil {
			t.Fatal("expected injected owner transfer failure")
		}

		var version int64
		if err := database.QueryRow("SELECT version FROM ledgers WHERE id = 'ledger-active'").Scan(&version); err != nil {
			t.Fatalf("read version after rollback: %v", err)
		}
		roles := queryMemberRoles(t, database, "ledger-active")
		if version != 3 || roles["user-a"] != string(RoleOwner) || roles["user-b"] != string(RoleEditor) {
			t.Fatalf("owner transfer failure was not rolled back: version=%d roles=%+v", version, roles)
		}
		var auditCount int
		if err := database.QueryRow(`
			SELECT COUNT(*) FROM audit_logs
			WHERE ledger_id = 'ledger-active' AND action = 'ledger_owner_transfer'
		`).Scan(&auditCount); err != nil {
			t.Fatalf("count rollback audits: %v", err)
		}
		if auditCount != 0 {
			t.Fatalf("rollback left %d transfer audits", auditCount)
		}
	})
}

func TestTask503BRemoveAndLeavePreserveHistoricalLedgerObjects(t *testing.T) {
	t.Run("owner removes partner", func(t *testing.T) {
		database := openLedgerRepositoryTestDB(t)
		seedLedgerRepositoryFixtures(t, database)
		insertMemberHistoryTransaction(t, database, "history-remove", "ledger-active", "user-b")
		service := NewService(NewRepository(database))
		lc := LedgerContext{
			UserID:     "user-a",
			LedgerID:   "ledger-active",
			Role:       RoleOwner,
			Status:     LedgerStatusActive,
			Version:    3,
			IsExplicit: true,
		}

		result, err := service.RemoveMemberVersioned(context.Background(), lc, 3, "user-b")
		if err != nil {
			t.Fatalf("remove member: %v", err)
		}
		if result.Ledger.Version != 4 || result.Ledger.MemberCount != 1 || len(result.Members) != 1 {
			t.Fatalf("unexpected member removal result: %+v", result)
		}
		assertHistoricalTransactionPreserved(t, database, "history-remove", "user-b")
	})

	t.Run("editor leaves with committed etag version", func(t *testing.T) {
		database := openLedgerRepositoryTestDB(t)
		seedLedgerRepositoryFixtures(t, database)
		insertMemberHistoryTransaction(t, database, "history-leave", "ledger-active", "user-b")
		service := NewService(NewRepository(database))
		lc := LedgerContext{
			UserID:     "user-b",
			LedgerID:   "ledger-active",
			Role:       RoleEditor,
			Status:     LedgerStatusActive,
			Version:    3,
			IsExplicit: true,
		}

		result, err := service.LeaveLedgerVersioned(context.Background(), lc, 3)
		if err != nil {
			t.Fatalf("leave ledger: %v", err)
		}
		if result.LedgerID != "ledger-active" || result.Version != 4 {
			t.Fatalf("unexpected leave result: %+v", result)
		}
		if _, err := NewRepository(database).GetMemberRole(context.Background(), "ledger-active", "user-b"); err == nil {
			t.Fatal("expected leaving member relationship to be removed")
		}
		assertHistoricalTransactionPreserved(t, database, "history-leave", "user-b")
	})

	t.Run("owner must transfer before leaving", func(t *testing.T) {
		database := openLedgerRepositoryTestDB(t)
		seedLedgerRepositoryFixtures(t, database)
		service := NewService(NewRepository(database))
		lc := LedgerContext{
			UserID:     "user-a",
			LedgerID:   "ledger-active",
			Role:       RoleOwner,
			Status:     LedgerStatusActive,
			Version:    3,
			IsExplicit: true,
		}

		_, err := service.LeaveLedgerVersioned(context.Background(), lc, 3)
		assertLedgerAppError(t, err, 409, "LEDGER_OWNER_TRANSFER_REQUIRED")

		_, err = service.RemoveMemberVersioned(context.Background(), lc, 3, "user-a")
		assertLedgerAppError(t, err, 409, "LEDGER_OWNER_TRANSFER_REQUIRED")

		var version int64
		if err := database.QueryRow("SELECT version FROM ledgers WHERE id = 'ledger-active'").Scan(&version); err != nil {
			t.Fatalf("read owner leave version: %v", err)
		}
		if version != 3 {
			t.Fatalf("rejected owner leave changed version to %d", version)
		}
	})
}

func insertMemberHistoryTransaction(t *testing.T, database *sql.DB, transactionID, ledgerID, userID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := database.Exec(`
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, currency, occurred_at,
			owner_user_id, created_by_user_id, payer_user_id, visibility,
			status, created_at, updated_at
		) VALUES (?, ?, 'expense', 'Historical member expense', 321, 'CNY', ?,
			?, ?, ?, 'partner_readable', 'normal', ?, ?)
	`, transactionID, ledgerID, now, userID, userID, userID, now, now); err != nil {
		t.Fatalf("insert historical transaction: %v", err)
	}
}

func assertHistoricalTransactionPreserved(t *testing.T, database *sql.DB, transactionID, userID string) {
	t.Helper()
	var amount int64
	var ownerID, creatorID, payerID string
	if err := database.QueryRow(`
		SELECT amount, owner_user_id, created_by_user_id, payer_user_id
		FROM transactions
		WHERE id = ?
	`, transactionID).Scan(&amount, &ownerID, &creatorID, &payerID); err != nil {
		t.Fatalf("read historical transaction: %v", err)
	}
	if amount != 321 || ownerID != userID || creatorID != userID || payerID != userID {
		t.Fatalf(
			"historical transaction changed: amount=%d owner=%s creator=%s payer=%s",
			amount,
			ownerID,
			creatorID,
			payerID,
		)
	}
}

func assertMemberLedgerVersionAndCount(t *testing.T, database *sql.DB, ledgerID string, version int64, memberCount int) {
	t.Helper()
	var actualVersion int64
	var actualCount int
	if err := database.QueryRow("SELECT version FROM ledgers WHERE id = ?", ledgerID).Scan(&actualVersion); err != nil {
		t.Fatalf("read member ledger version: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM ledger_members WHERE ledger_id = ?", ledgerID).Scan(&actualCount); err != nil {
		t.Fatalf("count member relationships: %v", err)
	}
	if actualVersion != version || actualCount != memberCount {
		t.Fatalf(
			"unexpected member ledger state: version=%d/%d members=%d/%d",
			actualVersion,
			version,
			actualCount,
			memberCount,
		)
	}
}

func memberRoles(members []MemberDetail) map[string]string {
	result := make(map[string]string, len(members))
	for _, member := range members {
		result[member.UserID] = member.Role
	}
	return result
}

func queryMemberRoles(t *testing.T, database *sql.DB, ledgerID string) map[string]string {
	t.Helper()
	rows, err := database.Query(`
		SELECT user_id, role
		FROM ledger_members
		WHERE ledger_id = ?
	`, ledgerID)
	if err != nil {
		t.Fatalf("query member roles: %v", err)
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var userID, role string
		if err := rows.Scan(&userID, &role); err != nil {
			t.Fatalf("scan member role: %v", err)
		}
		result[userID] = role
	}
	return result
}

func insertMemberTestLedger(t *testing.T, database *sql.DB, ledgerID, ownerUserID string, version int64) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := database.Exec(`
		INSERT INTO ledgers (id, name, default_currency, status, version, created_at, updated_at)
		VALUES (?, ?, 'CNY', 'active', ?, ?, ?)
	`, ledgerID, ledgerID, version, now, now); err != nil {
		t.Fatalf("insert member test ledger: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES (?, ?, 'owner', ?, ?)
	`, ledgerID, ownerUserID, now, now); err != nil {
		t.Fatalf("insert member test owner: %v", err)
	}
}
