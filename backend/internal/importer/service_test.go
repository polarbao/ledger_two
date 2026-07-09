package importer

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"

	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
	"ledger_two/internal/ledger"
	"ledger_two/migrations"

	_ "github.com/mattn/go-sqlite3"
)

func TestPreviewCSVCreatesReadyBatchWithoutTransactions(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	service := NewService(NewRepository(database))

	content := readImportFixture(t, "wechat-basic.csv")
	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(),
		Filename:      "wechat-basic.csv",
		SourceType:    SourceTypeWechat,
		Content:       content,
	})
	if err != nil {
		t.Fatalf("PreviewCSV returned error: %v", err)
	}

	if batch.Status != "ready" {
		t.Fatalf("expected ready batch, got %s", batch.Status)
	}
	if batch.TotalRows != 5 || batch.InvalidRows != 1 || batch.SkippedRows != 1 {
		t.Fatalf("unexpected batch counts: total=%d invalid=%d skipped=%d", batch.TotalRows, batch.InvalidRows, batch.SkippedRows)
	}
	if len(batch.Rows) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(batch.Rows))
	}
	if countRows(t, database, "transactions") != 0 {
		t.Fatalf("preview must not write transactions")
	}

	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("GetPreviewBatch returned error: %v", err)
	}
	if stored.ID != batch.ID || len(stored.Rows) != 5 {
		t.Fatalf("stored batch mismatch: id=%s rows=%d", stored.ID, len(stored.Rows))
	}
}

func TestPreviewCSVRequiresOwner(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	service := NewService(NewRepository(database))

	_, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ledger.LedgerContext{
			UserID:   "editor-user",
			LedgerID: "ledger-one",
			Role:     ledger.RoleEditor,
		},
		Filename:   "generic-basic.csv",
		SourceType: SourceTypeGeneric,
		Content:    readImportFixture(t, "generic-basic.csv"),
	})
	if err == nil {
		t.Fatalf("expected forbidden error for editor preview")
	}
}

func TestUpdatePreviewRowPersistsUserAdjustment(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	service := NewService(NewRepository(database))

	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(),
		Filename:      "generic-basic.csv",
		SourceType:    SourceTypeGeneric,
		Content:       readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("PreviewCSV returned error: %v", err)
	}

	row := batch.Rows[0]
	adjustedStatus := RowStatusAdjusted
	targetType := TargetTransactionExpense
	categoryID := "cat-food"
	accountID := "account-cash"
	visibility := "partner_readable"

	updated, err := service.UpdatePreviewRow(context.Background(), UpdateRowCommand{
		LedgerContext: ownerLedgerContext(),
		BatchID:       batch.ID,
		RowID:         row.ID,
		Patch: UpdateRowRequest{
			TargetTransactionType: &targetType,
			RowStatus:             &adjustedStatus,
			SelectedCategoryID:    &categoryID,
			SelectedAccountID:     &accountID,
			SelectedTagIDs:        []string{"tag-breakfast", "tag-workday"},
			Visibility:            &visibility,
		},
	})
	if err != nil {
		t.Fatalf("UpdatePreviewRow returned error: %v", err)
	}

	updatedRow := findPreviewRow(t, updated, row.ID)
	if updatedRow.RowStatus != RowStatusAdjusted {
		t.Fatalf("expected adjusted row status, got %s", updatedRow.RowStatus)
	}
	if updatedRow.SelectedCategoryID != categoryID || updatedRow.SelectedAccountID != accountID {
		t.Fatalf("selection not persisted: category=%s account=%s", updatedRow.SelectedCategoryID, updatedRow.SelectedAccountID)
	}
	if updatedRow.Visibility != visibility {
		t.Fatalf("expected visibility %s, got %s", visibility, updatedRow.Visibility)
	}
	if len(updatedRow.SelectedTagIDs) != 2 || updatedRow.SelectedTagIDs[0] != "tag-breakfast" {
		t.Fatalf("selected tags not persisted: %#v", updatedRow.SelectedTagIDs)
	}
	if countRows(t, database, "transactions") != 0 {
		t.Fatalf("row adjustment must not write transactions")
	}
}

