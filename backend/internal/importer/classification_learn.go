package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/importer/classifier"
	"ledger_two/internal/ledger"
)

const learnedRuleDefaultPriority = 500

// This namespace is part of the persisted learned-rule identity contract.
var learnedRuleNamespace = uuid.MustParse("d96c24c9-4ba8-5e56-9f30-f52ddcf8a94a")

type learnMerchantSnapshot struct {
	BatchID              string
	BatchStatus          string
	BatchSourceType      string
	BatchUpdatedAt       string
	BatchExpiresAt       string
	RowID                string
	Merchant             string
	TargetType           string
	DuplicateStatus      string
	RowStatus            string
	ClassificationStatus string
	ClassificationSource string
	SelectedCategoryID   string
	SelectedTagIDs       []string
}

type learnedRuleSpec struct {
	RuleID             string
	NormalizedMerchant string
	SourceScope        string
	SourceType         *string
	Result             ImportRuleResult
}

type manualRuleConflictError struct {
	RuleID string
}

func (e *manualRuleConflictError) Error() string {
	return "active manual merchant rule conflicts with explicit learning"
}

func (s *Service) LearnMerchantRule(ctx context.Context, cmd LearnMerchantCommand) (*LearnMerchantResult, error) {
	if cmd.LedgerContext.Role != ledger.RoleOwner {
		return nil, appErrors.NewAppError(http.StatusForbidden, appErrors.ErrCodeForbidden, "仅账本 Owner 可学习导入分类规则")
	}
	if strings.TrimSpace(cmd.BatchID) == "" || strings.TrimSpace(cmd.RowID) == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "导入批次 ID 和行 ID 不能为空")
	}
	if cmd.Request.SourceScope != LearnSourceScopeCurrent && cmd.Request.SourceScope != LearnSourceScopeAll {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "学习规则来源范围无效")
	}

	snapshot, err := s.repo.LoadLearnMerchantSnapshot(ctx, cmd.LedgerContext.LedgerID, cmd.BatchID, cmd.RowID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "导入预览行不存在或不属于当前账本")
		}
		return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取学习规则来源行失败")
	}
	if err := s.validateLearnMerchantSnapshot(ctx, cmd.LedgerContext.LedgerID, snapshot); err != nil {
		return nil, err
	}

	normalizedMerchant := classifier.NormalizeText(snapshot.Merchant)
	if normalizedMerchant == "" {
		return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeClassificationMerchantRequired, "当前行缺少可学习的商户信息")
	}
	var sourceType *string
	if cmd.Request.SourceScope == LearnSourceScopeCurrent {
		value := snapshot.BatchSourceType
		sourceType = &value
	}
	spec := learnedRuleSpec{
		RuleID:             learnedMerchantRuleID(cmd.LedgerContext.LedgerID, sourceType, normalizedMerchant),
		NormalizedMerchant: normalizedMerchant,
		SourceScope:        cmd.Request.SourceScope,
		SourceType:         sourceType,
		Result: ImportRuleResult{
			CategoryID: snapshot.SelectedCategoryID,
			TagIDs:     copyStrings(snapshot.SelectedTagIDs),
		},
	}
	result, err := s.repo.UpsertLearnedMerchantRule(ctx, cmd.LedgerContext, snapshot, spec)
	if err == nil {
		return result, nil
	}
	var conflict *manualRuleConflictError
	if errors.As(err, &conflict) {
		return nil, appErrors.NewAppErrorWithDetails(
			http.StatusConflict,
			appErrors.ErrCodeClassificationConflict,
			"同一来源范围已存在显式商户规则",
			map[string]string{"rule_id": conflict.RuleID},
		)
	}
	var appErr *appErrors.AppError
	if errors.As(err, &appErr) {
		return nil, appErr
	}
	return nil, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "保存学习规则失败")
}

