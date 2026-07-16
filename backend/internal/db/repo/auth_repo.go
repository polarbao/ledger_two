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
	ID            string `json:"id"`
	Username      string `json:"username"`
	DisplayName   string `json:"display_name"`
	AvatarURL     string `json:"avatar_url"`
	InstanceAdmin bool   `json:"instance_admin"`
}

func (r *AuthRepo) GetMe(ctx context.Context, userID string) (*MeData, error) {
	var me MeData
	var avatar sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id,
		       u.username,
		       u.display_name,
		       u.avatar_url,
		       EXISTS(SELECT 1 FROM instance_admins ia WHERE ia.user_id = u.id)
		FROM users u
		WHERE u.id = ?
	`, userID).Scan(&me.ID, &me.Username, &me.DisplayName, &avatar, &me.InstanceAdmin)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	if avatar.Valid {
		me.AvatarURL = avatar.String
	}

	return &me, nil
}
