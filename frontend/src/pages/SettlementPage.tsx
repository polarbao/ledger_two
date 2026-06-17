import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { DollarSign, HelpCircle, CheckCircle2, User, ArrowRight, Loader2, Calendar, X, AlertTriangle } from 'lucide-react';
import { useUIStore } from '../stores/ui.store';
import { useAuthStore } from '../stores/auth.store';
import { settlementApi } from '../api/settlement.api';
import { dashboardApi } from '../api/dashboard.api';
import { useLedgerStore } from '../stores/ledger.store';
import { centsToYuan } from '../utils/money';
import { formatDate } from '../utils/date';
import SkeletonCard from '../components/ui/SkeletonCard';
import SkeletonTable from '../components/ui/SkeletonTable';
import EmptyState from '../components/ui/EmptyState';
import ErrorState from '../components/ui/ErrorState';

/**
 * @brief 结算中心页面组件 (SettlementPage)
 * @details 负责双人账单差额轧差展示，提供二次确认的记账清偿以及历史结算流水查询
 * @return React.ReactElement 渲染的 React 节点
 */
export default function SettlementPage() {
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((state) => state.user);
  const { currentMonth, isOffline } = useUIStore();
  const activeRole = useLedgerStore((state) => state.activeRole);
  const [showConfirmModal, setShowConfirmModal] = useState(false);
  const [note, setNote] = useState('');

  // 1. 获取结算轧差详情 (Balance)
  const { data: balance, isLoading: isBalanceLoading, isError: isBalanceError, error: balanceError, refetch: refetchBalance } = useQuery({
    queryKey: ['settlement-balance'],
    queryFn: () => settlementApi.getBalance(),
    enabled: !!currentUser,
  });

  // 2. 获取当月结算明细列表
  const { data: historyList, isLoading: isHistoryLoading, isError: isHistoryError, error: historyError, refetch: refetchHistory } = useQuery({
    queryKey: ['settlements', currentMonth],
    queryFn: () => settlementApi.getSettlements(currentMonth),
    enabled: !!currentUser,
  });

  // 3. 复用当月 Dashboard 缓存以取得系统成员 display_name
  const { data: dashboardData } = useQuery({
    queryKey: ['dashboard', currentMonth],
    queryFn: () => dashboardApi.getDashboard(currentMonth),
    enabled: !!currentUser,
  });

  const users = dashboardData?.user_stats || [];

  // 提取结算人与收款人的名字
  const getUserDisplayName = (userId: string) => {
    if (userId === currentUser?.id) return '我';
    const match = users.find((u) => u.user_id === userId);
    return match ? match.display_name : '对方';
  };

  // 4. 获取多条建议转账路径
  const suggestedTransfers = balance?.suggested_transfers || [];
  const hasTransfers = suggestedTransfers.length > 0;

  // 结算操作的目标对象状态
  const [activeTransfer, setActiveTransfer] = useState<{
    from_user_id: string;
    to_user_id: string;
    amount_cents: number;
  } | null>(null);

  // 5. 结算发起 Mutation
  const createSettlementMutation = useMutation({
    mutationFn: async () => {
      if (!activeTransfer) return;
      const occurredAt = new Date().toISOString();

      return settlementApi.createSettlement({
        from_user_id: activeTransfer.from_user_id,
        to_user_id: activeTransfer.to_user_id,
        amount_cents: activeTransfer.amount_cents,
        occurred_at: occurredAt,
        note: note.trim(),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settlement-balance'] });
      queryClient.invalidateQueries({ queryKey: ['settlements'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      setShowConfirmModal(false);
      setNote('');
      setActiveTransfer(null);
    },
  });

  const handleConfirmSettlement = () => {
    createSettlementMutation.mutate();
  };

  // 获取个人对账单轧差
  const getPersonalNetDetails = () => {
    if (!balance || !balance.user_balances) {
      return [];
    }

    return balance.user_balances.map((u) => {
      const displayName = getUserDisplayName(u.user_id);
      return {
        userId: u.user_id,
        displayName: displayName,
        isMe: u.user_id === currentUser?.id,
        paidCents: u.paid_cents,
        shareCents: u.share_cents,
        netCents: u.net_cents,
      };
    });
  };

  const personalDetails = getPersonalNetDetails();
  const showPageError = isBalanceError || isHistoryError;
  const pageErrorMsg = (balanceError instanceof Error ? balanceError.message : '') || (historyError instanceof Error ? historyError.message : '') || '获取结算对账信息失败';

  const handleRetryAll = () => {
    refetchBalance();
    refetchHistory();
  };

  return (
    <div className="page-content animate-fade-in text-left">
      {/* 顶部迎宾 Banner */}
      <div className="glass-card header-banner">
        <DollarSign className="banner-icon" />
        <div>
          <h2>结算中心</h2>
          <p className="dimmed">在此进行双人债务清偿轧差与结账补款。</p>
        </div>
      </div>

      {showPageError && (
        <ErrorState title="对账数据加载失败" message={pageErrorMsg} onRetry={handleRetryAll} />
      )}

      {!showPageError && (
        <>
          {/* 1. 当前未结清清偿看板 */}
          <div className="glass-card settlement-glow-card balance-summary-card">
            {isBalanceLoading ? (
              <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '110px' }}>
                <div className="skeleton-item" style={{ height: '70px', width: '90%', borderRadius: '12px' }}></div>
              </div>
            ) : hasTransfers ? (
              <div className="balance-debt-state">
                <HelpCircle size={44} className="status-icon text-warning animate-float" />
                <div className="debt-details" style={{ width: '100%' }}>
                  <h4>当前待结账目</h4>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', marginTop: '12px' }}>
                    {suggestedTransfers.map((transfer, idx) => (
                      <div key={idx} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: 'rgba(255,255,255,0.03)', padding: '12px 16px', borderRadius: '8px', flexWrap: 'wrap', gap: '12px' }}>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                          <div className="debt-flow" style={{ margin: 0 }}>
                            <span className="user-glow">{getUserDisplayName(transfer.from_user_id)}</span>
                            <ArrowRight size={16} className="arrow-flow" />
                            <span className="user-glow">{getUserDisplayName(transfer.to_user_id)}</span>
                          </div>
                          <div className="text-muted" style={{ fontSize: '13px' }}>
                            {transfer.from_user_id === currentUser?.id ? '您需要向对方汇款补款的差额。' : '对方需要向您转账汇款的差额。'}
                          </div>
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '16px', flexWrap: 'wrap' }}>
                          <div className="debt-amount-large" style={{ fontSize: '24px' }}>
                            ¥{centsToYuan(transfer.amount_cents)}
                          </div>
                          <button
                            className="btn-primary"
                            onClick={() => {
                              setActiveTransfer(transfer);
                              setShowConfirmModal(true);
                            }}
                            disabled={activeRole === 'viewer'}
                            style={activeRole === 'viewer' ? { opacity: 0.5, cursor: 'not-allowed', padding: '8px 16px', fontSize: '14px' } : { padding: '8px 16px', fontSize: '14px' }}
                          >
                            登记结算
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            ) : (
              <div className="balance-settled-state">
                <CheckCircle2 size={48} className="status-icon text-success" />
                <div className="settled-details">
                  <h3>账目已完全结清</h3>
                  <p className="dimmed">本期共享记账对账单无任何未结欠款，继续保持吧！</p>
                </div>
              </div>
            )}
          </div>

          {/* 2. 双方对账详细卡片 (双栏格栅) */}
          <div className="dashboard-grid-2">
            {isBalanceLoading ? (
              <SkeletonCard count={2} height="170px" />
            ) : (
              personalDetails.map((item) => (
                <div
                  key={item.userId}
                  className={`glass-card user-balance-card ${item.isMe ? 'my-balance-glow' : ''}`}
                >
                  <div className="user-card-header">
                    <User size={18} className="user-card-icon" />
                    <h4>
                      {item.displayName} {item.isMe ? '(我)' : '(对方)'} 的对账单
                    </h4>
                  </div>
                  <div className="balance-card-body">
                    <div className="balance-row">
                      <span className="label">累计实际垫付</span>
                      <span className="value">¥{centsToYuan(item.paidCents)}</span>
                    </div>
                    <div className="balance-row">
                      <span className="label">累计应承担消费</span>
                      <span className="value">¥{centsToYuan(item.shareCents)}</span>
                    </div>
                    <div className="balance-divider"></div>
                    <div className="balance-row net-row">
                      <span className="label">当期应收/应付轧差</span>
                      {item.netCents > 0 ? (
                        <span className="value text-success font-heading">
                          +¥{centsToYuan(item.netCents)} (应收)
                        </span>
                      ) : item.netCents < 0 ? (
                        <span className="value text-expense font-heading">
                          -¥{centsToYuan(Math.abs(item.netCents))} (应付)
                        </span>
                      ) : (
                        <span className="value text-muted">¥0.00 (已结清)</span>
                      )}
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>

          {/* 3. 已结算记录历史列表 */}
          <div className="glass-card history-settlement-card">
            <div className="subcard-header">
              <Calendar size={18} className="subcard-icon" />
              <h3>当期结算流水历史 ({currentMonth})</h3>
            </div>
            {isHistoryLoading ? (
              <SkeletonTable rows={3} />
            ) : historyList && historyList.length > 0 ? (
              <div className="timeline-list">
                {historyList.map((item) => (
                  <div className="timeline-item" key={item.id}>
                    <div className="item-left">
                      <span className="type-badge badge-settle">结算</span>
                      <div className="tx-details">
                        <span className="tx-title">
                          {getUserDisplayName(item.from_user_id)} ➔ {getUserDisplayName(item.to_user_id)}
                        </span>
                        <span className="tx-meta">
                          {formatDate(item.occurred_at).substring(5, 16)}
                          {item.note && ` · 备注: ${item.note}`}
                        </span>
                      </div>
                    </div>
                    <div className="tx-amount val-settle">
                      ¥{centsToYuan(item.amount_cents)}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState 
                title="暂无历史结算记录"
                description="本月账期中双方尚未登记过清偿补款账单。在上方发起一键结算，或者新生成共同支出后即可自动激活。" 
              />
            )}
          </div>
        </>
      )}

      {/* ==========================================
         二次清偿确认模态框 Modal (Danger 危险操作按钮)
         ========================================== */}
      {showConfirmModal && balance && (
        <div className="drawer-overlay show" style={{ alignItems: 'center', justifyContent: 'center' }}>
          <div className="confirm-modal-box animate-fade-in">
            <div className="drawer-header" style={{ padding: '16px 20px' }}>
              <div className="header-title" style={{ color: 'var(--accent-purple)' }}>
                <DollarSign className="title-icon" style={{ color: 'inherit' }} />
                <h3 style={{ fontSize: '16px' }}>确认登记结算补款</h3>
              </div>
              <button 
                className="btn-close-drawer" 
                onClick={() => setShowConfirmModal(false)}
              >
                <X size={18} />
              </button>
            </div>
            <div className="modal-body-padding" style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <p className="modal-alert-text">
                确认执行以下轧差清偿结算补款吗？该操作将建立结算补款凭证，并在交易流水中生成一条共享的结算明细记录。
              </p>

              {/* 收付款关系卡片 */}
              <div className="modal-transfer-card" style={{ margin: 0 }}>
                <div className="transfer-party">
                  <span className="party-name">{activeTransfer ? getUserDisplayName(activeTransfer.from_user_id) : ''}</span>
                  <span className="party-role">付款方</span>
                </div>
                <ArrowRight size={24} className="transfer-arrow" />
                <div className="transfer-party">
                  <span className="party-name">{activeTransfer ? getUserDisplayName(activeTransfer.to_user_id) : ''}</span>
                  <span className="party-role">收款方</span>
                </div>
              </div>

              {/* 轧差大数额 */}
              <div className="modal-amount-display">
                <span className="amount-label">结算清偿金额</span>
                <span className="amount-val">¥{centsToYuan(activeTransfer?.amount_cents ?? 0)}</span>
              </div>

              {/* 备注表单 */}
              <div className="form-group" style={{ marginBottom: 0 }}>
                <label>结算备注 (选填)</label>
                <input
                  type="text"
                  placeholder="例如: 微信已转账结清"
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                  style={{ width: '100%', padding: '10px 14px', borderRadius: '8px', border: '1px solid rgba(255,255,255,0.08)', background: 'rgba(10,12,16,0.6)', color: '#fff', fontSize: '13px' }}
                />
              </div>

              {/* 审计日志提示 */}
              <div style={{ background: 'rgba(147, 51, 234, 0.04)', border: '1px solid rgba(147, 51, 234, 0.15)', borderRadius: '8px', padding: '10px 14px', display: 'flex', alignItems: 'flex-start', gap: '8px', fontSize: '11px', color: '#c084fc', textAlign: 'left' }}>
                <AlertTriangle size={14} style={{ marginTop: '2px', flexShrink: 0 }} />
                <span>此操作作为高风险数据变动动作，将被自动记录并同步写入系统的 `audit_logs` 审计表中以备历史追溯。</span>
              </div>

              {isOffline && (
                <div style={{ marginTop: '12px', background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgba(239, 68, 68, 0.2)', color: '#ef4444', borderRadius: '8px', padding: '10px 14px', fontSize: '12px', display: 'flex', gap: '8px', alignItems: 'center' }}>
                  <AlertTriangle size={14} />
                  <span>当前处于离线状态，无法登记高风险结算操作。</span>
                </div>
              )}

              {/* 模态框页脚操作 (Danger 样式按钮) */}
              <div className="drawer-footer" style={{ borderTop: 'none', paddingTop: 0, marginTop: '8px', display: 'flex', gap: '10px', justifyContent: 'flex-end', flexWrap: 'wrap' }}>
                <button
                  type="button"
                  className="btn-secondary mobile-full"
                  style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }}
                  onClick={() => setShowConfirmModal(false)}
                  disabled={createSettlementMutation.isPending}
                >
                  取消
                </button>
                <button
                  type="button"
                  className="btn-danger mobile-full"
                  style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px', background: 'linear-gradient(135deg, #a855f7 0%, #7e22ce 100%)', boxShadow: '0 8px 32px 0 rgba(126, 34, 206, 0.2)', borderColor: 'rgba(126, 34, 206, 0.2)' }}
                  onClick={handleConfirmSettlement}
                  disabled={createSettlementMutation.isPending || isOffline}
                >
                  {createSettlementMutation.isPending ? (
                    <>
                      <Loader2 size={16} className="spinner" />
                      <span>登记结算中...</span>
                    </>
                  ) : (
                    <span>确认已结算</span>
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
