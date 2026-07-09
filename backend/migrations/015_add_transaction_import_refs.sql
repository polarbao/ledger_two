-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_import_refs (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    transaction_id TEXT NOT NULL,
    import_batch_id TEXT NOT NULL,
    import_row_id TEXT NOT NULL,
    import_hash TEXT NOT NULL,
    external_order_id TEXT,
    source_type TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id) ON DELETE CASCADE,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    FOREIGN KEY (import_batch_id) REFERENCES import_batches(id) ON DELETE CASCADE,
    FOREIGN KEY (import_row_id) REFERENCES import_items(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_transaction_import_refs_hash ON transaction_import_refs(ledger_id, import_hash);
CREATE INDEX idx_transaction_import_refs_tx ON transaction_import_refs(transaction_id);
CREATE INDEX idx_transaction_import_refs_batch ON transaction_import_refs(import_batch_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transaction_import_refs_batch;
DROP INDEX IF EXISTS idx_transaction_import_refs_tx;
DROP INDEX IF EXISTS idx_transaction_import_refs_hash;
DROP TABLE IF EXISTS transaction_import_refs;
-- +goose StatementEnd