func TestUpdatePreviewRowRejectsInvalidRowAsAdjusted(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	service := NewService(NewRepository(database))

	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(),
		Filename:      "wechat-basic.csv",
		SourceType:    SourceTypeWechat,
		Content:       readImportFixture(t, "wechat-basic.csv"),
	})
	if err != nil {
		t.Fatalf("PreviewCSV returned error: %v", err)
	}

	invalidRow := batch.Rows[4]
	adjustedStatus := RowStatusAdjusted
	_, err = service.UpdatePreviewRow(context.Background(), UpdateRowCommand{
		LedgerContext: ownerLedgerContext(),
		BatchID:       batch.ID,
		RowID:         invalidRow.ID,
		Patch: UpdateRowRequest{
			RowStatus: &adjustedStatus,
		},
	})
	if err == nil {
		t.Fatalf("expected invalid row adjusted update to fail")
	}
}

func TestCommitPreviewBatchImportsRowsAndWritesAudit(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	service := NewService(NewRepository(database))

	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(),
		Filename:      "generic-basic.csv",
		SourceType:    SourceTypeGeneric,
		Content:       readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("PreviewCSV returned error: %v", err)
	}

	result, err := service.CommitPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("CommitPreviewBatch returned error: %v", err)
	}
	if result.Status != "committed" || result.ImportedRows != 3 || result.SkippedRows != 1 || result.FailedRows != 0 {
		t.Fatalf("unexpected commit result: %+v", result)
	}
	if len(result.GeneratedTransactionIDs) != 3 {
		t.Fatalf("expected 3 generated transactions, got %d", len(result.GeneratedTransactionIDs))
	}
	if countRows(t, database, "transactions") != 3 {
		t.Fatalf("expected 3 transactions after commit")
	}
	if countRows(t, database, "transaction_import_refs") != 3 {
		t.Fatalf("expected 3 import refs after commit")
	}
	if countWhere(t, database, "import_items", "row_status = 'imported'") != 3 {
		t.Fatalf("expected 3 imported rows")
	}
	if countWhere(t, database, "import_items", "row_status = 'skipped'") != 1 {
		t.Fatalf("expected 1 skipped row")
	}
	if countWhere(t, database, "import_batches", "status = 'committed' AND committed_at IS NOT NULL") != 1 {
		t.Fatalf("expected committed batch with committed_at")
	}
	if countWhere(t, database, "audit_logs", "action = 'import_commit' AND entity_type = 'import_batch'") != 1 {
		t.Fatalf("expected import commit audit log")
	}
}

func TestCommitPreviewBatchMakesRepeatedFileDuplicate(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	service := NewService(NewRepository(database))
	content := readImportFixture(t, "generic-basic.csv")

	first, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(),
		Filename:      "generic-basic.csv",
		SourceType:    SourceTypeGeneric,
		Content:       content,
	})
	if err != nil {
		t.Fatalf("first PreviewCSV returned error: %v", err)
	}
	if _, err := service.CommitPreviewBatch(context.Background(), ownerLedgerContext(), first.ID); err != nil {
		t.Fatalf("first CommitPreviewBatch returned error: %v", err)
	}

	second, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(),
		Filename:      "generic-basic.csv",
		SourceType:    SourceTypeGeneric,
		Content:       content,
	})
	if err != nil {
		t.Fatalf("second PreviewCSV returned error: %v", err)
	}
	if second.DuplicateRows != 3 || second.SkippedRows != 4 {
		t.Fatalf("expected 3 duplicate imported rows and 4 skipped rows, got duplicate=%d skipped=%d", second.DuplicateRows, second.SkippedRows)
	}
}

func TestCommitPreviewBatchRejectsInvalidRowWithoutPartialWrite(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	service := NewService(NewRepository(database))

	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(),
		Filename:      "wechat-basic.csv",
		SourceType:    SourceTypeWechat,
		Content:       readImportFixture(t, "wechat-basic.csv"),
	})
	if err != nil {
		t.Fatalf("PreviewCSV returned error: %v", err)
	}

	_, err = service.CommitPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err == nil {
		t.Fatalf("expected commit with invalid row to fail")
	}
	if countRows(t, database, "transactions") != 0 {
		t.Fatalf("invalid batch must not write partial transactions")
	}
	if countRows(t, database, "transaction_import_refs") != 0 {
		t.Fatalf("invalid batch must not write import refs")
	}
	if countWhere(t, database, "import_batches", "status = 'committed'") != 0 {
		t.Fatalf("invalid batch must not be marked committed")
	}
}

