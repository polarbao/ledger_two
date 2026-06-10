interface LoadingSpinnerProps {
  message?: string;
  size?: 'sm' | 'md' | 'lg';
}

export default function LoadingSpinner({ message = '加载中，请稍候...', size = 'md' }: LoadingSpinnerProps) {
  const spinnerSize = size === 'sm' ? '24px' : size === 'lg' ? '56px' : '40px';
  const borderWidth = size === 'sm' ? '2px' : '3px';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: '20px', color: 'var(--text-secondary)' }}>
      <div 
        className="loading-spinner" 
        style={{ 
          width: spinnerSize, 
          height: spinnerSize, 
          borderWidth: borderWidth, 
          margin: '0 auto 12px' 
        }}
      ></div>
      {message && <p style={{ fontSize: '13px', margin: 0 }}>{message}</p>}
    </div>
  );
}
