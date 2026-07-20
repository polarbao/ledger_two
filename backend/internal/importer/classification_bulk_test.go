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

func TestTask534ABulkAdjustAcceptsPersistedSuggestionsWithPartialResults(t *testing.T) {
	database, service, batch := newTask534ABulkFixture(t)
	rows := task534ARowsByNumber(t, batch)
	if _, err := database.Exec(`
		UPDATE import_items
		SET classification_status = 'conflict', classification_confidence = 'none'
		WHERE id = ?
	`, rows[2].ID); err != nil {
		t.Fatalf("mark conflict row: %v", err)
	}
	if _, err := database.Exec(`UPDATE import_items SET duplicate_status = 'duplicate' WHERE id = ?`, rows[3].ID); err != nil {
		t.Fatalf("mark duplicate row: %v", err)
	}

	result, err := service.BulkAdjustPreviewRows(context.Background(), BulkAdjustCommand{
		LedgerContext: ownerLedgerContext(),
		BatchID:       batch.ID,
		Request: BulkClassificationRequest{
			RowIDs: []string{rows[1].ID, rows[2].ID, rows[3].ID, rows[4].ID, "missing-row"},
			Action: BulkAdjustActionAcceptSuggestions,
		},
	})
	if err != nil {
		t.Fatalf("BulkAdjustPreviewRows returned error: %v", err)
	}
	if result.AffectedRows != 1 || result.SkippedRows != 2 || result.ConflictRows != 1 {
		t.Fatalf("unexpected partial counts: %+v", result)
	}
	assertStringList(t, result.UpdatedRowIDs, []string{rows[1].ID})
	assertStringList(t, result.ConflictRowIDs, []string{rows[2].ID})
	assertStringList(t, result.SkippedRowIDs, []string{rows[3].ID, rows[4].ID})
	if len(result.Errors) != 1 || result.Errors[0].RowID != "missing-row" || result.Errors[0].Code != appErrors.ErrCodeLedgerObjectNotFound {
		t.Fatalf("unexpected row errors: %+v", result.Errors)
	}
	if result.Summary.Bulk != 1 || result.Summary.Conflict != 1 {
		t.Fatalf("unexpected classification summary: %+v", result.Summary)
	}

	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("read adjusted batch: %v", err)
	}
	adjusted := findPreviewRow(t, stored, rows[1].ID)
	if adjusted.RowStatus != RowStatusAdjusted || adjusted.Classification.Status != ClassificationStatusBulk || adjusted.Classification.Source != "bulk" {
		t.Fatalf("suggestion was not marked as bulk adjustment: %+v", adjusted)
	}
	if adjusted.SelectedCategoryID != rows[1].SuggestedCategoryID || adjusted.SelectedAccountID != rows[1].SuggestedAccountID {
		t.Fatalf("persisted suggestion was not applied: %+v", adjusted)
	}
	assertStringList(t, adjusted.SelectedTagIDs, rows[1].SuggestedTagIDs)
	if adjusted.SuggestedCategoryID != rows[1].SuggestedCategoryID || adjusted.SuggestionReason != rows[1].SuggestionReason {
		t.Fatalf("suggestion source snapshot was overwritten: before=%+v after=%+v", rows[1], adjusted)
	}
	if countWhere(t, database, "audit_logs", "action = 'import_bulk_adjust'") != 1 {
		t.Fatalf("bulk request must write exactly one audit")
	}
	if countRows(t, database, "transactions") != 0 || countWhere(t, database, "import_rules", "origin = 'learned'") != 0 {
		t.Fatalf("bulk adjustment must not create transactions or learned rules")
	}
	var auditJSON string
	if err := database.QueryRow(`SELECT after_json FROM audit_logs WHERE action = 'import_bulk_adjust'`).Scan(&auditJSON); err != nil {
		t.Fatalf("read bulk audit: %v", err)
	}
	for _, forbidden := range []string{"早餐店", "早餐", "通用模板支出", "merchant", "title", "description"} {
		if strings.Contains(auditJSON, forbidden) {
			t.Fatalf("bulk audit leaked source text %q: %s", forbidden, auditJSON)
		}
	}
}

