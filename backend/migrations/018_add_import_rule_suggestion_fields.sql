-- +goose Up
-- +goose StatementBegin
ALTER TABLE import_items ADD COLUMN suggested_rule_id TEXT;
ALTER TABLE import_items ADD COLUMN suggestion_reason TEXT;

CREATE INDEX idx_import_items_suggested_rule ON import_items(suggested_rule_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_import_items_suggested_rule;
-- SQLite cannot drop columns without rebuilding tables; keep additive suggestion columns on down.
-- +goose StatementEnd
