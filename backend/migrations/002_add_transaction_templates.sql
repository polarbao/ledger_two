-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_templates (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    title TEXT,
    amount_cents INTEGER,
    category_id TEXT,
    account_id TEXT,
    payer_user_id TEXT,
    split_method TEXT,
    tag_names TEXT,
    note TEXT,
    created_by_user_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_templates_ledger ON transaction_templates(ledger_id);
CREATE INDEX idx_templates_created_by ON transaction_templates(created_by_user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_templates;
-- +goose StatementEnd
