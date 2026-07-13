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

  it('keeps Dark Glass as the fallback and applies a valid explicit theme', async () => {
    const themeModule = await import('./theme').catch(() => null);

    expect(themeModule).not.toBeNull();
    if (!themeModule) return;

    expect(themeModule.resolveTheme(null)).toBe('dark-glass');
    expect(themeModule.resolveTheme('unknown')).toBe('dark-glass');
    expect(themeModule.resolveTheme('fresh-light')).toBe('fresh-light');

    const target = { dataset: {}, style: {} };
    themeModule.applyTheme('fresh-light', target);

    expect(target.dataset).toEqual({ theme: 'fresh-light' });
    expect(target.style).toEqual({ colorScheme: 'light' });
  });
});
