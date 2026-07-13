export const UI_THEMES = ['dark-glass', 'fresh-light'] as const;

export type UiTheme = (typeof UI_THEMES)[number];

export interface ThemeTarget {
  dataset: { theme?: string };
  style: { colorScheme?: string };
}

export const DEFAULT_UI_THEME: UiTheme = 'dark-glass';

export function resolveTheme(value: string | null | undefined): UiTheme {
  return value === 'fresh-light' || value === 'dark-glass' ? value : DEFAULT_UI_THEME;
}

export function applyTheme(
  theme: UiTheme,
  target: ThemeTarget = document.documentElement,
): UiTheme {
  target.dataset.theme = theme;
  target.style.colorScheme = theme === 'fresh-light' ? 'light' : 'dark';
  return theme;
}
