import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const themeDirectory = dirname(fileURLToPath(import.meta.url));

function readSource(relativePath: string) {
  const filePath = resolve(themeDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('UI-FL-01 theme foundation', () => {
  it('defines Fresh Light and Dark Glass semantic token modes with legacy aliases', () => {
    const tokenCss = readSource('../styles/tokens.css');

    expect(tokenCss).toContain("[data-theme='dark-glass']");
    expect(tokenCss).toContain("[data-theme='fresh-light']");
    expect(tokenCss).toContain('--lt-bg-page:');
    expect(tokenCss).toContain('--lt-focus-ring:');
    expect(tokenCss).toContain('--bg-primary: var(--lt-bg-page);');
    expect(tokenCss).toContain('--accent-danger: var(--lt-danger);');
  });

  it('loads the token and primitive styles before the application renders', () => {
    const mainSource = readSource('../main.tsx');

    expect(mainSource).toContain("import './styles/tokens.css'");
    expect(mainSource).toContain("import './styles/ui-primitives.css'");
  });

  it('uses dedicated accessible action colors instead of raw accent colors', () => {
    const tokenCss = readSource('../styles/tokens.css');
    const primitiveCss = readSource('../styles/ui-primitives.css');

    expect(tokenCss).toContain('--lt-action-primary-bg:');
    expect(tokenCss).toContain('--lt-action-danger-bg:');
    expect(primitiveCss).toContain('background: var(--lt-action-primary-bg);');
    expect(primitiveCss).toContain('background: var(--lt-action-danger-bg);');
  });

  it('uses Fresh Light as the fallback and preserves an explicit Dark Glass choice', async () => {
    const themeModule = await import('./theme').catch(() => null);

    expect(themeModule).not.toBeNull();
    if (!themeModule) return;

    expect(themeModule.resolveTheme(null)).toBe('fresh-light');
    expect(themeModule.resolveTheme('unknown')).toBe('fresh-light');
    expect(themeModule.resolveTheme('fresh-light')).toBe('fresh-light');
    expect(themeModule.resolveInitialTheme('dark-glass', 'fresh-light')).toBe('dark-glass');
    expect(themeModule.resolveInitialTheme('fresh-light', 'dark-glass')).toBe('fresh-light');
    expect(themeModule.resolveInitialTheme(null, 'fresh-light')).toBe('fresh-light');

    const target = { dataset: {}, style: {} };
    themeModule.applyTheme('fresh-light', target);

    expect(target.dataset).toEqual({ theme: 'fresh-light' });
    expect(target.style).toEqual({ colorScheme: 'light' });
  });

  it('persists the explicit user choice and wires the shell toggle', async () => {
    const themeModule = await import('./theme').catch(() => null);
    expect(themeModule).not.toBeNull();
    if (!themeModule) return;

    const stored = new Map<string, string>();
    const storage = {
      getItem: (key: string) => stored.get(key) || null,
      setItem: (key: string, value: string) => stored.set(key, value),
    };
    themeModule.persistTheme('fresh-light', storage);
    expect(themeModule.readStoredTheme(storage)).toBe('fresh-light');

    const toggleSource = readSource('../components/theme/ThemeToggle.tsx');
    const shellSource = readSource('../components/layout/AppShell.tsx');
    expect(toggleSource).toContain('handleThemeToggle');
    expect(toggleSource).toContain('切换到白绿色浅色主题');
    expect(toggleSource).toContain('aria-pressed={uiTheme');
    expect(shellSource).toContain('<ThemeToggle');
  });
});
