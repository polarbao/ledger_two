import { beforeEach, describe, expect, it, vi } from 'vitest';
import { importsApi } from './imports.api';
import { useLedgerStore } from '../stores/ledger.store';

describe('import batch mutation API', () => {
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

  it('defaults Task53.3 reclassification to a non-persistent dry-run', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({
        success: true,
        data: {
          dry_run: true,
          eligible_rows: 3,
          changed_rows: 1,
          unchanged_rows: 2,
          protected_manual_rows: 0,
          protected_bulk_rows: 0,
          conflict_rows: 0,
          summary: {
            auto_selected: 1,
            suggested: 0,
            fallback: 2,
            manual: 0,
            bulk: 0,
            conflict: 0,
            unresolved: 1,
          },
          changes: [],
        },
      }),
    } as Response);

    const result = await importsApi.reclassify('batch-a');

    expect(result.dry_run).toBe(true);
    expect(fetch).toHaveBeenCalledWith('/api/imports/batch-a/reclassify', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ dry_run: true }),
      headers: expect.objectContaining({ 'X-Ledger-Id': 'ledger-a' }),
    }));
  });
});
