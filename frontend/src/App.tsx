import { RouterProvider } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { router } from './routes';
import './App.css';

// 初始化 React Query 客户端，并进行基础配置
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false, // 禁用聚焦自动刷新
      retry: 1, // 失败重试 1 次
      staleTime: 5 * 60 * 1000, // 默认缓存时间 5 分钟
    },
  },
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}
