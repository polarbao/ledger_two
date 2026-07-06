-- +goose Up
ALTER TABLE transaction_templates ADD COLUMN is_archived INTEGER NOT NULL DEFAULT 0;
ALTER TABLE transaction_templates ADD COLUMN archived_at TEXT;

CREATE INDEX IF NOT EXISTS idx_templates_ledger_archived
    ON transaction_templates (ledger_id, is_archived, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_templates_ledger_archived;
ALTER TABLE transaction_templates DROP COLUMN archived_at;
ALTER TABLE transaction_templates DROP COLUMN is_archived;
