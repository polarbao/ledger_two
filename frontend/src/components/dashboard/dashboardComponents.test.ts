import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import type { RecurringReminderResponse } from '../../types/transaction';
import type { DashboardSummaryMetric, SettlementAction } from './dashboardModel';

const metrics: DashboardSummaryMetric[] = [
  { id: 'expense', label: '本月总支出', amountCents: 428650, detail: '个人与共同消费', tone: 'expense' },
  { id: 'settlement', label: '待结算', amountCents: 32840, detail: '我应收', tone: 'receive' },
  { id: 'my-paid', label: '我的支付', amountCents: 253000, detail: '当月实际垫付', tone: 'neutral' },
  { id: 'partner-paid', label: '北北的支付', amountCents: 175650, detail: '当月实际垫付', tone: 'neutral' },
  { id: 'income', label: '本月总收入', amountCents: 860000, detail: '个人录入收入', tone: 'income' },
];

const reminder: RecurringReminderResponse = {
  id: 'reminder-id',
  rule_id: 'rule-id',
  rule_name: '每月房租',
  type: 'shared_expense',
  title: '房租',
  amount_cents: 320000,
  category_id: 'category-id',
  category_name: '共同居住',
  payer_user_id: 'payer-id',
  split_method: 'equal',
  tag_names: [],
  note: '',
  frequency: 'monthly',
  due_date: '2026-07-15',
  status: 'pending',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
};

describe('UI-FL-03 dashboard components', () => {
  it('renders the five summary metrics as one labelled compact group', async () => {
    const component = await import('./MonthlySummary').catch(() => null);

    expect(component).not.toBeNull();
    if (!component) return;

    const markup = renderToStaticMarkup(createElement(component.default, { metrics }));

    expect(markup).toContain('aria-label="月度摘要"');
    expect(markup.match(/lt-dashboard-metric/g)?.length).toBeGreaterThanOrEqual(5);
    expect(markup).toContain('本月总支出');
    expect(markup).toContain('待结算');
    expect(markup).toContain('¥4,286.50');
    expect(markup).toContain('¥328.40');
  });

  it('renders an explicit settlement destination and keeps history semantics visible', async () => {
    const component = await import('./SettlementActionCard').catch(() => null);

    expect(component).not.toBeNull();
    if (!component) return;

    const action: SettlementAction = {
      state: 'receive',
      eyebrow: '待结算',
      title: '北北应转给你',
      amountCents: 32840,
      description: '结算会新增独立记录，不会改写历史共同账单。',
    };
    const markup = renderToStaticMarkup(createElement(
      MemoryRouter,
      null,
      createElement(component.default, { action }),
    ));

    expect(markup).toContain('北北应转给你');
    expect(markup).toContain('¥328.40');
    expect(markup).toContain('不会改写历史共同账单');
    expect(markup).toContain('href="/settlement"');
    expect(markup).toContain('查看结算');
  });

  it('renders recurring reminders with readable context and separate actions', async () => {
    const component = await import('./RecurringReminderList').catch(() => null);

    expect(component).not.toBeNull();
    if (!component) return;

    const markup = renderToStaticMarkup(createElement(component.default, {
      reminders: [reminder],
      isMutating: false,
      onConfirm: () => undefined,
      onSkip: () => undefined,
    }));

    expect(markup).toContain('1 笔待确认');
    expect(markup).toContain('每月房租');
    expect(markup).toContain('每月');
    expect(markup).toContain('共同支出');
    expect(markup).toContain('跳过本期');
    expect(markup).toContain('确认记账');
  });
});
