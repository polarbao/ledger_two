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

  it('previews and applies the basic profile with explicit conflict resolutions', async () => {
    await metadataApi.previewDefaultProfile('basic_cn_v1');
    await metadataApi.applyDefaultProfile('basic_cn_v1', [{
      system_key: 'expense_food',
      action: 'reuse',
      existing_id: 'category-food',
    }]);

    const calls = vi.mocked(fetch).mock.calls;
    expect(calls[0]).toEqual([
      '/api/metadata/default-profile/preview',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ profile: 'basic_cn_v1' }),
      }),
    ]);
    expect(calls[1]).toEqual([
      '/api/metadata/default-profile/apply',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          profile: 'basic_cn_v1',
          resolutions: [{
            system_key: 'expense_food',
            action: 'reuse',
            existing_id: 'category-food',
          }],
        }),
      }),
    ]);
  });
});
