import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle,
  ArrowRight,
  CalendarDays,
  CheckCircle2,
  Copy,
  ReceiptText,
  Scale,
  ShieldCheck,
  User,
} from 'lucide-react';
import { settlementApi } from '../api/settlement.api';
import { dashboardApi } from '../api/dashboard.api';
import { queryKeys } from '../api/queryKeys';
import { useAuthStore } from '../stores/auth.store';
import { useLedgerContext } from '../components/ledger/useLedgerContext';
import { useUIStore } from '../stores/ui.store';
import type { SuggestedTransfer } from '../types/settlement';
import { centsToYuan } from '../utils/money';
import { formatDate } from '../utils/date';
import Button from '../components/ui/Button';
import ConfirmDialog from '../components/ui/ConfirmDialog';
import EmptyState from '../components/ui/EmptyState';
import ErrorState from '../components/ui/ErrorState';
import SegmentedControl from '../components/ui/SegmentedControl';
import SkeletonCard from '../components/ui/SkeletonCard';
import SkeletonTable from '../components/ui/SkeletonTable';
import StatusChip from '../components/ui/StatusChip';
import {
  buildSettlementCopyText,
  copyTextToClipboard,
  describeSettlementNet,
  formatSignedYuan,
  type SettlementBalanceDetail,
  type SettlementScope,
} from './settlementPageModel';
import './SettlementPage.css';

const scopeOptions = [
  { value: 'all', label: '全部未结' },
  { value: 'month', label: '仅本月' },
] as const;

