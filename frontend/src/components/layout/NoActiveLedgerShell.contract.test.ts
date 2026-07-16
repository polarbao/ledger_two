import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const layoutDirectory = dirname(fileURLToPath(import.meta.url));

function readSource(relativePath: string) {
  const filePath = resolve(layoutDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('Task50.4 no-active ledger shell contract', () => {
  it('offers ledger creation and archived-ledger discovery through global APIs only', () => {
    const source = readSource('./NoActiveLedgerShell.tsx');

    expect(source).toContain('ledgerApi.createLedger');
    expect(source).toContain("ledgerApi.listUserLedgers('archived'");
    expect(source).not.toContain('<Outlet');
    expect(source).not.toMatch(/dashboardApi|transactionsApi|reportsApi|importsApi/);
  });

  it('prevents AppShell from mounting business routes without an active context', () => {
    const source = readSource('./AppShell.tsx');

    expect(source).toContain('<NoActiveLedgerShell');
    expect(source).toContain('canMountBusinessRoutes ? <Outlet key={activeLedgerId} />');
  });

  it('routes existing settings-based ledger creation through the same safe switch path', () => {
    const source = readSource('../ledger/LedgerSettings.tsx');

    expect(source).toContain('switchActiveLedgerContext');
    expect(source).not.toContain('void queryClient.invalidateQueries();');
  });

  it('scopes local transaction form preferences to the active ledger', () => {
    const source = readSource('../transaction/TransactionFormDrawer.tsx');

    expect(source).toContain("const preferenceScope = activeLedgerId || 'no-active-ledger';");
    expect(source).toContain('ledger_two_last_category_id:${preferenceScope}');
    expect(source).toContain('ledger_two_recent_tags:${preferenceScope}');
  });
});
