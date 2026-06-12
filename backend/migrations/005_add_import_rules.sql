-- +goose Up
-- +goose StatementBegin
CREATE TABLE import_rules (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    keyword TEXT NOT NULL,
    category_id TEXT,
    tag_names TEXT, -- 逗号分隔的标签字符串，如 "咖啡,外卖"
    account_id TEXT,
    created_by_user_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL,
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_import_rules_ledger ON import_rules(ledger_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_import_rules_ledger;
DROP TABLE IF EXISTS import_rules;
-- +goose StatementEnd
