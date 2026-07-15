export const UI_THEMES = ['dark-glass', 'fresh-light'] as const;

export type UiTheme = (typeof UI_THEMES)[number];

export interface ThemeTarget {
  dataset: { theme?: string };
  style: { colorScheme?: string };
}

export const DEFAULT_UI_THEME: UiTheme = 'fresh-light';
export const UI_THEME_STORAGE_KEY = 'ledger-two-ui-theme';

export function resolveTheme(value: string | null | undefined): UiTheme {
  return value === 'fresh-light' || value === 'dark-glass' ? value : DEFAULT_UI_THEME;
}

export function resolveInitialTheme(
  storedValue: string | null | undefined,
  documentValue: string | null | undefined,
): UiTheme {
  if (storedValue === 'fresh-light' || storedValue === 'dark-glass') {
    return storedValue;
  }
  return resolveTheme(documentValue);
}

export function applyTheme(
  theme: UiTheme,
  target: ThemeTarget = document.documentElement,
): UiTheme {
  target.dataset.theme = theme;
  target.style.colorScheme = theme === 'fresh-light' ? 'light' : 'dark';
  return theme;
}

export function readStoredTheme(storage: Pick<Storage, 'getItem'> = window.localStorage): string | null {
  try {
    return storage.getItem(UI_THEME_STORAGE_KEY);
  } catch {
    return null;
  }
}

export function persistTheme(theme: UiTheme, storage: Pick<Storage, 'setItem'> = window.localStorage): UiTheme {
  try {
    storage.setItem(UI_THEME_STORAGE_KEY, theme);
  } catch {
    // Theme switching remains usable when browser storage is unavailable.
  }
  return theme;
}
