package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	appErrors "ledger_two/internal/errors"
	ledgerctx "ledger_two/internal/ledger"
	"ledger_two/internal/metadata/defaults"
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
	if archived {
		_, err := s.Archive(ctx, currentUserID, kind, id, ArchiveRequest{})
		return err
	}
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

func (s *Service) Archive(ctx context.Context, currentUserID string, kind Kind, id string, req ArchiveRequest) (*ArchiveResult, error) {
	ledgerID, role, err := s.resolveLedger(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	if !CanManage(role) {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅 Owner 可管理元数据")
	}
	id = strings.TrimSpace(id)
	req.ReplacementCategoryID = strings.TrimSpace(req.ReplacementCategoryID)
	if id == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "元数据 ID 不能为空")
	}
	if kind == KindCategory {
		result, err := s.repo.ArchiveCategory(ctx, ledgerID, currentUserID, id, req.ReplacementCategoryID)
		switch {
		case errors.Is(err, errFallbackReplacementRequired):
			return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeCategoryFallbackRequired, "归档兜底分类前必须指定同类型替代分类")
		case errors.Is(err, errFallbackReplacementInvalid):
			return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeCategoryFallbackReplacementInvalid, "兜底替代分类无效或状态已变化")
		case err != nil:
			return nil, mapNotFound(err, kind)
		default:
			return result, nil
		}
	}
	if err := s.repo.SetArchived(ctx, kind, ledgerID, id, true); err != nil {
		return nil, mapNotFound(err, kind)
	}
	return &ArchiveResult{ArchivedID: id}, nil
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

func (s *Service) GetDefaultProfile(ctx context.Context, currentUserID string, profileKey string) (*DefaultProfile, error) {
	preview, err := s.PreviewDefaultProfile(ctx, currentUserID, ProfilePreviewRequest{Profile: profileKey})
	if err != nil {
		return nil, err
	}
	return &preview.Profile, nil
}

func (s *Service) PreviewDefaultProfile(ctx context.Context, currentUserID string, req ProfilePreviewRequest) (*ProfilePreviewResult, error) {
	profile, err := resolveDefaultProfile(req.Profile)
	if err != nil {
		return nil, err
	}
	ledgerID, _, err := s.resolveLedger(ctx, currentUserID)
	if err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return s.previewDefaultProfileWithTx(ctx, tx, ledgerID, profile)
}

