import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageDirectory = dirname(fileURLToPath(import.meta.url));

function readPageFile(relativePath: string) {
  const filePath = resolve(pageDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('UI-FL-06 Settlement page contract', () => {
  it('uses shared primitives and preserves authoritative settlement behavior', () => {
    const source = readPageFile('./SettlementPage.tsx');

    expect(source).toContain("import ConfirmDialog from '../components/ui/ConfirmDialog'");
    expect(source).toContain("import SegmentedControl from '../components/ui/SegmentedControl'");
    expect(source).toContain('settlementApi.getBalance(balanceMonth, signal)');
    expect(source).toContain('settlementApi.createSettlement');
    expect(source).toContain('queryKeys.settlements.balanceRoot(ledgerId)');
    expect(source).toContain('queryKeys.reports.root(ledgerId)');
    expect(source).toContain('!isArchivedView');
    expect(source).toContain('不会修改历史共同支出');
    expect(source).toContain('复制失败，请手动选择下方文案');
    expect(source).not.toContain('style={{');
  });

  it('defines stable Fresh Light geometry without visual gradients', () => {
    const css = readPageFile('./SettlementPage.css');

    expect(css).toContain('max-width: 1180px;');
    expect(css).toContain('grid-template-columns: repeat(2, minmax(0, 1fr));');
    expect(css).toContain('@media (max-width: 1024px)');
    expect(css).toContain('@media (max-width: 768px)');
    expect(css).toContain('@media (max-width: 430px)');
    expect(css).toContain('var(--lt-bg-surface)');
    expect(css).toContain('font-variant-numeric: tabular-nums;');
    expect(css).not.toContain('linear-gradient');
    expect(css).not.toContain('letter-spacing: -');
  });
});
