import { describe, expect, it } from 'vitest';
import {
  buildMonthRange,
  buildTransactionsDrilldown,
  getChartPercent,
  getExpenseChange,
  getExpenseChangeLabel,
  getShortMonthLabel,
} from './analyticsPageModel';

describe('UI-FL-09 analytics page model', () => {
  it('builds a stable six-month range across a year boundary', () => {
    expect(buildMonthRange('2026-02', 6)).toEqual([
      '2025-09',
      '2025-10',
      '2025-11',
      '2025-12',
      '2026-01',
      '2026-02',
    ]);
    expect(buildMonthRange('invalid', 6)).toEqual([]);
  });

  it('explains expense movement without dividing by a zero baseline', () => {
    expect(getExpenseChange(15000, 10000)).toEqual({ direction: 'up', percent: 50 });
    expect(getExpenseChange(5000, 10000)).toEqual({ direction: 'down', percent: 50 });
    expect(getExpenseChange(0, 0)).toEqual({ direction: 'flat', percent: 0 });
    expect(getExpenseChange(100, 0)).toEqual({ direction: 'new', percent: null });
    expect(getExpenseChangeLabel(getExpenseChange(100, 0))).toBe('上月无支出基线');
  });

  it('creates transaction drilldowns with existing filter parameters', () => {
    expect(buildTransactionsDrilldown({ month: '2026-07', categoryId: 'cat food' }))
      .toBe('/transactions?month=2026-07&page=1&category_id=cat+food');
    expect(buildTransactionsDrilldown({ month: '2026-07', tag: '周末 外出' }))
      .toBe('/transactions?month=2026-07&page=1&tag=%E5%91%A8%E6%9C%AB+%E5%A4%96%E5%87%BA');
    expect(buildTransactionsDrilldown({ month: '2026-07', payerUserId: 'user-a' }))
      .toBe('/transactions?month=2026-07&page=1&payer_user_id=user-a');
    expect(buildTransactionsDrilldown({ month: '2026-07', archivedLedgerId: 'ledger-a' }))
      .toBe('/transactions?month=2026-07&page=1&archived_ledger_id=ledger-a');
  });

  it('keeps month labels and chart bars readable', () => {
    expect(getShortMonthLabel('2026-07')).toBe('7月');
    expect(getChartPercent(0, 100)).toBe(0);
    expect(getChartPercent(1, 100)).toBe(4);
    expect(getChartPercent(200, 100)).toBe(100);
  });
});
