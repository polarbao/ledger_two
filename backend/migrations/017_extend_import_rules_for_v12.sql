-- +goose Up
-- +goose StatementBegin
ALTER TABLE import_rules ADD COLUMN name TEXT;
ALTER TABLE import_rules ADD COLUMN match_type TEXT;
ALTER TABLE import_rules ADD COLUMN pattern TEXT;
ALTER TABLE import_rules ADD COLUMN amount_min_cents INTEGER;
ALTER TABLE import_rules ADD COLUMN amount_max_cents INTEGER;
ALTER TABLE import_rules ADD COLUMN priority INTEGER NOT NULL DEFAULT 100;
ALTER TABLE import_rules ADD COLUMN result_json TEXT;
ALTER TABLE import_rules ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE import_rules ADD COLUMN archived_at TEXT;

UPDATE import_rules
SET name = COALESCE(NULLIF(keyword, ''), '导入规则'),
    match_type = 'merchant_contains',
    pattern = keyword,
    result_json = json_object(
        'category_id', category_id,
        'account_id', account_id,
        'tag_ids', CASE
            WHEN tag_names IS NULL OR tag_names = '' THEN json('[]')
            ELSE json('[' || '"' || replace(tag_names, ',', '","') || '"' || ']')
        END,
        'visibility', 'private'
    )
WHERE match_type IS NULL;

CREATE INDEX idx_import_rules_ledger_status_priority ON import_rules(ledger_id, status, priority);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_import_rules_ledger_status_priority;
-- SQLite cannot drop columns without rebuilding tables; keep additive v1.2 rule columns on down.
-- +goose StatementEnd
