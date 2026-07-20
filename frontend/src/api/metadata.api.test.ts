import { beforeEach, describe, expect, it, vi } from 'vitest';
import { metadataApi } from './metadata.api';
import { useLedgerStore } from '../stores/ledger.store';

describe('Task53.4C metadata API', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    useLedgerStore.getState().setActiveLedger('ledger-a', 'owner');
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({
        success: true,
        data: {
          archived_id: 'category-old',
          fallback_replaced: true,
          transferred_system_key: 'expense_other',
          replacement_category_id: 'category-new',
        },
      }),
    }));
  });

  it('sends the explicit fallback replacement on the existing archive route', async () => {
    const result = await metadataApi.archive('categories', 'category-old', {
      replacement_category_id: 'category-new',
    });

    expect(result.fallback_replaced).toBe(true);
    expect(fetch).toHaveBeenCalledWith('/api/metadata/categories/category-old/archive', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ replacement_category_id: 'category-new' }),
      headers: expect.objectContaining({ 'X-Ledger-Id': 'ledger-a' }),
    }));
  });
});
