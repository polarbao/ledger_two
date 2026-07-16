import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageDirectory = dirname(fileURLToPath(import.meta.url));

function readSource(relativePath: string) {
  const filePath = resolve(pageDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('Task50.5 ledger management UI contract', () => {
  it('registers canonical management and detail routes before metadata routes', () => {
    const routes = readSource('../routes.tsx');

    expect(routes).toContain("path: 'settings/ledgers'");
    expect(routes).toContain("path: 'settings/ledgers/:ledgerId'");
    expect(routes.indexOf("path: 'settings/ledgers'"))
      .toBeLessThan(routes.indexOf("path: 'settings/:kind'"));
  });

  it('implements list, lifecycle, member and archived-view composites with existing primitives', () => {
    const management = readSource('./LedgerManagementPage.tsx');
    const detail = readSource('./LedgerDetailPage.tsx');
    const banner = readSource('../components/ledger/ArchivedLedgerBanner.tsx');
    const lifecycle = readSource('../components/ledger/LedgerLifecycleActions.tsx');
    const importPage = readSource('./ImportPage.tsx');

    expect(management).toContain('SegmentedControl');
    expect(management).toContain('ResponsiveDataList');
    expect(lifecycle).toContain('ledgerApi.getArchivePreflight');
    expect(lifecycle).toContain('ledgerApi.archiveLedger');
    expect(lifecycle).toContain('ledgerApi.restoreLedger');
    expect(importPage).toContain('importsApi.discard');
    expect(importPage).toContain('放弃预览');
    expect(detail).toContain('ledgerApi.addMember');
    expect(detail).toContain('ledgerApi.updateMemberRole');
    expect(detail).toContain('ledgerApi.transferOwner');
    expect(detail).toContain('ledgerApi.removeMember');
    expect(detail).toContain('ledgerApi.leaveLedger');
    expect(banner).toContain('正在查看已归档账本');
  });

  it('keeps the Fresh Light page responsive without decorative gradients', () => {
    const css = readSource('./LedgerManagementPage.css');

    expect(css).toContain('@media (max-width: 768px)');
    expect(css).toContain('@media (max-width: 430px)');
    expect(css).toContain('min-height: 44px;');
    expect(css).not.toContain('linear-gradient');
    expect(css).not.toContain('letter-spacing: -');
  });
});
