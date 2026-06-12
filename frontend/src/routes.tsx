import { createBrowserRouter, Navigate } from 'react-router-dom';

// 页面组件
import InitPage from './pages/InitPage';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import TransactionsPage from './pages/TransactionsPage';
import SettlementPage from './pages/SettlementPage';
import AnalyticsPage from './pages/AnalyticsPage';
import SettingsPage from './pages/SettingsPage';
import RecurringRulesPage from './pages/RecurringRulesPage';
import ImportPage from './pages/ImportPage';
import AppShell from './components/layout/AppShell';
import AppInitGuard from './components/layout/AppInitGuard';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppInitGuard />,
    children: [
      {
        path: 'init',
        element: <InitPage />,
      },
      {
        path: 'login',
        element: <LoginPage />,
      },
      {
        path: '/',
        element: <AppShell />,
        children: [
          {
            index: true,
            element: <DashboardPage />,
          },
          {
            path: 'transactions',
            element: <TransactionsPage />,
          },
          {
            path: 'settlement',
            element: <SettlementPage />,
          },
          {
            path: 'analytics',
            element: <AnalyticsPage />,
          },
          {
            path: 'settings',
            element: <SettingsPage />,
          },
          {
            path: 'recurring-rules',
            element: <RecurringRulesPage />,
          },
          {
            path: 'import',
            element: <ImportPage />,
          },
        ],
      },
      {
        path: '*',
        element: <Navigate to="/" replace />,
      },
    ],
  },
]);
