import { useEffect, useState } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '../../stores/auth.store';
import { initApi } from '../../api/init.api';
import { authApi } from '../../api/auth.api';
import { ledgerApi } from '../../api/ledger.api';
import { queryKeys } from '../../api/queryKeys';
import { useLedgerStore } from '../../stores/ledger.store';

export default function AppInitGuard() {
  const { user, isInitialized, setUser, setIsInitialized } = useAuthStore();
  const [loading, setLoading] = useState(true);
  const location = useLocation();
  const queryClient = useQueryClient();

  useEffect(() => {
    const controller = new AbortController();

    async function initApp() {
      try {
        const initStatus = await initApi.getStatus();
        setIsInitialized(initStatus.initialized);

        if (initStatus.initialized) {
          try {
            const me = await authApi.getMe();
            setUser(me);
          } catch {
            setUser(null);
            return;
          }

          const ledgerState = useLedgerStore.getState();
          ledgerState.beginLedgerValidation();
          try {
            const ledgers = await ledgerApi.listUserLedgers('active', controller.signal);
            queryClient.setQueryData(queryKeys.ledgers.list('active'), ledgers);
            useLedgerStore.getState().reconcileActiveLedgers(ledgers);
          } catch (error) {
            if (!controller.signal.aborted) {
              queryClient.removeQueries({
                queryKey: queryKeys.ledgers.list('active'),
                exact: true,
              });
              useLedgerStore.getState().failLedgerValidation(
                error instanceof Error ? error.message : '账本列表加载失败',
              );
            }
          }
        }
      } catch (err) {
        console.error('App start checklist failed:', err);
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }
    void initApp();

    return () => controller.abort();
  }, [queryClient, setIsInitialized, setUser]);

  if (loading) {
    return (
      <div className="app-loading">
        <div className="loading-spinner"></div>
        <p>系统准备中，请稍候...</p>
      </div>
    );
  }

  if (!isInitialized) {
    if (location.pathname !== '/init') {
      return <Navigate to="/init" replace />;
    }
    return <Outlet />;
  }

  if (isInitialized && location.pathname === '/init') {
    return <Navigate to="/login" replace />;
  }

  const isLoggedIn = !!user;

  if (!isLoggedIn) {
    if (location.pathname !== '/login') {
      return <Navigate to="/login" replace />;
    }
    return <Outlet />;
  }

  if (isLoggedIn && location.pathname === '/login') {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
