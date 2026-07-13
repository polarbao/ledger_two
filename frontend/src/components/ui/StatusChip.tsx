import type { HTMLAttributes, ReactNode } from 'react';

export type StatusChipTone = 'neutral' | 'success' | 'info' | 'warning' | 'danger' | 'accent';

export interface StatusChipProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: StatusChipTone;
  icon?: ReactNode;
  children: ReactNode;
}

export default function StatusChip({
  tone = 'neutral',
  icon,
  children,
  className,
  ...spanProps
}: StatusChipProps) {
  const classes = [
    'ui-status-chip',
    `ui-status-chip--${tone}`,
    className ?? '',
  ].filter(Boolean).join(' ');

  return (
    <span {...spanProps} className={classes}>
      {icon ? <span className="ui-status-chip__icon" aria-hidden="true">{icon}</span> : null}
      <span>{children}</span>
    </span>
  );
}
