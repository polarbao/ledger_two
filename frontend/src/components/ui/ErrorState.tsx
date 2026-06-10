import { AlertCircle, RefreshCw } from 'lucide-react';

interface ErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
}

export default function ErrorState({ 
  title = '加载失败', 
  message, 
  onRetry 
}: ErrorStateProps) {
  return (
    <div className="error-banner" style={{ margin: '16px 0', padding: '24px', borderRadius: '16px', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '12px', textAlign: 'center' }}>
      <AlertCircle size={36} style={{ color: '#ef4444' }} />
      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <h4 style={{ margin: 0, fontSize: '15px', fontWeight: 600, color: '#fca5a5' }}>{title}</h4>
        <p style={{ margin: 0, fontSize: '13px', color: 'rgba(252, 165, 165, 0.8)' }}>{message}</p>
      </div>
      {onRetry && (
        <button 
          className="btn-secondary" 
          onClick={onRetry} 
          style={{ 
            marginTop: '8px', 
            padding: '8px 16px', 
            fontSize: '12px', 
            borderRadius: '8px', 
            display: 'flex', 
            alignItems: 'center', 
            gap: '6px',
            background: 'rgba(239, 68, 68, 0.1)',
            borderColor: 'rgba(239, 68, 68, 0.2)',
            color: '#fca5a5'
          }}
        >
          <RefreshCw size={12} />
          立即重试
        </button>
      )}
    </div>
  );
}
