import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageDirectory = dirname(fileURLToPath(import.meta.url));

function readPageFile(relativePath: string) {
  const filePath = resolve(pageDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('UI-FL-07 metadata management contract', () => {
  it('uses shared controls and preserves archive history semantics', () => {
    const source = readPageFile('./MetadataManagePage.tsx');
    const ledgerSettings = readPageFile('../components/ledger/LedgerSettings.tsx');

    expect(source).toContain("import ConfirmDialog from '../components/ui/ConfirmDialog'");
    expect(source).toContain("import SegmentedControl from '../components/ui/SegmentedControl'");
    expect(source).toContain('metadataApi.archive(kind, item.id)');
    expect(source).toContain('历史引用仍保留原名称');
    expect(source).not.toContain('window.confirm');
    expect(source).not.toContain('style={{');
    expect(ledgerSettings).toContain('所有成员可以查看名单');
    expect(ledgerSettings).toContain('历史账单不会被删除');
    expect(ledgerSettings).toContain('acknowledgeHistoryVisibility');
    expect(ledgerSettings).toContain('新成员将按可见性规则读取当前账本的既有历史');
    expect(ledgerSettings).not.toContain('window.confirm');
    expect(ledgerSettings).not.toContain('style={{');
  });

  it('keeps list, editor and mobile controls within stable dimensions', () => {
    const css = readPageFile('./MetadataManagePage.css');

    expect(css).toContain('grid-template-columns: minmax(280px, 340px) minmax(0, 1fr);');
    expect(css).toContain('@media (max-width: 1024px)');
    expect(css).toContain('@media (max-width: 768px)');
    expect(css).toContain('@media (max-width: 430px)');
    expect(css).toContain('overflow-wrap: anywhere;');
    expect(css).not.toContain('linear-gradient');
    expect(css).not.toContain('letter-spacing: -');
  });
});
