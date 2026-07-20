import { useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Archive,
  ArrowLeft,
  BookOpen,
  ChevronRight,
  FolderOpen,
  LogOut,
  Plus,
  WifiOff,
} from 'lucide-react';
import { useForm } from 'react-hook-form';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { ledgerApi, type LedgerListStatus, type LedgerWithRole } from '../api/ledger.api';
import { queryKeys } from '../api/queryKeys';
import LedgerActionSurface from '../components/ledger/LedgerActionSurface';
import LedgerLifecycleActions from '../components/ledger/LedgerLifecycleActions';
import {
  buildArchivedLedgerPath,
  getLedgerCapabilities,
  getLedgerErrorPresentation,
  ledgerNameSchema,
  type LedgerNameValues,
} from '../components/ledger/ledgerManagementModel';
import { switchActiveLedgerContext } from '../components/layout/ledgerContextModel';
import Button from '../components/ui/Button';
import PageState from '../components/ui/PageState';
import ResponsiveDataList from '../components/ui/ResponsiveDataList';
import SegmentedControl from '../components/ui/SegmentedControl';
import StatePanel from '../components/ui/StatePanel';
import StatusChip from '../components/ui/StatusChip';
import { useLedgerStore } from '../stores/ledger.store';
import { useUIStore } from '../stores/ui.store';
import { formatDate } from '../utils/date';
import type { MetadataProfileKey } from '../types/metadata';
import './LedgerManagementPage.css';

type ManagementStatus = Extract<LedgerListStatus, 'active' | 'archived'>;

const roleLabels = {
  owner: 'Owner',
  editor: 'Editor',
  viewer: 'Viewer',
} as const;

function upsertLedger(list: LedgerWithRole[] | undefined, ledger: LedgerWithRole, status: ManagementStatus) {
  const withoutLedger = (list ?? []).filter((item) => item.id !== ledger.id);
  return ledger.status === status ? [ledger, ...withoutLedger] : withoutLedger;
}

function formatLedgerTimestamp(ledger: LedgerWithRole) {
  return formatDate(ledger.updated_at).substring(0, 16);
}

