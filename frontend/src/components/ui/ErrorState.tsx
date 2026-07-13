import { AlertCircle } from 'lucide-react';
import StatePanel from './StatePanel';

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
    <StatePanel
      tone="danger"
      title={title}
      description={message}
      icon={<AlertCircle size={36} />}
      action={onRetry ? {
        label: '立即重试',
        onClick: onRetry,
        variant: 'secondary',
      } : undefined}
    />
  );
}
