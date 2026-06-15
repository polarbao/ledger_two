import { useEffect, useState } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuthStore } from '../../stores/auth.store';
import { initApi } from '../../api/init.api';
import { authApi } from '../../api/auth.api';
import { ledgerApi } from '../../api/ledger.api';
import { useLedgerStore } from '../../stores/ledger.store';

export default function AppInitGuard() {
  const { user, isInitialized, setUser, setIsInitialized } = useAuthStore();
  const [loading, setLoading] = useState(true);
  const location = useLocation();

  useEffect(() => {
    async function initApp() {
      try {
        const initStatus = await initApi.getStatus();
        setIsInitialized(initStatus.initialized);

        if (initStatus.initialized) {
          try {
            const me = await authApi.getMe();
            setUser(me);

            const { activeLedgerId, setActiveLedger } = useLedgerStore.getState();
            const ledgers = await ledgerApi.listUserLedgers();
            
            if (ledgers.length > 0) {
              const activeLedger = ledgers.find((l) => l.id === activeLedgerId);
              if (activeLedger) {
                setActiveLedger(activeLedger.id, activeLedger.role);
              } else {
                setActiveLedger(ledgers[0].id, ledgers[0].role);
              }
            }
          } catch {
            setUser(null);
          }
        }
      } catch (err) {
        console.error('App start checklist failed:', err);
      } finally {
        setLoading(false);
      }
    }
    initApp();
  }, [setIsInitialized, setUser]);

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
