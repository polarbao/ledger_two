import { useMemo, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery } from '@tanstack/react-query';
import {
  Archive,
  MoreVertical,
  Pencil,
  RefreshCw,
  RotateCcw,
} from 'lucide-react';
import { useForm } from 'react-hook-form';
import { ledgerApi, type LedgerWithRole } from '../../api/ledger.api';
import { queryKeys } from '../../api/queryKeys';
import { centsToYuan } from '../../utils/money';
import { useUIStore } from '../../stores/ui.store';
import Button from '../ui/Button';
import StatusChip from '../ui/StatusChip';
import LedgerActionSurface from './LedgerActionSurface';
import {
  getLedgerCapabilities,
  getLedgerErrorPresentation,
  ledgerNameSchema,
  type LedgerNameValues,
} from './ledgerManagementModel';

type LifecycleAction = 'rename' | 'archive' | 'restore' | null;

interface LedgerLifecycleActionsProps {
  ledger: LedgerWithRole;
  mode?: 'menu' | 'inline';
  onUpdated: (ledger: LedgerWithRole) => void;
  onGoToImports: (ledger: LedgerWithRole) => void;
}

export default function LedgerLifecycleActions({
  ledger,
  mode = 'menu',
  onUpdated,
  onGoToImports,
}: LedgerLifecycleActionsProps) {
  const isOffline = useUIStore((state) => state.isOffline);
  const capabilities = getLedgerCapabilities(ledger);
  const [action, setAction] = useState<LifecycleAction>(null);
  const [acknowledgeUnsettled, setAcknowledgeUnsettled] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<LedgerNameValues>({
    resolver: zodResolver(ledgerNameSchema),
    defaultValues: { name: ledger.name },
  });

  const preflightQuery = useQuery({
    queryKey: [...queryKeys.ledgers.members(ledger.id), 'archive-preflight'],
    queryFn: ({ signal }) => ledgerApi.getArchivePreflight(ledger.id, signal),
    enabled: action === 'archive',
  });
  const membersQuery = useQuery({
    queryKey: queryKeys.ledgers.members(ledger.id),
    queryFn: ({ signal }) => ledgerApi.getLedgerMembers(ledger.id, signal),
    enabled: action === 'archive',
  });
  const memberNames = useMemo(
    () => new Map((membersQuery.data?.members ?? []).map((member) => [member.user_id, member.username])),
    [membersQuery.data?.members],
  );

  const renameMutation = useMutation({
    mutationFn: ({ name }: LedgerNameValues) =>
      ledgerApi.renameLedger(ledger.id, ledger.version, { name }),
    onSuccess: (updatedLedger) => {
      onUpdated(updatedLedger);
      setAction(null);
    },
  });
  const archiveMutation = useMutation({
    mutationFn: () => ledgerApi.archiveLedger(
      ledger.id,
      preflightQuery.data?.ledger.version ?? ledger.version,
      { acknowledge_unsettled_balance: acknowledgeUnsettled },
    ),
    onSuccess: (updatedLedger) => {
      onUpdated(updatedLedger);
      setAction(null);
    },
  });
  const restoreMutation = useMutation({
    mutationFn: () => ledgerApi.restoreLedger(ledger.id, ledger.version),
    onSuccess: (updatedLedger) => {
      onUpdated(updatedLedger);
      setAction(null);
    },
  });
  const refreshMutation = useMutation({
    mutationFn: () => ledgerApi.getLedger(ledger.id),
    onSuccess: (updatedLedger) => {
      onUpdated(updatedLedger);
      if (action === 'archive') void preflightQuery.refetch();
    },
  });

  const currentError = action === 'rename'
    ? renameMutation.error ?? refreshMutation.error
    : action === 'archive'
      ? archiveMutation.error ?? preflightQuery.error ?? refreshMutation.error
      : action === 'restore'
        ? restoreMutation.error ?? refreshMutation.error
        : null;
  const errorPresentation = currentError
    ? getLedgerErrorPresentation(currentError)
    : null;
  const preflight = preflightQuery.data;
  const unsettled = preflight?.unsettled_balance;
  const requiresAcknowledgement = Boolean(preflight?.requires_unsettled_acknowledgement);
  const archiveBlocked = Boolean(preflight && !preflight.can_archive);
  const archiveConfirmDisabled = isOffline
    || preflightQuery.isLoading
    || archiveBlocked
    || (requiresAcknowledgement && !acknowledgeUnsettled);

  const resetMutations = () => {
    renameMutation.reset();
    archiveMutation.reset();
    restoreMutation.reset();
    refreshMutation.reset();
  };
  const closeSurface = () => {
    if (renameMutation.isPending || archiveMutation.isPending || restoreMutation.isPending) return;
    resetMutations();
    setAction(null);
  };

  const actions = (
    <>
      {capabilities.canRename ? (
        <Button
          variant="ghost"
          startIcon={<Pencil size={16} />}
          disabled={isOffline}
          onClick={() => {
            reset({ name: ledger.name });
            setAction('rename');
          }}
        >
          重命名
        </Button>
      ) : null}
      {capabilities.canArchive ? (
        <Button
          variant="ghost"
          startIcon={<Archive size={16} />}
          disabled={isOffline}
          onClick={() => {
            setAcknowledgeUnsettled(false);
            setAction('archive');
          }}
        >
          归档
        </Button>
      ) : null}
      {capabilities.canRestore ? (
        <Button variant="ghost" startIcon={<RotateCcw size={16} />} disabled={isOffline} onClick={() => setAction('restore')}>
          恢复
        </Button>
      ) : null}
    </>
  );

  return (
    <>
      {mode === 'menu' ? (
        <details className="ledger-actions-menu">
          <summary aria-label={`打开 ${ledger.name} 的更多操作`} title="更多操作">
            <MoreVertical size={18} aria-hidden="true" />
          </summary>
          <div className="ledger-actions-menu__popover">
            {actions}
          </div>
        </details>
      ) : (
        <div className="ledger-lifecycle-actions">{actions}</div>
      )}

      <LedgerActionSurface
        open={action === 'rename'}
        title="重命名账本"
        description="名称可与其他账本重复；保存时会校验最新版本。"
        confirmLabel="保存名称"
        icon={<Pencil size={22} />}
        isConfirming={renameMutation.isPending}
        confirmDisabled={isOffline}
        onClose={closeSurface}
        onConfirm={() => void handleSubmit((values) => renameMutation.mutate(values))()}
      >
        <form
          className="ledger-action-form"
          onSubmit={handleSubmit((values) => renameMutation.mutate(values))}
        >
          <label>
            <span>账本名称</span>
            <input
              {...register('name')}
              type="text"
              maxLength={60}
              autoFocus
              aria-invalid={Boolean(errors.name)}
            />
            {errors.name ? <small role="alert">{errors.name.message}</small> : null}
          </label>
          {errorPresentation ? (
            <div className="ledger-action-feedback ledger-action-feedback--error" role="alert">
              <span>{errorPresentation.message}</span>
              {errorPresentation.recovery === 'refresh' ? (
                <Button
                  variant="ghost"
                  startIcon={<RefreshCw size={15} />}
                  isLoading={refreshMutation.isPending}
                  onClick={() => refreshMutation.mutate()}
                >
                  刷新账本信息
                </Button>
              ) : null}
            </div>
          ) : null}
        </form>
      </LedgerActionSurface>

      <LedgerActionSurface
        open={action === 'archive'}
        title="归档后全员只读"
        description="归档不会生成结算、删除历史或改变已有分摊。"
        confirmLabel="归档账本"
        tone="danger"
        icon={<Archive size={22} />}
        isConfirming={archiveMutation.isPending}
        confirmDisabled={archiveConfirmDisabled}
        onClose={closeSurface}
        onConfirm={() => archiveMutation.mutate()}
      >
        <div className="ledger-action-summary">
          <div><span>账本</span><strong>{preflight?.ledger.name ?? ledger.name}</strong></div>
          <div><span>状态变化</span><strong>活跃到已归档</strong></div>
          <div>
            <span>未结清</span>
            <strong>
              {preflightQuery.isLoading
                ? '正在计算'
                : unsettled?.amount_cents
                  ? `${memberNames.get(unsettled.from_user_id ?? '') ?? '一名成员'} 需向 ${memberNames.get(unsettled.to_user_id ?? '') ?? '另一名成员'} 支付 ¥${centsToYuan(unsettled.amount_cents)}`
                  : '当前已结清'}
            </strong>
          </div>
          <div>
            <span>待处理导入</span>
            <strong>{preflightQuery.isLoading ? '正在检查' : `${preflight?.ready_import_batch_count ?? 0} 个 ready 批次`}</strong>
          </div>
        </div>
        {archiveBlocked ? (
          <div className="ledger-action-feedback ledger-action-feedback--warning">
            <span>先处理待确认导入。放弃预览不会新增正式流水，批次会保留审计记录。</span>
            <Button variant="secondary" onClick={() => onGoToImports(ledger)}>
              前往导入处理
            </Button>
          </div>
        ) : null}
        {requiresAcknowledgement ? (
          <label className="ledger-action-checkbox">
            <input
              type="checkbox"
              checked={acknowledgeUnsettled}
              onChange={(event) => setAcknowledgeUnsettled(event.target.checked)}
            />
            <span>我知道归档不会自动生成结算记录</span>
          </label>
        ) : null}
        {errorPresentation ? (
          <div className="ledger-action-feedback ledger-action-feedback--error" role="alert">
            <span>{errorPresentation.message}</span>
            {errorPresentation.recovery === 'refresh' ? (
              <Button
                variant="ghost"
                startIcon={<RefreshCw size={15} />}
                isLoading={refreshMutation.isPending}
                onClick={() => refreshMutation.mutate()}
              >
                刷新账本信息
                </Button>
              ) : null}
            {errorPresentation.recovery === 'imports' ? (
              <Button variant="secondary" onClick={() => onGoToImports(ledger)}>
                前往导入处理
              </Button>
            ) : null}
          </div>
        ) : null}
      </LedgerActionSurface>

      <LedgerActionSurface
        open={action === 'restore'}
        title="恢复后重新开放写入"
        description="恢复后 Owner 和 Editor 可继续记账；系统不会自动补生成归档期间的周期账单。"
        confirmLabel="恢复账本"
        icon={<RotateCcw size={22} />}
        isConfirming={restoreMutation.isPending}
        confirmDisabled={isOffline}
        onClose={closeSurface}
        onConfirm={() => restoreMutation.mutate()}
      >
        <div className="ledger-action-summary">
          <div><span>账本</span><strong>{ledger.name}</strong></div>
          <div><span>当前状态</span><StatusChip tone="warning">已归档 · 只读</StatusChip></div>
        </div>
        {errorPresentation ? (
          <div className="ledger-action-feedback ledger-action-feedback--error" role="alert">
            <span>{errorPresentation.message}</span>
            {errorPresentation.recovery === 'refresh' ? (
              <Button
                variant="ghost"
                startIcon={<RefreshCw size={15} />}
                isLoading={refreshMutation.isPending}
                onClick={() => refreshMutation.mutate()}
              >
                刷新账本信息
              </Button>
            ) : null}
          </div>
        ) : null}
      </LedgerActionSurface>
    </>
  );
}
