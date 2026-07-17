-- +goose Up
-- +goose StatementBegin
ALTER TABLE ledgers ADD COLUMN metadata_profile_version INTEGER NOT NULL DEFAULT 0;

ALTER TABLE categories ADD COLUMN system_key TEXT;
ALTER TABLE tags ADD COLUMN system_key TEXT;

ALTER TABLE import_rules ADD COLUMN origin TEXT NOT NULL DEFAULT 'manual';
ALTER TABLE import_rules ADD COLUMN source_type TEXT;
ALTER TABLE import_rules ADD COLUMN apply_mode TEXT NOT NULL DEFAULT 'suggest';
ALTER TABLE import_rules ADD COLUMN confidence TEXT NOT NULL DEFAULT 'high';

ALTER TABLE import_items ADD COLUMN classification_status TEXT NOT NULL DEFAULT 'unresolved';
ALTER TABLE import_items ADD COLUMN classification_confidence TEXT NOT NULL DEFAULT 'none';
ALTER TABLE import_items ADD COLUMN classification_source TEXT;
ALTER TABLE import_items ADD COLUMN classification_reason_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE import_items ADD COLUMN matched_rule_ids_json TEXT NOT NULL DEFAULT '[]';

CREATE UNIQUE INDEX idx_categories_ledger_system_key
    ON categories(ledger_id, system_key)
    WHERE system_key IS NOT NULL;

CREATE UNIQUE INDEX idx_tags_ledger_system_key
    ON tags(ledger_id, system_key)
    WHERE system_key IS NOT NULL;

CREATE INDEX idx_import_rules_ledger_origin_status_priority
    ON import_rules(ledger_id, origin, status, priority);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_import_rules_ledger_origin_status_priority;
DROP INDEX IF EXISTS idx_tags_ledger_system_key;
DROP INDEX IF EXISTS idx_categories_ledger_system_key;
-- Task53 production rollback restores the paired pre-migration backup. The
-- additive columns remain because rebuilding referenced SQLite tables is unsafe.
-- +goose StatementEnd
