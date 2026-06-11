-- +goose Up
-- +goose StatementBegin
CREATE TABLE recurring_rules (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    title TEXT,
    amount_cents INTEGER,
    category_id TEXT,
    payer_user_id TEXT,
    split_method TEXT,
    tag_names TEXT,
    note TEXT,
    frequency TEXT NOT NULL,
    next_due_date TEXT NOT NULL,
    created_by_user_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE recurring_reminders (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    rule_id TEXT NOT NULL,
    due_date TEXT NOT NULL,
    status TEXT NOT NULL,
    transaction_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id) ON DELETE CASCADE,
    FOREIGN KEY (rule_id) REFERENCES recurring_rules(id) ON DELETE CASCADE,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE SET NULL
);

CREATE INDEX idx_recurring_rules_ledger ON recurring_rules(ledger_id);
CREATE INDEX idx_recurring_reminders_ledger ON recurring_reminders(ledger_id);
CREATE INDEX idx_recurring_reminders_status ON recurring_reminders(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recurring_reminders;
DROP TABLE IF EXISTS recurring_rules;
-- +goose StatementEnd
