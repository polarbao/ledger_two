import { useQuery } from '@tanstack/react-query';
import { useAuthStore } from '../stores/auth.store';
import { useUIStore } from '../stores/ui.store';
import { dashboardApi } from '../api/dashboard.api';
import { centsToYuan } from '../utils/money';
import { formatDate } from '../utils/date';
import {
  TrendingUp,
  TrendingDown,
  Sparkles,
  DollarSign,
  PlusCircle,
  Clock,
  PieChart,
  UserCheck,
} from 'lucide-react';

export default function DashboardPage() {
  const currentUser = useAuthStore((state) => state.user);
  const { currentMonth, setAddDrawerOpen } = useUIStore();

  // 1. 请求数据并绑定依赖 currentMonth 自动重载
  const { data, isLoading, error } = useQuery({
    queryKey: ['dashboard', currentMonth],
    queryFn: () => dashboardApi.getDashboard(currentMonth),
    enabled: !!currentUser,
  });

  if (isLoading) {
    return (
      <div className="app-loading">
        <div className="loading-spinner"></div>
        <p>数据加载中，请稍候...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="page-content">
        <div className="error-banner">
          <h3>数据加载失败</h3>
          <p>{error instanceof Error ? error.message : '请检查您的网络连接或尝试重新登录。'}</p>
        </div>
      </div>
    );
  }

  const dashboardData = data;
  if (!dashboardData) return null;

  const {
    total_expense_cents,
    total_income_cents,
    my_paid_cents,
    partner_paid_cents,
    shared_balance,
    recent_transactions,
    category_summary,
    user_stats,
  } = dashboardData;

  // 内存级计算从 ID 转换名字及债务状态，消除 DTO 不一致隐藏 Bug
  const fromUserName = user_stats?.find((u) => u.user_id === shared_balance?.from_user_id)?.display_name || '对方';
  const toUserName = user_stats?.find((u) => u.user_id === shared_balance?.to_user_id)?.display_name || '对方';
  const hasDebt = shared_balance ? shared_balance.amount_cents > 0 : false;

  // 2. 查找伙伴的显示名字，用来优化列表里的 payer 展示
  const partner = user_stats?.find((u) => u.user_id !== currentUser?.id);
  const getPayerName = (payerId: string) => {
    if (payerId === currentUser?.id) return '我';
    if (partner && payerId === partner.user_id) return partner.display_name;
    return '伙伴';
  };

  const handleQuickAdd = () => {
    setAddDrawerOpen(true);
  };

  return (
    <div className="page-content animate-fade-in">
      {/* 顶部迎宾栏与快捷按钮 */}
      <div className="glass-card header-banner dashboard-header-banner">
        <div className="banner-title-area">
          <Sparkles className="banner-icon" />
          <div>
            <h2>欢迎回来，{currentUser?.displayName}</h2>
            <p className="dimmed">这是你们在 {currentMonth} 账期的共享记账空间。</p>
          </div>
        </div>
        <button className="btn-primary quick-add-btn" onClick={handleQuickAdd}>
          <PlusCircle size={18} />
          <span>记一笔</span>
        </button>
      </div>

      {/* 本月收支大卡片 */}
      <div className="stats-grid">
        <div className="glass-card stat-card total-expense">
          <div className="card-header">
            <span className="card-title">本月总支出</span>
            <TrendingDown className="card-icon text-expense" />
          </div>
          <div className="card-value">¥{centsToYuan(total_expense_cents)}</div>
          <div className="card-desc">包含个人及共同分摊消费</div>
        </div>

        <div className="glass-card stat-card total-income">
          <div className="card-header">
            <span className="card-title">本月总收入</span>
            <TrendingUp className="card-icon text-income" />
          </div>
          <div className="card-value">¥{centsToYuan(total_income_cents)}</div>
          <div className="card-desc">个人独立录入的资金进账</div>
        </div>
      </div>

      {/* 两端垫付及结算卡片 */}
      <div className="dashboard-grid-2">
        {/* 双端垫付统计 */}
        <div className="glass-card dashboard-subcard">
          <div className="subcard-header">
            <UserCheck size={20} className="subcard-icon text-a" />
            <h3>当月垫付与实际承担</h3>
          </div>
          <div className="stat-split-list">
            <div className="stat-split-item">
              <span className="user-name">我 ({currentUser?.displayName})</span>
              <div className="amounts-line">
                <span className="paid-val">垫付 ¥{centsToYuan(my_paid_cents)}</span>
                <span className="divider">/</span>
                <span className="share-val">承担 ¥{centsToYuan(user_stats?.find(u => u.user_id === currentUser?.id)?.share_cents || 0)}</span>
              </div>
            </div>
            {partner && (
              <div className="stat-split-item">
                <span className="user-name">伙伴 ({partner.display_name})</span>
                <div className="amounts-line">
                  <span className="paid-val">垫付 ¥{centsToYuan(partner_paid_cents)}</span>
                  <span className="divider">/</span>
                  <span className="share-val">承担 ¥{centsToYuan(partner.share_cents)}</span>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* 全局未结清结算提醒 */}
        <div className="glass-card dashboard-subcard settlement-glow-card">
          <div className="subcard-header">
            <DollarSign size={20} className="subcard-icon text-b" />
            <h3>全局未结余额 (跨月)</h3>
          </div>
          <div className="settlement-alert-body">
            {hasDebt && shared_balance ? (
              <div className="debt-indicator">
                <div className="debt-status-text">
                  {shared_balance.from_user_id === currentUser?.id ? (
                    <>
                      <p className="debt-callout">你需要向 <strong className="partner-highlight">{toUserName}</strong> 结清</p>
                      <div className="debt-amount-big">¥{centsToYuan(shared_balance.amount_cents)}</div>
                    </>
                  ) : (
                    <>
                      <p className="debt-callout"><strong className="partner-highlight">{fromUserName}</strong> 需要向你支付</p>
                      <div className="debt-amount-big text-green">¥{centsToYuan(shared_balance.amount_cents)}</div>
                    </>
                  )}
                </div>
                <p className="debt-sub-info">结账可抵扣净额，在结算中心生成记录即可。</p>
              </div>
            ) : (
              <div className="debt-settled-state">
                <div className="settled-shield">✓</div>
                <p className="settled-text">账目已结清，本月暂无未结账单！</p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* 排行榜与最近流水 */}
      <div className="dashboard-grid-2">
        {/* 消费分类占比 Top N */}
        <div className="glass-card dashboard-subcard">
          <div className="subcard-header">
            <PieChart size={20} className="subcard-icon" />
            <h3>分类消费排行 (Top N)</h3>
          </div>
          <div className="ranking-list">
            {category_summary && category_summary.length > 0 ? (
              category_summary.map((item) => (
                <div className="category-ranking-row" key={item.id}>
                  <div className="category-ranking-info">
                    <span className="ranking-name">{item.name}</span>
                    <span className="ranking-amount">
                      ¥{centsToYuan(item.amount_cents)}
                      <span className="percent-text">({item.percent.toFixed(1)}%)</span>
                    </span>
                  </div>
                  <div className="progress-bar-bg">
                    <div
                      className="progress-bar-fill"
                      style={{ width: `${Math.min(100, Math.max(0, item.percent))}%` }}
                    ></div>
                  </div>
                </div>
              ))
            ) : (
              <div className="list-empty-state">
                <p>本月暂无分类消费数据</p>
              </div>
            )}
          </div>
        </div>

        {/* 最近流水 */}
        <div className="glass-card dashboard-subcard">
          <div className="subcard-header">
            <Clock size={20} className="subcard-icon" />
            <h3>最近流水时间线 (Max 10)</h3>
          </div>
          <div className="timeline-list">
            {recent_transactions && recent_transactions.length > 0 ? (
              recent_transactions.map((tx) => {
                let badgeClass = '';
                let badgeLabel = '';
                let amountSign = '';
                let amountClass = '';

                switch (tx.type) {
                  case 'expense':
                    badgeClass = 'badge-expense';
                    badgeLabel = '个人';
                    amountSign = '-';
                    amountClass = 'val-expense';
                    break;
                  case 'income':
                    badgeClass = 'badge-income';
                    badgeLabel = '收入';
                    amountSign = '+';
                    amountClass = 'val-income';
                    break;
                  case 'shared_expense':
                    badgeClass = 'badge-shared';
                    badgeLabel = '共享';
                    amountSign = '-';
                    amountClass = 'val-expense';
                    break;
                  case 'settlement':
                    badgeClass = 'badge-settle';
                    badgeLabel = '结算';
                    amountSign = '';
                    amountClass = 'val-settle';
                    break;
                }

                return (
                  <div className="timeline-item" key={tx.id}>
                    <div className="item-left">
                      <span className={`type-badge ${badgeClass}`}>{badgeLabel}</span>
                      <div className="tx-details">
                        <span className="tx-title">{tx.title}</span>
                        <span className="tx-meta">
                          {formatDate(tx.occurred_at)} · {getPayerName(tx.payer_user_id)}付
                        </span>
                      </div>
                    </div>
                    <div className={`tx-amount ${amountClass}`}>
                      {amountSign}¥{centsToYuan(tx.amount_cents)}
                    </div>
                  </div>
                );
              })
            ) : (
              <div className="list-empty-state">
                <p>本月暂无交易明细流水</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
