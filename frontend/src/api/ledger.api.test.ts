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

  it('forwards TanStack Query cancellation to ledger list requests', async () => {
    const controller = new AbortController();

    await ledgerApi.listUserLedgers('active', controller.signal);

    expect(fetch).toHaveBeenCalledWith('/api/ledgers?status=active', expect.objectContaining({
      signal: controller.signal,
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

describe('Task50.3B ledger member API', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    useLedgerStore.getState().clearActiveLedger();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ success: true, data: {} }),
    }));
  });

  it('uses PATCH and the ledger ETag for every member mutation', async () => {
    await ledgerApi.addMember('ledger-a', 3, {
      username: 'partner',
      role: 'editor',
      acknowledge_history_visibility: true,
    });
    await ledgerApi.updateMemberRole('ledger-a', 4, 'user-b', { role: 'viewer' });
    await ledgerApi.removeMember('ledger-a', 5, 'user-b');
    await ledgerApi.transferOwner('ledger-a', 6, 'user-b', {
      acknowledge_permission_change: true,
    });
    await ledgerApi.leaveLedger('ledger-a', 7);

    const calls = vi.mocked(fetch).mock.calls;
    expect(calls[0]).toEqual([
      '/api/ledgers/ledger-a/members',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          username: 'partner',
          role: 'editor',
          acknowledge_history_visibility: true,
        }),
        headers: expect.objectContaining({ 'if-match': '"ledger:ledger-a:v3"' }),
      }),
    ]);
    expect(calls[1]).toEqual([
      '/api/ledgers/ledger-a/members/user-b',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({ role: 'viewer' }),
        headers: expect.objectContaining({ 'if-match': '"ledger:ledger-a:v4"' }),
      }),
    ]);
    expect(calls[2]).toEqual([
      '/api/ledgers/ledger-a/members/user-b',
      expect.objectContaining({
        method: 'DELETE',
        headers: expect.objectContaining({ 'if-match': '"ledger:ledger-a:v5"' }),
      }),
    ]);
    expect(calls[3]).toEqual([
      '/api/ledgers/ledger-a/members/user-b/transfer-owner',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ acknowledge_permission_change: true }),
        headers: expect.objectContaining({ 'if-match': '"ledger:ledger-a:v6"' }),
      }),
    ]);
    expect(calls[4]).toEqual([
      '/api/ledgers/ledger-a/leave',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ 'if-match': '"ledger:ledger-a:v7"' }),
      }),
    ]);
  });
});
