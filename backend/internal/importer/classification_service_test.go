package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/ledger"
)

func TestTask533PreviewClassificationModesPersistFrozenSnapshots(t *testing.T) {
	tests := []struct {
		mode                  string
		wantStatus            string
		wantSelectedCategory  string
		wantSuggestedCategory string
		wantSuggestedCount    int
		wantAutoCount         int
	}{
		{
			mode: ClassificationModeOff, wantStatus: ClassificationStatusUnresolved,
		},
		{
			mode: ClassificationModeSuggest, wantStatus: ClassificationStatusSuggested,
			wantSuggestedCategory: "cat-food", wantSuggestedCount: 1,
		},
		{
			mode: ClassificationModeGraded, wantStatus: ClassificationStatusAutoSelected,
			wantSelectedCategory: "cat-food", wantSuggestedCategory: "cat-food", wantAutoCount: 1,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.mode, func(t *testing.T) {
			database := openImporterTestDB(t)
			seedTask533ClassificationData(t, database)
			service := NewService(NewRepository(database), WithClassificationMode(testCase.mode))

			batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
				LedgerContext: ownerLedgerContext(), Filename: "generic-basic.csv",
				SourceType: SourceTypeGeneric, Content: readImportFixture(t, "generic-basic.csv"),
			})
			if err != nil {
				t.Fatalf("PreviewCSV returned error: %v", err)
			}
			row := findPreviewRowByNumber(t, batch, 1)
			if row.Classification.Status != testCase.wantStatus {
				t.Fatalf("classification status = %q, want %q; row=%+v", row.Classification.Status, testCase.wantStatus, row)
			}
			if row.SelectedCategoryID != testCase.wantSelectedCategory || row.SuggestedCategoryID != testCase.wantSuggestedCategory {
				t.Fatalf("selected/suggested = %q/%q, want %q/%q", row.SelectedCategoryID, row.SuggestedCategoryID, testCase.wantSelectedCategory, testCase.wantSuggestedCategory)
			}
			if testCase.mode != ClassificationModeOff {
				if row.Classification.Confidence != "high" || row.Classification.Source != "user_rule" || row.Classification.ReasonCode != "merchant_equals" {
					t.Fatalf("classification explanation mismatch: %+v", row.Classification)
				}
				if len(row.Classification.MatchedRuleIDs) != 1 || row.Classification.MatchedRuleIDs[0] != "rule-breakfast" {
					t.Fatalf("matched rules = %#v", row.Classification.MatchedRuleIDs)
				}
			}
			if batch.ClassificationSummary.AutoSelected != testCase.wantAutoCount || batch.ClassificationSummary.Suggested != testCase.wantSuggestedCount {
				t.Fatalf("classification summary mismatch: %+v", batch.ClassificationSummary)
			}
			if batch.ClassificationSummary.Fallback != map[bool]int{true: 2, false: 0}[testCase.mode != ClassificationModeOff] {
				t.Fatalf("fallback summary mismatch: %+v", batch.ClassificationSummary)
			}
			if batch.ClassificationSummary.Unresolved != map[bool]int{true: 1, false: 4}[testCase.mode != ClassificationModeOff] {
				t.Fatalf("unresolved summary mismatch: %+v", batch.ClassificationSummary)
			}

			var status, confidence, source, reasonJSON, matchedJSON string
			if err := database.QueryRow(`
				SELECT classification_status, classification_confidence,
				       COALESCE(classification_source, ''), classification_reason_json, matched_rule_ids_json
				FROM import_items WHERE id = ?
			`, row.ID).Scan(&status, &confidence, &source, &reasonJSON, &matchedJSON); err != nil {
				t.Fatalf("read persisted classification: %v", err)
			}
			if status != row.Classification.Status || confidence != row.Classification.Confidence || source != row.Classification.Source {
				t.Fatalf("persisted classification mismatch: %s/%s/%s row=%+v", status, confidence, source, row.Classification)
			}
			if testCase.mode != ClassificationModeOff && (reasonJSON == "{}" || matchedJSON == "[]") {
				t.Fatalf("classification explanation was not persisted: reason=%s matched=%s", reasonJSON, matchedJSON)
			}
		})
	}
}

