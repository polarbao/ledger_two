-- +goose Up
-- +goose StatementBegin
ALTER TABLE import_items ADD COLUMN suggested_category_id TEXT;
ALTER TABLE import_items ADD COLUMN suggested_account_id TEXT;
ALTER TABLE import_items ADD COLUMN suggested_tag_ids_json TEXT;
ALTER TABLE import_items ADD COLUMN selected_category_id TEXT;
ALTER TABLE import_items ADD COLUMN selected_account_id TEXT;
ALTER TABLE import_items ADD COLUMN selected_tag_ids_json TEXT;
ALTER TABLE import_items ADD COLUMN visibility TEXT NOT NULL DEFAULT 'private';

CREATE INDEX idx_import_items_selected_category ON import_items(selected_category_id);
CREATE INDEX idx_import_items_selected_account ON import_items(selected_account_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_import_items_selected_account;
DROP INDEX IF EXISTS idx_import_items_selected_category;
-- SQLite cannot drop columns without rebuilding tables; keep additive row selection columns on down.
-- +goose StatementEnd
