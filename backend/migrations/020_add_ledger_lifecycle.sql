-- +goose Up
-- +goose StatementBegin
ALTER TABLE ledgers ADD COLUMN status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'archived'));
ALTER TABLE ledgers ADD COLUMN archived_at TEXT;
ALTER TABLE ledgers ADD COLUMN archived_by_user_id TEXT REFERENCES users(id);
ALTER TABLE ledgers ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE audit_logs ADD COLUMN actor_role TEXT;

CREATE INDEX idx_ledgers_status_created
    ON ledgers(status, created_at, id);
CREATE INDEX idx_ledger_members_user_ledger
    ON ledger_members(user_id, ledger_id);
CREATE UNIQUE INDEX idx_ledger_members_one_owner
    ON ledger_members(ledger_id)
    WHERE role = 'owner';
CREATE INDEX idx_audit_logs_ledger_created
    ON audit_logs(ledger_id, created_at, id);

CREATE TRIGGER trg_ledger_members_max_two_before_insert
BEFORE INSERT ON ledger_members
FOR EACH ROW
WHEN (
    SELECT COUNT(*)
    FROM ledger_members
    WHERE ledger_id = NEW.ledger_id
) >= 2
BEGIN
    SELECT RAISE(ABORT, 'ledger member limit reached');
END;
-- +goose StatementEnd

-- +goose Down
-- Production rollback restores the complete pre-migration backup. SQLite
-- cannot safely drop these columns without rebuilding referenced tables.
SELECT 1;
