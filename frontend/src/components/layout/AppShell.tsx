import { useEffect, useState, type ChangeEvent } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BarChart3,
  Calendar,
  ChevronDown,
  CloudOff,
  DollarSign,
  FileUp,
  LayoutDashboard,
  LogOut,
  Plus,
  ReceiptText,
  Repeat2,
  Settings,
  Sparkles,
  Wifi,
  WifiOff,
  type LucideIcon,
} from 'lucide-react';
import { authApi } from '../../api/auth.api';
import { ledgerApi, type LedgerWithRole } from '../../api/ledger.api';
import { queryKeys } from '../../api/queryKeys';
import { useAuthStore } from '../../stores/auth.store';
import { useDraftStore } from '../../stores/draft.store';
import { useLedgerStore } from '../../stores/ledger.store';
import { useUIStore } from '../../stores/ui.store';
import Button from '../ui/Button';
import StatusChip, { type StatusChipTone } from '../ui/StatusChip';
import ThemeToggle from '../theme/ThemeToggle';
import DraftListDrawer from '../transaction/DraftListDrawer';
import TransactionFormDrawer from '../transaction/TransactionFormDrawer';
import DeploymentBadge from './DeploymentBadge';
import {
  APP_PRIMARY_NAV_ITEMS,
  APP_TOOL_NAV_ITEMS,
  canCreateTransaction,
  getLedgerRoleLabel,
  isAppRouteActive,
  shouldShowQuickRecordAction,
  type AppPrimaryNavigationId,
  type AppToolNavigationId,
} from './appShellModel';
import './AppShell.css';

const PRIMARY_NAV_ICONS: Record<AppPrimaryNavigationId, LucideIcon> = {
  dashboard: LayoutDashboard,
  transactions: ReceiptText,
  analytics: BarChart3,
  settlement: DollarSign,
  settings: Settings,
};

const TOOL_NAV_ICONS: Record<AppToolNavigationId, LucideIcon> = {
  import: FileUp,
  recurring: Repeat2,
};

interface LedgerSelectorProps {
  ledgers: LedgerWithRole[];
  activeLedgerId: string | null;
  onChange: (event: ChangeEvent<HTMLSelectElement>) => void;
}

function LedgerSelector({ ledgers, activeLedgerId, onChange }: LedgerSelectorProps) {
  return (
    <label className="lt-shell__ledger-selector">
      <span className="lt-shell__field-label">当前账本</span>
      <span className="lt-shell__select-wrap">
        <select
          className="lt-shell__ledger-select"
          value={activeLedgerId ?? ''}
          onChange={onChange}
          aria-label="当前账本"
          disabled={ledgers.length === 0}
        >
          {ledgers.length === 0 ? <option value="">暂无可用账本</option> : null}
          {ledgers.map((ledger) => (
            <option key={ledger.id} value={ledger.id}>
              {ledger.name} · {getLedgerRoleLabel(ledger.role)}
            </option>
          ))}
        </select>
        <ChevronDown className="lt-shell__select-icon" size={16} aria-hidden="true" />
      </span>
    </label>
  );
}

interface MonthControlProps {
  value: string;
  onChange: (month: string) => void;
}

function MonthControl({ value, onChange }: MonthControlProps) {
  return (
    <label className="lt-shell__month-control">
      <Calendar className="lt-shell__month-icon" size={17} aria-hidden="true" />
      <input
        type="month"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="lt-shell__month-input"
        aria-label="当前月份"
      />
    </label>
  );
}