func TestTask533ConflictPersistsWithoutASelectedCategory(t *testing.T) {
	database := openImporterTestDB(t)
	seedTask533ClassificationData(t, database)
	if _, err := database.Exec(`
		INSERT INTO import_rules (
			id, ledger_id, keyword, created_by_user_id, name, match_type, pattern,
			priority, result_json, status, origin, source_type, apply_mode, confidence,
			created_at, updated_at
		) VALUES (
			'rule-breakfast-conflict', 'ledger-one', '早餐店', 'owner-user', '早餐冲突',
			'merchant_equals', '早餐店', 10, '{"category_id":"cat-travel","tag_ids":[]}',
			'active', 'manual', 'generic', 'auto', 'high',
			'2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed conflict rule: %v", err)
	}
	service := NewService(NewRepository(database), WithClassificationMode(ClassificationModeGraded))

	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(), Filename: "generic-basic.csv",
		SourceType: SourceTypeGeneric, Content: readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("PreviewCSV returned error: %v", err)
	}
	row := findPreviewRowByNumber(t, batch, 1)
	if row.Classification.Status != ClassificationStatusConflict || row.SelectedCategoryID != "" || row.SuggestedCategoryID != "" {
		t.Fatalf("conflict snapshot mismatch: %+v", row)
	}
	if batch.ClassificationSummary.Conflict != 1 {
		t.Fatalf("conflict summary = %+v", batch.ClassificationSummary)
	}
}

func TestTask533CommitUsesPersistedClassificationWithoutRerunningRules(t *testing.T) {
	database := openImporterTestDB(t)
	seedTask533ClassificationData(t, database)
	service := NewService(NewRepository(database), WithClassificationMode(ClassificationModeGraded))
	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(), Filename: "generic-basic.csv",
		SourceType: SourceTypeGeneric, Content: readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("PreviewCSV returned error: %v", err)
	}
	if _, err := database.Exec(`UPDATE import_rules SET result_json = '{"category_id":"cat-travel","tag_ids":[]}' WHERE id = 'rule-breakfast'`); err != nil {
		t.Fatalf("change rule after preview: %v", err)
	}
	if _, err := service.CommitPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID); err != nil {
		t.Fatalf("CommitPreviewBatch returned error: %v", err)
	}
	var categoryID sql.NullString
	if err := database.QueryRow(`SELECT category_id FROM transactions WHERE title = '早餐'`).Scan(&categoryID); err != nil {
		t.Fatalf("read imported breakfast transaction: %v", err)
	}
	if categoryID.String != "cat-food" {
		t.Fatalf("commit reran changed rule: category=%q", categoryID.String)
	}
}

func TestTask533ReclassifyDryRunAndExecuteAreDeterministicAndAudited(t *testing.T) {
	database := openImporterTestDB(t)
	seedTask533ClassificationData(t, database)
	service := NewService(NewRepository(database), WithClassificationMode(ClassificationModeGraded))
	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(), Filename: "generic-basic.csv",
		SourceType: SourceTypeGeneric, Content: readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("PreviewCSV returned error: %v", err)
	}
	if _, err := database.Exec(`UPDATE import_rules SET result_json = '{"category_id":"cat-travel","tag_ids":[]}' WHERE id = 'rule-breakfast'`); err != nil {
		t.Fatalf("change classification rule: %v", err)
	}
	dryRun, err := service.ReclassifyPreviewBatch(context.Background(), ReclassifyCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, DryRun: true,
	})
	if err != nil {
		t.Fatalf("dry-run reclassify returned error: %v", err)
	}
	if !dryRun.DryRun || dryRun.EligibleRows != 3 || dryRun.ChangedRows != 1 || len(dryRun.Changes) != 1 {
		t.Fatalf("unexpected dry-run result: %+v", dryRun)
	}
	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("read batch after dry-run: %v", err)
	}
	if findPreviewRowByNumber(t, stored, 1).SelectedCategoryID != "cat-food" {
		t.Fatalf("dry-run wrote classification changes")
	}
	if countWhere(t, database, "audit_logs", "action = 'import_reclassify'") != 0 {
		t.Fatalf("dry-run wrote an audit record")
	}
	if countRows(t, database, "transactions") != 0 {
		t.Fatalf("reclassify dry-run must not create transactions")
	}

	executed, err := service.ReclassifyPreviewBatch(context.Background(), ReclassifyCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, DryRun: false,
	})
	if err != nil {
		t.Fatalf("execute reclassify returned error: %v", err)
	}
	if executed.DryRun || executed.ChangedRows != 1 || executed.Summary.AutoSelected != 1 {
		t.Fatalf("unexpected executed result: %+v", executed)
	}
	stored, err = service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("read batch after execute: %v", err)
	}
	if findPreviewRowByNumber(t, stored, 1).SelectedCategoryID != "cat-travel" {
		t.Fatalf("execute did not persist new classification")
	}
	if countWhere(t, database, "audit_logs", "action = 'import_reclassify' AND entity_type = 'import_batch'") != 1 {
		t.Fatalf("execute must write one reclassify audit record")
	}
	if countRows(t, database, "transactions") != 0 {
		t.Fatalf("reclassify execute must not create transactions")
	}
}

func TestTask533ReclassifyProtectsManualRowsAndRejectsUnavailableBatches(t *testing.T) {
	database := openImporterTestDB(t)
	seedTask533ClassificationData(t, database)
	service := NewService(NewRepository(database), WithClassificationMode(ClassificationModeGraded))
	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(), Filename: "generic-basic.csv",
		SourceType: SourceTypeGeneric, Content: readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("PreviewCSV returned error: %v", err)
	}
	row := findPreviewRowByNumber(t, batch, 1)
	status := RowStatusAdjusted
	categoryID := "cat-travel"
	if _, err := service.UpdatePreviewRow(context.Background(), UpdateRowCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: row.ID,
		Patch: UpdateRowRequest{RowStatus: &status, SelectedCategoryID: &categoryID},
	}); err != nil {
		t.Fatalf("manual row adjustment: %v", err)
	}
	bulkRow := findPreviewRowByNumber(t, batch, 2)
	if _, err := database.Exec(`
		UPDATE import_items
		SET row_status = 'adjusted', classification_status = 'bulk',
		    classification_source = 'bulk', selected_category_id = 'cat-travel'
		WHERE id = ?
	`, bulkRow.ID); err != nil {
		t.Fatalf("mark bulk-protected row: %v", err)
	}
	if _, err := database.Exec(`UPDATE import_rules SET result_json = '{"category_id":"cat-food","tag_ids":[]}' WHERE id = 'rule-breakfast'`); err != nil {
		t.Fatalf("change rule: %v", err)
	}

	result, err := service.ReclassifyPreviewBatch(context.Background(), ReclassifyCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, DryRun: false,
	})
	if err != nil {
		t.Fatalf("reclassify with manual row returned error: %v", err)
	}
	if result.ProtectedManualRows != 1 || result.ProtectedBulkRows != 1 {
		t.Fatalf("protected manual/bulk rows = %d/%d, want 1/1", result.ProtectedManualRows, result.ProtectedBulkRows)
	}
	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("read batch: %v", err)
	}
	manual := findPreviewRowByNumber(t, stored, 1)
	if manual.SelectedCategoryID != "cat-travel" || manual.Classification.Status != ClassificationStatusManual {
		t.Fatalf("manual row was overwritten: %+v", manual)
	}

	if _, err := service.CommitPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID); err != nil {
		t.Fatalf("commit batch: %v", err)
	}
	_, err = service.ReclassifyPreviewBatch(context.Background(), ReclassifyCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, DryRun: true,
	})
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeImportReclassifyConflict)

	offService := NewService(NewRepository(database), WithClassificationMode(ClassificationModeOff))
	_, err = offService.ReclassifyPreviewBatch(context.Background(), ReclassifyCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, DryRun: true,
	})
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeImportReclassifyConflict)

	foreign := ownerLedgerContext()
	foreign.LedgerID = "ledger-two"
	_, err = service.ReclassifyPreviewBatch(context.Background(), ReclassifyCommand{
		LedgerContext: foreign, BatchID: batch.ID, DryRun: true,
	})
	var appErr *appErrors.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusNotFound {
		t.Fatalf("cross-ledger reclassify error = %v", err)
	}
}

func TestTask533DuplicateRowsRemainUnclassifiedAndExpiredBatchesAreRejected(t *testing.T) {
	database := openImporterTestDB(t)
	seedTask533ClassificationData(t, database)
	service := NewService(NewRepository(database), WithClassificationMode(ClassificationModeGraded))
	request := PreviewFileRequest{
		LedgerContext: ownerLedgerContext(), Filename: "generic-basic.csv",
		SourceType: SourceTypeGeneric, Content: readImportFixture(t, "generic-basic.csv"),
	}
	first, err := service.PreviewCSV(context.Background(), request)
	if err != nil {
		t.Fatalf("first preview: %v", err)
	}
	if _, err := service.CommitPreviewBatch(context.Background(), ownerLedgerContext(), first.ID); err != nil {
		t.Fatalf("commit first preview: %v", err)
	}
	duplicate, err := service.PreviewCSV(context.Background(), request)
	if err != nil {
		t.Fatalf("duplicate preview: %v", err)
	}
	for _, row := range duplicate.Rows {
		if row.DuplicateStatus == DuplicateStatusDuplicate && row.Classification.Status != ClassificationStatusUnresolved {
			t.Fatalf("duplicate row %d was classified: %+v", row.RowNumber, row.Classification)
		}
	}
	if duplicate.ClassificationSummary.Unresolved != len(duplicate.Rows) {
		t.Fatalf("duplicate preview summary mismatch: %+v", duplicate.ClassificationSummary)
	}
	if _, err := database.Exec(`UPDATE import_batches SET expires_at = '2026-01-01T00:00:00Z' WHERE id = ?`, duplicate.ID); err != nil {
		t.Fatalf("expire duplicate preview: %v", err)
	}
	_, err = service.ReclassifyPreviewBatch(context.Background(), ReclassifyCommand{
		LedgerContext: ownerLedgerContext(), BatchID: duplicate.ID, DryRun: true,
	})
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeImportReclassifyConflict)
}

func TestTask533ReclassifyHTTPDefaultsToDryRun(t *testing.T) {
	database := openImporterTestDB(t)
	seedTask533ClassificationData(t, database)
	service := NewService(NewRepository(database), WithClassificationMode(ClassificationModeGraded))
	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(), Filename: "generic-basic.csv",
		SourceType: SourceTypeGeneric, Content: readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if _, err := database.Exec(`UPDATE import_rules SET result_json = '{"category_id":"cat-travel","tag_ids":[]}' WHERE id = 'rule-breakfast'`); err != nil {
		t.Fatalf("change rule: %v", err)
	}

	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("batchID", batch.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/imports/"+batch.ID+"/reclassify", strings.NewReader(""))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeContext)
	ctx = context.WithValue(ctx, middleware.UserIDKey, "owner-user")
	ctx = ledger.ContextWithLedgerContext(ctx, ownerLedgerContext())
	req = req.WithContext(ctx)
	recorder := httptest.NewRecorder()
	NewHandler(service).HandleReclassify(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("reclassify status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Success bool             `json:"success"`
		Data    ReclassifyResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode reclassify response: %v", err)
	}
	if !body.Success || !body.Data.DryRun || body.Data.ChangedRows != 1 {
		t.Fatalf("unexpected reclassify response: %+v", body)
	}
	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("read preview after HTTP dry-run: %v", err)
	}
	if findPreviewRowByNumber(t, stored, 1).SelectedCategoryID != "cat-food" {
		t.Fatalf("default HTTP dry-run persisted changes")
	}
}

func TestTask533ReclassifyRepositoryRejectsConcurrentManualAdjustment(t *testing.T) {
	database := openImporterTestDB(t)
	seedTask533ClassificationData(t, database)
	repository := NewRepository(database)
	service := NewService(repository, WithClassificationMode(ClassificationModeGraded))
	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(), Filename: "generic-basic.csv",
		SourceType: SourceTypeGeneric, Content: readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	row := findPreviewRowByNumber(t, batch, 1)
	if _, err := database.Exec(`
		UPDATE import_items
		SET row_status = 'adjusted', classification_status = 'manual', classification_source = 'manual'
		WHERE id = ?
	`, row.ID); err != nil {
		t.Fatalf("concurrent manual update: %v", err)
	}
	row.SelectedCategoryID = "cat-travel"
	err = repository.ApplyReclassification(context.Background(), ownerLedgerContext(), batch.ID, batch.UpdatedAt, []PreviewRow{row}, &ReclassifyResult{
		DryRun: false, ChangedRows: 1, Changes: []ReclassifyRowChange{},
	})
	if !errors.Is(err, errReclassifyBatchChanged) {
		t.Fatalf("ApplyReclassification error = %v, want stale batch conflict", err)
	}
	if countWhere(t, database, "audit_logs", "action = 'import_reclassify'") != 0 {
		t.Fatalf("concurrent conflict must roll back audit")
	}
}

func TestTask533ReclassifyUsesPersistedSourceAccount(t *testing.T) {
	database := openImporterTestDB(t)
	seedTask533ClassificationData(t, database)
	if _, err := database.Exec(`
		INSERT INTO import_rules (
			id, ledger_id, keyword, created_by_user_id, name, match_type, pattern,
			priority, result_json, status, origin, source_type, apply_mode, confidence,
			created_at, updated_at
		) VALUES (
			'rule-alipay-account', 'ledger-one', 'alipay', 'owner-user', '支付宝来源账户',
			'source_account', 'alipay', 20, '{"category_id":"cat-travel","tag_ids":[]}',
			'active', 'manual', 'generic', 'auto', 'high',
			'2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed source-account rule: %v", err)
	}
	service := NewService(NewRepository(database), WithClassificationMode(ClassificationModeGraded))
	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(), Filename: "generic-basic.csv",
		SourceType: SourceTypeGeneric, Content: readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if findPreviewRowByNumber(t, batch, 2).SelectedCategoryID != "cat-travel" {
		t.Fatalf("initial source-account rule did not classify row")
	}
	if _, err := database.Exec(`UPDATE import_rules SET result_json = '{"category_id":"cat-food","tag_ids":[]}' WHERE id = 'rule-alipay-account'`); err != nil {
		t.Fatalf("change source-account rule: %v", err)
	}
	result, err := service.ReclassifyPreviewBatch(context.Background(), ReclassifyCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, DryRun: true,
	})
	if err != nil {
		t.Fatalf("reclassify: %v", err)
	}
	var sourceAccountChange *ReclassifyRowChange
	for index := range result.Changes {
		if result.Changes[index].RowID == findPreviewRowByNumber(t, batch, 2).ID {
			sourceAccountChange = &result.Changes[index]
			break
		}
	}
	if sourceAccountChange == nil || sourceAccountChange.NewCategoryID != "cat-food" {
		t.Fatalf("persisted source account was not used during reclassify: %+v", result)
	}
}

