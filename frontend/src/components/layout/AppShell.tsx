import { useEffect, useRef, useState } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle,
  Archive,
  BarChart3,
  BookOpen,
  Calendar,
  CloudOff,
  DollarSign,
  FileUp,
  LayoutDashboard,
  LogOut,
  Plus,
  ReceiptText,
  RefreshCw,
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
import { selectLedgerDrafts } from '../../stores/draftLedgerModel';
import { useLedgerStore } from '../../stores/ledger.store';
import { useUIStore } from '../../stores/ui.store';
import ArchivedLedgerBanner from '../ledger/ArchivedLedgerBanner';
import LedgerActionSurface from '../ledger/LedgerActionSurface';
import { getLedgerErrorPresentation } from '../ledger/ledgerManagementModel';
import Button from '../ui/Button';
import StatusChip, { type StatusChipTone } from '../ui/StatusChip';
import ThemeToggle from '../theme/ThemeToggle';
import DraftListDrawer from '../transaction/DraftListDrawer';
import TransactionFormDrawer from '../transaction/TransactionFormDrawer';
import StatePanel from '../ui/StatePanel';
import DeploymentBadge from './DeploymentBadge';
import LedgerSwitcher from './LedgerSwitcher';
import NoActiveLedgerShell from './NoActiveLedgerShell';
import {
  APP_PRIMARY_NAV_ITEMS,
  APP_TOOL_NAV_ITEMS,
  buildShellNavigationPath,
  canCreateTransaction,
  getLedgerRoleLabel,
  isArchivedContextCleanupPending,
  isArchivedHistoryRoute,
  isAppRouteActive,
  isLedgerManagementRoute,
  shouldShowQuickRecordAction,
  type AppPrimaryNavigationId,
  type AppToolNavigationId,
} from './appShellModel';
import { switchActiveLedgerContext } from './ledgerContextModel';
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
  const recentLedgerUsedAt = useLedgerStore((state) => state.recentLedgerUsedAt);
  const contextStatus = useLedgerStore((state) => state.contextStatus);
  const contextNotice = useLedgerStore((state) => state.contextNotice);
  const validationError = useLedgerStore((state) => state.validationError);
  const setActiveLedger = useLedgerStore((state) => state.setActiveLedger);
  const reconcileActiveLedgers = useLedgerStore((state) => state.reconcileActiveLedgers);
  const beginLedgerValidation = useLedgerStore((state) => state.beginLedgerValidation);
  const failLedgerValidation = useLedgerStore((state) => state.failLedgerValidation);
  const clearContextNotice = useLedgerStore((state) => state.clearContextNotice);
  const archivedViewingLedger = useLedgerStore((state) => state.archivedViewingLedger);
  const enterArchivedLedgerView = useLedgerStore((state) => state.enterArchivedLedgerView);
  const exitArchivedLedgerView = useLedgerStore((state) => state.exitArchivedLedgerView);
  const allDrafts = useDraftStore((state) => state.drafts);
  const drafts = selectLedgerDrafts(allDrafts, activeLedgerId);
  const [isDraftListOpen, setIsDraftListOpen] = useState(false);
  const [isSwitchingLedger, setIsSwitchingLedger] = useState(false);
  const [switchMessage, setSwitchMessage] = useState<string | null>(null);
  const [restoreConfirmOpen, setRestoreConfirmOpen] = useState(false);
  const pageRef = useRef<HTMLDivElement>(null);
  const activeLedgersQuery = useQuery({
    queryKey: queryKeys.ledgers.list('active'),
    queryFn: ({ signal }) => ledgerApi.listUserLedgers('active', signal),
    enabled: !!user,
  });
  const archivedLedgersQuery = useQuery({
    queryKey: queryKeys.ledgers.list('archived'),
    queryFn: ({ signal }) => ledgerApi.listUserLedgers('archived', signal),
    enabled: !!user,
    staleTime: 0,
  });
  const ledgers = activeLedgersQuery.data ?? [];
  const archivedRequestedId = isArchivedHistoryRoute(location.pathname)
    ? new URLSearchParams(location.search).get('archived_ledger_id')
    : null;
  const requestedArchivedLedger = archivedLedgersQuery.data?.find(
    (ledger) => ledger.id === archivedRequestedId,
  ) ?? null;
  const activeLedger = ledgers.find((ledger) => ledger.id === activeLedgerId);
  const archivedViewReady = Boolean(
    archivedRequestedId
    && requestedArchivedLedger
    && archivedViewingLedger?.id === archivedRequestedId,
  );
  const activeContextReady = contextStatus === 'active' && Boolean(activeLedger);
  const archivedContextCleanupPending = isArchivedContextCleanupPending(
    archivedViewingLedger?.id,
    archivedRequestedId,
  );
  const activeBusinessContextReady = activeContextReady && !archivedContextCleanupPending;
  const ledgerContextReady = archivedViewReady || activeBusinessContextReady;
  const globalLedgerRoute = isLedgerManagementRoute(location.pathname);
  const canMountOutlet = globalLedgerRoute || ledgerContextReady;
  const currentLedger = archivedViewReady ? requestedArchivedLedger : activeLedger;
  const currentRole = archivedViewReady ? requestedArchivedLedger?.role : activeRole;
  const canWriteLedger = activeBusinessContextReady && !archivedViewReady && canCreateTransaction(activeRole);
  const ledgerListRefreshError = activeLedgersQuery.isError && activeContextReady
    ? activeLedgersQuery.error instanceof Error
      ? activeLedgersQuery.error.message
      : '账本列表加载失败'
    : null;
  const showQuickRecordAction = shouldShowQuickRecordAction(location.pathname);
  const recordActionTitle = canWriteLedger ? '记一笔' : '当前账本为只读，无法记账';
  const visibleDraftCount = activeBusinessContextReady && !archivedViewReady ? drafts.length : 0;
  const networkTone: StatusChipTone = isOffline ? 'danger' : visibleDraftCount > 0 ? 'info' : 'success';
  const networkLabel = isOffline ? '网络离线' : visibleDraftCount > 0 ? `${visibleDraftCount} 条草稿` : '网络在线';

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
    if (activeLedgersQuery.data) reconcileActiveLedgers(activeLedgersQuery.data);
  }, [
    activeLedgersQuery.data,
    activeLedgersQuery.dataUpdatedAt,
    reconcileActiveLedgers,
  ]);

  useEffect(() => {
    if (!archivedRequestedId) {
      exitArchivedLedgerView();
      return;
    }
    if (requestedArchivedLedger) {
      enterArchivedLedgerView(requestedArchivedLedger);
      return;
    }
    if (!archivedLedgersQuery.isLoading) exitArchivedLedgerView();
  }, [
    archivedLedgersQuery.isLoading,
    archivedRequestedId,
    enterArchivedLedgerView,
    exitArchivedLedgerView,
    requestedArchivedLedger,
  ]);

  useEffect(() => {
    if (!activeLedgersQuery.isError) return;
    const message = activeLedgersQuery.error instanceof Error
      ? activeLedgersQuery.error.message
      : '账本列表加载失败';
    if (contextStatus === 'active' && activeLedger) {
      return;
    }
    failLedgerValidation(
      message,
    );
  }, [
    activeLedger,
    activeLedgersQuery.error,
    activeLedgersQuery.isError,
    contextStatus,
    failLedgerValidation,
  ]);

  useEffect(() => {
    if (!contextNotice || contextNotice.kind !== 'fallback') return undefined;
    const timer = window.setTimeout(() => clearContextNotice(), 6000);
    return () => window.clearTimeout(timer);
  }, [clearContextNotice, contextNotice]);

  useEffect(() => {
    pageRef.current?.focus({ preventScroll: true });
  }, [location.pathname]);

  const handleLedgerChange = async (nextLedger: LedgerWithRole) => {
    if ((nextLedger.id === activeLedgerId && !archivedRequestedId) || isSwitchingLedger) return;
    setIsSwitchingLedger(true);
    setSwitchMessage(null);
    setAddDrawerOpen(false);
    setCopySourceTransaction(null);
    setEditSourceTransaction(null);
    setEditingDraftId(null);
    try {
      await switchActiveLedgerContext({
        queryClient,
        currentLedgerId: activeLedgerId,
        nextLedger,
        commit: (ledger) => setActiveLedger(ledger.id, ledger.role),
      });
      exitArchivedLedgerView();
      if (archivedRequestedId) navigate(location.pathname);
      setSwitchMessage(`已切换到账本「${nextLedger.name}」`);
    } finally {
      setIsSwitchingLedger(false);
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
      beginLedgerValidation();
      navigate('/login');
    }
  };

  const retryLedgerList = () => {
    if (!activeBusinessContextReady && !globalLedgerRoute) beginLedgerValidation();
    void activeLedgersQuery.refetch();
  };

  const restoreArchivedMutation = useMutation({
    mutationFn: () => {
      if (!requestedArchivedLedger) throw new Error('归档账本信息尚未加载');
      return ledgerApi.restoreLedger(requestedArchivedLedger.id, requestedArchivedLedger.version);
    },
    onSuccess: (restoredLedger) => {
      const nextActive = [
        restoredLedger,
        ...ledgers.filter((ledger) => ledger.id !== restoredLedger.id),
      ];
      queryClient.setQueryData(queryKeys.ledgers.list('active'), nextActive);
      queryClient.setQueryData<LedgerWithRole[]>(
        queryKeys.ledgers.list('archived'),
        (current = []) => current.filter((ledger) => ledger.id !== restoredLedger.id),
      );
      setActiveLedger(restoredLedger.id, restoredLedger.role);
      exitArchivedLedgerView();
      setRestoreConfirmOpen(false);
      navigate(location.pathname);
    },
  });
  const restoreErrorPresentation = restoreArchivedMutation.error
    ? getLedgerErrorPresentation(restoreArchivedMutation.error)
    : null;

  const ledgerSwitcher = (
    <LedgerSwitcher
      ledgers={ledgers}
      activeLedgerId={activeLedgerId}
      recentLedgerUsedAt={recentLedgerUsedAt}
      contextStatus={contextStatus}
      errorMessage={validationError}
      archivedCount={archivedLedgersQuery.data?.length ?? 0}
      isSwitching={isSwitchingLedger}
      onSelect={handleLedgerChange}
      onRetry={retryLedgerList}
      onManage={() => navigate('/settings/ledgers')}
    />
  );

  const archivedContextControl = requestedArchivedLedger ? (
    <div className="lt-archived-context-control">
      <span className="lt-shell__field-label">正在查看</span>
      <button
        type="button"
        onClick={() => navigate(`/settings/ledgers/${requestedArchivedLedger.id}`)}
        title={requestedArchivedLedger.name}
      >
        <span>
          <strong>{requestedArchivedLedger.name}</strong>
          <small>已归档 · 只读</small>
        </span>
        <Archive size={17} aria-hidden="true" />
      </button>
    </div>
  ) : null;

  const archivedContextContent = archivedRequestedId
      ? archivedLedgersQuery.isLoading || requestedArchivedLedger
      ? (
          <StatePanel
            tone="info"
            icon={<Archive size={40} />}
            title="正在读取归档账本"
            description="系统正在建立临时只读查看上下文。"
          />
        )
      : (
          <StatePanel
            tone="danger"
            icon={<AlertTriangle size={40} />}
            title="无法查看该归档账本"
            description={archivedLedgersQuery.isError
              ? getLedgerErrorPresentation(archivedLedgersQuery.error).message
              : '该账本已恢复、你已失去访问权限，或链接已经失效。'}
            action={{ label: '返回账本管理', onClick: () => navigate('/settings/ledgers?status=archived') }}
          />
        )
    : null;
  const inactiveContextContent = archivedContextContent ?? (contextStatus === 'no-active'
    ? <NoActiveLedgerShell notice={contextNotice} />
    : contextStatus === 'error'
      ? (
          <StatePanel
            tone="danger"
            icon={<AlertTriangle size={40} />}
            title="无法读取账本列表"
            description={validationError || '请检查网络或登录状态后重试。'}
            action={{ label: '重试读取账本', onClick: retryLedgerList }}
          />
        )
      : (
          <StatePanel
            tone="info"
            icon={<BookOpen size={40} />}
            title="正在校验账本"
            description="系统正在确认最近使用的账本是否仍可访问。"
          />
        ));

  const navigationPath = (itemPath: string) =>
    buildShellNavigationPath(itemPath, archivedViewReady ? requestedArchivedLedger?.id ?? null : null);
  const returnFromArchivedView = () => {
    exitArchivedLedgerView();
    navigate(location.pathname);
  };

  return (
    <div className={`lt-shell${ledgerContextReady ? '' : ' lt-shell--inactive-context'}`}>
      <a className="lt-shell__skip-link" href="#main-content">跳到主要内容</a>
      <aside className="lt-shell__sidebar" aria-label="应用侧栏">
        <div className="lt-shell__brand">
          <span className="lt-shell__brand-mark"><Sparkles size={20} aria-hidden="true" /></span>
          <div className="lt-shell__brand-copy">
            <span className="lt-shell__brand-name">LedgerTwo</span>
            <DeploymentBadge />
          </div>
        </div>

        {archivedViewReady ? archivedContextControl : ledgerSwitcher}

        {activeBusinessContextReady && !archivedViewReady ? (
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
        ) : null}

        {ledgerContextReady ? <nav className="lt-shell__nav" aria-label="主导航">
          {APP_PRIMARY_NAV_ITEMS.map((item) => {
            const Icon = PRIMARY_NAV_ICONS[item.id];
            const isActive = isAppRouteActive(location.pathname, item.path);
            return (
              <Link
                key={item.id}
                to={navigationPath(item.path)}
                className="lt-shell__nav-link"
                aria-current={isActive ? 'page' : undefined}
              >
                <Icon size={19} aria-hidden="true" />
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav> : <div className="lt-shell__nav-spacer" />}

        {activeBusinessContextReady && !archivedViewReady ? <nav className="lt-shell__tools" aria-label="账本工具">
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
        </nav> : null}

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
        {archivedViewReady && requestedArchivedLedger ? (
          <ArchivedLedgerBanner
            ledger={requestedArchivedLedger}
            isRestoring={restoreArchivedMutation.isPending}
            errorMessage={restoreErrorPresentation?.message}
            onRestore={() => setRestoreConfirmOpen(true)}
            onReturn={returnFromArchivedView}
          />
        ) : activeBusinessContextReady && isOffline ? (
          <div className="lt-shell__notice lt-shell__notice--offline" role="status" aria-live="polite">
            <WifiOff size={17} aria-hidden="true" />
            <span>当前网络已离线，未提交内容会保存在本机草稿中。</span>
            {activeBusinessContextReady && drafts.length > 0 ? (
              <button type="button" className="lt-shell__notice-action" onClick={() => setIsDraftListOpen(true)}>
                查看 {drafts.length} 条草稿
              </button>
            ) : null}
          </div>
        ) : activeBusinessContextReady && drafts.length > 0 ? (
          <div className="lt-shell__notice lt-shell__notice--drafts" role="status" aria-live="polite">
            <CloudOff size={17} aria-hidden="true" />
            <span>有 {drafts.length} 条离线草稿待处理。</span>
            <button type="button" className="lt-shell__notice-action" onClick={() => setIsDraftListOpen(true)}>
              打开草稿箱
            </button>
          </div>
        ) : ledgerListRefreshError ? (
          <div className="lt-shell__notice lt-shell__notice--context" role="status" aria-live="polite">
            <AlertTriangle size={17} aria-hidden="true" />
            <span>账本列表暂未更新：{ledgerListRefreshError}</span>
            <button type="button" className="lt-shell__notice-action" onClick={retryLedgerList}>
              重试
            </button>
          </div>
        ) : contextNotice?.kind === 'fallback' ? (
          <div className="lt-shell__notice lt-shell__notice--context" role="status" aria-live="polite">
            <BookOpen size={17} aria-hidden="true" />
            <span>原账本已归档或无法访问，已切换到「{contextNotice.nextLedgerName}」。</span>
            <button type="button" className="lt-shell__notice-action" onClick={clearContextNotice}>
              知道了
            </button>
          </div>
        ) : switchMessage ? (
          <div className="lt-shell__notice lt-shell__notice--context" role="status" aria-live="polite">
            <BookOpen size={17} aria-hidden="true" />
            <span>{switchMessage}</span>
            <button type="button" className="lt-shell__notice-action" onClick={() => setSwitchMessage(null)}>
              关闭
            </button>
          </div>
        ) : null}

        <header className="lt-shell__topbar">
          <div className="lt-shell__desktop-context">
            <div className="lt-shell__ledger-summary">
              <span className="lt-shell__context-label">{archivedViewReady ? '正在查看' : '当前账本'}</span>
              <div className="lt-shell__ledger-summary-row">
                <strong className="lt-shell__ledger-name">{currentLedger?.name || '暂无活跃账本'}</strong>
                <StatusChip tone={archivedViewReady ? 'warning' : ledgerContextReady ? 'neutral' : 'warning'}>
                  {archivedViewReady ? '已归档 · 只读' : ledgerContextReady ? getLedgerRoleLabel(currentRole) : '全局状态'}
                </StatusChip>
              </div>
            </div>
            {ledgerContextReady ? <MonthControl value={currentMonth} onChange={setCurrentMonth} /> : null}
          </div>

          <div className="lt-shell__topbar-actions">
            <StatusChip tone={networkTone} icon={isOffline ? <WifiOff size={14} /> : <Wifi size={14} />}>
              {networkLabel}
            </StatusChip>
            {activeBusinessContextReady && !archivedViewReady && drafts.length > 0 ? (
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
            {ledgerContextReady ? (
              <div className="lt-shell__mobile-controls">
                {archivedViewReady ? archivedContextControl : ledgerSwitcher}
                <MonthControl value={currentMonth} onChange={setCurrentMonth} />
              </div>
            ) : null}
          </div>
        </header>

        <div ref={pageRef} id="main-content" className="lt-shell__page" tabIndex={-1}>
          {canMountOutlet ? <Outlet key={archivedViewReady ? requestedArchivedLedger?.id : activeLedgerId} /> : inactiveContextContent}
        </div>
      </main>

      {activeBusinessContextReady && !archivedViewReady && showQuickRecordAction ? (
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

      {ledgerContextReady ? <nav className="lt-shell__bottom-nav" aria-label="主导航">
        {APP_PRIMARY_NAV_ITEMS.map((item) => {
          const Icon = PRIMARY_NAV_ICONS[item.id];
          const isActive = isAppRouteActive(location.pathname, item.path);
          return (
            <Link
              key={item.id}
              to={navigationPath(item.path)}
              className="lt-shell__bottom-link"
              aria-current={isActive ? 'page' : undefined}
            >
              <Icon size={21} aria-hidden="true" />
              <span className="lt-shell__bottom-label">{item.label}</span>
            </Link>
          );
        })}
      </nav> : null}

      {activeBusinessContextReady && !archivedViewReady ? <TransactionFormDrawer /> : null}
      {activeBusinessContextReady && !archivedViewReady ? (
        <DraftListDrawer open={isDraftListOpen} onClose={() => setIsDraftListOpen(false)} />
      ) : null}
      <LedgerActionSurface
        open={restoreConfirmOpen && Boolean(requestedArchivedLedger)}
        title="恢复后重新开放写入"
        description="恢复后 Owner 和 Editor 可继续记账；系统不会自动补生成归档期间的周期账单。"
        confirmLabel="恢复账本"
        isConfirming={restoreArchivedMutation.isPending}
        confirmDisabled={isOffline}
        onClose={() => {
          if (restoreArchivedMutation.isPending) return;
          restoreArchivedMutation.reset();
          setRestoreConfirmOpen(false);
        }}
        onConfirm={() => restoreArchivedMutation.mutate()}
      >
        {restoreErrorPresentation ? (
          <div className="ledger-action-feedback ledger-action-feedback--error" role="alert">
            <span>{restoreErrorPresentation.message}</span>
            {restoreErrorPresentation?.recovery === 'refresh' ? (
              <Button
                variant="ghost"
                startIcon={<RefreshCw size={15} />}
                onClick={() => {
                  void archivedLedgersQuery.refetch();
                  restoreArchivedMutation.reset();
                }}
              >
                刷新账本信息
              </Button>
            ) : null}
          </div>
        ) : null}
      </LedgerActionSurface>
    </div>
  );
}
