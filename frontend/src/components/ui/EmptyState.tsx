import React from 'react';
import { Inbox } from 'lucide-react';
import StatePanel from './StatePanel';

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
    <StatePanel
      title={title}
      description={description}
      icon={icon || <Inbox size={40} />}
      action={actionText && onAction ? { label: actionText, onClick: onAction } : undefined}
    />
  );
}
