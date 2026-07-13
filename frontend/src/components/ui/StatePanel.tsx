import type { ReactNode } from 'react';
import Button, { type ButtonVariant } from './Button';

export type StatePanelTone = 'neutral' | 'info' | 'warning' | 'danger';

export interface StatePanelAction {
  label: string;
  onClick: () => void;
  variant?: ButtonVariant;
}

export interface StatePanelProps {
  title: string;
  description: string;
  tone?: StatePanelTone;
  icon?: ReactNode;
  action?: StatePanelAction;
  className?: string;
}

export default function StatePanel({
  title,
  description,
  tone = 'neutral',
  icon,
  action,
  className,
}: StatePanelProps) {
  const classes = [
    'ui-state-panel',
    `ui-state-panel--${tone}`,
    className ?? '',
  ].filter(Boolean).join(' ');

  return (
    <section className={classes} aria-label={title}>
      {icon ? <div className="ui-state-panel__icon" aria-hidden="true">{icon}</div> : null}
      <h3 className="ui-state-panel__title">{title}</h3>
      <p className="ui-state-panel__description">{description}</p>
      {action ? (
        <Button
          className="ui-state-panel__action"
          variant={action.variant ?? (tone === 'danger' ? 'danger' : 'primary')}
          onClick={action.onClick}
        >
          {action.label}
        </Button>
      ) : null}
    </section>
  );
}
