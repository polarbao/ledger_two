import { describe, expect, it } from 'vitest';
import { buildContinueTransactionFormValues } from './transactionFormState';

describe('buildContinueTransactionFormValues', () => {
  it('keeps daily-entry defaults while clearing one-time fields', () => {
    const nextValues = buildContinueTransactionFormValues(
      {
        type: 'expense',
        amount: '35.80',
        title: '午餐',
        category_id: 'cat-food',
        account_id: 'acc-wechat',
        tag_names: '工作餐, 报销',
        payer_user_id: 'user-a',
        split_method: 'equal',
        occurred_at: '2026-07-06',
        note: '有小票',
        visibility: 'partner_readable',
        attachment_paths: ['/uploads/receipt-a.jpg'],
      },
      '2026-07-07',
    );

    expect(nextValues).toEqual({
      type: 'expense',
      amount: '',
      title: '',
      category_id: 'cat-food',
      account_id: 'acc-wechat',
      tag_names: '工作餐, 报销',
      payer_user_id: 'user-a',
      split_method: 'equal',
      occurred_at: '2026-07-06',
      note: '',
      visibility: 'partner_readable',
      attachment_paths: [],
    });
  });

  it('preserves shared expense split context without carrying amount or attachments', () => {
    const nextValues = buildContinueTransactionFormValues(
      {
        type: 'shared_expense',
        amount: '200.00',
        title: '晚餐',
        category_id: 'cat-dinner',
        account_id: null,
        tag_names: '',
        payer_user_id: 'user-b',
        split_method: 'payer_only',
        occurred_at: '2026-07-06T00:00:00.000Z',
        note: '本期特殊处理',
        visibility: 'partner_readable',
        attachment_paths: ['/uploads/shared.jpg'],
      },
      '2026-07-07',
    );

    expect(nextValues.type).toBe('shared_expense');
    expect(nextValues.amount).toBe('');
    expect(nextValues.title).toBe('');
    expect(nextValues.category_id).toBe('cat-dinner');
    expect(nextValues.account_id).toBe('');
    expect(nextValues.payer_user_id).toBe('user-b');
    expect(nextValues.split_method).toBe('payer_only');
    expect(nextValues.occurred_at).toBe('2026-07-06');
    expect(nextValues.note).toBe('');
    expect(nextValues.attachment_paths).toEqual([]);
  });

  it('uses fallback date when the submitted date is empty', () => {
    const nextValues = buildContinueTransactionFormValues(
      {
        type: 'income',
        amount: '5000',
        title: '',
        category_id: '',
        account_id: '',
        tag_names: '',
        payer_user_id: 'user-a',
        split_method: 'equal',
        occurred_at: '',
        note: '',
        visibility: 'private',
        attachment_paths: [],
      },
      '2026-07-07',
    );

    expect(nextValues.occurred_at).toBe('2026-07-07');
    expect(nextValues.visibility).toBe('private');
  });
});
