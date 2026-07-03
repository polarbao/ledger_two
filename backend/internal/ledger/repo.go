package ledger

import (
	"context"
	"database/sql"
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
func (r *Repository) CreateLedger(ctx context.Context, name string, userID string) (*Ledger, error) {
	ledgerID := uuid.NewString()
	now := time.Now()

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

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Ledger{
		ID:        ledgerID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ListUserLedgers 获取用户加入的所有账本及对应角色
func (r *Repository) ListUserLedgers(ctx context.Context, userID string) ([]LedgerWithRole, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT l.id, l.name, l.created_at, l.updated_at, m.role
		FROM ledgers l
		JOIN ledger_members m ON l.id = m.ledger_id
		WHERE m.user_id = ?
		ORDER BY l.created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []LedgerWithRole
	for rows.Next() {
		var l LedgerWithRole
		var ca, ua string
		if err := rows.Scan(&l.ID, &l.Name, &ca, &ua, &l.Role); err != nil {
			return nil, err
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		l.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		result = append(result, l)
	}
	return result, nil
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
