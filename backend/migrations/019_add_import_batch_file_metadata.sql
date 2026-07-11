-- +goose Up
-- +goose StatementBegin
ALTER TABLE import_batches ADD COLUMN file_format TEXT NOT NULL DEFAULT 'csv';
ALTER TABLE import_batches ADD COLUMN parser_metadata_json TEXT NOT NULL DEFAULT '{}';
-- +goose StatementEnd

-- +goose Down
SELECT 1;