export default function AppShell() {
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clear);
  const currentMonth = useUIStore((state) => state.currentMonth);
  const setCurrentMonth = useUIStore((state) => state.setCurrentMonth);
  const setAddDrawerOpen = useUIStore((state) => state.setAddDrawerOpen);
  const setCopySourceTransaction = useUIStore((state) => state.setCopySourceTransaction);
  const setEditSourceTransaction = useUIStore((state) => state.setEditSourceTransaction);
  const setEditingDraftId = useUIStore((state) => state.setEditingDraftId);
  const isOffline = useUIStore((state) => state.isOffline);
  const setIsOffline = useUIStore((state) => state.setIsOffline);
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const activeRole = useLedgerStore((state) => state.activeRole);
  const setActiveLedger = useLedgerStore((state) => state.setActiveLedger);
  const drafts = useDraftStore((state) => state.drafts);
  const [isDraftListOpen, setIsDraftListOpen] = useState(false);
  const { data: ledgers = [] } = useQuery({
    queryKey: queryKeys.ledgers.all,
    queryFn: ledgerApi.listUserLedgers,
    enabled: !!user,
  });

  const activeLedger = ledgers.find((ledger) => ledger.id === activeLedgerId);
  const canWriteLedger = canCreateTransaction(activeRole);
  const showQuickRecordAction = shouldShowQuickRecordAction(location.pathname);
  const recordActionTitle = canWriteLedger ? '记一笔' : '当前账本为只读，无法记账';
  const networkTone: StatusChipTone = isOffline ? 'danger' : drafts.length > 0 ? 'info' : 'success';
  const networkLabel = isOffline ? '网络离线' : drafts.length > 0 ? `${drafts.length} 条草稿` : '网络在线';

  useEffect(() => {
    const handleOnline = () => setIsOffline(false);
    const handleOffline = () => setIsOffline(true);
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, [setIsOffline]);

  useEffect(() => {
    if (ledgers.length === 0) return;
    if (!activeLedger) {
      const firstLedger = ledgers[0];
      setActiveLedger(firstLedger.id, firstLedger.role);
      return;
    }
    if (activeLedger.role !== activeRole) {
      setActiveLedger(activeLedger.id, activeLedger.role);
    }
  }, [activeLedger, activeRole, ledgers, setActiveLedger]);

  const handleLedgerChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const nextLedger = ledgers.find((ledger) => ledger.id === event.target.value);
    if (nextLedger && nextLedger.id !== activeLedgerId) {
      setActiveLedger(nextLedger.id, nextLedger.role);
      queryClient.invalidateQueries();
    }
  };

  const openTransactionForm = () => {
    if (!canWriteLedger) return;
    setCopySourceTransaction(null);
    setEditSourceTransaction(null);
    setEditingDraftId(null);
    setAddDrawerOpen(true);
  };

  const handleLogout = async () => {
    try {
      await authApi.logout();
    } catch {
      // Local sign-out must remain available when the network is unavailable.
    } finally {
      clearAuth();
      navigate('/login');
    }
  };

  return (
    <div className="lt-shell">
      <aside className="lt-shell__sidebar" aria-label="应用侧栏">
        <div className="lt-shell__brand">
          <span className="lt-shell__brand-mark"><Sparkles size={20} aria-hidden="true" /></span>
          <div className="lt-shell__brand-copy">
            <span className="lt-shell__brand-name">LedgerTwo</span>
            <DeploymentBadge />
          </div>
        </div>

        <LedgerSelector ledgers={ledgers} activeLedgerId={activeLedgerId} onChange={handleLedgerChange} />

        <Button
          className="lt-shell__record-button"
          variant="primary"
          startIcon={<Plus size={18} />}
          disabled={!canWriteLedger}
          title={recordActionTitle}
          aria-label="记一笔"
          onClick={openTransactionForm}
        >
          记一笔
        </Button>

        <nav className="lt-shell__nav" aria-label="主导航">
          {APP_PRIMARY_NAV_ITEMS.map((item) => {
            const Icon = PRIMARY_NAV_ICONS[item.id];
            const isActive = isAppRouteActive(location.pathname, item.path);
            return (
              <Link
                key={item.id}
                to={item.path}
                className="lt-shell__nav-link"
                aria-current={isActive ? 'page' : undefined}
              >
                <Icon size={19} aria-hidden="true" />
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>

        <nav className="lt-shell__tools" aria-label="账本工具">
          <span className="lt-shell__tools-label">工具</span>
          {APP_TOOL_NAV_ITEMS.map((item) => {
            const Icon = TOOL_NAV_ICONS[item.id];
            const isActive = isAppRouteActive(location.pathname, item.path);
            return (
              <Link
                key={item.id}
                to={item.path}
                className="lt-shell__nav-link"
                aria-current={isActive ? 'page' : undefined}
              >
                <Icon size={19} aria-hidden="true" />
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>

        <div className="lt-shell__footer">
          <div className="lt-shell__user">
            <span className="lt-shell__avatar" aria-hidden="true">
              {user?.display_name?.charAt(0) || 'U'}
            </span>
            <div className="lt-shell__user-copy">
              <span className="lt-shell__user-name">{user?.display_name || '当前用户'}</span>
              <span className="lt-shell__user-handle">@{user?.username || 'unknown'}</span>
            </div>
          </div>
          <button type="button" className="lt-shell__logout" onClick={handleLogout}>
            <LogOut size={18} aria-hidden="true" />
            <span>退出登录</span>
          </button>
        </div>
      </aside>

      <main className="lt-shell__main">
        {isOffline ? (
          <div className="lt-shell__notice lt-shell__notice--offline" role="status" aria-live="polite">
            <WifiOff size={17} aria-hidden="true" />
            <span>当前网络已离线，未提交内容会保存在本机草稿中。</span>
            {drafts.length > 0 ? (
              <button type="button" className="lt-shell__notice-action" onClick={() => setIsDraftListOpen(true)}>
                查看 {drafts.length} 条草稿
              </button>
            ) : null}
          </div>
        ) : drafts.length > 0 ? (
          <div className="lt-shell__notice lt-shell__notice--drafts" role="status" aria-live="polite">
            <CloudOff size={17} aria-hidden="true" />
            <span>有 {drafts.length} 条离线草稿待处理。</span>
            <button type="button" className="lt-shell__notice-action" onClick={() => setIsDraftListOpen(true)}>
              打开草稿箱
            </button>
          </div>
        ) : null}

        <header className="lt-shell__topbar">
          <div className="lt-shell__desktop-context">
            <div className="lt-shell__ledger-summary">
              <span className="lt-shell__context-label">当前账本</span>
              <div className="lt-shell__ledger-summary-row">
                <strong className="lt-shell__ledger-name">{activeLedger?.name || '正在加载账本'}</strong>
                <StatusChip tone="neutral">{getLedgerRoleLabel(activeRole)}</StatusChip>
              </div>
            </div>
            <MonthControl value={currentMonth} onChange={setCurrentMonth} />
          </div>

          <div className="lt-shell__topbar-actions">
            <StatusChip tone={networkTone} icon={isOffline ? <WifiOff size={14} /> : <Wifi size={14} />}>
              {networkLabel}
            </StatusChip>
            {drafts.length > 0 ? (
              <Button
                variant="ghost"
                iconOnly
                aria-label={`打开草稿箱，共 ${drafts.length} 条`}
                title={`草稿箱，共 ${drafts.length} 条`}
                onClick={() => setIsDraftListOpen(true)}
              >
                <span className="lt-shell__draft-button-wrap">
                  <CloudOff size={20} aria-hidden="true" />
                  <span className="lt-shell__draft-count">{drafts.length}</span>
                </span>
              </Button>
            ) : null}
            <ThemeToggle className="lt-shell__theme-toggle" />
            <span className="lt-shell__welcome">你好，<strong>{user?.display_name || '用户'}</strong></span>
          </div>

          <div className="lt-shell__mobile-context">
            <div className="lt-shell__mobile-summary">
              <div className="lt-shell__mobile-brand">
                <span className="lt-shell__brand-mark"><Sparkles size={18} aria-hidden="true" /></span>
                <div className="lt-shell__mobile-brand-copy">
                  <span className="lt-shell__brand-name">LedgerTwo</span>
                  <DeploymentBadge />
                </div>
              </div>
              <div className="lt-shell__mobile-status-actions">
                <StatusChip tone={networkTone} icon={isOffline ? <WifiOff size={14} /> : <Wifi size={14} />}>
                  {networkLabel}
                </StatusChip>
                <ThemeToggle className="lt-shell__theme-toggle" />
              </div>
            </div>
            <div className="lt-shell__mobile-controls">
              <LedgerSelector ledgers={ledgers} activeLedgerId={activeLedgerId} onChange={handleLedgerChange} />
              <MonthControl value={currentMonth} onChange={setCurrentMonth} />
            </div>
          </div>
        </header>

        <div className="lt-shell__page">
          <Outlet />
        </div>
      </main>

      {showQuickRecordAction ? (
        <Button
          className="lt-shell__fab"
          variant="primary"
          startIcon={<Plus size={19} />}
          disabled={!canWriteLedger}
          title={recordActionTitle}
          aria-label="记一笔"
          onClick={openTransactionForm}
        >
          记一笔
        </Button>
      ) : null}

      <nav className="lt-shell__bottom-nav" aria-label="主导航">
        {APP_PRIMARY_NAV_ITEMS.map((item) => {
          const Icon = PRIMARY_NAV_ICONS[item.id];
          const isActive = isAppRouteActive(location.pathname, item.path);
          return (
            <Link
              key={item.id}
              to={item.path}
              className="lt-shell__bottom-link"
              aria-current={isActive ? 'page' : undefined}
            >
              <Icon size={21} aria-hidden="true" />
              <span className="lt-shell__bottom-label">{item.label}</span>
            </Link>
          );
        })}
      </nav>

      <TransactionFormDrawer />
      <DraftListDrawer open={isDraftListOpen} onClose={() => setIsDraftListOpen(false)} />
    </div>
  );
}
