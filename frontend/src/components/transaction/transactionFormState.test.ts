import { describe, expect, it } from 'vitest';
import {
  buildContinueTransactionFormValues,
  buildTransactionUpdatePayload,
  buildSharedExpensePreview,
  getTransactionEditBlockReason,
  shouldOpenAdvancedFields,
  transactionToFormValues,
} from './transactionFormState';
import type { TransactionResponse } from '../../types/transaction';

const transaction = (overrides: Partial<TransactionResponse> = {}): TransactionResponse => ({
  id: 'tx-1',
  type: 'expense',
  title: '午餐',
  amount_cents: 3580,
  currency: 'CNY',
  occurred_at: '2026-07-14T08:30:00+08:00',
  owner_user_id: 'user-a',
  created_by_user_id: 'user-a',
  payer_user_id: 'user-a',
  account_id: 'account-wechat',
  category_id: 'category-food',
  visibility: 'partner_readable',
  note: '工作餐',
  status: 'active',
  tags: ['报销', '工作'],
  attachment_paths: ['/uploads/receipt-a.jpg'],
  created_at: '2026-07-14T08:30:00+08:00',
  updated_at: '2026-07-14T08:30:00+08:00',
  ...overrides,
});

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

describe('UI-FL-05E transaction editing', () => {
  it('maps a transaction to a clean edit snapshot without changing its date or attachments', () => {
    expect(transactionToFormValues(transaction())).toEqual({
      type: 'expense',
      amount: '35.80',
      title: '午餐',
      category_id: 'category-food',
      account_id: 'account-wechat',
      tag_names: '报销, 工作',
      payer_user_id: 'user-a',
      split_method: 'equal',
      occurred_at: '2026-07-14',
      note: '工作餐',
      visibility: 'partner_readable',
      attachment_paths: ['/uploads/receipt-a.jpg'],
    });
  });

  it('builds a partial patch and does not resend unchanged archived metadata', () => {
    const source = transaction({
      category_id: 'archived-category',
      account_id: 'archived-account',
      tags: ['历史标签'],
    });
    const values = transactionToFormValues(source);

    expect(buildTransactionUpdatePayload(source, { ...values, title: '新的午餐', amount: '40.00' }))
      .toEqual({ title: '新的午餐', amount_cents: 4000 });
    expect(buildTransactionUpdatePayload(source, values)).toEqual({});
  });

  it('sends explicit nulls and complete attachment lists only when the user changes them', () => {
    const source = transaction();
    const values = transactionToFormValues(source);

    expect(buildTransactionUpdatePayload(source, {
      ...values,
      category_id: '',
      account_id: '',
      visibility: 'private',
      tag_names: '工作, 新标签',
      attachment_paths: [],
    })).toEqual({
      category_id: null,
      account_id: null,
      visibility: 'private',
      tag_names: ['工作', '新标签'],
      attachment_paths: [],
    });
  });

  it('allows only creator-owned online bills and protects unsupported shared participant sets', () => {
    const shared = transaction({
      type: 'shared_expense',
      visibility: 'shared',
      split_method: 'equal',
      participants: [
        { user_id: 'user-a', share_amount_cents: 1790 },
        { user_id: 'user-b', share_amount_cents: 1790 },
      ],
    });

    expect(getTransactionEditBlockReason(shared, 'user-a', true, ['user-a', 'user-b'], false)).toBeNull();
    expect(getTransactionEditBlockReason(shared, 'user-b', true, ['user-a', 'user-b'], false))
      .toBe('只能编辑自己创建的账单');
    expect(getTransactionEditBlockReason(shared, 'user-a', true, ['user-a', 'user-b'], true))
      .toBe('离线状态不能编辑已保存账单');
    expect(getTransactionEditBlockReason(
      { ...shared, participants: [{ user_id: 'user-a', share_amount_cents: 3580 }] },
      'user-a',
      true,
      ['user-a', 'user-b'],
      false,
    )).toBe('历史参与人和当前账本成员不一致，请先保留原账单');
    expect(getTransactionEditBlockReason(
      { ...shared, split_method: 'ratio' },
      'user-a',
      true,
      ['user-a', 'user-b'],
      false,
    )).toBe('自定义分摊账单暂不支持在快捷编辑器中修改');
  });
});
