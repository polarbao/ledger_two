package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"ledger_two/internal/db/repo"
)

var ErrAlreadyInitialized = errors.New("system is already initialized")

type InitService struct {
	repo *repo.InitRepo
}

func NewInitService(r *repo.InitRepo) *InitService {
	return &InitService{repo: r}
}

func (s *InitService) CheckStatus(ctx context.Context) (bool, error) {
	return s.repo.IsInitialized(ctx)
}

// SetupRequest 包含在初始化阶段所需的两个成员完整信息以及账本信息
type SetupRequest struct {
	LedgerName       string `json:"ledger_name"`
	DefaultCurrency  string `json:"default_currency"`
	UserAUsername    string `json:"user_a_username"`
	UserADisplayName string `json:"user_a_display_name"`
	UserAPassword    string `json:"user_a_password"`
	UserBUsername    string `json:"user_b_username"`
	UserBDisplayName string `json:"user_b_display_name"`
	UserBPassword    string `json:"user_b_password"`
}

func (s *InitService) RunSetup(ctx context.Context, req SetupRequest) error {
	// 校验防并发重入攻击
	isInit, err := s.repo.IsInitialized(ctx)
	if err != nil {
		return err
	}
	if isInit {
		return ErrAlreadyInitialized
	}

	if req.DefaultCurrency == "" {
		req.DefaultCurrency = "CNY"
	}

	// 必须加盐强哈希，绝不明文留存密码
	hashA, err := bcrypt.GenerateFromPassword([]byte(req.UserAPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashB, err := bcrypt.GenerateFromPassword([]byte(req.UserBPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	users := []repo.UserPayload{
		{
			Username:     req.UserAUsername,
			DisplayName:  req.UserADisplayName,
			PasswordHash: string(hashA),
		},
		{
			Username:     req.UserBUsername,
			DisplayName:  req.UserBDisplayName,
			PasswordHash: string(hashB),
		},
	}

	return s.repo.ExecuteSetupTx(ctx, req.LedgerName, req.DefaultCurrency, users)
}
