import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { DollarSign, HelpCircle, CheckCircle2, User, ArrowRight, Loader2, Calendar } from 'lucide-react';
import { useUIStore } from '../stores/ui.store';
import { useAuthStore } from '../stores/auth.store';
import { settlementApi } from '../api/settlement.api';
import { dashboardApi } from '../api/dashboard.api';
import { centsToYuan } from '../utils/money';
import { formatDate } from '../utils/date';

/**
 * @brief 结算中心页面组件 (SettlementPage)
 * @details 负责双人账单差额轧差展示，提供二次确认的记账清偿以及历史结算流水查询
 * @return React.ReactElement 渲染的 React 节点
 */
export default function SettlementPage() {
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((state) => state.user);
  const { currentMonth } = useUIStore();
  const [showConfirmModal, setShowConfirmModal] = useState(false);
  const [note, setNote] = useState('');

  // 1. 获取结算轧差详情 (Balance)
  const { data: balance, isLoading: isBalanceLoading } = useQuery({
    queryKey: ['settlement-balance'],
    queryFn: () => settlementApi.getBalance(),
    enabled: !!currentUser,
  });

  // 2. 获取当月结算明细列表
  const { data: historyList, isLoading: isHistoryLoading } = useQuery({
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

  // 获取指定 ID 成员的显示名字
  const getUserDisplayName = (userId: string) => {
    if (userId === currentUser?.id) return '我';
    const match = users.find((u) => u.user_id === userId);
    return match ? match.display_name : '对方';
  };

  // 4. 判定是否有债务关系
  const hasDebt = balance ? balance.amount_cents > 0 : false;
  const isDebtor = balance ? balance.from_user_id === currentUser?.id : false;

  // 提取结算人与收款人的名字
  const debtorName = balance ? getUserDisplayName(balance.from_user_id) : '付款人';
  const creditorName = balance ? getUserDisplayName(balance.to_user_id) : '收款人';

  // 5. 结算发起 Mutation
  const createSettlementMutation = useMutation({
    mutationFn: async () => {
      if (!balance) return;
      // 组装 UTC 时间格式
      const occurredAt = new Date().toISOString();

      return settlementApi.createSettlement({
        from_user_id: balance.from_user_id,
        to_user_id: balance.to_user_id,
        amount_cents: balance.amount_cents,
        occurred_at: occurredAt,
        note: note.trim(),
      });
    },
    onSuccess: () => {
      // 成功后全局失效相关缓存，触发全链路数据更新
      queryClient.invalidateQueries({ queryKey: ['settlement-balance'] });
      queryClient.invalidateQueries({ queryKey: ['settlements'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      setShowConfirmModal(false);
      setNote('');
    },
  });

  const handleConfirmSettlement = () => {
    createSettlementMutation.mutate();
  };

  const isLoading = isBalanceLoading || isHistoryLoading;

  if (isLoading) {
    return (
      <div className="app-loading">
        <div className="loading-spinner"></div>
        <p>结算中心数据加载中...</p>
      </div>
    );
  }

  // 内存级计算个人对账单轧差
  const getPersonalNetDetails = () => {
    if (!balance || users.length === 0) return [];

    return users.map((u) => {
      // 最终净额判定：如果是债务人，净额为负值；如果是债权人，净额为正值；已结清为 0
      let netCents = 0;
      if (hasDebt && balance) {
        if (u.user_id === balance.from_user_id) {
          netCents = -balance.amount_cents;
        } else if (u.user_id === balance.to_user_id) {
          netCents = balance.amount_cents;
        }
      }

      return {
        userId: u.user_id,
        displayName: u.display_name,
        isMe: u.user_id === currentUser?.id,
        paidCents: u.paid_cents,
        shareCents: u.share_cents,
        netCents: netCents,
      };
    });
  };

  const personalDetails = getPersonalNetDetails();

  return (
    <div className="page-content animate-fade-in">
      {/* 顶部迎宾 Banner */}
      <div className="glass-card header-banner">
        <DollarSign className="banner-icon" />
        <div>
          <h2>结算中心</h2>
          <p className="dimmed">在此进行双人债务清偿轧差与结账补款。</p>
        </div>
      </div>

      {/* 1. 当前未结清清偿看板 */}
      <div className="glass-card settlement-glow-card balance-summary-card">
        {hasDebt && balance ? (
          <div className="balance-debt-state">
            <HelpCircle size={44} className="status-icon text-warning animate-float" />
            <div className="debt-details">
              <h4>当前待结账目</h4>
              <div className="debt-flow">
                <span className="user-glow">{debtorName}</span>
                <ArrowRight size={20} className="arrow-flow" />
                <span className="user-glow">{creditorName}</span>
              </div>
              <div className="debt-amount-large">
                ¥{centsToYuan(balance.amount_cents)}
              </div>
              <p className="dimmed-desc">
                {isDebtor
                  ? '这是您本期需要向对方汇款补款的最终差额。'
                  : '这是对方本期需要向您转账汇款的最终差额。'}
              </p>
            </div>
            <button
              className="btn-primary btn-settle-action"
              onClick={() => setShowConfirmModal(true)}
            >
              一键登记结算
            </button>
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
        {personalDetails.map((item) => (
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
        ))}
      </div>

      {/* 3. 已结算记录历史列表 */}
      <div className="glass-card history-settlement-card">
        <div className="subcard-header">
          <Calendar size={18} className="subcard-icon" />
          <h3>当期结算流水历史 ({currentMonth})</h3>
        </div>
        <div className="timeline-list">
          {historyList && historyList.length > 0 ? (
            historyList.map((item) => (
              <div className="timeline-item" key={item.id}>
                <div className="item-left">
                  <span className="type-badge badge-settle">结算</span>
                  <div className="tx-details">
                    <span className="tx-title">
                      {getUserDisplayName(item.from_user_id)} ➔ {getUserDisplayName(item.to_user_id)}
                    </span>
                    <span className="tx-meta">
                      {formatDate(item.occurred_at)}
                      {item.note && ` · 备注: ${item.note}`}
                    </span>
                  </div>
                </div>
                <div className="tx-amount val-settle">
                  ¥{centsToYuan(item.amount_cents)}
                </div>
              </div>
            ))
          ) : (
            <div className="list-empty-state">
              <p>该账期暂无已执行的结算记录</p>
            </div>
          )}
        </div>
      </div>

      {/* 4. 二次确认模态框 */}
      {showConfirmModal && balance && (
        <div className="drawer-overlay glass-blur show" onClick={() => setShowConfirmModal(false)}>
          <div
            className="drawer-container glass-card confirm-modal-box"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="drawer-header">
              <div className="header-title">
                <DollarSign className="title-icon text-glow" />
                <h3>确认登记结算补款</h3>
              </div>
            </div>
            <div className="drawer-body modal-body-padding">
              <p className="modal-alert-text">
                确认执行以下轧差清偿结算补款吗？该操作将建立结算补款凭证，并在交易流水中生成一条共享的结算明细记录。
              </p>

              <div className="modal-transfer-card">
                <div className="transfer-party">
                  <span className="party-name">{debtorName}</span>
                  <span className="party-role">付款方</span>
                </div>
                <ArrowRight size={24} className="transfer-arrow" />
                <div className="transfer-party">
                  <span className="party-name">{creditorName}</span>
                  <span className="party-role">收款方</span>
                </div>
              </div>

              <div className="modal-amount-display">
                <span className="amount-label">结算清偿金额</span>
                <span className="amount-val">¥{centsToYuan(balance.amount_cents)}</span>
              </div>

              <div className="form-group">
                <label className="form-label">结算备注 (选填)</label>
                <input
                  type="text"
                  placeholder="例如: 微信已转账结清"
                  className="form-input"
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                />
              </div>
            </div>
            <div className="drawer-footer">
              <button
                type="button"
                className="btn-secondary"
                onClick={() => setShowConfirmModal(false)}
              >
                取消
              </button>
              <button
                type="button"
                className="btn-primary btn-submit"
                onClick={handleConfirmSettlement}
                disabled={createSettlementMutation.isPending}
              >
                {createSettlementMutation.isPending ? (
                  <>
                    <Loader2 size={16} className="spinner" />
                    <span>执行结算中...</span>
                  </>
                ) : (
                  <span>确认已结算</span>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
