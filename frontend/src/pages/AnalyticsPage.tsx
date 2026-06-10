import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useUIStore } from '../stores/ui.store';
import { reportsApi } from '../api/reports.api';
import { centsToYuan } from '../utils/money';
import { 
  BarChart3, 
  TrendingUp, 
  TrendingDown, 
  PieChart, 
  Users, 
  Tag, 
  DollarSign,
  Info,
  Calendar,
  Layers
} from 'lucide-react';

type TabType = 'monthly' | 'category' | 'member' | 'tag';

export default function AnalyticsPage() {
  const { currentMonth } = useUIStore();
  const [activeTab, setActiveTab] = useState<TabType>('monthly');

  // 1. 获取月度汇总
  const { data: monthlyData, isLoading: monthlyLoading, error: monthlyError } = useQuery({
    queryKey: ['reports-monthly', currentMonth],
    queryFn: () => reportsApi.getMonthlySummary(currentMonth),
    enabled: activeTab === 'monthly',
  });

  // 2. 获取分类汇总
  const { data: categoryData, isLoading: categoryLoading, error: categoryError } = useQuery({
    queryKey: ['reports-category', currentMonth],
    queryFn: () => reportsApi.getCategorySummary(currentMonth),
    enabled: activeTab === 'category',
  });

  // 3. 获取标签汇总
  const { data: tagData, isLoading: tagLoading, error: tagError } = useQuery({
    queryKey: ['reports-tag', currentMonth],
    queryFn: () => reportsApi.getTagSummary(currentMonth),
    enabled: activeTab === 'tag',
  });

  // 4. 获取成员汇总
  const { data: memberData, isLoading: memberLoading, error: memberError } = useQuery({
    queryKey: ['reports-member', currentMonth],
    queryFn: () => reportsApi.getMemberSummary(currentMonth),
    enabled: activeTab === 'member',
  });

  const isLoading = monthlyLoading || categoryLoading || tagLoading || memberLoading;
  const hasError = monthlyError || categoryError || tagError || memberError;

  return (
    <div className="page-content animate-fade-in text-left">
      {/* 头部 Banner */}
      <div className="glass-card header-banner">
        <BarChart3 className="banner-icon" />
        <div>
          <h2>财务统计与分析</h2>
          <p className="dimmed">基于真实消费口径的账本多维报表分析</p>
        </div>
      </div>

      {/* 选项卡 Tab 控制栏 */}
      <div className="segmented-control" style={{ marginBottom: '20px' }}>
        <button 
          onClick={() => setActiveTab('monthly')}
          className={`segment-btn ${activeTab === 'monthly' ? 'active' : ''}`}
        >
          <Calendar size={14} style={{ display: 'inline', marginRight: '6px', verticalAlign: 'middle' }} />
          月度汇总
        </button>
        <button 
          onClick={() => setActiveTab('category')}
          className={`segment-btn ${activeTab === 'category' ? 'active' : ''}`}
        >
          <PieChart size={14} style={{ display: 'inline', marginRight: '6px', verticalAlign: 'middle' }} />
          分类占比
        </button>
        <button 
          onClick={() => setActiveTab('member')}
          className={`segment-btn ${activeTab === 'member' ? 'active' : ''}`}
        >
          <Users size={14} style={{ display: 'inline', marginRight: '6px', verticalAlign: 'middle' }} />
          成员对账
        </button>
        <button 
          onClick={() => setActiveTab('tag')}
          className={`segment-btn ${activeTab === 'tag' ? 'active' : ''}`}
        >
          <Tag size={14} style={{ display: 'inline', marginRight: '6px', verticalAlign: 'middle' }} />
          标签排行
        </button>
      </div>

      {/* 加载中状态 */}
      {isLoading && (
        <div style={{ textAlign: 'center', padding: '60px 0', color: 'var(--text-muted)' }}>
          <div className="loading-spinner" style={{ margin: '0 auto 16px' }}></div>
          <p>报表数据加载中，请稍候...</p>
        </div>
      )}

      {/* 错误态 */}
      {hasError && !isLoading && (
        <div className="error-banner" style={{ margin: '0 0 20px 0', borderRadius: '12px' }}>
          <h3>报表拉取失败</h3>
          <p>无法从服务器获取当前账期的财务数据，请检查网络或刷新重试。</p>
        </div>
      )}

      {/* 数据内容呈现区 */}
      {!isLoading && !hasError && (
        <div className="animate-fade-in">
          {/* A. 月度汇总 Tab */}
          {activeTab === 'monthly' && monthlyData && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
              <div className="stats-grid">
                <div className="glass-card stat-card total-expense">
                  <div className="card-header">
                    <span className="card-title">本月总支出</span>
                    <TrendingDown className="card-icon text-expense" />
                  </div>
                  <div className="card-value">¥{centsToYuan(monthlyData.total_expense)}</div>
                  <div className="card-desc">普通个人消费 + 共享消费之和</div>
                </div>

                <div className="glass-card stat-card total-income">
                  <div className="card-header">
                    <span className="card-title">本月总收入</span>
                    <TrendingUp className="card-icon text-income" />
                  </div>
                  <div className="card-value">¥{centsToYuan(monthlyData.total_income)}</div>
                  <div className="card-desc">本期所有的个人资金流入</div>
                </div>
              </div>

              {/* 细分账目 */}
              <div className="dashboard-grid-2">
                <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '10px' }}>
                    <Layers size={18} className="partner-highlight" />
                    <strong style={{ fontSize: '15px' }}>消费支出结构细分</strong>
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '14px' }}>
                      <span className="dimmed">共同支出 (Shared)</span>
                      <strong style={{ color: '#c084fc' }}>¥{centsToYuan(monthlyData.shared_expense)}</strong>
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '14px' }}>
                      <span className="dimmed">个人支出 (Personal)</span>
                      <strong style={{ color: 'var(--text-primary)' }}>¥{centsToYuan(monthlyData.personal_expense)}</strong>
                    </div>
                  </div>
                </div>

                <div className="glass-card settlement-glow-card" style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '10px' }}>
                    <DollarSign size={18} className="partner-highlight" />
                    <strong style={{ fontSize: '15px' }}>当前未结差额</strong>
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
                    <span className="dimmed" style={{ fontSize: '12px' }}>全局结算中心轧差金额</span>
                    <strong style={{ fontSize: '28px', color: '#ef4444', background: 'linear-gradient(90deg, #fca5a5 0%, #ef4444 100%)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
                      ¥{centsToYuan(monthlyData.settlement_amount)}
                    </strong>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* B. 分类占比 Tab */}
          {activeTab === 'category' && categoryData && (
            <div className="glass-card" style={{ padding: '24px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '20px', borderBottom: '1px solid rgba(255, 255, 255, 0.05)', paddingBottom: '12px' }}>
                <PieChart size={18} className="partner-highlight" />
                <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>本月分类消费分布</h3>
              </div>

              {categoryData.length > 0 ? (
                <div className="ranking-list">
                  {categoryData.map((item) => (
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
                  ))}
                </div>
              ) : (
                <div className="list-empty-state">
                  <p>本月暂无分类统计数据</p>
                </div>
              )}
            </div>
          )}

          {/* C. 标签排行 Tab */}
          {activeTab === 'tag' && tagData && (
            <div className="glass-card" style={{ padding: '24px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '20px', borderBottom: '1px solid rgba(255, 255, 255, 0.05)', paddingBottom: '12px' }}>
                <Tag size={18} className="partner-highlight" />
                <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>本月标签命中排行</h3>
              </div>

              <div style={{ background: 'rgba(147, 51, 234, 0.04)', border: '1px solid rgba(147, 51, 234, 0.1)', padding: '10px 14px', borderRadius: '8px', display: 'flex', alignItems: 'flex-start', gap: '8px', fontSize: '11px', color: '#c084fc', marginBottom: '20px' }}>
                <Info size={14} style={{ marginTop: '2px', flexShrink: 0 }} />
                <span>标签采用“全额合并命中口径”统计，多标签账单会被分别重复计入对应标签中。标签总和不作为账期总校验额。</span>
              </div>

              {tagData.length > 0 ? (
                <div className="ranking-list">
                  {tagData.map((item) => (
                    <div className="category-ranking-row" key={item.name}>
                      <div className="category-ranking-info">
                        <span className="ranking-name">#{item.name}</span>
                        <span className="ranking-amount">
                          ¥{centsToYuan(item.amount_cents)}
                          <span className="percent-text">({item.percent.toFixed(1)}%)</span>
                        </span>
                      </div>
                      <div className="progress-bar-bg">
                        <div
                          className="progress-bar-fill"
                          style={{ width: `${Math.min(100, Math.max(0, item.percent))}%`, background: 'linear-gradient(90deg, #3b82f6 0%, #10b981 100%)' }}
                        ></div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="list-empty-state">
                  <p>本月暂无标签统计数据</p>
                </div>
              )}
            </div>
          )}

          {/* D. 成员对账 Tab */}
          {activeTab === 'member' && memberData && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
              <div className="glass-card" style={{ padding: '24px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '12px', marginBottom: '20px' }}>
                  <Users size={18} className="partner-highlight" />
                  <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>成员收支承担明细</h3>
                </div>

                <div className="form-row-2">
                  {memberData.map((m) => {
                    const finalNet = m.final_net;
                    let statusLabel = '';
                    let statusClass = '';

                    if (finalNet > 0) {
                      statusLabel = `本月仍应收款 ¥${centsToYuan(finalNet)}`;
                      statusClass = 'text-green';
                    } else if (finalNet < 0) {
                      statusLabel = `本月仍应付款 ¥${centsToYuan(-finalNet)}`;
                      statusClass = 'val-expense';
                    } else {
                      statusLabel = '本期账目已结清';
                      statusClass = 'dimmed';
                    }

                    return (
                      <div 
                        key={m.user_id} 
                        className="glass-card" 
                        style={{ padding: '16px', display: 'flex', flexDirection: 'column', gap: '14px', background: 'rgba(255,255,255,0.01)', border: '1px solid rgba(255,255,255,0.03)' }}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '8px' }}>
                          <strong style={{ fontSize: '14px', color: 'var(--text-primary)' }}>{m.display_name}</strong>
                          <span style={{ fontSize: '11px' }} className={statusClass}>● {statusLabel}</span>
                        </div>

                        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', fontSize: '12px' }}>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <span className="dimmed">实际支付 (Paid)</span>
                            <span>¥{centsToYuan(m.paid_amount)}</span>
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <span className="dimmed">消费承担 (Share)</span>
                            <span>¥{centsToYuan(m.share_amount)}</span>
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between', fontWeight: 500, borderTop: '1px dashed rgba(255,255,255,0.04)', paddingTop: '6px' }}>
                            <span className="dimmed">垫付净额 (Raw Net)</span>
                            <span className={m.raw_net >= 0 ? 'text-green' : 'val-expense'}>
                              {m.raw_net >= 0 ? '+' : ''}¥{centsToYuan(m.raw_net)}
                            </span>
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <span className="dimmed">已付结算 (Settlement Paid)</span>
                            <span>¥{centsToYuan(m.settlement_paid)}</span>
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <span className="dimmed">已收结算 (Settlement Received)</span>
                            <span>¥{centsToYuan(m.settlement_received)}</span>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
