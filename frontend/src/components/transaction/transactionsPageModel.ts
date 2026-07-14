import type { StatusChipTone } from '../ui/StatusChip';
import type { TransactionResponse } from '../../types/transaction';

export type TransactionQuickType = '' | 'expense' | 'income' | 'shared_expense' | 'settlement';

export interface TransactionPresentation {
  typeLabel: string;
  typeTone: StatusChipTone;
  amountPrefix: string;
  amountTone: 'expense' | 'income' | 'neutral';
  scopeLabel: string;
  splitLabel: string;
}

export interface TransactionFilterState {
  type: string;
  categoryId: string;
  keyword: string;
  minAmount: string;
  maxAmount: string;
  payerUserId: string;
  visibility: string;
  tag: string;
}

export interface TransactionFilterChip {
  key: keyof TransactionFilterState;
  label: string;
}

export function getTransactionPresentation(tx: TransactionResponse): TransactionPresentation {
  if (tx.type === 'income') {
    return {
      typeLabel: '个人收入',
      typeTone: 'success',
      amountPrefix: '+',
      amountTone: 'income',
      scopeLabel: tx.visibility === 'private' ? '仅自己可见' : '对方可见，只读',
      splitLabel: '个人账单',
    };
  }
  if (tx.type === 'shared_expense') {
    return {
      typeLabel: '共同支出',
      typeTone: 'accent',
      amountPrefix: '-',
      amountTone: 'expense',
      scopeLabel: '共同账本可见',
      splitLabel: tx.split_method === 'payer_only' ? '付款人承担' : '双方均摊',
    };
  }
  if (tx.type === 'settlement') {
    return {
      typeLabel: '结算记录',
      typeTone: 'info',
      amountPrefix: '',
      amountTone: 'neutral',
      scopeLabel: '共同账本可见',
      splitLabel: '独立结算记录',
    };
  }
  return {
    typeLabel: '个人支出',
    typeTone: 'warning',
    amountPrefix: '-',
    amountTone: 'expense',
    scopeLabel: tx.visibility === 'private' ? '仅自己可见' : '对方可见，只读',
    splitLabel: '个人账单',
  };
}

export function buildTransactionFilterChips(
  filters: TransactionFilterState,
  categoryNames: Record<string, string>,
  payerNames: Record<string, string>,
): TransactionFilterChip[] {
  const chips: TransactionFilterChip[] = [];
  if (filters.type) {
    const typeLabels: Record<string, string> = {
      expense: '个人支出',
      income: '个人收入',
      shared_expense: '共同支出',
      settlement: '结算记录',
    };
    chips.push({ key: 'type', label: `类型：${typeLabels[filters.type] || '其他'}` });
  }
  if (filters.categoryId) {
    chips.push({ key: 'categoryId', label: `分类：${categoryNames[filters.categoryId] || '已设分类'}` });
  }
  if (filters.keyword) chips.push({ key: 'keyword', label: `关键词：${filters.keyword}` });
  if (filters.minAmount) chips.push({ key: 'minAmount', label: `最低：¥${filters.minAmount}` });
  if (filters.maxAmount) chips.push({ key: 'maxAmount', label: `最高：¥${filters.maxAmount}` });
  if (filters.payerUserId) {
    chips.push({ key: 'payerUserId', label: `付款人：${payerNames[filters.payerUserId] || '账本成员'}` });
  }
  if (filters.visibility) {
    chips.push({
      key: 'visibility',
      label: filters.visibility === 'private' ? '仅自己可见' : '对方可见，只读',
    });
  }
  if (filters.tag) chips.push({ key: 'tag', label: `标签：${filters.tag}` });
  return chips;
}

export function yuanFilterToCents(value: string): number | undefined {
  if (!value.trim()) return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0 ? Math.round(parsed * 100) : undefined;
}
