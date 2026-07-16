package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateLedger 创建新账本并同时将会话用户设为 owner
func (r *Repository) CreateLedger(ctx context.Context, name string, userID string) (*LedgerWithRole, error) {
	ledgerID := uuid.NewString()
	now := time.Now().UTC()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. 创建账本
	_, err = tx.ExecContext(ctx, "INSERT INTO ledgers (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)",
		ledgerID, name, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	// 2. 将创建者加入成员，角色为 owner
	_, err = tx.ExecContext(ctx, "INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		ledgerID, userID, "owner", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	created := &LedgerWithRole{
		Ledger: Ledger{
			ID:          ledgerID,
			Name:        name,
			Status:      LedgerStatusActive,
			Version:     1,
			MemberCount: 1,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Role: string(RoleOwner),
	}
	afterJSON, err := json.Marshal(created)
	if err != nil {
		return nil, err
	}
	if err := r.CreateLedgerAuditWithTx(ctx, tx, ledgerID, userID, RoleOwner, "ledger_create", nil, afterJSON); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return created, nil
}

// ListUserLedgers 获取用户加入的所有账本及对应角色
func (r *Repository) ListUserLedgers(ctx context.Context, userID string) ([]LedgerWithRole, error) {
	return r.ListUserLedgersByStatus(ctx, userID, LedgerListAll)
}

func (r *Repository) ListUserLedgersByStatus(ctx context.Context, userID string, status LedgerListStatus) ([]LedgerWithRole, error) {
	if status != LedgerListActive && status != LedgerListArchived && status != LedgerListAll {
		return nil, fmt.Errorf("invalid ledger list status %q", status)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT l.id,
		       l.name,
		       l.status,
		       l.archived_at,
		       l.archived_by_user_id,
		       l.version,
		       (
		           SELECT COUNT(*)
		           FROM ledger_members member_count
		           WHERE member_count.ledger_id = l.id
		       ),
		       l.created_at,
		       l.updated_at,
		       m.role
		FROM ledgers l
		JOIN ledger_members m ON l.id = m.ledger_id
		WHERE m.user_id = ?
		  AND (? = 'all' OR l.status = ?)
		ORDER BY CASE l.status WHEN 'active' THEN 0 ELSE 1 END,
		         l.created_at ASC,
		         l.id ASC
	`, userID, status, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []LedgerWithRole
	for rows.Next() {
		ledgerWithRole, err := scanLedgerWithRole(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, ledgerWithRole)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) GetLedgerByID(ctx context.Context, ledgerID string) (*Ledger, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT l.id,
		       l.name,
		       l.status,
		       l.archived_at,
		       l.archived_by_user_id,
		       l.version,
		       (
		           SELECT COUNT(*)
		           FROM ledger_members member_count
		           WHERE member_count.ledger_id = l.id
		       ),
		       l.created_at,
		       l.updated_at
		FROM ledgers l
		WHERE l.id = ?
	`, ledgerID)

	ledgerModel, err := scanLedger(row)
	if err != nil {
		return nil, err
	}
	return &ledgerModel, nil
}

func (r *Repository) GetLedgerWithRole(ctx context.Context, tx *sql.Tx, ledgerID, userID string) (*LedgerWithRole, error) {
	var executor ledgerQueryExecutor = r.db
	if tx != nil {
		executor = tx
	}
	row := executor.QueryRowContext(ctx, `
		SELECT l.id,
		       l.name,
		       l.status,
		       l.archived_at,
		       l.archived_by_user_id,
		       l.version,
		       (
		           SELECT COUNT(*)
		           FROM ledger_members member_count
		           WHERE member_count.ledger_id = l.id
		       ),
		       l.created_at,
		       l.updated_at,
		       m.role
		FROM ledgers l
		JOIN ledger_members m ON m.ledger_id = l.id
		WHERE l.id = ? AND m.user_id = ?
	`, ledgerID, userID)

	ledgerWithRole, err := scanLedgerWithRole(row)
	if err != nil {
		return nil, err
	}
	return &ledgerWithRole, nil
}

func (r *Repository) CountMembers(ctx context.Context, ledgerID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM ledger_members WHERE ledger_id = ?",
		ledgerID,
	).Scan(&count)
	return count, err
}

func (r *Repository) CountOwners(ctx context.Context, ledgerID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM ledger_members WHERE ledger_id = ? AND role = 'owner'",
		ledgerID,
	).Scan(&count)
	return count, err
}

func (r *Repository) IsInstanceAdmin(ctx context.Context, userID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM instance_admins WHERE user_id = ?",
		userID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) GetLedgerAccess(ctx context.Context, ledgerID string, userID string) (Role, LedgerStatus, int64, error) {
	return r.GetLedgerAccessWithTx(ctx, nil, ledgerID, userID)
}