func (s *Service) ApplyDefaultProfile(ctx context.Context, currentUserID string, req ProfileApplyRequest) (*ProfileApplyResult, error) {
	profile, err := resolveDefaultProfile(req.Profile)
	if err != nil {
		return nil, err
	}
	ledgerID, role, err := s.resolveLedger(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	if !CanManage(role) {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅 Owner 可应用默认分类和标签")
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	currentVersion, err := s.repo.profileVersionWithTx(ctx, tx, ledgerID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	result := &ProfileApplyResult{
		Profile:                profile.Key,
		MetadataProfileVersion: currentVersion,
	}
	if currentVersion >= profile.Version {
		if err := s.writeProfileAudit(ctx, tx, ledgerID, currentUserID, profile.Key, currentVersion, result, now); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return result, nil
	}

	preview, err := s.previewDefaultProfileWithTx(ctx, tx, ledgerID, profile)
	if err != nil {
		return nil, err
	}
	resolutions, err := validateProfileResolutions(req.Resolutions, preview.Profile.Items)
	if err != nil {
		return nil, err
	}

	for index, previewItem := range preview.Profile.Items {
		switch previewItem.Action {
		case ProfileActionExisting:
			result.ReusedCount++
		case ProfileActionCreate:
			if err := defaults.InsertItem(ctx, tx, ledgerID, currentUserID, profile.Items[index], now); err != nil {
				return nil, err
			}
			result.CreatedCount++
		case ProfileActionConflict:
			resolution := resolutions[previewItem.SystemKey]
			if resolution.Action == ProfileResolutionReuse {
				result.ReusedCount++
			} else {
				result.SkippedCount++
			}
		}
	}

	if err := s.repo.updateProfileVersionWithTx(ctx, tx, ledgerID, profile.Version, now); err != nil {
		return nil, err
	}
	result.MetadataProfileVersion = profile.Version
	if err := s.writeProfileAudit(ctx, tx, ledgerID, currentUserID, profile.Key, currentVersion, result, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) previewDefaultProfileWithTx(ctx context.Context, tx *sql.Tx, ledgerID string, profile defaults.Profile) (*ProfilePreviewResult, error) {
	state, err := s.repo.listProfileStateWithTx(ctx, tx, ledgerID)
	if err != nil {
		return nil, err
	}
	bySystemKey := make(map[string]profileStateItem, len(state))
	byNameAndKind := make(map[string]profileStateItem, len(state))
	for _, existing := range state {
		if existing.SystemKey != "" {
			bySystemKey[existing.SystemKey] = existing
		}
		byNameAndKind[profileNameKey(existing.Kind, existing.Name)] = existing
	}

	result := &ProfilePreviewResult{
		Profile: DefaultProfile{Key: profile.Key, Version: profile.Version, Items: make([]ProfileItem, 0, len(profile.Items))},
	}
	for _, definition := range profile.Items {
		item := ProfileItem{
			SystemKey: definition.SystemKey,
			Kind:      string(definition.Kind),
			Name:      definition.Name,
			Icon:      definition.Icon,
			Color:     definition.Color,
			Action:    ProfileActionCreate,
		}
		if existing, ok := bySystemKey[definition.SystemKey]; ok {
			item.Action = ProfileActionExisting
			item.ExistingID = existing.ID
			result.ReuseCount++
		} else if existing, ok := byNameAndKind[profileNameKey(string(definition.Kind), definition.Name)]; ok {
			item.Action = ProfileActionConflict
			item.ExistingID = existing.ID
			result.ConflictCount++
		} else {
			result.CreateCount++
		}
		result.Profile.Items = append(result.Profile.Items, item)
	}
	return result, nil
}

func resolveDefaultProfile(profileKey string) (defaults.Profile, error) {
	profileKey = strings.TrimSpace(profileKey)
	if profileKey == "" {
		profileKey = defaults.ProfileBasicCNV1
	}
	profile, ok := defaults.Get(profileKey)
	if !ok {
		return defaults.Profile{}, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "默认元数据模板无效")
	}
	return profile, nil
}

func validateProfileResolutions(resolutions []ProfileConflictResolution, items []ProfileItem) (map[string]ProfileConflictResolution, error) {
	conflicts := make(map[string]ProfileItem)
	for _, item := range items {
		if item.Action == ProfileActionConflict {
			conflicts[item.SystemKey] = item
		}
	}

	resolved := make(map[string]ProfileConflictResolution, len(resolutions))
	for _, resolution := range resolutions {
		resolution.SystemKey = strings.TrimSpace(resolution.SystemKey)
		resolution.Action = strings.TrimSpace(resolution.Action)
		resolution.ExistingID = strings.TrimSpace(resolution.ExistingID)
		conflict, ok := conflicts[resolution.SystemKey]
		if !ok || resolution.SystemKey == "" {
			return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "冲突解决项不属于当前默认模板预览")
		}
		if _, duplicate := resolved[resolution.SystemKey]; duplicate {
			return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "冲突解决项不能重复")
		}
		if resolution.Action != ProfileResolutionReuse && resolution.Action != ProfileResolutionSkip {
			return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "冲突解决操作必须为 reuse 或 skip")
		}
		if resolution.Action == ProfileResolutionReuse && resolution.ExistingID != conflict.ExistingID {
			return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeMetadataProfileConflict, "默认元数据冲突对象已变化，请重新预览")
		}
		if resolution.Action == ProfileResolutionSkip && resolution.ExistingID != "" && resolution.ExistingID != conflict.ExistingID {
			return nil, appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeMetadataProfileConflict, "默认元数据冲突对象已变化，请重新预览")
		}
		resolved[resolution.SystemKey] = resolution
	}

	var unresolved []string
	for systemKey := range conflicts {
		if _, ok := resolved[systemKey]; !ok {
			unresolved = append(unresolved, systemKey)
		}
	}
	sort.Strings(unresolved)
	if len(unresolved) > 0 {
		return nil, appErrors.NewAppErrorWithDetails(
			http.StatusConflict,
			appErrors.ErrCodeMetadataProfileConflict,
			"默认元数据存在未解决的同名冲突",
			map[string]any{"system_keys": unresolved},
		)
	}
	return resolved, nil
}

func (s *Service) writeProfileAudit(
	ctx context.Context,
	tx *sql.Tx,
	ledgerID string,
	actorUserID string,
	profileKey string,
	beforeVersion int,
	result *ProfileApplyResult,
	now time.Time,
) error {
	beforeJSON, err := json.Marshal(map[string]int{"metadata_profile_version": beforeVersion})
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return s.repo.createProfileAuditWithTx(ctx, tx, ledgerID, actorUserID, profileKey, beforeJSON, afterJSON, now)
}

func profileNameKey(kind string, name string) string {
	return kind + "\x00" + name
}

func (s *Service) resolveLedger(ctx context.Context, userID string) (string, string, error) {
	if userID == "" {
		return "", "", appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统")
	}
	lc, err := ledgerctx.RequireExplicitLedgerContext(ctx, userID)
	if err != nil {
		if errors.Is(err, ledgerctx.ErrLedgerContextMismatch) {
			return "", "", appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeLedgerAccessDenied, "账本上下文与当前用户不匹配")
		}
		return "", "", appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerRequired, "请选择账本后再继续")
	}
	return lc.LedgerID, string(lc.Role), nil
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
		return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "分类不存在或不属于当前账本")
	case KindTag:
		return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "标签不存在或不属于当前账本")
	case KindAccount:
		return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "账户不存在或不属于当前账本")
	default:
		return appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "元数据不存在或不属于当前账本")
	}
}
