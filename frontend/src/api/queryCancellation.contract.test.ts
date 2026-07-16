import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const apiDirectory = dirname(fileURLToPath(import.meta.url));
const sourceDirectory = resolve(apiDirectory, '..');

function readSource(relativePath: string) {
  return readFileSync(resolve(sourceDirectory, relativePath), 'utf8');
}

describe('Task50.4 query cancellation contract', () => {
  it('lets ledger-scoped read APIs receive an AbortSignal', () => {
    expect(readSource('api/dashboard.api.ts')).toContain('signal?: AbortSignal');
    expect(readSource('api/transactions.api.ts')).toContain('signal?: AbortSignal');
    expect(readSource('api/reports.api.ts')).toContain('signal?: AbortSignal');
    expect(readSource('api/settlement.api.ts')).toContain('signal?: AbortSignal');
    expect(readSource('api/metadata.api.ts')).toContain('signal?: AbortSignal');
    expect(readSource('api/imports.api.ts')).toContain('signal?: AbortSignal');
  });

  it('passes TanStack Query signals from business pages into read APIs', () => {
    expect(readSource('pages/DashboardPage.tsx')).toContain('queryFn: ({ signal })');
    expect(readSource('pages/TransactionsPage.tsx')).toContain('queryFn: ({ signal })');
    expect(readSource('pages/AnalyticsPage.tsx')).toContain('queryFn: ({ signal })');
    expect(readSource('pages/SettlementPage.tsx')).toContain('queryFn: ({ signal })');
    expect(readSource('pages/ImportPage.tsx')).toContain('queryFn: ({ signal })');
    expect(readSource('components/transaction/TransactionFormDrawer.tsx'))
      .toContain('queryFn: ({ signal })');
  });
});