func (s *Service) validateLearnMerchantSnapshot(ctx context.Context, ledgerID string, snapshot learnMerchantSnapshot) error {
	if snapshot.BatchStatus != batchStatusReady || previewExpired(snapshot.BatchExpiresAt, time.Now()) {
		return learnRowStale("仅有效的待确认导入行可以学习规则")
	}
	if snapshot.RowStatus != RowStatusAdjusted ||
		(snapshot.ClassificationStatus != ClassificationStatusManual && snapshot.ClassificationStatus != ClassificationStatusBulk) ||
		(snapshot.ClassificationSource != string(classifier.SourceManual) && snapshot.ClassificationSource != string(classifier.SourceBulk)) ||
		isLearnIneligibleRow(snapshot) || snapshot.SelectedCategoryID == "" {
		return learnRowStale("请先保存当前行的最终分类和完整标签")
	}
	if len(snapshot.SelectedTagIDs) > 8 || hasDuplicateStrings(snapshot.SelectedTagIDs) {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeTagLimitExceeded, "单条账单最多选择 8 个不重复标签")
	}
	for _, tagID := range snapshot.SelectedTagIDs {
		if tagID == "" || tagID != strings.TrimSpace(tagID) {
			return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeTagLimitExceeded, "最终标签内容无效")
		}
	}
	if !isValidImportSourceType(snapshot.BatchSourceType) {
		return learnRowStale("导入来源已不可用")
	}

	metadata, err := s.repo.LoadClassificationMetadata(ctx, ledgerID)
	if err != nil {
		return appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, "读取学习规则分类元数据失败")
	}
	active := activeClassificationMetadata(metadata)
	category, exists := active[snapshot.SelectedCategoryID]
	if !exists || (category.Kind != classifier.MetadataExpenseCategory && category.Kind != classifier.MetadataIncomeCategory) {
		return learnRowStale("已保存分类不存在、已归档或不属于当前账本")
	}
	if !categoryMatchesBulkRow(category, snapshot.TargetType) {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeCategoryTypeMismatch, "分类与账单收支类型不匹配")
	}
	for _, tagID := range snapshot.SelectedTagIDs {
		tag, exists := active[tagID]
		if !exists || tag.Kind != classifier.MetadataTag {
			return learnRowStale("已保存标签不存在、已归档或不属于当前账本")
		}
	}
	return nil
}

func (r *Repository) LoadLearnMerchantSnapshot(ctx context.Context, ledgerID string, batchID string, rowID string) (learnMerchantSnapshot, error) {
	return loadLearnMerchantSnapshot(ctx, r.db, ledgerID, batchID, rowID)
}

