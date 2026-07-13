interface LoadingSpinnerProps {
  message?: string;
  size?: 'sm' | 'md' | 'lg';
}

export default function LoadingSpinner({ message = '加载中，请稍候...', size = 'md' }: LoadingSpinnerProps) {
  return (
    <div className="ui-loading-state" role="status" aria-live="polite">
      <span className={`ui-loading-spinner ui-loading-spinner--${size}`} aria-hidden="true" />
      {message && <p className="ui-loading-state__message">{message}</p>}
    </div>
  );
}
