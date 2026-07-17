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
	"ledger_two/internal/metadata/defaults"
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
	Name            string `json:"name"`
	MetadataProfile string `json:"metadata_profile,omitempty"`
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
	profileKey := strings.TrimSpace(req.MetadataProfile)
	if profileKey == "" {
		profileKey = defaults.ProfileBasicCNV1
	}
	if _, ok := defaults.Get(profileKey); !ok {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "默认元数据模板无效")
	}
	return s.repo.CreateLedger(ctx, name, userID, profileKey)
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

func (s *Service) GetMemberList(ctx context.Context, lc LedgerContext) (*MemberListData, error) {
	result, err := s.getMemberListWithTx(ctx, nil, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, mapLedgerAccessError(err)
	}
	return result, nil
}

type AddMemberReq struct {
	Username                     string `json:"username"`
	Role                         string `json:"role"`
	AcknowledgeHistoryVisibility bool   `json:"acknowledge_history_visibility"`
}

func (s *Service) AddMemberVersioned(
	ctx context.Context,
	lc LedgerContext,
	expectedVersion int64,
	req AddMemberReq,
) (*MemberListData, error) {
	if req.Username == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "用户名不能为空")
	}
	role := Role(req.Role)
	if role != RoleEditor && role != RoleViewer {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "成员角色必须是 editor 或 viewer")
	}
	if !req.AcknowledgeHistoryVisibility {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "添加成员前必须确认历史可见性")
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	beforeLedger, err := s.requireLifecycleOwner(ctx, tx, lc, LedgerStatusActive)
	if err != nil {
		return nil, err
	}
	beforeMembers, err := s.repo.GetLedgerMembersWithTx(ctx, tx, lc.LedgerID)
	if err != nil {
		return nil, err
	}
	if err := s.claimLifecycleVersion(ctx, tx, lc.LedgerID, expectedVersion); err != nil {
		return nil, err
	}

	memberCount, err := s.repo.CountMembersWithTx(ctx, tx, lc.LedgerID)
	if err != nil {
		return nil, err
	}
	if memberCount >= 2 {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerMemberLimitReached, "账本最多只能有两名成员")
	}

	targetUserID, err := s.repo.FindActiveUserByUsernameWithTx(ctx, tx, req.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeNotFound, "用户不存在或不可用")
		}
		return nil, err
	}
	for _, member := range beforeMembers {
		if member.UserID == targetUserID {
			return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeConflict, "该用户已是账本成员")
		}
	}
	if err := s.repo.AddMemberWithTx(ctx, tx, lc.LedgerID, targetUserID, role); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "ledger member limit reached") {
			return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerMemberLimitReached, "账本最多只能有两名成员")
		}
		return nil, err
	}

	after, err := s.getMemberListWithTx(ctx, tx, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, err
	}
	before := MemberListData{Ledger: *beforeLedger, Members: beforeMembers}
	if err := s.writeLedgerAudit(ctx, tx, lc.LedgerID, lc.UserID, RoleOwner, "ledger_member_add", before, after); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

type UpdateMemberReq struct {
	Role string `json:"role"`
}

type TransferOwnerReq struct {
	AcknowledgePermissionChange bool `json:"acknowledge_permission_change"`
}

