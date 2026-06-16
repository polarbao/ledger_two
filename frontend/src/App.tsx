import { RouterProvider } from 'react-router-dom';
import { QueryClient } from '@tanstack/react-query';
import { PersistQueryClientProvider } from '@tanstack/react-query-persist-client';
import { createSyncStoragePersister } from '@tanstack/query-sync-storage-persister';
import { router } from './routes';
import './App.css';

// 初始化 React Query 客户端，并进行基础配置
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false, // 禁用聚焦自动刷新
      retry: 1, // 失败重试 1 次
      staleTime: 5 * 60 * 1000, // 默认缓存时间 5 分钟
      gcTime: 24 * 60 * 60 * 1000, // 数据垃圾回收前保留 24 小时（为离线模式提供缓存存活基础）
    },
  },
});

// 使用 localStorage 创建同步持久化器
const persister = createSyncStoragePersister({
  storage: window.localStorage,
});

export default function App() {
  return (
    <PersistQueryClientProvider 
      client={queryClient} 
      persistOptions={{ persister }}
    >
      <RouterProvider router={router} />
    </PersistQueryClientProvider>
  );
}