func TestImportRuleLifecycle(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	seedImportRuleMetadata(t, database)
	service := NewService(NewRepository(database))
	priority := 10

	created, err := service.CreateImportRule(context.Background(), ownerLedgerContext(), ImportRuleUpsertRequest{
		Name:      "咖啡规则",
		MatchType: "merchant_contains",
		Pattern:   "星巴克",
		Priority:  &priority,
		Result: ImportRuleResult{
			CategoryID: "cat-food",
			AccountID:  "account-cash",
			TagIDs:     []string{"tag-coffee"},
			Visibility: "private",
		},
	})
	if err != nil {
		t.Fatalf("CreateImportRule returned error: %v", err)
	}
	if created.Status != "active" || created.Priority != priority || created.Result.CategoryID != "cat-food" {
		t.Fatalf("unexpected created rule: %+v", created)
	}

	list, err := service.ListImportRules(context.Background(), ownerLedgerContext(), "active")
	if err != nil {
		t.Fatalf("ListImportRules returned error: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("expected created rule in active list, got %+v", list)
	}

	archived, err := service.ArchiveImportRule(context.Background(), ownerLedgerContext(), created.ID)
	if err != nil {
		t.Fatalf("ArchiveImportRule returned error: %v", err)
	}
	if archived.Status != "archived" || archived.ArchivedAt == "" {
		t.Fatalf("expected archived rule with archived_at, got %+v", archived)
	}

	restored, err := service.RestoreImportRule(context.Background(), ownerLedgerContext(), created.ID)
	if err != nil {
		t.Fatalf("RestoreImportRule returned error: %v", err)
	}
	if restored.Status != "active" || restored.ArchivedAt != "" {
		t.Fatalf("expected restored active rule without archived_at, got %+v", restored)
	}

	if countWhere(t, database, "audit_logs", "entity_type = 'import_rule'") != 3 {
		t.Fatalf("expected create/archive/restore audit logs")
	}
}

func TestImportRuleRejectsEditorAndArchivedMetadata(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	seedImportRuleMetadata(t, database)
	service := NewService(NewRepository(database))

	_, err := service.CreateImportRule(context.Background(), ledger.LedgerContext{
		UserID:   "editor-user",
		LedgerID: "ledger-one",
		Role:     ledger.RoleEditor,
	}, ImportRuleUpsertRequest{
		MatchType: "merchant_contains",
		Pattern:   "星巴克",
		Result:    ImportRuleResult{CategoryID: "cat-food"},
	})
	if err == nil {
		t.Fatalf("expected editor rule creation to be rejected")
	}

	_, err = service.CreateImportRule(context.Background(), ownerLedgerContext(), ImportRuleUpsertRequest{
		MatchType: "merchant_contains",
		Pattern:   "星巴克",
		Result:    ImportRuleResult{CategoryID: "cat-archived"},
	})
	if err == nil {
		t.Fatalf("expected archived category to be rejected")
	}
}

func TestPreviewCSVAppliesActiveImportRuleAsSuggestion(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	seedImportRuleMetadata(t, database)
	service := NewService(NewRepository(database))

	_, err := service.CreateImportRule(context.Background(), ownerLedgerContext(), ImportRuleUpsertRequest{
		Name:      "早餐店规则",
		MatchType: "merchant_contains",
		Pattern:   "早餐店",
		Result: ImportRuleResult{
			CategoryID: "cat-food",
			AccountID:  "account-cash",
			TagIDs:     []string{"tag-coffee"},
			Visibility: "private",
		},
	})
	if err != nil {
		t.Fatalf("CreateImportRule returned error: %v", err)
	}

	batch, err := service.PreviewCSV(context.Background(), PreviewFileRequest{
		LedgerContext: ownerLedgerContext(),
		Filename:      "generic-basic.csv",
		SourceType:    SourceTypeGeneric,
		Content:       readImportFixture(t, "generic-basic.csv"),
	})
	if err != nil {
		t.Fatalf("PreviewCSV returned error: %v", err)
	}

	row := findPreviewRowByNumber(t, batch, 1)
	if row.SuggestedCategoryID != "cat-food" || row.SuggestedAccountID != "account-cash" {
		t.Fatalf("expected rule suggestion on row 1, got category=%s account=%s", row.SuggestedCategoryID, row.SuggestedAccountID)
	}
	if len(row.SuggestedTagIDs) != 1 || row.SuggestedTagIDs[0] != "tag-coffee" {
		t.Fatalf("expected suggested tag, got %#v", row.SuggestedTagIDs)
	}
	if row.SuggestedRuleID == "" || row.SuggestionReason == "" {
		t.Fatalf("expected rule id and reason, got rule=%s reason=%s", row.SuggestedRuleID, row.SuggestionReason)
	}
	if row.SelectedCategoryID != "" || row.SelectedAccountID != "" || len(row.SelectedTagIDs) != 0 {
		t.Fatalf("rule suggestions must not overwrite selected fields: %+v", row)
	}

	stored, err := service.GetPreviewBatch(context.Background(), ownerLedgerContext(), batch.ID)
	if err != nil {
		t.Fatalf("GetPreviewBatch returned error: %v", err)
	}
	storedRow := findPreviewRowByNumber(t, stored, 1)
	if storedRow.SuggestedRuleID != row.SuggestedRuleID || storedRow.SuggestionReason == "" {
		t.Fatalf("expected persisted suggestion fields, got %+v", storedRow)
	}
}

func TestHandlePreviewAcceptsMultipartCSV(t *testing.T) {
	t.Parallel()

	database := openImporterTestDB(t)
	handler := NewHandler(NewService(NewRepository(database)))
	body, contentType := buildMultipartPreviewBody(t, "source_type", SourceTypeGeneric, "file", "generic-basic.csv", readImportFixture(t, "generic-basic.csv"))

	req := httptest.NewRequest(http.MethodPost, "/api/imports/preview", body)
	req.Header.Set("Content-Type", contentType)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "owner-user")
	ctx = ledger.ContextWithLedgerContext(ctx, ownerLedgerContext())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.HandlePreview(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var res response.SuccessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success response")
	}
}

