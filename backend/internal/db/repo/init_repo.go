package repo

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"

	"ledger_two/internal/metadata/defaults"
)

type InitRepo struct {
	db *sql.DB
}

func NewInitRepo(db *sql.DB) *InitRepo {
	return &InitRepo{db: db}
}

// IsInitialized 直接利用 sql 查询探测设置表中的保护锁
func (r *InitRepo) IsInitialized(ctx context.Context) (bool, error) {
	var val string
	err := r.db.QueryRowContext(ctx, "SELECT value FROM app_settings WHERE key = 'initialized'").Scan(&val)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return val == "true", nil
}

type UserPayload struct {
	Username     string
	DisplayName  string
	PasswordHash string
}

// ExecuteSetupTx 强制事务执行所有初始化写入
func (r *InitRepo) ExecuteSetupTx(ctx context.Context, ledgerName, currency string, users []UserPayload) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// 利用 defer 兜底，只有 Commit 后 err 为 nil 才会跳过
	defer tx.Rollback()

	nowTime := time.Now().UTC()
	now := nowTime.Format(time.RFC3339)

	// 1. 创建全局唯一固定账本
	ledgerID := uuid.NewString()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO ledgers (id, name, default_currency, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, ledgerID, ledgerName, currency, now, now)
	if err != nil {
		return err
	}

	// 2. 依次生成两个受控用户，以及各自对应的默认账户
	var ownerUserID string
	for i, u := range users {
		userID := uuid.NewString()
		if i == 0 {
			ownerUserID = userID
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'user', ?, ?)
		`, userID, u.Username, u.DisplayName, u.PasswordHash, now, now)
		if err != nil {
			return err
		}
		if i == 0 {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO instance_admins (user_id, granted_at, granted_by_user_id)
				VALUES (?, ?, NULL)
			`, userID, now)
			if err != nil {
				return err
			}
		}

		memberRole := "editor"
		if i == 0 {
			memberRole = "owner"
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, ledgerID, userID, memberRole, now, now)
		if err != nil {
			return err
		}

		accountID := uuid.NewString()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO accounts (id, ledger_id, owner_user_id, name, type, currency, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'cash', ?, ?, ?)
		`, accountID, ledgerID, userID, defaultAccountName(u, len(users)), currency, now, now)
		if err != nil {
			return err
		}
	}

	// 3. 在同一事务内创建版本化默认分类和标签。
	if _, err = defaults.ApplyFresh(ctx, tx, ledgerID, ownerUserID, defaults.ProfileBasicCNV1, nowTime); err != nil {
		return err
	}

	// 4. 最终锁定系统门阀
	_, err = tx.ExecContext(ctx, `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ('initialized', 'true', ?)
	`, now)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func defaultAccountName(user UserPayload, userCount int) string {
	const baseName = "日常账户"
	if userCount <= 1 {
		return baseName
	}

	label := strings.TrimSpace(user.DisplayName)
	if label == "" {
		label = strings.TrimSpace(user.Username)
	}
	if label == "" {
		return baseName
	}
	return label + baseName
}
