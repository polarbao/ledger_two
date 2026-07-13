import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const layoutDirectory = dirname(fileURLToPath(import.meta.url));

function readSource(relativePath: string) {
  const filePath = resolve(layoutDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('UI-FL-02 AppShell contract', () => {
  it('keeps exactly five primary destinations in the mobile navigation order', async () => {
    const shellModel = await import('./appShellModel').catch(() => null);

    expect(shellModel).not.toBeNull();
    if (!shellModel) return;

    expect(shellModel.APP_PRIMARY_NAV_ITEMS).toEqual([
      { id: 'dashboard', label: '首页', path: '/' },
      { id: 'transactions', label: '流水', path: '/transactions' },
      { id: 'analytics', label: '分析', path: '/analytics' },
      { id: 'settlement', label: '结算', path: '/settlement' },
      { id: 'settings', label: '设置', path: '/settings' },
    ]);
  });

  it('keeps nested settings routes active without making the dashboard a wildcard', async () => {
    const shellModel = await import('./appShellModel').catch(() => null);

    expect(shellModel).not.toBeNull();
    if (!shellModel) return;

    expect(shellModel.isAppRouteActive('/', '/')).toBe(true);
    expect(shellModel.isAppRouteActive('/transactions', '/')).toBe(false);
    expect(shellModel.isAppRouteActive('/settings/categories', '/settings')).toBe(true);
    expect(shellModel.isAppRouteActive('/settlement', '/settings')).toBe(false);
  });

  it('turns ledger roles into readable labels and protects the write action', async () => {
    const shellModel = await import('./appShellModel').catch(() => null);

    expect(shellModel).not.toBeNull();
    if (!shellModel) return;

    expect(shellModel.getLedgerRoleLabel('owner')).toBe('所有者');
    expect(shellModel.getLedgerRoleLabel('editor')).toBe('可编辑');
    expect(shellModel.getLedgerRoleLabel('viewer')).toBe('只读');
    expect(shellModel.canCreateTransaction('owner')).toBe(true);
    expect(shellModel.canCreateTransaction('editor')).toBe(true);
    expect(shellModel.canCreateTransaction('viewer')).toBe(false);
    expect(shellModel.canCreateTransaction(null)).toBe(false);
  });

  it('exposes semantic navigation, status and record actions in the shell source', () => {
    const shellSource = readSource('./AppShell.tsx');

    expect(shellSource).toContain('aria-label="主导航"');
    expect(shellSource).toContain('aria-current={isActive ? \'page\' : undefined}');
    expect(shellSource).toContain('setAddDrawerOpen(true)');
    expect(shellSource).toContain('aria-label="记一笔"');
    expect(shellSource).toContain('aria-live="polite"');
  });

  it('defines stable desktop and mobile shell geometry without horizontal overflow', () => {
    const shellCss = readSource('./AppShell.css');

    expect(shellCss).toContain('grid-template-columns: 248px minmax(0, 1fr);');
    expect(shellCss).toContain('@media (max-width: 1024px)');
    expect(shellCss).toContain('@media (max-width: 430px)');
    expect(shellCss).toContain('env(safe-area-inset-bottom)');
    expect(shellCss).toContain('min-height: 44px;');
    expect(shellCss).toContain('overflow-x: hidden;');
  });
});