export default function LedgerManagementPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const setActiveLedger = useLedgerStore((state) => state.setActiveLedger);
  const reconcileActiveLedgers = useLedgerStore((state) => state.reconcileActiveLedgers);
  const exitArchivedLedgerView = useLedgerStore((state) => state.exitArchivedLedgerView);
  const isOffline = useUIStore((state) => state.isOffline);
  const [createOpen, setCreateOpen] = useState(false);
  const [metadataProfile, setMetadataProfile] = useState<MetadataProfileKey>('basic_cn_v1');
  const [leaveTarget, setLeaveTarget] = useState<LedgerWithRole | null>(null);
  const requestedStatus = searchParams.get('status');
  const status: ManagementStatus = requestedStatus === 'archived' ? 'archived' : 'active';

  const activeQuery = useQuery({
    queryKey: queryKeys.ledgers.list('active'),
    queryFn: ({ signal }) => ledgerApi.listUserLedgers('active', signal),
  });
  const archivedQuery = useQuery({
    queryKey: queryKeys.ledgers.list('archived'),
    queryFn: ({ signal }) => ledgerApi.listUserLedgers('archived', signal),
  });
  const currentQuery = status === 'active' ? activeQuery : archivedQuery;
  const currentLedgers = currentQuery.data ?? [];

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<LedgerNameValues>({
    resolver: zodResolver(ledgerNameSchema),
    defaultValues: { name: '' },
  });

  const syncLedger = (ledger: LedgerWithRole) => {
    const nextActive = upsertLedger(
      queryClient.getQueryData<LedgerWithRole[]>(queryKeys.ledgers.list('active')),
      ledger,
      'active',
    );
    queryClient.setQueryData(queryKeys.ledgers.list('active'), nextActive);
    queryClient.setQueryData<LedgerWithRole[]>(
      queryKeys.ledgers.list('archived'),
      (current) => upsertLedger(current, ledger, 'archived'),
    );
    queryClient.setQueryData<LedgerWithRole[]>(
      queryKeys.ledgers.list('all'),
      (current = []) => [ledger, ...current.filter((item) => item.id !== ledger.id)],
    );
    if (ledger.id === activeLedgerId && ledger.status === 'archived') {
      reconcileActiveLedgers(nextActive);
    }
  };

  const removeLedgerAccess = (ledger: LedgerWithRole) => {
    const nextActive = (queryClient.getQueryData<LedgerWithRole[]>(
      queryKeys.ledgers.list('active'),
    ) ?? []).filter((item) => item.id !== ledger.id);
    queryClient.setQueryData(queryKeys.ledgers.list('active'), nextActive);
    queryClient.setQueryData<LedgerWithRole[]>(
      queryKeys.ledgers.list('archived'),
      (current = []) => current.filter((item) => item.id !== ledger.id),
    );
    queryClient.setQueryData<LedgerWithRole[]>(
      queryKeys.ledgers.list('all'),
      (current = []) => current.filter((item) => item.id !== ledger.id),
    );
    if (ledger.id === activeLedgerId) reconcileActiveLedgers(nextActive);
  };

  const createMutation = useMutation({
    mutationFn: (values: LedgerNameValues) => ledgerApi.createLedger({
      ...values,
      metadata_profile: metadataProfile,
    }),
    onSuccess: async (ledger) => {
      syncLedger(ledger);
      await switchActiveLedgerContext({
        queryClient,
        currentLedgerId: activeLedgerId,
        nextLedger: ledger,
        commit: (nextLedger) => setActiveLedger(nextLedger.id, nextLedger.role),
      });
      reset();
      setMetadataProfile('basic_cn_v1');
      setCreateOpen(false);
      navigate('/');
    },
  });
  const leaveMutation = useMutation({
    mutationFn: (ledger: LedgerWithRole) => ledgerApi.leaveLedger(ledger.id, ledger.version),
    onSuccess: (_result, ledger) => {
      removeLedgerAccess(ledger);
      setLeaveTarget(null);
    },
  });

  const createError = createMutation.error
    ? getLedgerErrorPresentation(createMutation.error)
    : null;
  const leaveError = leaveMutation.error
    ? getLedgerErrorPresentation(leaveMutation.error)
    : null;

  const switchLedger = async (ledger: LedgerWithRole, destination = '/') => {
    exitArchivedLedgerView();
    if (ledger.id !== activeLedgerId) {
      await switchActiveLedgerContext({
        queryClient,
        currentLedgerId: activeLedgerId,
        nextLedger: ledger,
        commit: (nextLedger) => setActiveLedger(nextLedger.id, nextLedger.role),
      });
    }
    navigate(destination);
  };

  const renderPrimaryAction = (ledger: LedgerWithRole) => {
    if (ledger.status === 'archived') {
      return (
        <Button
          variant="secondary"
          startIcon={<FolderOpen size={16} />}
          onClick={() => navigate(buildArchivedLedgerPath('/', ledger.id))}
        >
          查看历史
        </Button>
      );
    }
    if (ledger.id === activeLedgerId) {
      return (
        <Button
          variant="secondary"
          endIcon={<ChevronRight size={16} />}
          onClick={() => navigate(`/settings/ledgers/${ledger.id}`)}
        >
          查看详情
        </Button>
      );
    }
    return (
      <Button
        variant="primary"
        startIcon={<BookOpen size={16} />}
        disabled={isOffline}
        onClick={() => void switchLedger(ledger)}
      >
        切换到账本
      </Button>
    );
  };

  const renderSecondaryActions = (ledger: LedgerWithRole) => {
    const capabilities = getLedgerCapabilities(ledger);
    return (
      <div className="ledger-list-item__secondary">
        <Link
          className="ledger-list-item__detail-link"
          to={`/settings/ledgers/${ledger.id}`}
          aria-label={`查看 ${ledger.name} 详情`}
          title="账本详情"
        >
          <ChevronRight size={18} aria-hidden="true" />
        </Link>
        {capabilities.canRename || capabilities.canArchive || capabilities.canRestore ? (
          <LedgerLifecycleActions
            ledger={ledger}
            onUpdated={syncLedger}
            onGoToImports={(target) => void switchLedger(target, '/import')}
          />
        ) : null}
        {capabilities.canLeave ? (
          <Button
            variant="ghost"
            iconOnly
            aria-label={`离开 ${ledger.name}`}
            title="离开账本"
            disabled={isOffline}
            onClick={() => setLeaveTarget(ledger)}
          >
            <LogOut size={17} aria-hidden="true" />
          </Button>
        ) : null}
      </div>
    );
  };

  const desktopList = (
    <div className="ledger-table" role="table" aria-label={`${status === 'active' ? '活跃' : '已归档'}账本`}>
      <div className="ledger-table__header" role="row">
        <span role="columnheader">账本</span>
        <span role="columnheader">状态与角色</span>
        <span role="columnheader">成员</span>
        <span role="columnheader">最近更新</span>
        <span role="columnheader">操作</span>
      </div>
      {currentLedgers.map((ledger) => (
        <div className="ledger-table__row" role="row" key={ledger.id}>
          <div role="cell" className="ledger-list-item__identity">
            <span className="ledger-list-item__icon" aria-hidden="true">
              {ledger.status === 'active' ? <BookOpen size={18} /> : <Archive size={18} />}
            </span>
            <div>
              <strong title={ledger.name}>{ledger.name}</strong>
              <small>{ledger.id === activeLedgerId && ledger.status === 'active' ? '当前账本' : '独立记账空间'}</small>
            </div>
          </div>
          <div role="cell" className="ledger-list-item__chips">
            <StatusChip tone={ledger.status === 'active' ? 'success' : 'warning'}>
              {ledger.status === 'active' ? '活跃' : '已归档 · 只读'}
            </StatusChip>
            <StatusChip>{roleLabels[ledger.role]}</StatusChip>
          </div>
          <span role="cell">{ledger.member_count}/2</span>
          <time role="cell" dateTime={ledger.updated_at}>{formatLedgerTimestamp(ledger)}</time>
          <div role="cell" className="ledger-list-item__actions">
            {renderPrimaryAction(ledger)}
            {renderSecondaryActions(ledger)}
          </div>
        </div>
      ))}
    </div>
  );

  const mobileList = (
    <div className="ledger-mobile-list">
      {currentLedgers.map((ledger) => (
        <article className="ledger-mobile-card" key={ledger.id}>
          <header>
            <span className="ledger-list-item__icon" aria-hidden="true">
              {ledger.status === 'active' ? <BookOpen size={18} /> : <Archive size={18} />}
            </span>
            <div>
              <strong title={ledger.name}>{ledger.name}</strong>
              <span>{ledger.member_count}/2 名成员 · {formatLedgerTimestamp(ledger)}</span>
            </div>
            {renderSecondaryActions(ledger)}
          </header>
          <div className="ledger-list-item__chips">
            <StatusChip tone={ledger.status === 'active' ? 'success' : 'warning'}>
              {ledger.status === 'active' ? '活跃' : '已归档 · 只读'}
            </StatusChip>
            <StatusChip>{roleLabels[ledger.role]}</StatusChip>
            {ledger.id === activeLedgerId && ledger.status === 'active'
              ? <StatusChip tone="info">当前</StatusChip>
              : null}
          </div>
          {renderPrimaryAction(ledger)}
        </article>
      ))}
    </div>
  );

  return (
    <main className="ledger-management-page">
      <header className="ledger-management-page__header">
        <div>
          <Link className="ledger-management-page__back" to="/settings">
            <ArrowLeft size={16} aria-hidden="true" />
            返回设置
          </Link>
          <span className="ledger-management-page__eyebrow">多账本与访问边界</span>
          <h1>账本管理</h1>
          <p>活跃账本用于日常记账；归档账本保持只读，可显式查看历史或由 Owner 恢复。</p>
        </div>
        <Button
          variant="primary"
          startIcon={<Plus size={17} />}
          disabled={isOffline}
          onClick={() => {
            createMutation.reset();
            setMetadataProfile('basic_cn_v1');
            setCreateOpen(true);
          }}
        >
          创建账本
        </Button>
      </header>

      {isOffline ? (
        <div className="ledger-management-page__notice" role="status">
          <WifiOff size={17} aria-hidden="true" />
          <span>当前离线。你仍可查看缓存列表，但创建、切换和管理操作已暂停。</span>
        </div>
      ) : null}

      <div className="ledger-management-page__toolbar">
        <SegmentedControl
          ariaLabel="账本状态"
          value={status}
          options={[
            { value: 'active', label: '活跃', count: activeQuery.data?.length ?? 0 },
            { value: 'archived', label: '已归档', count: archivedQuery.data?.length ?? 0 },
          ]}
          onChange={(nextStatus) => {
            const next = new URLSearchParams(searchParams);
            if (nextStatus === 'archived') next.set('status', 'archived');
            else next.delete('status');
            setSearchParams(next);
          }}
          fullWidth
        />
        {currentQuery.isError && currentLedgers.length > 0 ? (
          <div className="ledger-management-page__stale" role="status">
            列表暂未更新。
            <Button variant="ghost" onClick={() => void currentQuery.refetch()}>重试</Button>
          </div>
        ) : null}
      </div>

      <PageState
        isLoading={currentQuery.isLoading}
        isError={currentQuery.isError && currentLedgers.length === 0}
        isEmpty={false}
        errorMsg={getLedgerErrorPresentation(currentQuery.error).message}
        emptyMessage={status === 'active' ? '暂无活跃账本。' : '暂无已归档账本。'}
        skeletonType="table"
        onRetry={() => void currentQuery.refetch()}
      >
        <ResponsiveDataList
          desktop={desktopList}
          mobile={mobileList}
          desktopLabel="桌面账本列表"
          mobileLabel="移动端账本列表"
        />
      </PageState>

      {currentLedgers.length === 0 && !currentQuery.isLoading && !currentQuery.isError ? (
        <StatePanel
          tone={status === 'active' ? 'info' : 'neutral'}
          icon={status === 'active' ? <BookOpen size={40} /> : <Archive size={40} />}
          title={status === 'active' ? '暂无活跃账本' : '暂无已归档账本'}
          description={status === 'active'
            ? '创建账本后即可恢复日常记账；归档历史不会自动成为当前账本。'
            : '归档账本会在这里集中展示。'}
          action={{
            label: status === 'active' ? '创建账本' : '返回活跃账本',
            onClick: () => status === 'active'
              ? setCreateOpen(true)
              : setSearchParams(new URLSearchParams()),
          }}
        />
      ) : null}

      <LedgerActionSurface
        open={createOpen}
        title="创建账本"
        description="新账本会成为当前账本，创建者为唯一 Owner；允许与已有账本同名。"
        confirmLabel="创建并进入账本"
        icon={<Plus size={22} />}
        isConfirming={createMutation.isPending}
        confirmDisabled={isOffline}
        onClose={() => {
          if (createMutation.isPending) return;
          createMutation.reset();
          setMetadataProfile('basic_cn_v1');
          setCreateOpen(false);
        }}
        onConfirm={() => void handleSubmit((values) => createMutation.mutate(values))()}
      >
        <form
          className="ledger-action-form"
          onSubmit={handleSubmit((values) => createMutation.mutate(values))}
        >
          <label>
            <span>账本名称</span>
            <input
              {...register('name')}
              type="text"
              maxLength={60}
              placeholder="例如：共同生活"
              autoFocus
              aria-invalid={Boolean(errors.name)}
            />
            {errors.name ? <small role="alert">{errors.name.message}</small> : null}
          </label>
          <fieldset className="ledger-create-profile">
            <legend>初始分类与标签</legend>
            <SegmentedControl
              ariaLabel="新账本初始分类与标签"
              value={metadataProfile}
              onChange={setMetadataProfile}
              options={[
                { value: 'basic_cn_v1', label: '基础分类与标签' },
                { value: 'empty', label: '空白账本' },
              ]}
              fullWidth
            />
            <small>
              {metadataProfile === 'basic_cn_v1'
                ? '创建常用收支分类、兜底分类和 8 个基础标签，之后可在设置中修改。'
                : '不创建分类和标签；导入前需要自行补充至少一个支出和收入分类。'}
            </small>
          </fieldset>
          {createError ? (
            <div className="ledger-action-feedback ledger-action-feedback--error" role="alert">
              {createError.message}
            </div>
          ) : null}
        </form>
      </LedgerActionSurface>

      <LedgerActionSurface
        open={leaveTarget !== null}
        title="你将立即失去访问"
        description="离开不会删除你创建、支付或参与的历史账单，也不会改写结算和审计记录。"
        confirmLabel="离开账本"
        tone="danger"
        icon={<LogOut size={22} />}
        isConfirming={leaveMutation.isPending}
        confirmDisabled={isOffline}
        onClose={() => {
          if (leaveMutation.isPending) return;
          leaveMutation.reset();
          setLeaveTarget(null);
        }}
        onConfirm={() => leaveTarget && leaveMutation.mutate(leaveTarget)}
      >
        <div className="ledger-action-summary">
          <div><span>账本</span><strong>{leaveTarget?.name}</strong></div>
          <div><span>当前角色</span><strong>{leaveTarget ? roleLabels[leaveTarget.role] : ''}</strong></div>
        </div>
        {leaveError ? (
          <div className="ledger-action-feedback ledger-action-feedback--error" role="alert">
            {leaveError.message}
          </div>
        ) : null}
      </LedgerActionSurface>
    </main>
  );
}
