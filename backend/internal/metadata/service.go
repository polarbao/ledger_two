package metadata

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	ledgerctx "ledger_two/internal/ledger"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func ParseKind(value string) (Kind, bool) {
	switch Kind(value) {
	case KindCategory, KindTag, KindAccount:
		return Kind(value), true
	default:
		return "", false
	}
}

func CanManage(role string) bool {
	return role == string(ledgerctx.RoleOwner)
}

func (s *Service) List(ctx context.Context, currentUserID string, kind Kind, includeArchived bool) ([]Item, error) {
	ledgerID, _, err := s.resolveLedger(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, kind, ledgerID, includeArchived)
}

func (s *Service) Create(ctx context.Context, currentUserID string, kind Kind, req UpsertRequest) (*Item, error) {
	req = normalize(req)
	ledgerID, role, err := s.resolveLedger(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	if !CanManage(role) {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅 Owner 可管理元数据")
	}
	if err := validate(kind, req); err != nil {
		return nil, err
	}
	exists, err := s.repo.NameExists(ctx, kind, ledgerID, req.Type, req.Name, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeDuplicateName, "同账本内名称已存在")
	}
	return s.repo.Create(ctx, kind, ledgerID, currentUserID, req)
}

func (s *Service) Update(ctx context.Context, currentUserID string, kind Kind, id string, req UpsertRequest) error {
	req = normalize(req)
	ledgerID, role, err := s.resolveLedger(ctx, currentUserID)
	if err != nil {
		return err
	}
	if !CanManage(role) {
		return appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅 Owner 可管理元数据")
	}
	if id == "" {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "元数据 ID 不能为空")
	}
	if err := validate(kind, req); err != nil {
		return err
	}
	exists, err := s.repo.NameExists(ctx, kind, ledgerID, req.Type, req.Name, id)
	if err != nil {
		return err
	}
	if exists {
		return appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeDuplicateName, "同账本内名称已存在")
	}
	if err := s.repo.Update(ctx, kind, ledgerID, id, req); err != nil {
		return mapNotFound(err, kind)
	}
	return nil
}

func (s *Service) SetArchived(ctx context.Context, currentUserID string, kind Kind, id string, archived bool) error {
	ledgerID, role, err := s.resolveLedger(ctx, currentUserID)
	if err != nil {
		return err
	}
	if !CanManage(role) {
		return appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅 Owner 可管理元数据")
	}
	if id == "" {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "元数据 ID 不能为空")
	}
	if err := s.repo.SetArchived(ctx, kind, ledgerID, id, archived); err != nil {
		return mapNotFound(err, kind)
	}
	return nil
}

func (s *Service) Reorder(ctx context.Context, currentUserID string, kind Kind, req ReorderRequest) error {
	ledgerID, role, err := s.resolveLedger(ctx, currentUserID)
	if err != nil {
		return err
	}
	if !CanManage(role) {
		return appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅 Owner 可管理元数据")
	}
	if len(req.OrderedIDs) == 0 {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "排序列表不能为空")
	}
	seen := make(map[string]bool, len(req.OrderedIDs))
	for index, id := range req.OrderedIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "排序 ID 不能为空")
		}
		if seen[id] {
			return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "排序 ID 不能重复")
		}
		seen[id] = true
		req.OrderedIDs[index] = id
	}
	if err := s.repo.Reorder(ctx, kind, ledgerID, req.OrderedIDs); err != nil {
		return mapNotFound(err, kind)
	}
	return nil
}

func (s *Service) resolveLedger(ctx context.Context, userID string) (string, string, error) {
	if userID == "" {
		return "", "", appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统")
	}
	if lc, ok := ledgerctx.LedgerContextFromContext(ctx); ok && lc.UserID == userID {
		return lc.LedgerID, string(lc.Role), nil
	}
	headerLedgerID := middleware.GetHeaderLedgerIDFromContext(ctx)
	if headerLedgerID != "" {
		role, err := s.repo.GetMemberRole(ctx, headerLedgerID, userID)
		if err != nil {
			return "", "", appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "您不是该账本的成员")
		}
		return headerLedgerID, role, nil
	}
	ledgerID, role, err := s.repo.GetFirstLedgerRole(ctx, userID)
	if err != nil {
		return "", "", appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "获取系统账本失败")
	}
	return ledgerID, role, nil
}

func validate(kind Kind, req UpsertRequest) error {
	if req.Name == "" {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "名称不能为空")
	}
	switch kind {
	case KindCategory:
		if req.Type != "expense" && req.Type != "income" {
			return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "分类类型必须为 expense 或 income")
		}
	case KindAccount:
		if req.Type == "" {
			return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "账户类型不能为空")
		}
	case KindTag:
		return nil
	default:
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "不支持的元数据类型")
	}
	return nil
}

func normalize(req UpsertRequest) UpsertRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.TrimSpace(req.Type)
	req.Icon = strings.TrimSpace(req.Icon)
	req.Color = strings.TrimSpace(req.Color)
	return req
}

func mapNotFound(err error, kind Kind) error {
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	switch kind {
	case KindCategory:
		return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeCategoryNotFound, "分类不存在")
	case KindTag:
		return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeTagNotFound, "标签不存在")
	case KindAccount:
		return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeAccountNotFound, "账户不存在")
	default:
		return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeNotFound, "元数据不存在")
	}
}
