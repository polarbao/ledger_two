import { describe, expect, it } from 'vitest';
import type { TransactionDraft } from './draft.store';
import {
  migrateLegacyDrafts,
  selectLedgerDrafts,
} from './draftLedgerModel';

function draft(id: string, ledgerId?: string): TransactionDraft {
  return {
    id,
    ledgerId: ledgerId ?? null,
    formValues: {
      type: 'expense',
      amount: '12.34',
      payer_user_id: 'user-a',
      split_method: 'equal',
      occurred_at: '2026-07-16',
      visibility: 'partner_readable',
    },
    createdAt: '2026-07-16T00:00:00Z',
  };
}

describe('Task50.4 ledger-scoped offline drafts', () => {
  it('shows only drafts that belong to the active ledger', () => {
    expect(selectLedgerDrafts([
      draft('draft-a', 'ledger-a'),
      draft('draft-b', 'ledger-b'),
      draft('draft-legacy'),
    ], 'ledger-a').map((item) => item.id)).toEqual(['draft-a']);
  });

  it('assigns legacy drafts to the last persisted ledger during migration', () => {
    expect(migrateLegacyDrafts([
      draft('draft-a', 'ledger-a'),
      draft('draft-legacy'),
    ], 'ledger-b')).toEqual([
      expect.objectContaining({ id: 'draft-a', ledgerId: 'ledger-a' }),
      expect.objectContaining({ id: 'draft-legacy', ledgerId: 'ledger-b' }),
    ]);
  });
});