func findPreviewRow(t *testing.T, batch *PreviewBatch, rowID string) PreviewRow {
	t.Helper()

	for _, row := range batch.Rows {
		if row.ID == rowID {
			return row
		}
	}
	t.Fatalf("row %s not found", rowID)
	return PreviewRow{}
}

func openImporterTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	database.SetMaxOpenConns(1)

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}
	if err := goose.Up(database, "."); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	_, err = database.Exec(`
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at)
		VALUES
			('owner-user', 'owner', 'Owner', 'hash', 'user', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('editor-user', 'editor', 'Editor', 'hash', 'user', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO ledgers (id, name, default_currency, created_at, updated_at)
		VALUES ('ledger-one', 'Ledger One', 'CNY', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
		VALUES
			('ledger-one', 'owner-user', 'owner', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('ledger-one', 'editor-user', 'editor', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	return database
}

func ownerLedgerContext() ledger.LedgerContext {
	return ledger.LedgerContext{
		UserID:     "owner-user",
		LedgerID:   "ledger-one",
		Role:       ledger.RoleOwner,
		IsExplicit: true,
	}
}

func readImportFixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "fixtures", "imports", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func findPreviewRowByNumber(t *testing.T, batch *PreviewBatch, rowNumber int) PreviewRow {
	t.Helper()

	for _, row := range batch.Rows {
		if row.RowNumber == rowNumber {
			return row
		}
	}
	t.Fatalf("row number %d not found", rowNumber)
	return PreviewRow{}
}

func seedImportRuleMetadata(t *testing.T, database *sql.DB) {
	t.Helper()

	_, err := database.Exec(`
		INSERT INTO categories (id, ledger_id, owner_user_id, name, type, color, is_archived, created_at, updated_at)
		VALUES
			('cat-food', 'ledger-one', 'owner-user', '餐饮', 'expense', '#22c55e', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
			('cat-archived', 'ledger-one', 'owner-user', '旧分类', 'expense', '#94a3b8', 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO accounts (id, ledger_id, owner_user_id, name, type, currency, initial_balance, is_archived, created_at, updated_at)
		VALUES ('account-cash', 'ledger-one', 'owner-user', '现金', 'cash', 'CNY', 0, 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO tags (id, ledger_id, owner_user_id, name, color, is_archived, created_at, updated_at)
		VALUES ('tag-coffee', 'ledger-one', 'owner-user', '咖啡', '#0f766e', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("seed import rule metadata: %v", err)
	}
}

func countRows(t *testing.T, database *sql.DB, table string) int {
	t.Helper()

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func countWhere(t *testing.T, database *sql.DB, table string, where string) int {
	t.Helper()

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM " + table + " WHERE " + where).Scan(&count); err != nil {
		t.Fatalf("count %s where %s: %v", table, where, err)
	}
	return count
}

func buildMultipartPreviewBody(t *testing.T, sourceField, sourceValue, fileField, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField(sourceField, sourceValue); err != nil {
		t.Fatalf("write source field: %v", err)
	}
	part, err := writer.CreateFormFile(fileField, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}
