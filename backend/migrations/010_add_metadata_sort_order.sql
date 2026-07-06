-- +goose Up
ALTER TABLE tags ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_tags_ledger_sort ON tags(ledger_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_accounts_ledger_sort ON accounts(ledger_id, sort_order);

-- +goose Down
DROP INDEX IF EXISTS idx_accounts_ledger_sort;
DROP INDEX IF EXISTS idx_tags_ledger_sort;

