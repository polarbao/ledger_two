-- +goose Up
CREATE TABLE transaction_defaults (
    ledger_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'expense',
    category_id TEXT,
    account_id TEXT,
    payer_user_id TEXT,
    visibility TEXT NOT NULL DEFAULT 'partner_readable',
    split_method TEXT NOT NULL DEFAULT 'equal',
    tag_names TEXT,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (ledger_id, user_id),
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (category_id) REFERENCES categories(id),
    FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_transaction_defaults_ledger_user
    ON transaction_defaults (ledger_id, user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_transaction_defaults_ledger_user;
DROP TABLE IF EXISTS transaction_defaults;
