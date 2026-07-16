import { beforeEach, describe, expect, it, vi } from 'vitest';
import { importsApi } from './imports.api';
import { useLedgerStore } from '../stores/ledger.store';

describe('Task50.3A import batch discard API', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    useLedgerStore.getState().setActiveLedger('ledger-a', 'owner');
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({
        success: true,
        data: { batch_id: 'batch-a', status: 'expired', discard_reason: 'user_requested' },
      }),
    }));
  });

  it('explicitly discards a ready batch in the active ledger', async () => {
    const result = await importsApi.discard('batch-a');

    expect(result).toEqual({ batch_id: 'batch-a', status: 'expired', discard_reason: 'user_requested' });
    expect(fetch).toHaveBeenCalledWith('/api/imports/batch-a/discard', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ reason: 'user_requested' }),
      headers: expect.objectContaining({ 'X-Ledger-Id': 'ledger-a' }),
    }));
  });
});
