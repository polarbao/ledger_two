export interface ExpenseChange {
  direction: 'up' | 'down' | 'flat' | 'new';
  percent: number | null;
}

export interface TransactionDrilldown {
  month: string;
  categoryId?: string;
  tag?: string;
  payerUserId?: string;
  archivedLedgerId?: string;
}

const monthPattern = /^(\d{4})-(0[1-9]|1[0-2])$/;

export function buildMonthRange(endMonth: string, count: number) {
  const match = monthPattern.exec(endMonth);
  if (!match || count <= 0) return [];

  const year = Number(match[1]);
  const monthIndex = Number(match[2]) - 1;

  return Array.from({ length: count }, (_, index) => {
    const date = new Date(Date.UTC(year, monthIndex - (count - index - 1), 1));
    return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`;
  });
}

export function getExpenseChange(currentExpense: number, previousExpense: number): ExpenseChange {
  if (previousExpense === 0) {
    return currentExpense === 0
      ? { direction: 'flat', percent: 0 }
      : { direction: 'new', percent: null };
  }

  const percent = Math.abs((currentExpense - previousExpense) / previousExpense * 100);
  if (currentExpense === previousExpense) return { direction: 'flat', percent: 0 };
  return {
    direction: currentExpense > previousExpense ? 'up' : 'down',
    percent,
  };
}

export function getExpenseChangeLabel(change: ExpenseChange) {
  if (change.direction === 'new') return '上月无支出基线';
  if (change.direction === 'flat') return '与上月持平';
  return `${change.direction === 'up' ? '较上月增加' : '较上月减少'} ${change.percent?.toFixed(1)}%`;
}

export function getShortMonthLabel(month: string) {
  const match = monthPattern.exec(month);
  return match ? `${Number(match[2])}月` : month;
}

export function buildTransactionsDrilldown({
  month,
  categoryId,
  tag,
  payerUserId,
  archivedLedgerId,
}: TransactionDrilldown) {
  const params = new URLSearchParams({ month, page: '1' });
  if (categoryId) params.set('category_id', categoryId);
  if (tag) params.set('tag', tag);
  if (payerUserId) params.set('payer_user_id', payerUserId);
  if (archivedLedgerId) params.set('archived_ledger_id', archivedLedgerId);
  return `/transactions?${params.toString()}`;
}

export function getChartScale(values: number[]) {
  return Math.max(1, ...values);
}

export function getChartPercent(value: number, scale: number) {
  if (value <= 0 || scale <= 0) return 0;
  return Math.max(4, Math.min(100, value / scale * 100));
}
