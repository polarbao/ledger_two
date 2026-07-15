import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const srcRoot = resolve(process.cwd(), 'src');

function readSource(relativePath: string) {
  return readFileSync(resolve(srcRoot, relativePath), 'utf8');
}

describe('UI-FL-10 global experience contract', () => {
  it('makes Fresh Light the new-session default while retaining Dark Glass support', () => {
    const theme = readSource('theme/theme.ts');

    expect(theme).toContain("DEFAULT_UI_THEME: UiTheme = 'fresh-light'");
    expect(theme).toContain("value === 'fresh-light' || value === 'dark-glass'");
  });

  it('provides a keyboard skip target and route focus landing point', () => {
    const shell = readSource('components/layout/AppShell.tsx');
    const shellCss = readSource('components/layout/AppShell.css');

    expect(shell).toContain('href="#main-content"');
    expect(shell).toContain('id="main-content"');
    expect(shell).toContain('tabIndex={-1}');
    expect(shell).toContain('pageRef.current?.focus');
    expect(shell).toContain('}, [location.pathname]);');
    expect(shell).not.toContain('[location.pathname, location.search]');
    expect(shellCss).toContain('.lt-shell__skip-link:focus');
  });

  it('keeps authentication controls keyboard and screen-reader accessible', () => {
    const login = readSource('pages/LoginPage.tsx');
    const setup = readSource('pages/InitPage.tsx');

    expect(login).toContain('aria-label={showPassword');
    expect(login).toContain('aria-pressed={showPassword}');
    expect(login).not.toContain('tabIndex={-1}');
    expect(login).toContain('role="alert"');
    expect(setup).toContain('htmlFor="setup-ledger-name"');
    expect(setup).toContain('aria-invalid={Boolean(errors.user_b_password)}');
  });

  it('uses shared modal primitives for drafts and recurring-rule deletion', () => {
    const drafts = readSource('components/transaction/DraftListDrawer.tsx');
    const recurring = readSource('pages/RecurringRulesPage.tsx');

    expect(drafts).toContain('<BottomSheet');
    expect(drafts).toContain('<ConfirmDialog');
    expect(drafts).not.toContain('confirm(');
    expect(drafts).not.toMatch(/#[0-9a-f]{3,8}|rgba\(/i);
    expect(recurring).toContain('<ConfirmDialog');
    expect(recurring).toContain('<SegmentedControl');
    expect(recurring).not.toMatch(/#[0-9a-f]{3,8}|rgba\(/i);
  });

  it('provides semantic Fresh Light compatibility and reduced-motion fallback', () => {
    const tokens = readSource('styles/tokens.css');
    const globalCss = readSource('index.css');

    expect(tokens).toContain("[data-theme='fresh-light'] .glass-card");
    expect(tokens).toContain("[data-theme='fresh-light'] .btn-primary");
    expect(tokens).toContain("[data-theme='fresh-light'] .form-group input");
    expect(tokens).toContain("[data-theme='fresh-light'] .form-select");
    expect(tokens).toContain("[data-theme='fresh-light'] .login-header h1");
    expect(tokens).toContain('-webkit-text-fill-color: currentColor;');
    expect(globalCss).toContain('@media (prefers-reduced-motion: reduce)');
    expect(globalCss).toContain('animation-duration: 1ms !important;');
  });
});
