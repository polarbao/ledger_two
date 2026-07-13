import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageDirectory = dirname(fileURLToPath(import.meta.url));

function readPageFile(relativePath: string) {
  const filePath = resolve(pageDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('UI-FL-03 Dashboard page contract', () => {
  it('composes the frozen dashboard sections without changing the quick-add contract', () => {
    const source = readPageFile('./DashboardPage.tsx');

    expect(source).toContain("import MonthlySummary from '../components/dashboard/MonthlySummary'");
    expect(source).toContain("import SettlementActionCard from '../components/dashboard/SettlementActionCard'");
    expect(source).toContain("import RecurringReminderList from '../components/dashboard/RecurringReminderList'");
    expect(source).toContain("import CategorySummary from '../components/dashboard/CategorySummary'");
    expect(source).toContain("import RecentTransactionList from '../components/dashboard/RecentTransactionList'");
    expect(source).toContain('setAddDrawerOpen(true)');
    expect(source).toContain('transactionsApi.confirmReminder(id)');
    expect(source).toContain('transactionsApi.skipReminder(id)');
    expect(source).toContain('confirmReminderMutation.isPending ? confirmReminderMutation.variables : undefined');
    expect(source).not.toContain('style={{');
  });

  it('defines stable desktop and mobile dashboard geometry with semantic tokens', () => {
    const css = readPageFile('./DashboardPage.css');

    expect(css).toContain('grid-template-columns: repeat(5, minmax(0, 1fr));');
    expect(css).toContain('@media (max-width: 1024px)');
    expect(css).toContain('@media (max-width: 768px)');
    expect(css).toContain('@media (max-width: 430px)');
    expect(css).toContain('.lt-dashboard-metric:last-child:nth-child(odd)');
    expect(css).toContain('grid-column: 1 / -1;');
    expect(css).toContain('var(--lt-bg-surface)');
    expect(css).toContain('font-variant-numeric: tabular-nums;');
    expect(css).toContain('overflow-wrap: anywhere;');
    expect(css).not.toContain('letter-spacing: -');
  });
});
