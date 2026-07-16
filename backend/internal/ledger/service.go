package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	appErrors "ledger_two/internal/errors"
)

type Service struct {
	repo            *Repository
	balanceProvider UnsettledBalanceProvider
}

type UnsettledBalanceProvider interface {
	GetUnsettledBalance(context.Context, *sql.Tx, LedgerContext) (UnsettledBalanceSnapshot, error)
}

func NewService(repo *Repository, balanceProviders ...UnsettledBalanceProvider) *Service {
	service := &Service{repo: repo}
	if len(balanceProviders) > 0 {
		service.balanceProvider = balanceProviders[0]
	}
	return service
}

type CreateLedgerReq struct {
	Name string `json:"name"`
}

type RenameLedgerReq struct {
	Name string `json:"name"`
}

type ArchiveLedgerReq struct {
	AcknowledgeUnsettledBalance *bool `json:"acknowledge_unsettled_balance"`
}

func (s *Service) CreateLedger(ctx context.Context, userID string, req CreateLedgerReq) (*LedgerWithRole, error) {
	name, err := normalizeLedgerName(req.Name)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateLedger(ctx, name, userID)
}

func (s *Service) ListUserLedgers(ctx context.Context, userID string, status LedgerListStatus) ([]LedgerWithRole, error) {
	if status != LedgerListActive && status != LedgerListArchived && status != LedgerListAll {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "账本状态筛选值无效")
	}
	return s.repo.ListUserLedgersByStatus(ctx, userID, status)
}

func (s *Service) GetLedger(ctx context.Context, lc LedgerContext) (*LedgerWithRole, error) {
	ledgerModel, err := s.repo.GetLedgerWithRole(ctx, nil, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, mapLedgerAccessError(err)
	}
	return ledgerModel, nil
}

func (s *Service) RenameLedger(ctx context.Context, lc LedgerContext, expectedVersion int64, req RenameLedgerReq) (*LedgerWithRole, error) {
	name, err := normalizeLedgerName(req.Name)
	if err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	before, err := s.requireLifecycleOwner(ctx, tx, lc, LedgerStatusActive)
	if err != nil {
		return nil, err
	}
	if err := s.claimLifecycleVersion(ctx, tx, lc.LedgerID, expectedVersion); err != nil {
		return nil, err
	}
	if err := s.repo.RenameLedgerWithTx(ctx, tx, lc.LedgerID, name); err != nil {
		return nil, err
	}
	after, err := s.repo.GetLedgerWithRole(ctx, tx, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, err
	}
	if err := s.writeLifecycleAudit(ctx, tx, lc.UserID, RoleOwner, "ledger_rename", before, after); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func (s *Service) GetArchivePreflight(ctx context.Context, lc LedgerContext) (*ArchivePreflight, error) {
	ledgerModel, err := s.requireLifecycleOwner(ctx, nil, lc, LedgerStatusActive)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	readyCount, err := s.repo.CountBlockingReadyImportBatches(ctx, nil, lc.LedgerID, now)
	if err != nil {
		return nil, err
	}
	balance, err := s.getUnsettledBalance(ctx, nil, lifecycleContextFromLedger(lc.UserID, *ledgerModel))
	if err != nil {
		return nil, err
	}
	return &ArchivePreflight{
		Ledger:                           *ledgerModel,
		UnsettledBalance:                 balance,
		ReadyImportBatchCount:            readyCount,
		CanArchive:                       readyCount == 0,
		RequiresUnsettledAcknowledgement: balance.AmountCents > 0,
	}, nil
}

func (s *Service) ArchiveLedger(ctx context.Context, lc LedgerContext, expectedVersion int64, req ArchiveLedgerReq) (*LedgerWithRole, error) {
	if req.AcknowledgeUnsettledBalance == nil {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "必须明确确认是否接受未结清净额")
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	before, err := s.requireLifecycleOwner(ctx, tx, lc, LedgerStatusActive)
	if err != nil {
		return nil, err
	}
	if err := s.claimLifecycleVersion(ctx, tx, lc.LedgerID, expectedVersion); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	readyCount, err := s.repo.CountBlockingReadyImportBatches(ctx, tx, lc.LedgerID, now)
	if err != nil {
		return nil, err
	}
	if readyCount > 0 {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerReadyImportExists, "存在待确认的导入批次，请先完成或放弃导入")
	}
	if err := s.repo.ExpireReadyImportBatchesWithTx(ctx, tx, lc.LedgerID, now); err != nil {
		return nil, err
	}
	balanceContext := lifecycleContextFromLedger(lc.UserID, *before)
	balanceContext.Version = expectedVersion + 1
	balance, err := s.getUnsettledBalance(ctx, tx, balanceContext)
	if err != nil {
		return nil, err
	}
	if balance.AmountCents > 0 && !*req.AcknowledgeUnsettledBalance {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "账本存在未结清净额，请确认后再归档")
	}
	if err := s.repo.ArchiveLedgerWithTx(ctx, tx, lc.LedgerID, lc.UserID, now); err != nil {
		return nil, err
	}
	after, err := s.repo.GetLedgerWithRole(ctx, tx, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, err
	}
	auditAfter := struct {
		LedgerWithRole
		UnsettledBalance      UnsettledBalanceSnapshot `json:"unsettled_balance"`
		ReadyImportBatchCount int                      `json:"ready_import_batch_count"`
	}{
		LedgerWithRole:        *after,
		UnsettledBalance:      balance,
		ReadyImportBatchCount: readyCount,
	}
	if err := s.writeLifecycleAudit(ctx, tx, lc.UserID, RoleOwner, "ledger_archive", before, auditAfter); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func (s *Service) RestoreLedger(ctx context.Context, lc LedgerContext, expectedVersion int64) (*LedgerWithRole, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	before, err := s.requireLifecycleOwner(ctx, tx, lc, LedgerStatusArchived)
	if err != nil {
		return nil, err
	}
	if err := s.claimLifecycleVersion(ctx, tx, lc.LedgerID, expectedVersion); err != nil {
		return nil, err
	}
	if err := s.repo.RestoreLedgerWithTx(ctx, tx, lc.LedgerID); err != nil {
		return nil, err
	}
	after, err := s.repo.GetLedgerWithRole(ctx, tx, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, err
	}
	if err := s.writeLifecycleAudit(ctx, tx, lc.UserID, RoleOwner, "ledger_restore", before, after); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func normalizeLedgerName(value string) (string, error) {
	name := strings.TrimSpace(value)
	length := utf8.RuneCountInString(name)
	if length < 1 || length > 60 {
		return "", appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "账本名称长度必须为 1 到 60 个字符")
	}
	return name, nil
}