func (r *Repository) GetLedgerAccessWithTx(ctx context.Context, tx *sql.Tx, ledgerID string, userID string) (Role, LedgerStatus, int64, error) {
	var executor ledgerQueryExecutor = r.db
	if tx != nil {
		executor = tx
	}
	var role Role
	var status LedgerStatus
	var version int64
	err := executor.QueryRowContext(ctx, `
		SELECT m.role, l.status, l.version
		FROM ledger_members m
		JOIN ledgers l ON l.id = m.ledger_id
		WHERE m.ledger_id = ? AND m.user_id = ?
	`, ledgerID, userID).Scan(&role, &status, &version)
	return role, status, version, err
}

func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *Repository) ClaimLedgerVersion(ctx context.Context, tx *sql.Tx, ledgerID string, expectedVersion int64) (bool, error) {
	var executor ledgerExecutor = r.db
	if tx != nil {
		executor = tx
	}
	result, err := executor.ExecContext(ctx, `
		UPDATE ledgers
		SET version = version + 1,
		    updated_at = ?
		WHERE id = ?
		  AND version = ?
	`, time.Now().Format(time.RFC3339), ledgerID, expectedVersion)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected == 1, nil
}

func (r *Repository) RenameLedgerWithTx(ctx context.Context, tx *sql.Tx, ledgerID, name string) error {
	_, err := tx.ExecContext(ctx, "UPDATE ledgers SET name = ? WHERE id = ?", name, ledgerID)
	return err
}

func (r *Repository) ArchiveLedgerWithTx(ctx context.Context, tx *sql.Tx, ledgerID, userID string, archivedAt time.Time) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE ledgers
		SET status = 'archived', archived_at = ?, archived_by_user_id = ?
		WHERE id = ?
	`, archivedAt.UTC().Format(time.RFC3339Nano), userID, ledgerID)
	return err
}

func (r *Repository) RestoreLedgerWithTx(ctx context.Context, tx *sql.Tx, ledgerID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE ledgers
		SET status = 'active', archived_at = NULL, archived_by_user_id = NULL
		WHERE id = ?
	`, ledgerID)
	return err
}

