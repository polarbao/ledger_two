-- +goose Up
-- +goose StatementBegin
CREATE TABLE import_batches (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    created_by_user_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'completed',
    created_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE import_items (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL,
    transaction_id TEXT,
    import_hash TEXT NOT NULL,
    status TEXT NOT NULL, -- 'imported' | 'skipped'
    created_at TEXT NOT NULL,
    FOREIGN KEY (batch_id) REFERENCES import_batches(id) ON DELETE CASCADE,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE SET NULL
);

CREATE INDEX idx_import_items_hash ON import_items(import_hash);
CREATE INDEX idx_import_items_batch ON import_items(batch_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_import_items_batch;
DROP INDEX IF EXISTS idx_import_items_hash;
DROP TABLE IF EXISTS import_items;
DROP TABLE IF EXISTS import_batches;
-- +goose StatementEnd
