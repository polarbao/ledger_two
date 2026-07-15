-- +goose Up
-- +goose StatementBegin
CREATE TABLE instance_admins (
    user_id TEXT PRIMARY KEY,
    granted_at TEXT NOT NULL,
    granted_by_user_id TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (granted_by_user_id) REFERENCES users(id)
);

CREATE TABLE instance_audit_logs (
    id TEXT PRIMARY KEY,
    actor_user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    before_json TEXT,
    after_json TEXT,
    created_at TEXT NOT NULL,
    FOREIGN KEY (actor_user_id) REFERENCES users(id)
);

CREATE INDEX idx_instance_audit_created
    ON instance_audit_logs(created_at, id);

INSERT INTO instance_admins (user_id, granted_at, granted_by_user_id)
SELECT lm.user_id, CURRENT_TIMESTAMP, NULL
FROM ledgers l
JOIN ledger_members lm
  ON lm.ledger_id = l.id
 AND lm.role = 'owner'
ORDER BY l.created_at ASC,
         l.id ASC,
         lm.created_at ASC,
         lm.user_id ASC
LIMIT 1;
-- +goose StatementEnd

-- +goose Down
-- Instance administration and its audit trail are retained on rollback.
SELECT 1;
