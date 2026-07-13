export interface SegmentedControlOption<T extends string> {
  value: T;
  label: string;
  count?: number;
  disabled?: boolean;
}

export interface SegmentedControlProps<T extends string> {
  ariaLabel: string;
  value: T;
  options: readonly SegmentedControlOption<T>[];
  onChange: (value: T) => void;
  fullWidth?: boolean;
  className?: string;
}

export default function SegmentedControl<T extends string>({
  ariaLabel,
  value,
  options,
  onChange,
  fullWidth = false,
  className,
}: SegmentedControlProps<T>) {
  const classes = [
    'ui-segmented-control',
    fullWidth ? 'ui-segmented-control--full-width' : '',
    className ?? '',
  ].filter(Boolean).join(' ');

  return (
    <div className={classes} role="group" aria-label={ariaLabel}>
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          className="ui-segmented-control__option"
          aria-pressed={option.value === value}
          disabled={option.disabled}
          onClick={() => onChange(option.value)}
        >
          <span>{option.label}</span>
          {option.count !== undefined ? (
            <span className="ui-segmented-control__count" aria-label={`${option.count} 项`}>
              {option.count}
            </span>
          ) : null}
        </button>
      ))}
    </div>
  );
}
