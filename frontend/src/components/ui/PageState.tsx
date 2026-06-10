import React from 'react';
import LoadingSpinner from './LoadingSpinner';
import EmptyState from './EmptyState';
import ErrorState from './ErrorState';
import SkeletonCard from './SkeletonCard';
import SkeletonTable from './SkeletonTable';

interface PageStateProps {
  isLoading: boolean;
  isError: boolean;
  isEmpty?: boolean;
  errorMsg?: string;
  emptyMessage?: string;
  loadingMessage?: string;
  skeletonType?: 'card' | 'table' | 'spinner';
  onRetry?: () => void;
  children: React.ReactNode;
}

export default function PageState({
  isLoading,
  isError,
  isEmpty = false,
  errorMsg = '获取数据失败，请检查网络或重试。',
  emptyMessage = '暂无相关账目数据。',
  loadingMessage,
  skeletonType = 'spinner',
  onRetry,
  children
}: PageStateProps) {
  if (isLoading) {
    if (skeletonType === 'card') {
      return <SkeletonCard count={2} />;
    } else if (skeletonType === 'table') {
      return <SkeletonTable rows={5} />;
    }
    return <LoadingSpinner message={loadingMessage} />;
  }

  if (isError) {
    return <ErrorState message={errorMsg} onRetry={onRetry} />;
  }

  if (isEmpty) {
    return <EmptyState description={emptyMessage} />;
  }

  return <>{children}</>;
}