func (r *Repository) UpsertLearnedMerchantRule(
	ctx context.Context,
	lc ledger.LedgerContext,
	expected learnMerchantSnapshot,
	spec learnedRuleSpec,
) (*LearnMerchantResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	current, err := loadLearnMerchantSnapshot(ctx, tx, lc.LedgerID, expected.BatchID, expected.RowID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.NewAppError(http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound, "导入预览行不存在或不属于当前账本")
		}
		return nil, err
	}
	if !learnMerchantSnapshotsEqual(current, expected) {
		return nil, learnRowStale("导入行已变化，请刷新后重新保存")
	}
	if current.BatchStatus != batchStatusReady || previewExpired(current.BatchExpiresAt, time.Now()) ||
		current.RowStatus != RowStatusAdjusted ||
		(current.ClassificationStatus != ClassificationStatusManual && current.ClassificationStatus != ClassificationStatusBulk) ||
		(current.ClassificationSource != string(classifier.SourceManual) && current.ClassificationSource != string(classifier.SourceBulk)) ||
		isLearnIneligibleRow(current) || current.SelectedCategoryID == "" || !isValidImportSourceType(current.BatchSourceType) {
		return nil, learnRowStale("导入行已不再满足学习条件，请刷新后重新保存")
	}
	if err := validateLearnMetadataTx(ctx, tx, lc.LedgerID, current); err != nil {
		return nil, err
	}
	conflictID, err := findActiveManualMerchantConflict(ctx, tx, lc.LedgerID, spec.SourceType, spec.NormalizedMerchant)
	if err != nil {
		return nil, err
	}
	if conflictID != "" {
		return nil, &manualRuleConflictError{RuleID: conflictID}
	}

	resultJSON, err := json.Marshal(spec.Result)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	action := LearnActionCreated
	var existingOrigin, existingStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(origin, 'manual'), COALESCE(status, 'active')
		FROM import_rules
		WHERE id = ? AND ledger_id = ?
	`, spec.RuleID, lc.LedgerID).Scan(&existingOrigin, &existingStatus)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.ExecContext(ctx, `
			INSERT INTO import_rules (
				id, ledger_id, keyword, category_id, tag_names, account_id,
				created_by_user_id, created_at, updated_at,
				name, match_type, pattern, amount_min_cents, amount_max_cents,
				priority, result_json, status, archived_at,
				origin, source_type, apply_mode, confidence
			) VALUES (?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, 'merchant_equals', ?, NULL, NULL, ?, ?, 'active', NULL, 'learned', ?, 'auto', 'high')
		`,
			spec.RuleID, lc.LedgerID, spec.NormalizedMerchant, nullString(spec.Result.CategoryID),
			nullString(strings.Join(spec.Result.TagIDs, ",")), lc.UserID, now, now,
			"已学习商户规则", spec.NormalizedMerchant, learnedRuleDefaultPriority, string(resultJSON), nullableSourceType(spec.SourceType),
		)
	case err != nil:
		return nil, err
	case existingOrigin != string(classifier.OriginLearned):
		return nil, &manualRuleConflictError{RuleID: spec.RuleID}
	default:
		action = LearnActionUpdated
		if existingStatus == "archived" {
			action = LearnActionRestored
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE import_rules
			SET keyword = ?, category_id = ?, tag_names = ?, account_id = NULL,
			    match_type = 'merchant_equals', pattern = ?, amount_min_cents = NULL, amount_max_cents = NULL,
			    result_json = ?, status = 'active', archived_at = NULL,
			    origin = 'learned', source_type = ?, confidence = 'high', updated_at = ?
			WHERE id = ? AND ledger_id = ?
		`, spec.NormalizedMerchant, nullString(spec.Result.CategoryID), nullString(strings.Join(spec.Result.TagIDs, ",")),
			spec.NormalizedMerchant, string(resultJSON), nullableSourceType(spec.SourceType), now, spec.RuleID, lc.LedgerID)
	}
	if err != nil {
		return nil, err
	}

	auditJSON, err := json.Marshal(struct {
		RuleID      string `json:"rule_id"`
		RowID       string `json:"row_id"`
		SourceScope string `json:"source_scope"`
		Action      string `json:"action"`
	}{RuleID: spec.RuleID, RowID: expected.RowID, SourceScope: spec.SourceScope, Action: action})
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_logs (
			id, ledger_id, actor_user_id, actor_role, action, entity_type,
			entity_id, before_json, after_json, created_at
		) VALUES (?, ?, ?, ?, 'import_rule_learn', 'import_rule', ?, NULL, ?, ?)
	`, uuid.NewString(), lc.LedgerID, lc.UserID, lc.Role, spec.RuleID, string(auditJSON), now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &LearnMerchantResult{
		RuleID: spec.RuleID, Action: action, NormalizedMerchant: spec.NormalizedMerchant,
		SourceScope: spec.SourceScope, SourceType: cloneStringPointer(spec.SourceType),
	}, nil
}

type learnRowQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type learnRuleQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func loadLearnMerchantSnapshot(ctx context.Context, queryer learnRowQueryer, ledgerID string, batchID string, rowID string) (learnMerchantSnapshot, error) {
	var snapshot learnMerchantSnapshot
	var expiresAt, classificationSource, categoryID, tagIDsJSON sql.NullString
	err := queryer.QueryRowContext(ctx, `
		SELECT batch.id, batch.status, batch.source_type, COALESCE(batch.updated_at, ''), batch.expires_at,
		       item.id, COALESCE(item.merchant, ''), item.target_transaction_type,
		       item.duplicate_status, item.row_status, item.classification_status,
		       item.classification_source, item.selected_category_id, item.selected_tag_ids_json
		FROM import_batches AS batch
		JOIN import_items AS item ON item.batch_id = batch.id
		WHERE batch.id = ? AND batch.ledger_id = ? AND item.id = ?
	`, batchID, ledgerID, rowID).Scan(
		&snapshot.BatchID, &snapshot.BatchStatus, &snapshot.BatchSourceType, &snapshot.BatchUpdatedAt, &expiresAt,
		&snapshot.RowID, &snapshot.Merchant, &snapshot.TargetType, &snapshot.DuplicateStatus,
		&snapshot.RowStatus, &snapshot.ClassificationStatus, &classificationSource, &categoryID, &tagIDsJSON,
	)
	if err != nil {
		return learnMerchantSnapshot{}, err
	}
	snapshot.BatchExpiresAt = valueOf(expiresAt)
	snapshot.ClassificationSource = valueOf(classificationSource)
	snapshot.SelectedCategoryID = valueOf(categoryID)
	if tagIDsJSON.Valid && strings.TrimSpace(tagIDsJSON.String) != "" {
		if err := json.Unmarshal([]byte(tagIDsJSON.String), &snapshot.SelectedTagIDs); err != nil {
			return learnMerchantSnapshot{}, err
		}
	}
	if snapshot.SelectedTagIDs == nil {
		snapshot.SelectedTagIDs = []string{}
	}
	return snapshot, nil
}

func validateLearnMetadataTx(ctx context.Context, tx *sql.Tx, ledgerID string, snapshot learnMerchantSnapshot) error {
	var categoryType string
	if err := tx.QueryRowContext(ctx, `
		SELECT type FROM categories
		WHERE id = ? AND ledger_id = ? AND COALESCE(is_archived, 0) = 0
	`, snapshot.SelectedCategoryID, ledgerID).Scan(&categoryType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return learnRowStale("已保存分类不存在、已归档或不属于当前账本")
		}
		return err
	}
	if (snapshot.TargetType == TargetTransactionExpense && categoryType != "expense") ||
		(snapshot.TargetType == TargetTransactionIncome && categoryType != "income") {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeCategoryTypeMismatch, "分类与账单收支类型不匹配")
	}
	if len(snapshot.SelectedTagIDs) > 8 || hasDuplicateStrings(snapshot.SelectedTagIDs) {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeTagLimitExceeded, "单条账单最多选择 8 个不重复标签")
	}
	for _, tagID := range snapshot.SelectedTagIDs {
		var exists bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM tags
			WHERE id = ? AND ledger_id = ? AND COALESCE(is_archived, 0) = 0)
		`, tagID, ledgerID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return learnRowStale("已保存标签不存在、已归档或不属于当前账本")
		}
	}
	return nil
}

