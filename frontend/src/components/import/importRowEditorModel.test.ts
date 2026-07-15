import { describe, expect, it } from 'vitest';
import type { ImportPreviewRow } from '../../types/imports';
import type { MetadataItem } from '../../types/metadata';
import { buildImportRowUpdatePayload, createImportRowEditorDraft } from './importRowEditorModel';

const metadata = (id: string, isArchived = false): MetadataItem => ({
  id,
  ledger_id: 'ledger-1',
  name: id,
  sort_order: 0,
  usage_count: 0,
  is_archived: isArchived,
});

const row = (overrides: Partial<ImportPreviewRow> = {}): ImportPreviewRow => ({
  id: 'row-1',
  batch_id: 'batch-1',
  row_number: 1,
  title: '午餐',
  merchant: '餐厅',
  amount_cents: 3200,
  direction: 'expense',
  target_transaction_type: 'expense',
  duplicate_status: 'new',
  row_status: 'pending',
  ...overrides,
});

describe('import row editor model', () => {
  it('prefills only active rule suggestions', () => {
    const draft = createImportRowEditorDraft(
      row({
        suggested_category_id: 'category-active',
        suggested_account_id: 'account-archived',
        suggested_tag_ids: ['tag-active', 'tag-archived'],
      }),
      [metadata('category-active')],
      [metadata('account-archived', true)],
      [metadata('tag-active'), metadata('tag-archived', true)],
    );

    expect(draft.categoryId).toBe('category-active');
    expect(draft.accountId).toBe('');
    expect(draft.tagIds).toEqual(['tag-active']);
  });

  it('marks manual adjustments and preserves integer-safe metadata identifiers', () => {
    expect(buildImportRowUpdatePayload({
      targetTransactionType: 'income',
      categoryId: 'category-1',
      accountId: 'account-1',
      tagIds: ['tag-1', 'tag-1', 'tag-2'],
      visibility: 'partner_readable',
    })).toEqual({
      target_transaction_type: 'income',
      row_status: 'adjusted',
      selected_category_id: 'category-1',
      selected_account_id: 'account-1',
      selected_tag_ids: ['tag-1', 'tag-2'],
      visibility: 'partner_readable',
    });
  });

  it('clears recommendations when the user explicitly skips a row', () => {
    expect(buildImportRowUpdatePayload({
      targetTransactionType: 'skipped',
      categoryId: 'category-1',
      accountId: 'account-1',
      tagIds: ['tag-1'],
      visibility: 'private',
    })).toEqual({
      target_transaction_type: 'skipped',
      row_status: 'skipped',
      selected_category_id: '',
      selected_account_id: '',
      selected_tag_ids: [],
      visibility: 'private',
    });
  });
});