func (s *Service) UpdateMemberRoleVersioned(
	ctx context.Context,
	lc LedgerContext,
	expectedVersion int64,
	targetUserID string,
	req UpdateMemberReq,
) (*MemberListData, error) {
	role := Role(req.Role)
	if role != RoleEditor && role != RoleViewer {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "成员角色必须是 editor 或 viewer")
	}
	if targetUserID == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "成员 ID 不能为空")
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	beforeLedger, err := s.requireLifecycleOwner(ctx, tx, lc, LedgerStatusActive)
	if err != nil {
		return nil, err
	}
	beforeMembers, err := s.repo.GetLedgerMembersWithTx(ctx, tx, lc.LedgerID)
	if err != nil {
		return nil, err
	}
	if err := s.claimLifecycleVersion(ctx, tx, lc.LedgerID, expectedVersion); err != nil {
		return nil, err
	}
	currentRole, err := s.repo.GetMemberRoleWithTx(ctx, tx, lc.LedgerID, targetUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "账本成员不存在")
		}
		return nil, err
	}
	if currentRole == RoleOwner {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerOwnerTransferRequired, "Owner 角色只能通过所有权移交变更")
	}
	if err := s.repo.UpdateMemberRoleWithTx(ctx, tx, lc.LedgerID, targetUserID, role); err != nil {
		return nil, err
	}

	after, err := s.getMemberListWithTx(ctx, tx, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, err
	}
	before := MemberListData{Ledger: *beforeLedger, Members: beforeMembers}
	if err := s.writeLedgerAudit(ctx, tx, lc.LedgerID, lc.UserID, RoleOwner, "ledger_member_role_update", before, after); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func (s *Service) TransferOwnerVersioned(
	ctx context.Context,
	lc LedgerContext,
	expectedVersion int64,
	targetUserID string,
	req TransferOwnerReq,
) (*MemberListData, error) {
	if !req.AcknowledgePermissionChange {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "必须确认移交后的权限变化")
	}
	if targetUserID == "" || targetUserID == lc.UserID {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "请选择另一名账本成员作为新 Owner")
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	beforeLedger, err := s.requireLifecycleOwner(ctx, tx, lc, LedgerStatusActive)
	if err != nil {
		return nil, err
	}
	beforeMembers, err := s.repo.GetLedgerMembersWithTx(ctx, tx, lc.LedgerID)
	if err != nil {
		return nil, err
	}
	if err := s.claimLifecycleVersion(ctx, tx, lc.LedgerID, expectedVersion); err != nil {
		return nil, err
	}
	targetRole, err := s.repo.GetMemberRoleWithTx(ctx, tx, lc.LedgerID, targetUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "目标成员不存在")
		}
		return nil, err
	}
	if targetRole != RoleEditor && targetRole != RoleViewer {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "目标成员角色不允许接收所有权")
	}

	if err := s.repo.UpdateMemberRoleWithTx(ctx, tx, lc.LedgerID, lc.UserID, RoleEditor); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateMemberRoleWithTx(ctx, tx, lc.LedgerID, targetUserID, RoleOwner); err != nil {
		return nil, err
	}
	ownerCount, err := s.repo.CountOwnersWithTx(ctx, tx, lc.LedgerID)
	if err != nil {
		return nil, err
	}
	if ownerCount != 1 {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerOwnerInvariantViolation, "账本必须且只能有一名 Owner")
	}

	after, err := s.getMemberListWithTx(ctx, tx, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, err
	}
	before := MemberListData{Ledger: *beforeLedger, Members: beforeMembers}
	if err := s.writeLedgerAudit(ctx, tx, lc.LedgerID, lc.UserID, RoleOwner, "ledger_owner_transfer", before, after); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func (s *Service) RemoveMemberVersioned(
	ctx context.Context,
	lc LedgerContext,
	expectedVersion int64,
	targetUserID string,
) (*MemberListData, error) {
	if targetUserID == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "成员 ID 不能为空")
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	beforeLedger, err := s.requireLifecycleOwner(ctx, tx, lc, LedgerStatusActive)
	if err != nil {
		return nil, err
	}
	beforeMembers, err := s.repo.GetLedgerMembersWithTx(ctx, tx, lc.LedgerID)
	if err != nil {
		return nil, err
	}
	if err := s.claimLifecycleVersion(ctx, tx, lc.LedgerID, expectedVersion); err != nil {
		return nil, err
	}
	targetRole, err := s.repo.GetMemberRoleWithTx(ctx, tx, lc.LedgerID, targetUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "账本成员不存在")
		}
		return nil, err
	}
	if targetRole == RoleOwner {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerOwnerTransferRequired, "Owner 必须先移交所有权")
	}
	if err := s.repo.RemoveMemberWithTx(ctx, tx, lc.LedgerID, targetUserID); err != nil {
		return nil, err
	}

	after, err := s.getMemberListWithTx(ctx, tx, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, err
	}
	before := MemberListData{Ledger: *beforeLedger, Members: beforeMembers}
	if err := s.writeLedgerAudit(ctx, tx, lc.LedgerID, lc.UserID, RoleOwner, "ledger_member_remove", before, after); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func (s *Service) LeaveLedgerVersioned(
	ctx context.Context,
	lc LedgerContext,
	expectedVersion int64,
) (*LeaveLedgerResult, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	beforeLedger, err := s.repo.GetLedgerWithRole(ctx, tx, lc.LedgerID, lc.UserID)
	if err != nil {
		return nil, mapLedgerAccessError(err)
	}
	if beforeLedger.Status == LedgerStatusArchived {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerArchived, "归档账本为只读状态")
	}
	if beforeLedger.Status != LedgerStatusActive {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerInvalidState, "账本状态不允许此操作")
	}
	actorRole := Role(beforeLedger.Role)
	if actorRole == RoleOwner {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerOwnerTransferRequired, "Owner 必须先移交所有权")
	}
	if actorRole != RoleEditor && actorRole != RoleViewer {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeLedgerAccessDenied, "当前成员角色不允许离开账本")
	}
	beforeMembers, err := s.repo.GetLedgerMembersWithTx(ctx, tx, lc.LedgerID)
	if err != nil {
		return nil, err
	}
	if err := s.claimLifecycleVersion(ctx, tx, lc.LedgerID, expectedVersion); err != nil {
		return nil, err
	}
	if err := s.repo.RemoveMemberWithTx(ctx, tx, lc.LedgerID, lc.UserID); err != nil {
		return nil, err
	}
	ownerCount, err := s.repo.CountOwnersWithTx(ctx, tx, lc.LedgerID)
	if err != nil {
		return nil, err
	}
	if ownerCount != 1 {
		return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeLedgerOwnerInvariantViolation, "账本必须且只能有一名 Owner")
	}

	result := &LeaveLedgerResult{LedgerID: lc.LedgerID, Version: expectedVersion + 1}
	before := MemberListData{Ledger: *beforeLedger, Members: beforeMembers}
	if err := s.writeLedgerAudit(ctx, tx, lc.LedgerID, lc.UserID, actorRole, "ledger_member_leave", before, result); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) getMemberListWithTx(ctx context.Context, tx *sql.Tx, ledgerID, userID string) (*MemberListData, error) {
	ledgerModel, err := s.repo.GetLedgerWithRole(ctx, tx, ledgerID, userID)
	if err != nil {
		return nil, err
	}
	members, err := s.repo.GetLedgerMembersWithTx(ctx, tx, ledgerID)
	if err != nil {
		return nil, err
	}
	return &MemberListData{Ledger: *ledgerModel, Members: members}, nil
}

func (s *Service) writeLedgerAudit(
	ctx context.Context,
	tx *sql.Tx,
	ledgerID, actorUserID string,
	actorRole Role,
	action string,
	before, after any,
) error {
	var beforeJSON []byte
	var err error
	if before != nil {
		beforeJSON, err = json.Marshal(before)
		if err != nil {
			return err
		}
	}
	var afterJSON []byte
	if after != nil {
		afterJSON, err = json.Marshal(after)
		if err != nil {
			return err
		}
	}
	return s.repo.CreateLedgerAuditWithTx(ctx, tx, ledgerID, actorUserID, actorRole, action, beforeJSON, afterJSON)
}