func findActiveManualMerchantConflict(ctx context.Context, queryer learnRuleQueryer, ledgerID string, sourceType *string, normalizedMerchant string) (string, error) {
	rows, err := queryer.QueryContext(ctx, `
		SELECT id, COALESCE(source_type, ''), COALESCE(pattern, keyword)
		FROM import_rules
		WHERE ledger_id = ? AND COALESCE(status, 'active') = 'active'
		  AND COALESCE(origin, 'manual') = 'manual'
		  AND COALESCE(match_type, '') = 'merchant_equals'
		ORDER BY priority ASC, created_at DESC, id ASC
	`, ledgerID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	wantedSource := pointerValue(sourceType)
	for rows.Next() {
		var ruleID, ruleSource, pattern string
		if err := rows.Scan(&ruleID, &ruleSource, &pattern); err != nil {
			return "", err
		}
		if ruleSource == wantedSource && classifier.NormalizeText(pattern) == normalizedMerchant {
			return ruleID, nil
		}
	}
	return "", rows.Err()
}

func (r *Repository) FindActiveManualMerchantConflict(ctx context.Context, ledgerID string, sourceType *string, normalizedMerchant string) (string, error) {
	return findActiveManualMerchantConflict(ctx, r.db, ledgerID, sourceType, normalizedMerchant)
}

func learnedMerchantRuleID(ledgerID string, sourceType *string, normalizedMerchant string) string {
	sourceIdentity := LearnSourceScopeAll
	if sourceType != nil {
		sourceIdentity = "source:" + *sourceType
	}
	identity := strings.Join([]string{ledgerID, sourceIdentity, normalizedMerchant}, "\x1f")
	return uuid.NewSHA1(learnedRuleNamespace, []byte(identity)).String()
}

func previewExpired(value string, now time.Time) bool {
	if value == "" {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, value)
	return err != nil || !expiresAt.After(now)
}

func isLearnIneligibleRow(snapshot learnMerchantSnapshot) bool {
	return snapshot.DuplicateStatus == DuplicateStatusDuplicate || snapshot.DuplicateStatus == DuplicateStatusInvalid ||
		snapshot.RowStatus == RowStatusSkipped || snapshot.RowStatus == RowStatusImported || snapshot.RowStatus == RowStatusFailed ||
		(snapshot.TargetType != TargetTransactionExpense && snapshot.TargetType != TargetTransactionIncome)
}

func learnMerchantSnapshotsEqual(left learnMerchantSnapshot, right learnMerchantSnapshot) bool {
	return left.BatchID == right.BatchID && left.BatchStatus == right.BatchStatus &&
		left.BatchSourceType == right.BatchSourceType && left.BatchUpdatedAt == right.BatchUpdatedAt &&
		left.BatchExpiresAt == right.BatchExpiresAt && left.RowID == right.RowID && left.Merchant == right.Merchant &&
		left.TargetType == right.TargetType && left.DuplicateStatus == right.DuplicateStatus &&
		left.RowStatus == right.RowStatus && left.ClassificationStatus == right.ClassificationStatus &&
		left.ClassificationSource == right.ClassificationSource && left.SelectedCategoryID == right.SelectedCategoryID &&
		reflect.DeepEqual(left.SelectedTagIDs, right.SelectedTagIDs)
}

func isValidImportSourceType(value string) bool {
	return value == SourceTypeWechat || value == SourceTypeAlipay || value == SourceTypeGeneric
}

func learnRowStale(message string) *appErrors.AppError {
	return appErrors.NewAppError(http.StatusConflict, appErrors.ErrCodeClassificationRuleStale, message)
}

func nullableSourceType(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