func TestTask534ABulkApplyValuesReportsCategoryMismatchPerRow(t *testing.T) {
	_, service, batch := newTask534ABulkFixture(t)
	rows := task534ARowsByNumber(t, batch)
	manualStatus := RowStatusAdjusted
	manualCategory := "cat-travel"
	if _, err := service.UpdatePreviewRow(context.Background(), UpdateRowCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID, RowID: rows[1].ID,
		Patch: UpdateRowRequest{RowStatus: &manualStatus, SelectedCategoryID: &manualCategory},
	}); err != nil {
		t.Fatalf("prepare manual row: %v", err)
	}
	accountID := "account-cash"
	tags := []string{"tag-breakfast"}
	result, err := service.BulkAdjustPreviewRows(context.Background(), BulkAdjustCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID,
		Request: BulkClassificationRequest{
			RowIDs: rowsInOrder(rows, 1, 3, 4), Action: BulkAdjustActionApplyValues,
			CategoryID: NullableString{Set: true, Value: stringPointer("cat-food")}, AccountID: NullableString{Set: true, Value: &accountID}, TagIDs: &tags,
		},
	})
	if err != nil {
		t.Fatalf("BulkAdjustPreviewRows returned error: %v", err)
	}
	assertStringList(t, result.UpdatedRowIDs, []string{rows[1].ID})
	assertStringList(t, result.SkippedRowIDs, []string{rows[4].ID})
	if len(result.Errors) != 1 || result.Errors[0].RowID != rows[3].ID || result.Errors[0].Code != appErrors.ErrCodeCategoryTypeMismatch {
		t.Fatalf("category mismatch must be a row error: %+v", result.Errors)
	}
	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("read adjusted batch: %v", err)
	}
	adjusted := findPreviewRow(t, stored, rows[1].ID)
	if adjusted.SelectedCategoryID != "cat-food" || adjusted.SelectedAccountID != accountID {
		t.Fatalf("explicit values were not applied: %+v", adjusted)
	}
	if adjusted.Classification.Status != ClassificationStatusBulk {
		t.Fatalf("apply_values did not explicitly overwrite the manual classification: %+v", adjusted.Classification)
	}
	assertStringList(t, adjusted.SelectedTagIDs, tags)
	if findPreviewRow(t, stored, rows[3].ID).Classification.Status == ClassificationStatusBulk {
		t.Fatalf("income row was modified by an expense category")
	}
}

func TestTask534AAcceptSuggestionsReportsArchivedReferencesAsRowErrors(t *testing.T) {
	database, service, batch := newTask534ABulkFixture(t)
	row := findPreviewRowByNumber(t, batch, 1)
	if _, err := database.Exec(`UPDATE tags SET is_archived = 1 WHERE id = 'tag-breakfast'`); err != nil {
		t.Fatalf("archive suggested tag: %v", err)
	}

	result, err := service.BulkAdjustPreviewRows(context.Background(), BulkAdjustCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID,
		Request: BulkClassificationRequest{RowIDs: []string{row.ID}, Action: BulkAdjustActionAcceptSuggestions},
	})
	if err != nil {
		t.Fatalf("BulkAdjustPreviewRows returned top-level error: %v", err)
	}
	if len(result.Errors) != 1 || result.Errors[0].RowID != row.ID || result.Errors[0].Code != appErrors.ErrCodeClassificationRuleStale {
		t.Fatalf("archived suggestion reference was not reported safely: %+v", result)
	}
	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("read batch: %v", err)
	}
	if findPreviewRow(t, stored, row.ID).Classification.Status == ClassificationStatusBulk {
		t.Fatalf("stale suggestion was applied")
	}
	if countWhere(t, database, "audit_logs", "action = 'import_bulk_adjust'") != 1 {
		t.Fatalf("partially handled request must write one redacted audit")
	}
}

func TestTask534ABulkAdjustValidatesPayloadAndActiveLedgerMetadata(t *testing.T) {
	database, service, batch := newTask534ABulkFixture(t)
	row := findPreviewRowByNumber(t, batch, 1)
	accountID := "account-cash"
	nineTags := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}

	tests := []struct {
		name    string
		request BulkClassificationRequest
		status  int
		code    string
	}{
		{name: "duplicate row ids", request: BulkClassificationRequest{RowIDs: []string{row.ID, row.ID}, Action: BulkAdjustActionAcceptSuggestions}, status: http.StatusBadRequest, code: appErrors.ErrCodeValidationError},
		{name: "too many tags", request: BulkClassificationRequest{RowIDs: []string{row.ID}, Action: BulkAdjustActionApplyValues, CategoryID: NullableString{Set: true, Value: stringPointer("cat-food")}, AccountID: NullableString{Set: true, Value: &accountID}, TagIDs: &nineTags}, status: http.StatusBadRequest, code: appErrors.ErrCodeTagLimitExceeded},
		{name: "missing account field", request: BulkClassificationRequest{RowIDs: []string{row.ID}, Action: BulkAdjustActionApplyValues, CategoryID: NullableString{Set: true, Value: stringPointer("cat-food")}, TagIDs: &[]string{}}, status: http.StatusBadRequest, code: appErrors.ErrCodeValidationError},
		{name: "archived category", request: BulkClassificationRequest{RowIDs: []string{row.ID}, Action: BulkAdjustActionApplyValues, CategoryID: NullableString{Set: true, Value: stringPointer("cat-food")}, AccountID: NullableString{Set: true}, TagIDs: &[]string{}}, status: http.StatusNotFound, code: appErrors.ErrCodeLedgerObjectNotFound},
	}
	if _, err := database.Exec(`UPDATE categories SET is_archived = 1 WHERE id = 'cat-food'`); err != nil {
		t.Fatalf("archive category: %v", err)
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := service.BulkAdjustPreviewRows(context.Background(), BulkAdjustCommand{
				LedgerContext: ownerLedgerContext(), BatchID: batch.ID, Request: testCase.request,
			})
			assertAppError(t, err, testCase.status, testCase.code)
		})
	}
	if countWhere(t, database, "audit_logs", "action = 'import_bulk_adjust'") != 0 {
		t.Fatalf("top-level validation failures must not write audits")
	}
}

