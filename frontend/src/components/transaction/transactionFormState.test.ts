import { describe, expect, it } from 'vitest';
import {
  buildContinueTransactionFormValues,
  buildSharedExpensePreview,
  shouldOpenAdvancedFields,
} from './transactionFormState';

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

  it('matches the backend equal-split remainder rule in the shared preview', () => {
    const preview = buildSharedExpensePreview(
      '100.01',
      [
        { user_id: 'user-a', display_name: '林然' },
        { user_id: 'user-b', display_name: '北北' },
      ],
      'user-a',
      'equal',
    );

    expect(preview).toEqual([
      {
        userId: 'user-a',
        displayName: '林然',
        shareAmountCents: 5001,
        isPayer: true,
        isParticipating: true,
      },
      {
        userId: 'user-b',
        displayName: '北北',
        shareAmountCents: 5000,
        isPayer: false,
        isParticipating: true,
      },
    ]);
  });

  it('assigns payer-only preview to the payer and tolerates incomplete amounts', () => {
    const members = [
      { user_id: 'user-a', display_name: '林然' },
      { user_id: 'user-b', display_name: '北北' },
    ];

    expect(buildSharedExpensePreview('88', members, 'user-b', 'payer_only'))
      .toEqual([
        {
          userId: 'user-a',
          displayName: '林然',
          shareAmountCents: 0,
          isPayer: false,
          isParticipating: false,
        },
        {
          userId: 'user-b',
          displayName: '北北',
          shareAmountCents: 8800,
          isPayer: true,
          isParticipating: true,
        },
      ]);
    expect(buildSharedExpensePreview('', members, 'user-a', 'equal')
      .every((item) => item.shareAmountCents === 0)).toBe(true);
  });

  it('opens low-frequency fields only when they carry meaningful data', () => {
    const base = {
      type: 'expense' as const,
      amount: '',
      title: '',
      category_id: '',
      account_id: '',
      tag_names: '',
      payer_user_id: 'user-a',
      split_method: 'equal' as const,
      occurred_at: '2026-07-14',
      note: '',
      visibility: 'partner_readable' as const,
      attachment_paths: [],
    };

    expect(shouldOpenAdvancedFields(base)).toBe(false);
    expect(shouldOpenAdvancedFields({ ...base, title: '午餐' })).toBe(true);
    expect(shouldOpenAdvancedFields({ ...base, visibility: 'private' })).toBe(true);
    expect(shouldOpenAdvancedFields({ ...base, attachment_paths: ['/receipt.jpg'] })).toBe(true);
  });
});