func (s *Service) requireLifecycleOwner(ctx context.Context, tx *sql.Tx, lc LedgerContext, expectedStatus LedgerStatus) (*LedgerWithRole, error) {
	ledgerModel, err := s.repo.GetLedgerWithRole(ctx, tx, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, mapLedgerAccessError(err)
	}
	if Role(ledgerModel.Role) != RoleOwner {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeLedgerAccessDenied, "仅 Owner 可管理账本生命周期")
	}
	if ledgerModel.Status != expectedStatus {
		if ledgerModel.Status == LedgerStatusArchived && expectedStatus == LedgerStatusActive {
			return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerArchived, "归档账本为只读状态")
		}
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerInvalidState, "账本状态不允许此操作")
	}
	return ledgerModel, nil
}

func (s *Service) claimLifecycleVersion(ctx context.Context, tx *sql.Tx, ledgerID string, expectedVersion int64) error {
	if expectedVersion < 1 {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "If-Match 版本无效")
	}
	claimed, err := s.repo.ClaimLedgerVersion(ctx, tx, ledgerID, expectedVersion)
	if err != nil {
		return err
	}
	if !claimed {
		return appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerVersionConflict, "账本已被更新，请刷新后重试")
	}
	return nil
}

func (s *Service) writeLifecycleAudit(ctx context.Context, tx *sql.Tx, actorUserID string, actorRole Role, action string, before *LedgerWithRole, after any) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	return s.repo.CreateLedgerAuditWithTx(ctx, tx, before.ID, actorUserID, actorRole, action, beforeJSON, afterJSON)
}

func (s *Service) getUnsettledBalance(ctx context.Context, tx *sql.Tx, lc LedgerContext) (UnsettledBalanceSnapshot, error) {
	if s.balanceProvider == nil {
		return UnsettledBalanceSnapshot{}, nil
	}
	return s.balanceProvider.GetUnsettledBalance(ctx, tx, lc)
}

func lifecycleContextFromLedger(userID string, ledgerModel LedgerWithRole) LedgerContext {
	return LedgerContext{
		UserID:     userID,
		LedgerID:   ledgerModel.ID,
		Role:       Role(ledgerModel.Role),
		Status:     ledgerModel.Status,
		Version:    ledgerModel.Version,
		IsExplicit: true,
	}
}

func mapLedgerAccessError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeLedgerAccessDenied, "您无权访问该账本")
	}
	return err
}

func (s *Service) ResolveLedgerContext(ctx context.Context, currentUserID string, ledgerID string, isExplicit bool) (LedgerContext, error) {
	lc, err := ResolveLedgerContext(ctx, currentUserID, ledgerID, isExplicit, s.repo.GetLedgerAccess)
	if err == nil {
		return lc, nil
	}
	if errors.Is(err, ErrLedgerUserRequired) {
		return LedgerContext{}, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统")
	}
	if errors.Is(err, ErrLedgerIDRequired) {
		return LedgerContext{}, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerRequired, "请选择账本后再执行此操作")
	}
	if errors.Is(err, ErrLedgerRoleInvalid) {
		return LedgerContext{}, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeLedgerAccessDenied, "当前账本成员身份无效")
	}
	if errors.Is(err, ErrLedgerStateInvalid) {
		return LedgerContext{}, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "账本状态数据无效")
	}
	if errors.Is(err, sql.ErrNoRows) {
		return LedgerContext{}, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeLedgerAccessDenied, "您无权访问该账本")
	}
	return LedgerContext{}, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "解析账本成员身份失败")
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
