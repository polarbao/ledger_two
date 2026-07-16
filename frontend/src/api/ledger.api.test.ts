import { beforeEach, describe, expect, it, vi } from 'vitest';
import { formatLedgerETag, ledgerApi } from './ledger.api';
import { useLedgerStore } from '../stores/ledger.store';

describe('Task50.3A ledger lifecycle API', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    useLedgerStore.getState().clearActiveLedger();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ success: true, data: {} }),
    }));
  });

  it('lists ledgers with an explicit lifecycle filter outside ledger scope', async () => {
    await ledgerApi.listUserLedgers('archived');

    expect(fetch).toHaveBeenCalledWith('/api/ledgers?status=archived', expect.objectContaining({
      method: 'GET',
      headers: expect.not.objectContaining({ 'X-Ledger-Id': expect.anything() }),
    }));
  });

  it('reads lifecycle detail in the path ledger scope', async () => {
    await ledgerApi.getLedger('ledger-a');

    expect(fetch).toHaveBeenCalledWith('/api/ledgers/ledger-a', expect.objectContaining({
      method: 'GET',
      headers: expect.objectContaining({ 'X-Ledger-Id': 'ledger-a' }),
    }));
  });

  it('sends the frozen ETag for rename archive and restore mutations', async () => {
    expect(formatLedgerETag('ledger-a', 7)).toBe('"ledger:ledger-a:v7"');

    await ledgerApi.renameLedger('ledger-a', 7, { name: 'Renamed' });
    await ledgerApi.archiveLedger('ledger-a', 8, { acknowledge_unsettled_balance: true });
    await ledgerApi.restoreLedger('ledger-a', 9);

    const calls = vi.mocked(fetch).mock.calls;
    expect(calls[0]).toEqual([
      '/api/ledgers/ledger-a',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({ name: 'Renamed' }),
        headers: expect.objectContaining({
          'X-Ledger-Id': 'ledger-a',
          'if-match': '"ledger:ledger-a:v7"',
        }),
      }),
    ]);
    expect(calls[1]).toEqual([
      '/api/ledgers/ledger-a/archive',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ acknowledge_unsettled_balance: true }),
        headers: expect.objectContaining({ 'if-match': '"ledger:ledger-a:v8"' }),
      }),
    ]);
    expect(calls[2]).toEqual([
      '/api/ledgers/ledger-a/restore',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ 'if-match': '"ledger:ledger-a:v9"' }),
      }),
    ]);
  });

  it('loads archive preflight without writing', async () => {
    await ledgerApi.getArchivePreflight('ledger-a');

    expect(fetch).toHaveBeenCalledWith('/api/ledgers/ledger-a/archive-preflight', expect.objectContaining({
      method: 'GET',
      headers: expect.objectContaining({ 'X-Ledger-Id': 'ledger-a' }),
    }));
  });
});
