-- +goose Up
-- +goose StatementBegin
UPDATE ledger_members
SET role = 'owner',
    updated_at = CURRENT_TIMESTAMP
WHERE role <> 'owner'
  AND NOT EXISTS (
      SELECT 1
      FROM ledger_members existing_owner
      WHERE existing_owner.ledger_id = ledger_members.ledger_id
        AND existing_owner.role = 'owner'
  )
  AND user_id = (
      SELECT candidate.user_id
      FROM ledger_members candidate
      WHERE candidate.ledger_id = ledger_members.ledger_id
      ORDER BY candidate.created_at ASC, candidate.user_id ASC
      LIMIT 1
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
