-- +goose Up
-- +goose StatementBegin
CREATE TABLE ledger_members (
    ledger_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'editor',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (ledger_id, user_id),
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- 为当前 Demo 系统中已存在的所有用户和默认账本创建绑定关系
INSERT INTO ledger_members (ledger_id, user_id, role, created_at, updated_at)
SELECT l.id, u.id, 'editor', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM ledgers l
CROSS JOIN users u;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE ledger_members;
-- +goose StatementEnd
