-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    avatar_url TEXT,
    role TEXT NOT NULL DEFAULT 'user',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE ledgers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    default_currency TEXT NOT NULL DEFAULT 'CNY',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'CNY',
    initial_balance INTEGER NOT NULL DEFAULT 0,
    is_archived INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (owner_user_id) REFERENCES users(id)
);

CREATE TABLE categories (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    owner_user_id TEXT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    icon TEXT,
    color TEXT,
    parent_id TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_system INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (owner_user_id) REFERENCES users(id),
    FOREIGN KEY (parent_id) REFERENCES categories(id)
);

CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    name TEXT NOT NULL,
    owner_user_id TEXT,
    color TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (owner_user_id) REFERENCES users(id)
);

CREATE TABLE transactions (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL DEFAULT 'CNY',
    occurred_at TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    created_by_user_id TEXT NOT NULL,
    payer_user_id TEXT,
    account_id TEXT,
    category_id TEXT,
    visibility TEXT NOT NULL DEFAULT 'private',
    split_method TEXT,
    note TEXT,
    status TEXT NOT NULL DEFAULT 'normal',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (owner_user_id) REFERENCES users(id),
    FOREIGN KEY (created_by_user_id) REFERENCES users(id),
    FOREIGN KEY (payer_user_id) REFERENCES users(id),
    FOREIGN KEY (account_id) REFERENCES accounts(id),
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE TABLE transaction_splits (
    id TEXT PRIMARY KEY,
    transaction_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    share_amount INTEGER NOT NULL,
    share_ratio INTEGER,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE transaction_tags (
    transaction_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (transaction_id, tag_id),
    FOREIGN KEY (transaction_id) REFERENCES transactions(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE settlements (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    from_user_id TEXT NOT NULL,
    to_user_id TEXT NOT NULL,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL DEFAULT 'CNY',
    occurred_at TEXT NOT NULL,
    note TEXT,
    created_by_user_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (from_user_id) REFERENCES users(id),
    FOREIGN KEY (to_user_id) REFERENCES users(id),
    FOREIGN KEY (created_by_user_id) REFERENCES users(id)
);

CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    actor_user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    before_json TEXT,
    after_json TEXT,
    created_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (actor_user_id) REFERENCES users(id)
);

CREATE TABLE app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_transactions_ledger_month ON transactions(ledger_id, occurred_at);
CREATE INDEX idx_transactions_payer ON transactions(ledger_id, payer_user_id, occurred_at);
CREATE INDEX idx_transactions_category ON transactions(ledger_id, category_id, occurred_at);
CREATE INDEX idx_transactions_type ON transactions(ledger_id, type, occurred_at);
CREATE INDEX idx_transactions_visibility ON transactions(ledger_id, visibility);
CREATE INDEX idx_transactions_status ON transactions(ledger_id, status);
CREATE INDEX idx_transactions_owner ON transactions(ledger_id, owner_user_id);
CREATE INDEX idx_transactions_created_by ON transactions(ledger_id, created_by_user_id);
CREATE INDEX idx_splits_transaction ON transaction_splits(transaction_id);
CREATE INDEX idx_splits_user ON transaction_splits(user_id);
CREATE INDEX idx_settlements_users ON settlements(ledger_id, from_user_id, to_user_id, occurred_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE app_settings;
DROP TABLE audit_logs;
DROP TABLE settlements;
DROP TABLE transaction_tags;
DROP TABLE transaction_splits;
DROP TABLE transactions;
DROP TABLE tags;
DROP TABLE categories;
DROP TABLE accounts;
DROP TABLE ledgers;
DROP TABLE users;
-- +goose StatementEnd
