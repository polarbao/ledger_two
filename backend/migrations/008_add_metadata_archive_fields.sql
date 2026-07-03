-- +goose Up
-- +goose StatementBegin
ALTER TABLE categories ADD COLUMN is_archived INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tags ADD COLUMN is_archived INTEGER NOT NULL DEFAULT 0;

CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_ledger_type_name ON categories(ledger_id, type, name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_ledger_name ON tags(ledger_id, name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_ledger_name ON accounts(ledger_id, name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_accounts_ledger_name;
DROP INDEX IF EXISTS idx_tags_ledger_name;
DROP INDEX IF EXISTS idx_categories_ledger_type_name;
-- SQLite does not support DROP COLUMN in older versions; keep added archive columns on down migration.
-- +goose StatementEnd
