import { useState } from 'react';
import { Moon, Sun } from 'lucide-react';
import { applyTheme, persistTheme, resolveTheme, type UiTheme } from '../../theme/theme';
import Button from '../ui/Button';
import './ThemeToggle.css';

interface ThemeToggleProps {
  className?: string;
}

export default function ThemeToggle({ className }: ThemeToggleProps) {
  const [uiTheme, setUiTheme] = useState<UiTheme>(() => resolveTheme(document.documentElement.dataset.theme));
  const nextTheme = uiTheme === 'fresh-light' ? 'dark-glass' : 'fresh-light';
  const actionLabel = nextTheme === 'fresh-light' ? '切换到白绿色浅色主题' : '切换到深色主题';

  const handleThemeToggle = () => {
    setUiTheme(nextTheme);
    applyTheme(nextTheme);
    persistTheme(nextTheme);
  };

  return (
    <Button
      className={['ui-theme-toggle', className].filter(Boolean).join(' ')}
      variant="ghost"
      iconOnly
      aria-label={actionLabel}
      aria-pressed={uiTheme === 'fresh-light'}
      title={actionLabel}
      onClick={handleThemeToggle}
    >
      {nextTheme === 'fresh-light'
        ? <Sun size={19} aria-hidden="true" />
        : <Moon size={19} aria-hidden="true" />}
    </Button>
  );
}
