import { useQuery } from '@tanstack/react-query';
import type { UserStatItem } from '../types/dashboard';
import { useAuthStore } from '../stores/auth.store';
import { useUIStore } from '../stores/ui.store';
import { dashboardApi } from '../api/dashboard.api';
import { centsToYuan } from '../utils/money';
import { formatDate } from '../utils/date';
import SkeletonTable from '../components/ui/SkeletonTable';
import ErrorState from '../components/ui/ErrorState';
import EmptyState from '../components/ui/EmptyState';
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
  const { data: dashboardData, isLoading, error, refetch } = useQuery({
    queryKey: ['dashboard', currentMonth],
    queryFn: () => dashboardApi.getDashboard(currentMonth),
    enabled: !!currentUser,
  });

  const handleQuickAdd = () => {
    setAddDrawerOpen(true);
  };

  const getPayerName = (payerId: string, userStats: UserStatItem[] | undefined) => {
    if (payerId === currentUser?.id) return '我';
    const partner = userStats?.find((u) => u.user_id !== currentUser?.id);
    if (partner && payerId === partner.user_id) return partner.display_name;
    return '伙伴';
  };

  // 判定当月账期是否彻底为空，以便显示空状态引导
  const isMonthEmpty = dashboardData 
    ? (dashboardData.total_expense_cents === 0 && 
       dashboardData.total_income_cents === 0 && 
       (!dashboardData.recent_transactions || dashboardData.recent_transactions.length === 0))
    : false;

  return (
    <div className="page-content animate-fade-in text-left">
      {/* 顶部迎宾栏与快捷按钮 */}
      <div className="glass-card header-banner dashboard-header-banner">
        <div className="banner-title-area">
          <Sparkles className="banner-icon" />
          <div>
            <h2>欢迎回来，{currentUser?.display_name}</h2>
            <p className="dimmed">这是你们在 {currentMonth} 账期的共享记账空间。</p>
          </div>
        </div>
        <button className="btn-primary quick-add-btn" onClick={handleQuickAdd}>
          <PlusCircle size={18} />
          <span>记一笔</span>
        </button>
      </div>

      {/* 异常错误态渲染 */}
      {error && (
        <ErrorState 
          title="系统账务大屏获取失败" 
          message={error instanceof Error ? error.message : '请检查网络或重新登录后重试'} 
          onRetry={refetch} 
        />
      )}

      {!error && (
        <>
          {/* 本月收支大卡片 */}
          <div className="stats-grid">
            <div className="glass-card stat-card total-expense">
              <div className="card-header">
                <span className="card-title">本月总支出</span>
                <TrendingDown className="card-icon text-expense" />
              </div>
              <div className="card-value">
                {isLoading ? (
                  <div className="skeleton-item" style={{ height: '36px', width: '120px' }}></div>
                ) : (
                  `¥${centsToYuan(dashboardData?.total_expense_cents || 0)}`
                )}
              </div>
              <div className="card-desc">
                {isLoading ? (
                  <div className="skeleton-item" style={{ height: '14px', width: '160px', marginTop: '8px' }}></div>
                ) : (
                  '包含个人及共同分摊消费'
                )}
              </div>
            </div>

            <div className="glass-card stat-card total-income">
              <div className="card-header">
                <span className="card-title">本月总收入</span>
                <TrendingUp className="card-icon text-income" />
              </div>
              <div className="card-value">
                {isLoading ? (
                  <div className="skeleton-item" style={{ height: '36px', width: '120px' }}></div>
                ) : (
                  `¥${centsToYuan(dashboardData?.total_income_cents || 0)}`
                )}
              </div>
              <div className="card-desc">
                {isLoading ? (
                  <div className="skeleton-item" style={{ height: '14px', width: '160px', marginTop: '8px' }}></div>
                ) : (
                  '个人独立录入的资金进账'
                )}
              </div>
            </div>
          </div>

          {/* 当月空状态处理 */}
          {isMonthEmpty && !isLoading ? (
            <EmptyState 
              title="本月账期暂无账单数据"
              description="你们在这个月份还没有记录过任何消费或收入流水。点击右上角「记一笔」，记录这笔账单吧！"
              actionText="开始记账"
              onAction={handleQuickAdd}
            />
          ) : (
            <>
              {/* 两端垫付及结算卡片 */}
              <div className="dashboard-grid-2">
                {/* 双端垫付统计 */}
                <div className="glass-card dashboard-subcard">
                  <div className="subcard-header">
                    <UserCheck size={20} className="subcard-icon text-a" />
                    <h3>当月垫付与实际承担</h3>
                  </div>
                  {isLoading ? (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', padding: '8px 0' }}>
                      <div className="skeleton-item" style={{ height: '40px', width: '100%', borderRadius: '8px' }}></div>
                      <div className="skeleton-item" style={{ height: '40px', width: '100%', borderRadius: '8px' }}></div>
                    </div>
                  ) : (
                    <div className="stat-split-list">
                      <div className="stat-split-item">
                        <span className="user-name">我 ({currentUser?.display_name})</span>
                        <div className="amounts-line">
                          <span className="paid-val">垫付 ¥{centsToYuan(dashboardData?.my_paid_cents || 0)}</span>
                          <span className="divider">/</span>
                          <span className="share-val">
                            承担 ¥{centsToYuan(dashboardData?.user_stats?.find(u => u.user_id === currentUser?.id)?.share_cents || 0)}
                          </span>
                        </div>
                      </div>
                      {dashboardData?.user_stats?.find(u => u.user_id !== currentUser?.id) && (
                        <div className="stat-split-item">
                          <span className="user-name">
                            伙伴 ({dashboardData.user_stats.find(u => u.user_id !== currentUser?.id)?.display_name})
                          </span>
                          <div className="amounts-line">
                            <span className="paid-val">垫付 ¥{centsToYuan(dashboardData.partner_paid_cents)}</span>
                            <span className="divider">/</span>
                            <span className="share-val">
                              承担 ¥{centsToYuan(dashboardData.user_stats.find(u => u.user_id !== currentUser?.id)?.share_cents || 0)}
                            </span>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>

                {/* 全局未结清结算提醒 */}
                <div className="glass-card dashboard-subcard settlement-glow-card">
                  <div className="subcard-header">
                    <DollarSign size={20} className="subcard-icon text-b" />
                    <h3>全局未结余额 (跨月)</h3>
                  </div>
                  {isLoading ? (
                    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100px' }}>
                      <div className="skeleton-item" style={{ height: '60px', width: '80%', borderRadius: '12px' }}></div>
                    </div>
                  ) : (
                    <div className="settlement-alert-body">
                      {dashboardData?.shared_balance && dashboardData.shared_balance.amount_cents > 0 ? (
                        <div className="debt-indicator">
                          <div className="debt-status-text">
                            {dashboardData.shared_balance.from_user_id === currentUser?.id ? (
                              <>
                                <p className="debt-callout">
                                  你需要向 <strong className="partner-highlight">
                                    {dashboardData.user_stats?.find(u => u.user_id === dashboardData.shared_balance.to_user_id)?.display_name || '对方'}
                                  </strong> 结清
                                </p>
                                <div className="debt-amount-big">¥{centsToYuan(dashboardData.shared_balance.amount_cents)}</div>
                              </>
                            ) : (
                              <>
                                <p className="debt-callout">
                                  <strong className="partner-highlight">
                                    {dashboardData.user_stats?.find(u => u.user_id === dashboardData.shared_balance.from_user_id)?.display_name || '对方'}
                                  </strong> 需要向你支付
                                </p>
                                <div className="debt-amount-big text-green">¥{centsToYuan(dashboardData.shared_balance.amount_cents)}</div>
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
                  )}
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
                  {isLoading ? (
                    <SkeletonTable rows={3} />
                  ) : (
                    <div className="ranking-list">
                      {dashboardData?.category_summary && dashboardData.category_summary.length > 0 ? (
                        dashboardData.category_summary.map((item) => (
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
                  )}
                </div>

                {/* 最近流水 */}
                <div className="glass-card dashboard-subcard">
                  <div className="subcard-header">
                    <Clock size={20} className="subcard-icon" />
                    <h3>最近流水时间线 (Max 10)</h3>
                  </div>
                  {isLoading ? (
                    <SkeletonTable rows={4} />
                  ) : (
                    <div className="timeline-list">
                      {dashboardData?.recent_transactions && dashboardData.recent_transactions.length > 0 ? (
                        dashboardData.recent_transactions.map((tx) => {
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
                                    {formatDate(tx.occurred_at).substring(5, 16)} · {getPayerName(tx.payer_user_id, dashboardData.user_stats)}付
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
                  )}
                </div>
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
}
