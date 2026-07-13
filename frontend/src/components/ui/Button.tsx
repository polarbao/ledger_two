import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react';

export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost';
export type ButtonSize = 'md' | 'lg';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  isLoading?: boolean;
  fullWidth?: boolean;
  iconOnly?: boolean;
  startIcon?: ReactNode;
  endIcon?: ReactNode;
}

const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button({
  variant = 'secondary',
  size = 'md',
  isLoading = false,
  fullWidth = false,
  iconOnly = false,
  startIcon,
  endIcon,
  className,
  children,
  disabled,
  type = 'button',
  ...buttonProps
}, ref) {
  const classes = [
    'ui-button',
    `ui-button--${variant}`,
    `ui-button--${size}`,
    fullWidth ? 'ui-button--full-width' : '',
    iconOnly ? 'ui-button--icon-only' : '',
    className ?? '',
  ].filter(Boolean).join(' ');

  return (
    <button
      {...buttonProps}
      ref={ref}
      type={type}
      className={classes}
      disabled={disabled || isLoading}
      aria-busy={isLoading || undefined}
    >
      {isLoading ? (
        <span className="ui-button__spinner" aria-hidden="true" />
      ) : startIcon ? (
        <span className="ui-button__icon" aria-hidden="true">{startIcon}</span>
      ) : null}
      {children ? <span className="ui-button__label">{children}</span> : null}
      {!isLoading && endIcon ? (
        <span className="ui-button__icon" aria-hidden="true">{endIcon}</span>
      ) : null}
    </button>
  );
});

export default Button;
