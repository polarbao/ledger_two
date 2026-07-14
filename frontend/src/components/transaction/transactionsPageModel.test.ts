import { describe, expect, it } from 'vitest';
import type { TransactionResponse } from '../../types/transaction';
import {
  buildTransactionFilterChips,
  getTransactionPresentation,
  yuanFilterToCents,
} from './transactionsPageModel';

const transaction = (overrides: Partial<TransactionResponse> = {}): TransactionResponse => ({
  id: 'tx-1',
  type: 'expense',
  title: '晚餐',
  amount_cents: 8800,
  currency: 'CNY',
  occurred_at: '2026-07-13T18:30:00+08:00',
  owner_user_id: 'user-a',
  created_by_user_id: 'user-a',
  payer_user_id: 'user-a',
  visibility: 'private',
  status: 'active',
  created_at: '2026-07-13T18:30:00+08:00',
  updated_at: '2026-07-13T18:30:00+08:00',
  ...overrides,
});

describe('transactions page model', () => {
  it('uses explicit text for personal, shared and settlement semantics', () => {
    expect(getTransactionPresentation(transaction())).toMatchObject({
      typeLabel: '个人支出',
      scopeLabel: '仅自己可见',
      splitLabel: '个人账单',
      amountPrefix: '-',
    });
    expect(getTransactionPresentation(transaction({
      type: 'shared_expense',
      visibility: 'shared',
      split_method: 'equal',
    }))).toMatchObject({
      typeLabel: '共同支出',
      scopeLabel: '共同账本可见',
      splitLabel: '成员均摊',
    });
    expect(getTransactionPresentation(transaction({ type: 'settlement', visibility: 'shared' }))).toMatchObject({
      typeLabel: '结算记录',
      splitLabel: '独立结算记录',
      amountPrefix: '',
    });
    expect(getTransactionPresentation(transaction({
      type: 'shared_expense',
      visibility: 'shared',
      split_method: 'ratio',
    }))).toMatchObject({ splitLabel: '自定义分摊' });
  });

  it('builds removable filter labels without exposing ids', () => {
    const chips = buildTransactionFilterChips({
      type: 'shared_expense',
      categoryId: 'category-uuid',
      keyword: '晚餐',
      minAmount: '20',
      maxAmount: '200',
      payerUserId: 'user-uuid',
      visibility: 'partner_readable',
      tag: '聚餐',
    }, { 'category-uuid': '餐饮（已归档）' }, { 'user-uuid': 'Lynn' });

    expect(chips.map((chip) => chip.label)).toEqual([
      '类型：共同支出',
      '分类：餐饮（已归档）',
      '关键词：晚餐',
      '最低：¥20',
      '最高：¥200',
      '付款人：Lynn',
      '对方可见，只读',
      '标签：聚餐',
    ]);
    expect(chips.map((chip) => chip.label).join(' ')).not.toContain('uuid');
  });

  it('converts valid yuan filters to integer cents', () => {
    expect(yuanFilterToCents('12.34')).toBe(1234);
    expect(yuanFilterToCents('')).toBeUndefined();
    expect(yuanFilterToCents('-1')).toBeUndefined();
    expect(yuanFilterToCents('not-a-number')).toBeUndefined();
  });
});
