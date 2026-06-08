package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"ledger_two/internal/db/repo"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

type AuthService struct {
	repo      *repo.AuthRepo
	jwtSecret string
}

func NewAuthService(r *repo.AuthRepo, secret string) *AuthService {
	return &AuthService{
		repo:      r,
		jwtSecret: secret,
	}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	// 签发 JWT (设置一星期的强有效性)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(24 * 7 * time.Hour).Unix(),
	})

	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) GetMe(ctx context.Context, userID string) (*repo.MeData, error) {
	return s.repo.GetMe(ctx, userID)
}
