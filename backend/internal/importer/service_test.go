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

func countRows(t *testing.T, database *sql.DB, table string) int {
	t.Helper()

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
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
