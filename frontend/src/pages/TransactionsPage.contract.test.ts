import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageDirectory = dirname(fileURLToPath(import.meta.url));

function readPageFile(relativePath: string) {
  const filePath = resolve(pageDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('UI-FL-05 transactions page contract', () => {
  it('composes the responsive workbench without changing frozen mutation behavior', () => {
    const source = readPageFile('./TransactionsPage.tsx');

    expect(source).toContain("import ResponsiveDataList from '../components/ui/ResponsiveDataList'");
    expect(source).toContain("import ActiveFilterChips from '../components/ui/ActiveFilterChips'");
    expect(source).toContain("import TransactionTable from '../components/transaction/TransactionTable'");
    expect(source).toContain("import TransactionDetailDrawer from '../components/transaction/TransactionDetailDrawer'");
    expect(source).toContain('transactionsApi.deleteTransaction(id)');
    expect(source).toContain('transactionsApi.batchTag(payload)');
    expect(source).toContain('getTransactionEditBlockReason(');
    expect(source).toContain('setEditSourceTransaction(tx)');
    expect(source).toContain('queryKeys.transactions.root(activeLedgerId)');
    expect(source).toContain('queryKeys.dashboard.root(activeLedgerId)');
    expect(source).toContain('queryKeys.reports.root(activeLedgerId)');
    expect(source).not.toContain('style={{');
  });

  it('keeps desktop tables out of mobile layout and protects long content', () => {
    const css = readPageFile('./TransactionsPage.css');
    const primitiveCss = readPageFile('../styles/ui-primitives.css');

    expect(css).toContain('@media (max-width: 768px)');
    expect(css).toContain('@media (max-width: 430px)');
    expect(css).toContain('min-width: 1040px;');
    expect(css).toContain('overflow-wrap: anywhere;');
    expect(css).toContain('font-variant-numeric: tabular-nums;');
    expect(css).not.toContain('letter-spacing: -');
    expect(primitiveCss).toContain('.ui-responsive-data-list__desktop');
    expect(primitiveCss).toContain('.ui-responsive-data-list__mobile');
    expect(primitiveCss).toContain('display: none;');
  });
});
