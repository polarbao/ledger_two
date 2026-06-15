package ledger

import (
	"context"
	"database/sql"
	"errors"

	appErrors "ledger_two/internal/errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type CreateLedgerReq struct {
	Name string `json:"name"`
}

func (s *Service) CreateLedger(ctx context.Context, userID string, req CreateLedgerReq) (*Ledger, error) {
	if req.Name == "" {
		return nil, appErrors.NewAppError(400, "VALIDATION_ERROR", "账本名称不能为空")
	}
	// Create Ledger
	return s.repo.CreateLedger(ctx, req.Name, userID)
}

func (s *Service) ListUserLedgers(ctx context.Context, userID string) ([]LedgerWithRole, error) {
	return s.repo.ListUserLedgers(ctx, userID)
}

func (s *Service) GetLedgerMembers(ctx context.Context, currentUserID, ledgerID string) ([]MemberDetail, error) {
	// Require membership
	if err := s.repo.CheckRole(ctx, ledgerID, currentUserID, "owner", "editor", "viewer"); err != nil {
		return nil, appErrors.NewAppError(403, "FORBIDDEN", "您无权查看该账本成员")
	}
	return s.repo.GetLedgerMembers(ctx, ledgerID)
}

type AddMemberReq struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (s *Service) AddMember(ctx context.Context, currentUserID, ledgerID string, req AddMemberReq) error {
	if req.Role != "editor" && req.Role != "viewer" {
		return appErrors.NewAppError(400, "VALIDATION_ERROR", "无效的角色")
	}
	// Only owner can add
	if err := s.repo.CheckRole(ctx, ledgerID, currentUserID, "owner"); err != nil {
		return appErrors.NewAppError(403, "FORBIDDEN", "仅 Owner 可管理成员")
	}

	targetUserID, err := s.repo.FindUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return appErrors.NewAppError(404, "NOT_FOUND", "邀请的用户不存在")
		}
		return err
	}

	// check if already a member
	members, err := s.repo.GetLedgerMembers(ctx, ledgerID)
	if err == nil {
		for _, m := range members {
			if m.UserID == targetUserID {
				return appErrors.NewAppError(400, "VALIDATION_ERROR", "该用户已是账本成员")
			}
		}
	}

	return s.repo.AddMember(ctx, ledgerID, targetUserID, req.Role)
}

type UpdateMemberReq struct {
	Role string `json:"role"`
}

func (s *Service) UpdateMemberRole(ctx context.Context, currentUserID, ledgerID, targetUserID string, req UpdateMemberReq) error {
	if req.Role != "editor" && req.Role != "viewer" {
		return appErrors.NewAppError(400, "VALIDATION_ERROR", "无效的角色")
	}
	if currentUserID == targetUserID {
		return appErrors.NewAppError(400, "VALIDATION_ERROR", "不能修改自己的角色")
	}
	// Only owner can update
	if err := s.repo.CheckRole(ctx, ledgerID, currentUserID, "owner"); err != nil {
		return appErrors.NewAppError(403, "FORBIDDEN", "仅 Owner 可管理成员")
	}
	return s.repo.UpdateMemberRole(ctx, ledgerID, targetUserID, req.Role)
}

func (s *Service) RemoveMember(ctx context.Context, currentUserID, ledgerID, targetUserID string) error {
	if currentUserID == targetUserID {
		return appErrors.NewAppError(400, "VALIDATION_ERROR", "不能移除自己")
	}
	// Only owner can remove
	if err := s.repo.CheckRole(ctx, ledgerID, currentUserID, "owner"); err != nil {
		return appErrors.NewAppError(403, "FORBIDDEN", "仅 Owner 可管理成员")
	}
	return s.repo.RemoveMember(ctx, ledgerID, targetUserID)
}