func (r *Repository) CountBlockingReadyImportBatches(ctx context.Context, tx *sql.Tx, ledgerID string, now time.Time) (int, error) {
	var executor ledgerQueryExecutor = r.db
	if tx != nil {
		executor = tx
	}
	var count int
	err := executor.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM import_batches
		WHERE ledger_id = ?
		  AND status = 'ready'
		  AND (expires_at IS NULL OR julianday(expires_at) > julianday(?))
	`, ledgerID, now.UTC().Format(time.RFC3339Nano)).Scan(&count)
	return count, err
}

func (r *Repository) ExpireReadyImportBatchesWithTx(ctx context.Context, tx *sql.Tx, ledgerID string, now time.Time) error {
	formattedNow := now.UTC().Format(time.RFC3339Nano)
	_, err := tx.ExecContext(ctx, `
		UPDATE import_batches
		SET status = 'expired', updated_at = ?
		WHERE ledger_id = ?
		  AND status = 'ready'
		  AND expires_at IS NOT NULL
		  AND julianday(expires_at) <= julianday(?)
	`, formattedNow, ledgerID, formattedNow)
	return err
}

func (r *Repository) CreateLedgerAuditWithTx(
	ctx context.Context,
	tx *sql.Tx,
	ledgerID, actorUserID string,
	actorRole Role,
	action string,
	beforeJSON, afterJSON []byte,
) error {
	var beforeValue any
	if len(beforeJSON) > 0 {
		beforeValue = string(beforeJSON)
	}
	var afterValue any
	if len(afterJSON) > 0 {
		afterValue = string(afterJSON)
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO audit_logs (
			id, ledger_id, actor_user_id, actor_role, action, entity_type,
			entity_id, before_json, after_json, created_at
		) VALUES (?, ?, ?, ?, ?, 'ledger', ?, ?, ?, ?)
	`, uuid.NewString(), ledgerID, actorUserID, actorRole, action, ledgerID,
		beforeValue, afterValue, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// GetLedgerMembers 获取某账本下的所有成员
func (r *Repository) GetLedgerMembers(ctx context.Context, ledgerID string) ([]MemberDetail, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.username, m.role
		FROM users u
		JOIN ledger_members m ON u.id = m.user_id
		WHERE m.ledger_id = ?
		ORDER BY m.created_at ASC
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MemberDetail
	for rows.Next() {
		var m MemberDetail
		if err := rows.Scan(&m.UserID, &m.Username, &m.Role); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

// FindUserByUsername 按用户名查找用户ID
func (r *Repository) FindUserByUsername(ctx context.Context, username string) (string, error) {
	var userID string
	err := r.db.QueryRowContext(ctx, "SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	return userID, err
}

// AddMember 添加成员
func (r *Repository) AddMember(ctx context.Context, ledgerID, userID, role string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?)
	`, ledgerID, userID, role, now, now)
	return err
}

// UpdateMemberRole 修改成员角色
func (r *Repository) UpdateMemberRole(ctx context.Context, ledgerID, userID, role string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE ledger_members SET role = ?, updated_at = ?
		WHERE ledger_id = ? AND user_id = ?
	`, role, now, ledgerID, userID)
	return err
}

// RemoveMember 移除成员
func (r *Repository) RemoveMember(ctx context.Context, ledgerID, userID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ledger_members WHERE ledger_id = ? AND user_id = ?", ledgerID, userID)
	return err
}

// GetMemberRole 查询用户在指定账本中的角色
func (r *Repository) GetMemberRole(ctx context.Context, ledgerID, userID string) (Role, error) {
	var role string
	err := r.db.QueryRowContext(ctx, "SELECT role FROM ledger_members WHERE ledger_id = ? AND user_id = ?", ledgerID, userID).Scan(&role)
	if err != nil {
		return "", err
	}

	return Role(role), nil
}

// CheckRole 校验角色
func (r *Repository) CheckRole(ctx context.Context, ledgerID, userID string, allowedRoles ...string) error {
	role, err := r.GetMemberRole(ctx, ledgerID, userID)
	if err != nil {
		return sql.ErrNoRows
	}

	for _, allowed := range allowedRoles {
		if role == Role(allowed) {
			return nil
		}
	}
	return sql.ErrNoRows // 表示不符合要求的角色
}

type ledgerScanner interface {
	Scan(dest ...any) error
}

type ledgerExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type ledgerQueryExecutor interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func scanLedger(scanner ledgerScanner) (Ledger, error) {
	var ledgerModel Ledger
	var status string
	var archivedAt sql.NullString
	var archivedByUserID sql.NullString
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(
		&ledgerModel.ID,
		&ledgerModel.Name,
		&status,
		&archivedAt,
		&archivedByUserID,
		&ledgerModel.Version,
		&ledgerModel.MemberCount,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Ledger{}, err
	}

	ledgerModel.Status = LedgerStatus(status)
	if archivedAt.Valid {
		parsed, err := parseLedgerTime(archivedAt.String)
		if err != nil {
			return Ledger{}, fmt.Errorf("parse archived_at for ledger %s: %w", ledgerModel.ID, err)
		}
		ledgerModel.ArchivedAt = &parsed
	}
	if archivedByUserID.Valid {
		value := archivedByUserID.String
		ledgerModel.ArchivedByUserID = &value
	}

	var err error
	ledgerModel.CreatedAt, err = parseLedgerTime(createdAt)
	if err != nil {
		return Ledger{}, fmt.Errorf("parse created_at for ledger %s: %w", ledgerModel.ID, err)
	}
	ledgerModel.UpdatedAt, err = parseLedgerTime(updatedAt)
	if err != nil {
		return Ledger{}, fmt.Errorf("parse updated_at for ledger %s: %w", ledgerModel.ID, err)
	}
	return ledgerModel, nil
}

func scanLedgerWithRole(scanner ledgerScanner) (LedgerWithRole, error) {
	var result LedgerWithRole
	var status string
	var archivedAt sql.NullString
	var archivedByUserID sql.NullString
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(
		&result.ID,
		&result.Name,
		&status,
		&archivedAt,
		&archivedByUserID,
		&result.Version,
		&result.MemberCount,
		&createdAt,
		&updatedAt,
		&result.Role,
	); err != nil {
		return LedgerWithRole{}, err
	}

	result.Status = LedgerStatus(status)
	if archivedAt.Valid {
		parsed, err := parseLedgerTime(archivedAt.String)
		if err != nil {
			return LedgerWithRole{}, fmt.Errorf("parse archived_at for ledger %s: %w", result.ID, err)
		}
		result.ArchivedAt = &parsed
	}
	if archivedByUserID.Valid {
		value := archivedByUserID.String
		result.ArchivedByUserID = &value
	}

	var err error
	result.CreatedAt, err = parseLedgerTime(createdAt)
	if err != nil {
		return LedgerWithRole{}, fmt.Errorf("parse created_at for ledger %s: %w", result.ID, err)
	}
	result.UpdatedAt, err = parseLedgerTime(updatedAt)
	if err != nil {
		return LedgerWithRole{}, fmt.Errorf("parse updated_at for ledger %s: %w", result.ID, err)
	}
	return result, nil
}

func parseLedgerTime(value string) (time.Time, error) {
	formats := []string{time.RFC3339Nano, "2006-01-02 15:04:05"}
	for _, format := range formats {
		parsed, err := time.Parse(format, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp %q", value)
}
