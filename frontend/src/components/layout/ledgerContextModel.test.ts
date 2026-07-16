import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';
import type { LedgerWithRole } from '../../api/ledger.api';
import {
  migratePersistedLedgerState,
  selectActiveLedger,
  selectPersistedLedgerState,
  switchActiveLedgerContext,
} from './ledgerContextModel';

function ledger(
  id: string,
  overrides: Partial<LedgerWithRole> = {},
): LedgerWithRole {
  return {
    id,
    name: id,
    role: 'owner',
    status: 'active',
    version: 1,
    member_count: 1,
    archived_at: null,
    archived_by_user_id: null,
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
    ...overrides,
  };
}

describe('Task50.4 active ledger selection', () => {
  it('keeps a persisted preference when it is still an accessible active ledger', () => {
    const result = selectActiveLedger(
      [ledger('ledger-a'), ledger('ledger-b')],
      'ledger-a',
      { 'ledger-b': 200 },
    );

    expect(result).toEqual({
      ledger: expect.objectContaining({ id: 'ledger-a' }),
      reason: 'preferred-valid',
      previousLedgerId: 'ledger-a',
    });
  });

  it('falls back to the most recently used accessible active ledger', () => {
    const result = selectActiveLedger(
      [
        ledger('ledger-a', { status: 'archived' }),
        ledger('ledger-b'),
        ledger('ledger-c'),
      ],
      'ledger-a',
      { 'ledger-b': 100, 'ledger-c': 300 },
    );

    expect(result).toEqual({
      ledger: expect.objectContaining({ id: 'ledger-c' }),
      reason: 'preferred-unavailable',
      previousLedgerId: 'ledger-a',
    });
  });

  it('returns an explicit no-active state instead of selecting an archived ledger', () => {
    const result = selectActiveLedger(
      [ledger('ledger-a', { status: 'archived' })],
      'ledger-a',
      { 'ledger-a': 500 },
    );

    expect(result).toEqual({
      ledger: null,
      reason: 'no-active',
      previousLedgerId: 'ledger-a',
    });
  });

  it('persists only the ledger preference and recent-use timestamps', () => {
    expect(selectPersistedLedgerState({
      activeLedgerId: 'ledger-a',
      activeRole: 'owner',
      contextStatus: 'active',
      recentLedgerUsedAt: { 'ledger-a': 100 },
    })).toEqual({
      activeLedgerId: 'ledger-a',
      recentLedgerUsedAt: { 'ledger-a': 100 },
    });
  });

  it('drops legacy role and status snapshots while migrating persisted state', () => {
    expect(migratePersistedLedgerState({
      activeLedgerId: 'ledger-a',
      activeRole: 'owner',
      contextStatus: 'active',
      recentLedgerUsedAt: { 'ledger-a': 100, 'ledger-b': Number.NaN },
    })).toEqual({
      activeLedgerId: 'ledger-a',
      recentLedgerUsedAt: { 'ledger-a': 100 },
    });
  });
});

describe('Task50.4 ledger query isolation', () => {
  it('cancels the old ledger request before committing the next context', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const events: string[] = [];
    const inFlight = queryClient.fetchQuery({
      queryKey: ['dashboard', 'ledger-a'],
      queryFn: ({ signal }) => new Promise<string>((_resolve, reject) => {
        signal.addEventListener('abort', () => {
          events.push('abort-ledger-a');
          reject(new DOMException('aborted', 'AbortError'));
        });
      }),
    }).catch(() => undefined);

    await switchActiveLedgerContext({
      queryClient,
      currentLedgerId: 'ledger-a',
      nextLedger: ledger('ledger-b'),
      commit: (nextLedger) => {
        events.push(`commit-${nextLedger.id}`);
      },
    });
    await inFlight;

    expect(events).toEqual(['abort-ledger-a', 'commit-ledger-b']);
  });

  it('keeps a late ledger A response outside the ledger B cache', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    let resolveLedgerA: ((value: string) => void) | undefined;
    const lateLedgerA = queryClient.fetchQuery({
      queryKey: ['transactions', 'ledger-a'],
      queryFn: () => new Promise<string>((resolve) => {
        resolveLedgerA = resolve;
      }),
    }).catch(() => undefined);
    queryClient.setQueryData(['transactions', 'ledger-b'], 'ledger-b-data');

    await switchActiveLedgerContext({
      queryClient,
      currentLedgerId: 'ledger-a',
      nextLedger: ledger('ledger-b'),
      commit: () => undefined,
    });
    resolveLedgerA?.('late-ledger-a-data');
    await lateLedgerA;

    expect(queryClient.getQueryData(['transactions', 'ledger-b'])).toBe('ledger-b-data');
  });
});
