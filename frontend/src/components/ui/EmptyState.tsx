import React from 'react';
import { Inbox } from 'lucide-react';

interface EmptyStateProps {
  title?: string;
  description: string;
  actionText?: string;
  onAction?: () => void;
  icon?: React.ReactNode;
}

export default function EmptyState({ 
  title = '暂无数据', 
  description, 
  actionText, 
  onAction, 
  icon 
}: EmptyStateProps) {
  return (
    <div className="page-state-container glass-card" style={{ padding: '32px', margin: '10px 0' }}>
      <div className="page-state-icon">
        {icon || <Inbox size={40} style={{ color: 'var(--text-muted)' }} />}
      </div>
      <h3 className="page-state-title">{title}</h3>
      <p className="page-state-desc">{description}</p>
      {actionText && onAction && (
        <button className="btn-primary" onClick={onAction} style={{ padding: '8px 20px', fontSize: '14px', borderRadius: '10px' }}>
          {actionText}
        </button>
      )}
    </div>
  );
}
