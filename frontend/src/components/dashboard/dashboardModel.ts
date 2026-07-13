import type { DashboardResponse } from '../../types/dashboard';
import type { TransactionResponse } from '../../types/transaction';

export type DashboardMetricTone = 'expense' | 'income' | 'neutral' | 'pay' | 'receive';

export interface DashboardSummaryMetric {
  id: 'expense' | 'settlement' | 'my-paid' | 'partner-paid' | 'income';
  label: string;
  amountCents: number;
  detail: string;
  tone: DashboardMetricTone;
}

export interface SettlementAction {
  state: 'pay' | 'receive' | 'settled';
  eyebrow: string;
  title: string;
  amountCents: number;
  description: string;
}

export interface TransactionTypePresentation {
  label: string;
  amountSign: '' | '+' | '-';
  tone: 'expense' | 'income' | 'shared' | 'settlement';
}

export function formatDashboardAmount(amountCents: number) {
  const normalizedCents = Math.trunc(amountCents);
  const absoluteCents = Math.abs(normalizedCents);
  const yuan = Math.floor(absoluteCents / 100).toLocaleString('en-US');
  const cents = String(absoluteCents % 100).padStart(2, '0');
  const sign = normalizedCents < 0 ? '-' : '';
  return `${sign}¥${yuan}.${cents}`;
}

function getPartnerName(data: DashboardResponse, currentUserId: string | undefined) {
  return data.user_stats.find((user) => user.user_id !== currentUserId)?.display_name || '伙伴';
}

export function getDashboardSummaryMetrics(
  data: DashboardResponse,
  currentUserId: string | undefined,
): DashboardSummaryMetric[] {
  const settlementAmount = data.shared_balance?.amount_cents ?? 0;
  const settlementDetail = settlementAmount <= 0
    ? '已结清'
    : data.shared_balance?.from_user_id === currentUserId
      ? '我应付'
      : '我应收';

  return [
    {
      id: 'expense',
      label: '本月总支出',
      amountCents: data.total_expense_cents,
      detail: '个人与共同消费',
      tone: 'expense',
    },
    {
      id: 'settlement',
      label: '待结算',
      amountCents: settlementAmount,
      detail: settlementDetail,
      tone: settlementDetail === '我应付' ? 'pay' : settlementDetail === '我应收' ? 'receive' : 'neutral',
    },
    {
      id: 'my-paid',
      label: '我的支付',
      amountCents: data.my_paid_cents,
      detail: '当月实际垫付',
      tone: 'neutral',
    },
    {
      id: 'partner-paid',
      label: `${getPartnerName(data, currentUserId)}的支付`,
      amountCents: data.partner_paid_cents,
      detail: '当月实际垫付',
      tone: 'neutral',
    },
    {
      id: 'income',
      label: '本月总收入',
      amountCents: data.total_income_cents,
      detail: '个人录入收入',
      tone: 'income',
    },
  ];
}

export function getSettlementAction(
  data: DashboardResponse,
  currentUserId: string | undefined,
): SettlementAction {
  const balance = data.shared_balance;
  const amountCents = balance?.amount_cents ?? 0;

  if (amountCents <= 0 || !balance?.from_user_id || !balance.to_user_id) {
    return {
      state: 'settled',
      eyebrow: '结算状态',
      title: '共同账目已结清',
      amountCents: 0,
      description: '当前没有需要转账的未结余额。',
    };
  }

  const partnerName = getPartnerName(data, currentUserId);
  const state = balance.from_user_id === currentUserId ? 'pay' : 'receive';

  return {
    state,
    eyebrow: '待结算',
    title: state === 'pay' ? `你应转给${partnerName}` : `${partnerName}应转给你`,
    amountCents,
    description: '结算会新增独立记录，不会改写历史共同账单。',
  };
}

export function getRecurringFrequencyLabel(frequency: 'weekly' | 'monthly' | 'yearly') {
  return {
    weekly: '每周',
    monthly: '每月',
    yearly: '每年',
  }[frequency];
}

export function getRecurringTypeLabel(type: 'expense' | 'income' | 'shared_expense') {
  return {
    expense: '个人支出',
    income: '个人收入',
    shared_expense: '共同支出',
  }[type];
}

export function getTransactionTypePresentation(
  type: TransactionResponse['type'],
): TransactionTypePresentation {
  const presentations: Record<TransactionResponse['type'], TransactionTypePresentation> = {
    expense: { label: '个人', amountSign: '-', tone: 'expense' },
    income: { label: '收入', amountSign: '+', tone: 'income' },
    shared_expense: { label: '共同', amountSign: '-', tone: 'shared' },
    settlement: { label: '结算', amountSign: '', tone: 'settlement' },
  };

  return presentations[type];
}

export function getPayerName(
  payerId: string,
  data: DashboardResponse,
  currentUserId: string | undefined,
) {
  if (payerId === currentUserId) return '我';
  return data.user_stats.find((user) => user.user_id === payerId)?.display_name || '伙伴';
}
