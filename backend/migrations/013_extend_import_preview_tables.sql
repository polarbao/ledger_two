-- +goose Up
-- +goose StatementBegin
ALTER TABLE import_batches ADD COLUMN source_type TEXT NOT NULL DEFAULT 'generic';
ALTER TABLE import_batches ADD COLUMN file_sha256 TEXT NOT NULL DEFAULT '';
ALTER TABLE import_batches ADD COLUMN total_rows INTEGER NOT NULL DEFAULT 0;
ALTER TABLE import_batches ADD COLUMN new_rows INTEGER NOT NULL DEFAULT 0;
ALTER TABLE import_batches ADD COLUMN duplicate_rows INTEGER NOT NULL DEFAULT 0;
ALTER TABLE import_batches ADD COLUMN suspicious_rows INTEGER NOT NULL DEFAULT 0;
ALTER TABLE import_batches ADD COLUMN invalid_rows INTEGER NOT NULL DEFAULT 0;
ALTER TABLE import_batches ADD COLUMN imported_rows INTEGER NOT NULL DEFAULT 0;
ALTER TABLE import_batches ADD COLUMN skipped_rows INTEGER NOT NULL DEFAULT 0;
ALTER TABLE import_batches ADD COLUMN updated_at TEXT;

ALTER TABLE import_items ADD COLUMN row_number INTEGER NOT NULL DEFAULT 0;
ALTER TABLE import_items ADD COLUMN source_type TEXT NOT NULL DEFAULT 'generic';
ALTER TABLE import_items ADD COLUMN external_order_id TEXT;
ALTER TABLE import_items ADD COLUMN occurred_at TEXT;
ALTER TABLE import_items ADD COLUMN title TEXT;
ALTER TABLE import_items ADD COLUMN merchant TEXT;
ALTER TABLE import_items ADD COLUMN description TEXT;
ALTER TABLE import_items ADD COLUMN amount_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE import_items ADD COLUMN direction TEXT NOT NULL DEFAULT 'unknown';
ALTER TABLE import_items ADD COLUMN target_transaction_type TEXT NOT NULL DEFAULT 'skipped';
ALTER TABLE import_items ADD COLUMN duplicate_status TEXT NOT NULL DEFAULT 'new';
ALTER TABLE import_items ADD COLUMN row_status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE import_items ADD COLUMN normalized_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE import_items ADD COLUMN user_adjustment_json TEXT;
ALTER TABLE import_items ADD COLUMN error_code TEXT;
ALTER TABLE import_items ADD COLUMN error_message TEXT;
ALTER TABLE import_items ADD COLUMN generated_transaction_id TEXT;

CREATE INDEX idx_import_batches_ledger_status ON import_batches(ledger_id, status);
CREATE INDEX idx_import_items_batch_row_number ON import_items(batch_id, row_number);
CREATE INDEX idx_import_items_batch_duplicate_status ON import_items(batch_id, duplicate_status);
CREATE INDEX idx_import_items_batch_row_status ON import_items(batch_id, row_status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_import_items_batch_row_status;
DROP INDEX IF EXISTS idx_import_items_batch_duplicate_status;
DROP INDEX IF EXISTS idx_import_items_batch_row_number;
DROP INDEX IF EXISTS idx_import_batches_ledger_status;
-- SQLite cannot drop columns without rebuilding tables; keep additive preview columns on down.
-- +goose StatementEnd
