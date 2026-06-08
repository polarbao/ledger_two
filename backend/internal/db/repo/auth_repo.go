package repo

import (
	"context"
	"database/sql"
	"errors"
)

type AuthRepo struct {
	db *sql.DB
}

func NewAuthRepo(db *sql.DB) *AuthRepo {
	return &AuthRepo{db: db}
}

type AuthUser struct {
	ID           string
	PasswordHash string
}

func (r *AuthRepo) GetUserByUsername(ctx context.Context, username string) (*AuthUser, error) {
	var user AuthUser
	err := r.db.QueryRowContext(ctx, "SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&user.ID, &user.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

type MeData struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	LedgerID    string `json:"ledger_id"`
}

func (r *AuthRepo) GetMe(ctx context.Context, userID string) (*MeData, error) {
	var me MeData
	var avatar sql.NullString
	err := r.db.QueryRowContext(ctx, "SELECT id, username, display_name, avatar_url FROM users WHERE id = ?", userID).Scan(&me.ID, &me.Username, &me.DisplayName, &avatar)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	if avatar.Valid {
		me.AvatarURL = avatar.String
	}

	// 注入系统的唯一 Ledger 以方便后续鉴权
	err = r.db.QueryRowContext(ctx, "SELECT id FROM ledgers LIMIT 1").Scan(&me.LedgerID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return &me, nil
}
