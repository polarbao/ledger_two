-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN attachment_paths TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions DROP COLUMN attachment_paths;
-- +goose StatementEnd
