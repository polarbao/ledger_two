import { describe, expect, it } from 'vitest';
import { queryKeys, UNSELECTED_LEDGER_ID } from './queryKeys';

describe('queryKeys', () => {
  it('scopes ledger data by active ledger id', () => {
    const filter = { month: '2026-07', page: 1, page_size: 15 };

    expect(queryKeys.dashboard.month('ledger-a', '2026-07')).not.toEqual(
      queryKeys.dashboard.month('ledger-b', '2026-07')
    );
    expect(queryKeys.transactions.list('ledger-a', filter)).not.toEqual(
      queryKeys.transactions.list('ledger-b', filter)
    );
    expect(queryKeys.categories('ledger-a')).not.toEqual(queryKeys.categories('ledger-b'));
    expect(queryKeys.metadata.list('ledger-a', 'categories')).not.toEqual(
      queryKeys.metadata.list('ledger-b', 'categories')
    );
    expect(queryKeys.settlements.balance('ledger-a', '2026-07')).not.toEqual(
      queryKeys.settlements.balance('ledger-a')
    );
  });

  it('uses an explicit placeholder when no ledger is selected', () => {
    expect(queryKeys.dashboard.root(null)).toEqual(['dashboard', UNSELECTED_LEDGER_ID]);
    expect(queryKeys.transactions.root(undefined)).toEqual(['transactions', UNSELECTED_LEDGER_ID]);
  });

  it('keeps ledger list outside ledger scoped data', () => {
    expect(queryKeys.ledgers.all).toEqual(['ledgers']);
		expect(queryKeys.safety.diagnostics).toEqual(['safety', 'diagnostics']);
  });
});
