-- +goose Up
-- +goose StatementBegin
ALTER TABLE import_batches ADD COLUMN failed_rows INTEGER NOT NULL DEFAULT 0;
ALTER TABLE import_batches ADD COLUMN committed_at TEXT;
ALTER TABLE import_batches ADD COLUMN expires_at TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite cannot drop columns without rebuilding tables; keep additive commit columns on down.
-- +goose StatementEnd