func TestTask534ABulkAdjustRollsBackRowsWhenAuditFails(t *testing.T) {
	database, service, batch := newTask534ABulkFixture(t)
	row := findPreviewRowByNumber(t, batch, 1)
	if _, err := database.Exec(`
		CREATE TRIGGER fail_bulk_audit BEFORE INSERT ON audit_logs
		WHEN NEW.action = 'import_bulk_adjust'
		BEGIN
			SELECT RAISE(ABORT, 'injected audit failure');
		END;
	`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	_, err := service.BulkAdjustPreviewRows(context.Background(), BulkAdjustCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID,
		Request: BulkClassificationRequest{RowIDs: []string{row.ID}, Action: BulkAdjustActionAcceptSuggestions},
	})
	assertAppError(t, err, http.StatusInternalServerError, appErrors.ErrCodeInternalError)
	stored, readErr := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if readErr != nil {
		t.Fatalf("read batch after rollback: %v", readErr)
	}
	after := findPreviewRow(t, stored, row.ID)
	if after.RowStatus != row.RowStatus || after.Classification.Status != row.Classification.Status || after.SelectedCategoryID != row.SelectedCategoryID {
		t.Fatalf("audit failure left a partial row update: before=%+v after=%+v", row, after)
	}
	if countWhere(t, database, "audit_logs", "action = 'import_bulk_adjust'") != 0 {
		t.Fatalf("failed transaction left an audit")
	}
}

func TestTask534ABulkAdjustRejectsUnauthorizedUnavailableAndConcurrentBatches(t *testing.T) {
	database, service, batch := newTask534ABulkFixture(t)
	row := findPreviewRowByNumber(t, batch, 1)
	editor := ownerLedgerContext()
	editor.UserID = "editor-user"
	editor.Role = ledger.RoleEditor
	_, err := service.BulkAdjustPreviewRows(context.Background(), BulkAdjustCommand{
		LedgerContext: editor, BatchID: batch.ID,
		Request: BulkClassificationRequest{RowIDs: []string{row.ID}, Action: BulkAdjustActionAcceptSuggestions},
	})
	assertAppError(t, err, http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound)

	foreign := ownerLedgerContext()
	foreign.LedgerID = "ledger-two"
	_, err = service.BulkAdjustPreviewRows(context.Background(), BulkAdjustCommand{
		LedgerContext: foreign, BatchID: batch.ID,
		Request: BulkClassificationRequest{RowIDs: []string{row.ID}, Action: BulkAdjustActionAcceptSuggestions},
	})
	assertAppError(t, err, http.StatusNotFound, appErrors.ErrCodeLedgerObjectNotFound)

	if _, err := database.Exec(`UPDATE import_batches SET expires_at = '2026-01-01T00:00:00Z' WHERE id = ?`, batch.ID); err != nil {
		t.Fatalf("expire batch: %v", err)
	}
	_, err = service.BulkAdjustPreviewRows(context.Background(), BulkAdjustCommand{
		LedgerContext: ownerLedgerContext(), BatchID: batch.ID,
		Request: BulkClassificationRequest{RowIDs: []string{row.ID}, Action: BulkAdjustActionAcceptSuggestions},
	})
	assertAppError(t, err, http.StatusConflict, appErrors.ErrCodeImportBulkAdjustConflict)
	if countWhere(t, database, "audit_logs", "action = 'import_bulk_adjust'") != 0 {
		t.Fatalf("rejected batch operations must not write audits")
	}
}

