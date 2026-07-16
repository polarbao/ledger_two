import { describe, expect, it } from 'vitest';
import type { DashboardResponse } from '../../types/dashboard';

const dashboardFixture: DashboardResponse = {
  month: '2026-07',
  total_expense_cents: 428650,
  total_income_cents: 860000,
  my_paid_cents: 253000,
  partner_paid_cents: 175650,
  shared_balance: {
    from_user_id: 'partner-id',
    to_user_id: 'current-id',
    amount_cents: 32840,
  },
  recent_transactions: [],
  category_summary: [],
  tag_summary: [],
  user_stats: [
    {
      user_id: 'current-id',
      display_name: '林恩',
      paid_cents: 253000,
      share_cents: 220160,
    },
    {
      user_id: 'partner-id',
      display_name: '北北',
      paid_cents: 175650,
      share_cents: 208490,
    },
  ],
};

describe('UI-FL-03 dashboard presentation model', () => {
  it('normalizes persisted null collections before rendering an empty ledger', async () => {
    const model = await import('./dashboardModel').catch(() => null);

    expect(model).not.toBeNull();
    if (!model) return;

    const normalized = model.normalizeDashboardResponse({
      ...dashboardFixture,
      recent_transactions: null,
      category_summary: null,
      tag_summary: null,
      user_stats: null,
      shared_balance: {
        ...dashboardFixture.shared_balance,
        user_balances: null,
        suggested_transfers: null,
      },
    } as never);

    expect(normalized.recent_transactions).toEqual([]);
    expect(normalized.category_summary).toEqual([]);
    expect(normalized.tag_summary).toEqual([]);
    expect(normalized.user_stats).toEqual([]);
    expect(normalized.shared_balance.user_balances).toEqual([]);
    expect(normalized.shared_balance.suggested_transfers).toEqual([]);
  });

  it('keeps the high-priority monthly metrics in a stable mobile-first order', async () => {
    const model = await import('./dashboardModel').catch(() => null);

    expect(model).not.toBeNull();
    if (!model) return;

    const metrics = model.getDashboardSummaryMetrics(dashboardFixture, 'current-id');

    expect(metrics.map((metric) => metric.id)).toEqual([
      'expense',
      'settlement',
      'my-paid',
      'partner-paid',
      'income',
    ]);
    expect(metrics.map((metric) => metric.amountCents)).toEqual([
      428650,
      32840,
      253000,
      175650,
      860000,
    ]);
    expect(metrics[1].detail).toBe('我应收');
  });

  it('describes the settlement action without leaking user ids', async () => {
    const model = await import('./dashboardModel').catch(() => null);

    expect(model).not.toBeNull();
    if (!model) return;

    const action = model.getSettlementAction(dashboardFixture, 'current-id');

    expect(action).toEqual({
      state: 'receive',
      eyebrow: '待结算',
      title: '北北应转给你',
      amountCents: 32840,
      description: '结算会新增独立记录，不会改写历史共同账单。',
    });
    expect(JSON.stringify(action)).not.toContain('partner-id');
    expect(JSON.stringify(action)).not.toContain('current-id');
  });

  it('uses a calm settled state when no transfer is required', async () => {
    const model = await import('./dashboardModel').catch(() => null);

    expect(model).not.toBeNull();
    if (!model) return;

    const action = model.getSettlementAction({
      ...dashboardFixture,
      shared_balance: { amount_cents: 0 },
    }, 'current-id');

    expect(action.state).toBe('settled');
    expect(action.title).toBe('共同账目已结清');
    expect(action.amountCents).toBe(0);
  });

  it('maps recurring and transaction types to readable product language', async () => {
    const model = await import('./dashboardModel').catch(() => null);

    expect(model).not.toBeNull();
    if (!model) return;

    expect(model.getRecurringFrequencyLabel('weekly')).toBe('每周');
    expect(model.getRecurringFrequencyLabel('monthly')).toBe('每月');
    expect(model.getRecurringFrequencyLabel('yearly')).toBe('每年');
    expect(model.getTransactionTypePresentation('shared_expense')).toEqual({
      label: '共同',
      amountSign: '-',
      tone: 'shared',
    });
    expect(model.getTransactionTypePresentation('settlement')).toEqual({
      label: '结算',
      amountSign: '',
      tone: 'settlement',
    });
  });

  it('formats integer cents with stable grouping and two decimal places', async () => {
    const model = await import('./dashboardModel').catch(() => null);

    expect(model).not.toBeNull();
    if (!model) return;

    expect(model.formatDashboardAmount(428650)).toBe('¥4,286.50');
    expect(model.formatDashboardAmount(0)).toBe('¥0.00');
    expect(model.formatDashboardAmount(-32840)).toBe('-¥328.40');
  });
});