func seedTask533ClassificationData(t *testing.T, database *sql.DB) {
	t.Helper()
	_, err := database.Exec(`
		INSERT INTO categories (
			id, ledger_id, owner_user_id, name, type, system_key, color, is_archived, created_at, updated_at
		) VALUES
			('cat-food', 'ledger-one', 'owner-user', '餐饮', 'expense', 'expense_food', '#22c55e', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('cat-travel', 'ledger-one', 'owner-user', '差旅', 'expense', 'expense_travel', '#3b82f6', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('cat-expense-other', 'ledger-one', 'owner-user', '其他支出', 'expense', 'expense_other', '#94a3b8', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('cat-income-other', 'ledger-one', 'owner-user', '其他收入', 'income', 'income_other', '#94a3b8', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO tags (
			id, ledger_id, owner_user_id, name, system_key, color, is_archived, created_at, updated_at
		) VALUES (
			'tag-breakfast', 'ledger-one', 'owner-user', '早餐', 'tag_work', '#0f766e', 0,
			'2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		);
		INSERT INTO import_rules (
			id, ledger_id, keyword, created_by_user_id, name, match_type, pattern,
			priority, result_json, status, origin, source_type, apply_mode, confidence,
			created_at, updated_at
		) VALUES (
			'rule-breakfast', 'ledger-one', '早餐店', 'owner-user', '早餐精确规则',
			'merchant_equals', '早餐店', 10,
			'{"category_id":"cat-food","tag_ids":["tag-breakfast"]}',
			'active', 'manual', 'generic', 'auto', 'high',
			'2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		)
	`)
	if err != nil {
		t.Fatalf("seed Task53.3 classification data: %v", err)
	}
}
