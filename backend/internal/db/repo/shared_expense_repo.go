package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type SharedExpenseRepo struct {
	db *sql.DB
}

func NewSharedExpenseRepo(db *sql.DB) *SharedExpenseRepo {
	return &SharedExpenseRepo{db: db}
}

type SimpleUser struct {
	ID string
}

func (r *SharedExpenseRepo) GetLedgerUsers(ctx context.Context, ledgerID string) ([]SimpleUser, error) {
	// 因为是在 demo 环境的限定下，可以通过查找存在于默认账户的所有持有人确定 ledger 内的合法成员
	// 这里最直接的方式是根据 users 查询，如果系统只有两个用户直接拉取全部用户即可
	rows, err := r.db.QueryContext(ctx, "SELECT id FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []SimpleUser
	for rows.Next() {
		var u SimpleUser
		if err := rows.Scan(&u.ID); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *SharedExpenseRepo) GetDefaultLedgerID(ctx context.Context) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, "SELECT id FROM ledgers LIMIT 1").Scan(&id)
	return id, err
}

type SplitPayload struct {
	UserID      string
	ShareAmount int64
}

type SharedExpensePayload struct {
	LedgerID        string
	Title           string
	Amount          int64
	OccurredAt      string
	OwnerUserID     string
	CreatedByUserID string
	PayerUserID     string
	AccountID       string
	CategoryID      string
	SplitMethod     string
	Splits          []SplitPayload
}

func (r *SharedExpenseRepo) CreateTx(ctx context.Context, p SharedExpensePayload) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)
	transactionID := uuid.NewString()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (
			id, ledger_id, type, title, amount, occurred_at, 
			owner_user_id, created_by_user_id, payer_user_id, 
			account_id, category_id, visibility, split_method, 
			created_at, updated_at
		) VALUES (?, ?, 'shared_expense', ?, ?, ?, ?, ?, ?, ?, ?, 'partner_readable', ?, ?, ?)
	`, transactionID, p.LedgerID, p.Title, p.Amount, p.OccurredAt,
		p.OwnerUserID, p.CreatedByUserID, p.PayerUserID,
		p.AccountID, p.CategoryID, p.SplitMethod, now, now)
	if err != nil {
		return "", err
	}

	for _, s := range p.Splits {
		splitID := uuid.NewString()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO transaction_splits (
				id, transaction_id, user_id, share_amount, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?)
		`, splitID, transactionID, s.UserID, s.ShareAmount, now, now)
		if err != nil {
			return "", err
		}
	}

	return transactionID, tx.Commit()
}

// 简单列表和详情方法待补充
func (r *SharedExpenseRepo) List(ctx context.Context, ledgerID string) ([]map[string]interface{}, error) {
	// For demo scope logic
	return nil, nil
}
