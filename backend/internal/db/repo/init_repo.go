package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
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

	now := time.Now().Format(time.RFC3339)

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
	for _, u := range users {
		userID := uuid.NewString()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'user', ?, ?)
		`, userID, u.Username, u.DisplayName, u.PasswordHash, now, now)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
			VALUES (?, ?, 'editor', ?, ?)
		`, ledgerID, userID, now, now)
		if err != nil {
			return err
		}

		accountID := uuid.NewString()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO accounts (id, ledger_id, owner_user_id, name, type, currency, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'cash', ?, ?, ?)
		`, accountID, ledgerID, userID, "日常账户", currency, now, now)
		if err != nil {
			return err
		}
	}

	// 3. 构建默认系统分类
	defaultCategories := []string{"餐饮美食", "交通出行", "购物娱乐", "生活缴费", "共同居住"}
	for i, cname := range defaultCategories {
		catID := uuid.NewString()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO categories (id, ledger_id, name, type, sort_order, is_system, created_at, updated_at)
			VALUES (?, ?, ?, 'expense', ?, 1, ?, ?)
		`, catID, ledgerID, cname, i, now, now)
		if err != nil {
			return err
		}
	}

	// 4. 生成默认的默认通用标签（由于文档提及）
	tagID := uuid.NewString()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tags (id, ledger_id, name, created_at, updated_at)
		VALUES (?, ?, '系统默认', ?, ?)
	`, tagID, ledgerID, now, now)
	if err != nil {
		return err
	}

	// 5. 最终锁定系统门阀
	_, err = tx.ExecContext(ctx, `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ('initialized', 'true', ?)
	`, now)
	if err != nil {
		return err
	}

	return tx.Commit()
}
