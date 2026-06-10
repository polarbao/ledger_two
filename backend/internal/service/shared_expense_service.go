package service

import (
	"context"
	"errors"

	"ledger_two/internal/db/repo"
)

type SharedExpenseService struct {
	repo *repo.SharedExpenseRepo
}

func NewSharedExpenseService(r *repo.SharedExpenseRepo) *SharedExpenseService {
	return &SharedExpenseService{repo: r}
}

func (s *SharedExpenseService) GetDefaultLedgerID(ctx context.Context) (string, error) {
	return s.repo.GetDefaultLedgerID(ctx)
}

type CreateSharedExpenseReq struct {
	Title       string `json:"title"`
	Amount      int64  `json:"amount"`
	PayerUserID string `json:"payer_user_id"`
	SplitMethod string `json:"split_method"`
	AccountID   string `json:"account_id"`
	CategoryID  string `json:"category_id"`
	OccurredAt  string `json:"occurred_at"`
}

func (s *SharedExpenseService) Create(ctx context.Context, ledgerID, currentUserID string, req CreateSharedExpenseReq) (string, error) {
	if req.Amount <= 0 {
		return "", errors.New("amount must be greater than 0")
	}
	if req.SplitMethod != "equal" && req.SplitMethod != "payer_only" {
		return "", errors.New("unsupported split method")
	}

	users, err := s.repo.GetLedgerUsers(ctx, ledgerID)
	if err != nil {
		return "", err
	}
	if len(users) != 2 {
		return "", errors.New("shared expense requires exactly 2 users in the system")
	}

	var splits []repo.SplitPayload

	if req.SplitMethod == "payer_only" {
		for _, u := range users {
			share := int64(0)
			if u.ID == req.PayerUserID {
				share = req.Amount
			}
			splits = append(splits, repo.SplitPayload{
				UserID:      u.ID,
				ShareAmount: share,
			})
		}
	} else if req.SplitMethod == "equal" {
		baseShare := req.Amount / 2
		remainder := req.Amount % 2

		for _, u := range users {
			share := baseShare
			if u.ID == req.PayerUserID {
				share += remainder
			}
			splits = append(splits, repo.SplitPayload{
				UserID:      u.ID,
				ShareAmount: share,
			})
		}
	}

	payload := repo.SharedExpensePayload{
		LedgerID:        ledgerID,
		Title:           req.Title,
		Amount:          req.Amount,
		OccurredAt:      req.OccurredAt,
		OwnerUserID:     currentUserID, // 创建者为 Owner
		CreatedByUserID: currentUserID,
		PayerUserID:     req.PayerUserID,
		AccountID:       req.AccountID,
		CategoryID:      req.CategoryID,
		SplitMethod:     req.SplitMethod,
		Splits:          splits,
	}

	return s.repo.CreateTx(ctx, payload)
}