export default function SettlementPage() {
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((state) => state.user);
  const { ledgerId, role, isArchivedView } = useLedgerContext();
  const canCreateSettlement = !isArchivedView && (role === 'owner' || role === 'editor');
  const { currentMonth, isOffline } = useUIStore();
  const [scope, setScope] = useState<SettlementScope>('all');
  const [activeTransfer, setActiveTransfer] = useState<SuggestedTransfer | null>(null);
  const [note, setNote] = useState('');
  const [copyStatus, setCopyStatus] = useState<string | null>(null);
  const [copyFallbackText, setCopyFallbackText] = useState<string | null>(null);
  const balanceMonth = scope === 'month' ? currentMonth : undefined;

  const {
    data: balance,
    isLoading: isBalanceLoading,
    isError: isBalanceError,
    error: balanceError,
    refetch: refetchBalance,
  } = useQuery({
    queryKey: queryKeys.settlements.balance(ledgerId, balanceMonth),
    queryFn: ({ signal }) => settlementApi.getBalance(balanceMonth, signal),
    enabled: Boolean(currentUser && ledgerId),
  });

  const {
    data: historyList,
    isLoading: isHistoryLoading,
    isError: isHistoryError,
    error: historyError,
    refetch: refetchHistory,
  } = useQuery({
    queryKey: queryKeys.settlements.list(ledgerId, currentMonth),
    queryFn: ({ signal }) => settlementApi.getSettlements(currentMonth, signal),
    enabled: Boolean(currentUser && ledgerId),
  });

  const { data: dashboardData } = useQuery({
    queryKey: queryKeys.dashboard.month(ledgerId, currentMonth),
    queryFn: ({ signal }) => dashboardApi.getDashboard(currentMonth, signal),
    enabled: Boolean(currentUser && ledgerId),
  });

  const users = dashboardData?.user_stats || [];
  const getUserDisplayName = (userId: string) => {
    if (userId === currentUser?.id) return '我';
    return users.find((user) => user.user_id === userId)?.display_name || '对方';
  };

  const personalDetails: SettlementBalanceDetail[] = (balance?.user_balances || []).map((item) => ({
    userId: item.user_id,
    displayName: getUserDisplayName(item.user_id),
    isMe: item.user_id === currentUser?.id,
    paidCents: item.paid_cents,
    shareCents: item.share_cents,
    rawNetCents: item.raw_net_cents,
    settlementNetCents: item.settlement_net_cents,
    finalNetCents: item.final_net_cents,
  }));
  const suggestedTransfers = balance?.suggested_transfers || [];
  const hasTransfers = suggestedTransfers.length > 0;

  const createSettlementMutation = useMutation({
    mutationFn: () => {
      if (!activeTransfer) throw new Error('缺少待登记的结算关系');
      return settlementApi.createSettlement({
        from_user_id: activeTransfer.from_user_id,
        to_user_id: activeTransfer.to_user_id,
        amount_cents: activeTransfer.amount_cents,
        occurred_at: new Date().toISOString(),
        note: note.trim(),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.settlements.balanceRoot(ledgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.settlements.root(ledgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.root(ledgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.root(ledgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.reports.root(ledgerId) });
      setActiveTransfer(null);
      setNote('');
    },
  });

  const closeConfirmDialog = () => {
    if (createSettlementMutation.isPending) return;
    setActiveTransfer(null);
    setNote('');
    createSettlementMutation.reset();
  };

  const legacyCopy = (value: string) => {
    const textarea = document.createElement('textarea');
    textarea.value = value;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    try {
      return document.execCommand('copy');
    } finally {
      document.body.removeChild(textarea);
    }
  };

  const handleCopySettlementText = async (transfer: SuggestedTransfer) => {
    const text = buildSettlementCopyText({
      scope,
      month: currentMonth,
      transfer,
      details: personalDetails,
      getUserDisplayName,
    });
    setCopyFallbackText(null);

    try {
      const writeText = navigator.clipboard?.writeText
        ? navigator.clipboard.writeText.bind(navigator.clipboard)
        : undefined;
      await copyTextToClipboard(text, writeText, legacyCopy);
      setCopyStatus('结算文案已复制');
      window.setTimeout(() => setCopyStatus(null), 2400);
    } catch {
      setCopyFallbackText(text);
      setCopyStatus('复制失败，请手动选择下方文案');
      window.setTimeout(() => setCopyStatus(null), 5000);
    }
  };

  const handleScopeChange = (nextScope: SettlementScope) => {
    setScope(nextScope);
    setCopyStatus(null);
    setCopyFallbackText(null);
  };

  const showPageError = isBalanceError || isHistoryError;
  const pageErrorMessage = (balanceError instanceof Error ? balanceError.message : '')
    || (historyError instanceof Error ? historyError.message : '')
    || '获取结算对账信息失败';
  const sharedExpenseDetailUrl = `/transactions?month=${encodeURIComponent(currentMonth)}&type=shared_expense&page=1${isArchivedView && ledgerId ? `&archived_ledger_id=${encodeURIComponent(ledgerId)}` : ''}`;
  const scopeLabel = scope === 'month' ? `${currentMonth} 本月` : '全部未结账期';

  return (
    <main className="settlement-page animate-fade-in">
      <header className="settlement-page__header">
        <div className="settlement-page__title-copy">
          <span className="settlement-page__eyebrow">共同账目 · {currentMonth}</span>
          <h1>结算中心</h1>
          <p>先核对实际支付与承担，再生成独立结算记录。</p>
        </div>
        <SegmentedControl
          ariaLabel="结算统计范围"
          value={scope}
          options={scopeOptions}
          onChange={handleScopeChange}
          className="settlement-page__scope"
        />
      </header>

      {showPageError ? (
        <section className="settlement-page__panel">
          <ErrorState
            title="对账数据加载失败"
            message={pageErrorMessage}
            onRetry={() => {
              refetchBalance();
              refetchHistory();
            }}
          />
        </section>
      ) : (
        <>
          <section className="settlement-page__panel settlement-page__conclusion" aria-labelledby="settlement-conclusion-title">
            <div className="settlement-page__section-header">
              <div>
                <StatusChip tone={hasTransfers ? 'warning' : 'success'} icon={hasTransfers ? <Scale size={14} /> : <CheckCircle2 size={14} />}>
                  {scopeLabel}
                </StatusChip>
                <h2 id="settlement-conclusion-title">当前结论</h2>
              </div>
              <span className="settlement-page__authority">以服务端对账结果为准</span>
            </div>

            {isBalanceLoading ? (
              <div className="settlement-page__conclusion-loading" aria-label="正在加载结算结果">
                <div className="skeleton-item" />
                <div className="skeleton-item" />
              </div>
            ) : hasTransfers ? (
              <div className="settlement-page__transfer-list">
                {suggestedTransfers.map((transfer) => (
                  <article
                    key={`${transfer.from_user_id}-${transfer.to_user_id}`}
                    className="settlement-page__transfer"
                  >
                    <div className="settlement-page__transfer-copy">
                      <div className="settlement-page__transfer-route">
                        <strong>{getUserDisplayName(transfer.from_user_id)}</strong>
                        <ArrowRight size={18} aria-label="转给" />
                        <strong>{getUserDisplayName(transfer.to_user_id)}</strong>
                      </div>
                      <p>
                        {transfer.from_user_id === currentUser?.id
                          ? '我需要向对方补齐共同支出差额'
                          : '对方需要向我补齐共同支出差额'}
                      </p>
                    </div>
                    <strong className="settlement-page__transfer-amount">¥{centsToYuan(transfer.amount_cents)}</strong>
                    <div className="settlement-page__transfer-actions">
                      <Button
                        variant="secondary"
                        startIcon={<Copy size={17} />}
                        onClick={() => handleCopySettlementText(transfer)}
                      >
                        复制文案
                      </Button>
                      {canCreateSettlement ? (
                        <Button
                          variant="primary"
                          startIcon={<Scale size={17} />}
                          onClick={() => {
                            createSettlementMutation.reset();
                            setActiveTransfer(transfer);
                          }}
                        >
                          生成结算记录
                        </Button>
                      ) : null}
                    </div>
                  </article>
                ))}
              </div>
            ) : (
              <div className="settlement-page__settled">
                <CheckCircle2 size={38} aria-hidden="true" />
                <div>
                  <h3>当前双方已结清</h3>
                  <p>{scopeLabel}暂无需要结算的金额。</p>
                </div>
              </div>
            )}

            {copyStatus ? (
              <p
                className={`settlement-page__copy-status ${copyFallbackText ? 'settlement-page__copy-status--error' : ''}`}
                role={copyFallbackText ? 'alert' : 'status'}
              >
                {copyStatus}
              </p>
            ) : null}
            {copyFallbackText ? (
              <label className="settlement-page__copy-fallback">
                <span>可手动复制的结算文案</span>
                <textarea
                  readOnly
                  value={copyFallbackText}
                  onFocus={(event) => event.currentTarget.select()}
                />
              </label>
            ) : null}
          </section>

          <section className="settlement-page__explanation" aria-labelledby="settlement-explanation-title">
            <div className="settlement-page__section-heading">
              <div>
                <span className="settlement-page__eyebrow">对账拆解</span>
                <h2 id="settlement-explanation-title">金额如何得出</h2>
              </div>
              <p>共同支出净额 + 已登记结算 = 最终未结</p>
            </div>
            <div className="settlement-page__balance-grid">
              {isBalanceLoading ? (
                <SkeletonCard count={2} height="250px" />
              ) : personalDetails.length > 0 ? (
                personalDetails.map((item) => (
                  <article className="settlement-page__balance-card" key={item.userId}>
                    <header>
                      <span className="settlement-page__member-icon"><User size={18} /></span>
                      <div>
                        <h3>{item.displayName}的对账单</h3>
                        <span>{item.isMe ? '当前账号' : '共同记账成员'}</span>
                      </div>
                      <StatusChip tone={item.finalNetCents > 0 ? 'success' : item.finalNetCents < 0 ? 'warning' : 'neutral'}>
                        {item.finalNetCents > 0 ? '应收' : item.finalNetCents < 0 ? '应付' : '已结清'}
                      </StatusChip>
                    </header>
                    <dl>
                      <div>
                        <dt>实际支付 <small>paid</small></dt>
                        <dd>¥{centsToYuan(item.paidCents)}</dd>
                      </div>
                      <div>
                        <dt>实际承担 <small>share</small></dt>
                        <dd>¥{centsToYuan(item.shareCents)}</dd>
                      </div>
                      <div>
                        <dt>共同支出净额 <small>raw_net</small></dt>
                        <dd>{formatSignedYuan(item.rawNetCents)}</dd>
                      </div>
                      <div>
                        <dt>已登记结算 <small>settlement</small></dt>
                        <dd>{formatSignedYuan(item.settlementNetCents)}</dd>
                      </div>
                      <div className="settlement-page__balance-final">
                        <dt>最终未结 <small>final_net</small></dt>
                        <dd>{describeSettlementNet(item.finalNetCents)}</dd>
                      </div>
                    </dl>
                  </article>
                ))
              ) : (
                <EmptyState title="暂无对账数据" description="当前账本还没有可用于结算的共同支出。" />
              )}
            </div>
          </section>

          <section className="settlement-page__panel settlement-page__impact" aria-labelledby="settlement-impact-title">
            <div className="settlement-page__section-header">
              <div className="settlement-page__section-title">
                <ReceiptText size={20} aria-hidden="true" />
                <div>
                  <h2 id="settlement-impact-title">影响结算的共同支出</h2>
                  <p>历史共同支出只参与对账计算，生成结算记录不会修改原账单。</p>
                </div>
              </div>
              <Link className="ui-button ui-button--secondary settlement-page__detail-link" to={sharedExpenseDetailUrl}>
                <ReceiptText size={17} aria-hidden="true" />
                <span>查看本月共同支出</span>
              </Link>
            </div>
          </section>

          <section className="settlement-page__panel settlement-page__history" aria-labelledby="settlement-history-title">
            <div className="settlement-page__section-header">
              <div className="settlement-page__section-title">
                <CalendarDays size={20} aria-hidden="true" />
                <div>
                  <h2 id="settlement-history-title">历史结算</h2>
                  <p>{currentMonth} 生成的独立结算记录</p>
                </div>
              </div>
              <StatusChip>{historyList?.length || 0} 条</StatusChip>
            </div>
            {isHistoryLoading ? (
              <SkeletonTable rows={3} />
            ) : historyList && historyList.length > 0 ? (
              <div className="settlement-page__history-list">
                {historyList.map((item) => (
                  <article className="settlement-page__history-row" key={item.id}>
                    <StatusChip tone="info">结算</StatusChip>
                    <div className="settlement-page__history-copy">
                      <strong>
                        {getUserDisplayName(item.from_user_id)}
                        <ArrowRight size={14} aria-label="转给" />
                        {getUserDisplayName(item.to_user_id)}
                      </strong>
                      <span>
                        {formatDate(item.occurred_at).substring(5, 16)}
                        {item.note ? ` · ${item.note}` : ''}
                      </span>
                    </div>
                    <strong className="settlement-page__history-amount">¥{centsToYuan(item.amount_cents)}</strong>
                  </article>
                ))}
              </div>
            ) : (
              <EmptyState
                title="暂无历史结算记录"
                description={`${currentMonth} 尚未生成结算记录。`}
              />
            )}
          </section>
        </>
      )}

      <ConfirmDialog
        open={activeTransfer !== null}
        title="生成结算记录"
        description="确认后会新增一条 settlement 记录和对应流水，不会修改历史共同支出。"
        confirmLabel="生成结算记录"
        icon={<Scale size={22} />}
        isConfirming={createSettlementMutation.isPending}
        confirmDisabled={isOffline || isArchivedView || !activeTransfer}
        onConfirm={() => createSettlementMutation.mutate()}
        onClose={closeConfirmDialog}
      >
        {activeTransfer ? (
          <div className="settlement-confirm">
            <div className="settlement-confirm__route">
              <div>
                <span>付款方</span>
                <strong>{getUserDisplayName(activeTransfer.from_user_id)}</strong>
              </div>
              <ArrowRight size={22} aria-label="转给" />
              <div>
                <span>收款方</span>
                <strong>{getUserDisplayName(activeTransfer.to_user_id)}</strong>
              </div>
            </div>
            <div className="settlement-confirm__amount">
              <span>结算金额</span>
              <strong>¥{centsToYuan(activeTransfer.amount_cents)}</strong>
            </div>
            <label className="settlement-confirm__field" htmlFor="settlement-note">
              <span>结算备注（选填）</span>
              <input
                id="settlement-note"
                type="text"
                maxLength={200}
                placeholder="例如：微信转账结清"
                value={note}
                onChange={(event) => setNote(event.target.value)}
              />
            </label>
            <div className="settlement-confirm__notice">
              <ShieldCheck size={17} aria-hidden="true" />
              <span>本次操作会保留审计记录，便于双方后续核对。</span>
            </div>
            {isOffline ? (
              <div className="settlement-confirm__message settlement-confirm__message--warning" role="alert">
                <AlertTriangle size={17} aria-hidden="true" />
                <span>当前处于离线状态，暂时无法生成结算记录。</span>
              </div>
            ) : null}
            {createSettlementMutation.isError ? (
              <div className="settlement-confirm__message settlement-confirm__message--error" role="alert">
                <AlertTriangle size={17} aria-hidden="true" />
                <span>
                  {createSettlementMutation.error instanceof Error
                    ? createSettlementMutation.error.message
                    : '生成结算记录失败，请稍后重试。'}
                </span>
              </div>
            ) : null}
          </div>
        ) : null}
      </ConfirmDialog>
    </main>
  );
}