func TestTask534ABulkRepositoryRejectsConcurrentRowAdjustment(t *testing.T) {
	database, service, batch := newTask534ABulkFixture(t)
	repository := service.repo
	row := findPreviewRowByNumber(t, batch, 1)
	adjusted := row
	adjusted.SelectedCategoryID = row.SuggestedCategoryID
	adjusted.SelectedTagIDs = copyStrings(row.SuggestedTagIDs)
	markBulkClassification(&adjusted, "bulk_accept_suggestions", "已批量接受持久化建议")
	if _, err := database.Exec(`
		UPDATE import_items
		SET row_status = 'adjusted', classification_status = 'manual', classification_source = 'manual'
		WHERE id = ?
	`, row.ID); err != nil {
		t.Fatalf("inject concurrent manual adjustment: %v", err)
	}
	result := &BulkClassificationResult{
		AffectedRows: 1, UpdatedRowIDs: []string{row.ID}, SkippedRowIDs: []string{},
		ConflictRowIDs: []string{}, Errors: []ClassificationRowError{}, Summary: batch.ClassificationSummary,
	}
	err := repository.ApplyBulkAdjustment(
		context.Background(), ownerLedgerContext(), batch.ID, batch.UpdatedAt,
		[]bulkRowUpdate{{Before: row, After: adjusted}}, result, BulkAdjustActionAcceptSuggestions,
	)
	if !errors.Is(err, errBulkAdjustBatchChanged) {
		t.Fatalf("ApplyBulkAdjustment error = %v, want concurrent conflict", err)
	}
	if countWhere(t, database, "audit_logs", "action = 'import_bulk_adjust'") != 0 {
		t.Fatalf("concurrent conflict must not write an audit")
	}
}

func TestTask534ABulkAdjustHTTPPreservesExplicitNullAccount(t *testing.T) {
	_, service, batch := newTask534ABulkFixture(t)
	row := findPreviewRowByNumber(t, batch, 1)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("batchID", batch.ID)
	body := `{"row_ids":["` + row.ID + `"],"action":"apply_values","category_id":"cat-food","account_id":null,"tag_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/imports/"+batch.ID+"/rows/bulk-adjust", strings.NewReader(body))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeContext)
	ctx = context.WithValue(ctx, middleware.UserIDKey, "owner-user")
	ctx = ledger.ContextWithLedgerContext(ctx, ownerLedgerContext())
	recorder := httptest.NewRecorder()
	NewHandler(service).HandleBulkAdjust(recorder, req.WithContext(ctx))
	if recorder.Code != http.StatusOK {
		t.Fatalf("bulk adjust status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var bodyResult struct {
		Success bool                     `json:"success"`
		Data    BulkClassificationResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &bodyResult); err != nil {
		t.Fatalf("decode bulk response: %v", err)
	}
	if !bodyResult.Success || bodyResult.Data.AffectedRows != 1 {
		t.Fatalf("unexpected bulk response: %+v", bodyResult)
	}
	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("read adjusted batch: %v", err)
	}
	if findPreviewRow(t, stored, row.ID).SelectedAccountID != "" {
		t.Fatalf("explicit null account did not clear the selected account")
	}
}

func newTask534ABulkFixture(t *testing.T) (*sql.DB, *Service, *PreviewBatch) {
	t.Helper()
	database := openImporterTestDB(t)
	seedTask533ClassificationData(t, database)
	if _, err := database.Exec(`
		INSERT INTO accounts (
			id, ledger_id, owner_user_id, name, type, currency, initial_balance,
			is_archived, created_at, updated_at
		) VALUES (
			'account-cash', 'ledger-one', 'owner-user', '现金', 'cash', 'CNY', 0,
			0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed bulk account: %v", err)
	}
	service := NewService(NewRepository(database), WithClassificationMode(ClassificationModeGraded))
	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(), Filename: "generic-basic.csv",
		SourceType: SourceTypeGeneric, Content: readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("preview bulk fixture: %v", err)
	}
	return database, service, batch
}

func task534ARowsByNumber(t *testing.T, batch *PreviewBatch) map[int]PreviewRow {
	t.Helper()
	return map[int]PreviewRow{
		1: findPreviewRowByNumber(t, batch, 1),
		2: findPreviewRowByNumber(t, batch, 2),
		3: findPreviewRowByNumber(t, batch, 3),
		4: findPreviewRowByNumber(t, batch, 4),
	}
}

func rowsInOrder(rows map[int]PreviewRow, numbers ...int) []string {
	result := make([]string, 0, len(numbers))
	for _, number := range numbers {
		result = append(result, rows[number].ID)
	}
	return result
}

func stringPointer(value string) *string {
	return &value
}

func assertStringList(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("list length = %d, want %d: got=%v want=%v", len(got), len(want), got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("list[%d] = %q, want %q: got=%v want=%v", index, got[index], want[index], got, want)
		}
	}
}
