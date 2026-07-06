import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useUIStore } from '../stores/ui.store';
import { useAuthStore } from '../stores/auth.store';
import { transactionsApi } from '../api/transactions.api';
import { dashboardApi } from '../api/dashboard.api';
import { queryKeys } from '../api/queryKeys';
import { useLedgerStore } from '../stores/ledger.store';
import { centsToYuan } from '../utils/money';
import { formatDate } from '../utils/date';
import PageState from '../components/ui/PageState';
import PermissionGate from '../components/ledger/PermissionGate';
import { 
  ReceiptText, 
  Trash2, 
  ChevronLeft, 
  ChevronRight, 
  X, 
  Info,
  SlidersHorizontal,
  RotateCcw,
  Tags
} from 'lucide-react';
import type { TransactionResponse } from '../types/transaction';
import TransactionCard from '../components/transaction/TransactionCard';

export default function TransactionsPage() {
  const currentMonth = useUIStore((state) => state.currentMonth);
  const currentUser = useAuthStore((state) => state.user);
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const queryClient = useQueryClient();

  const [searchParams, setSearchParams] = useSearchParams();

  // 从 URL 读取筛选参数
  const month = searchParams.get('month') || currentMonth;
  const type = searchParams.get('type') || '';
  const categoryId = searchParams.get('category_id') || '';
  const keyword = searchParams.get('keyword') || '';
  const minAmountStr = searchParams.get('min_amount') || ''; // 元
  const maxAmountStr = searchParams.get('max_amount') || ''; // 元
  const payerUserId = searchParams.get('payer_user_id') || '';
  const visibility = searchParams.get('visibility') || '';
  const tag = searchParams.get('tag') || '';
  const page = parseInt(searchParams.get('page') || '1', 10);
  const pageSize = 15;

  // 局部输入框状态（防止打字时频繁触发 API 请求）
  const [localMinAmount, setLocalMinAmount] = useState(minAmountStr);
  const [localMaxAmount, setLocalMaxAmount] = useState(maxAmountStr);
  const [localKeyword, setLocalKeyword] = useState(keyword);
  const [localTag, setLocalTag] = useState(tag);

  // 渲染期间同步外部 URL 的筛选值到局部输入框状态，防范 eslint/set-state-in-effect 规则报错
  const [prevMinAmount, setPrevMinAmount] = useState(minAmountStr);
  const [prevMaxAmount, setPrevMaxAmount] = useState(maxAmountStr);
  const [prevKeyword, setPrevKeyword] = useState(keyword);
  const [prevTag, setPrevTag] = useState(tag);

  if (minAmountStr !== prevMinAmount) {
    setPrevMinAmount(minAmountStr);
    setLocalMinAmount(minAmountStr);
  }
  if (maxAmountStr !== prevMaxAmount) {
    setPrevMaxAmount(maxAmountStr);
    setLocalMaxAmount(maxAmountStr);
  }
  if (keyword !== prevKeyword) {
    setPrevKeyword(keyword);
    setLocalKeyword(keyword);
  }
  if (tag !== prevTag) {
    setPrevTag(tag);
    setLocalTag(tag);
  }

  // 0. 获取账本成员信息
  const { data: dashboardData } = useQuery({
    queryKey: queryKeys.dashboard.month(activeLedgerId, month),
    queryFn: () => dashboardApi.getDashboard(month),
  });
  const users = dashboardData?.user_stats || [];

  const [selectedTx, setSelectedTx] = useState<TransactionResponse | null>(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [activeLightboxImg, setActiveLightboxImg] = useState<string | null>(null);

  // 移动端筛选控制
  const [mobileFilterOpen, setMobileFilterOpen] = useState(false);

  // 批量操作状态
  const [batchMode, setBatchMode] = useState(false);
  const [selectedTxIds, setSelectedTxIds] = useState<string[]>([]);
  const [showBatchTagModal, setShowBatchTagModal] = useState(false);
  const [batchTagsInput, setBatchTagsInput] = useState('');

  const setCopySourceTransaction = useUIStore((state) => state.setCopySourceTransaction);
  const setOpenTemplateSaveOnDrawerOpen = useUIStore((state) => state.setOpenTemplateSaveOnDrawerOpen);
  const setAddDrawerOpen = useUIStore((state) => state.setAddDrawerOpen);

  const updateFilter = (newFilters: Record<string, string | number | undefined>) => {
    const nextParams = new URLSearchParams(searchParams);
    if (!('page' in newFilters)) {
      nextParams.set('page', '1');
    }
    Object.entries(newFilters).forEach(([key, val]) => {
      if (val === undefined || val === '') {
        nextParams.delete(key);
      } else {
        nextParams.set(key, String(val));
      }
    });
    setSearchParams(nextParams);
  };

  const handleClearFilters = () => {
    const nextParams = new URLSearchParams();
    nextParams.set('month', month);
    nextParams.set('page', '1');
    setSearchParams(nextParams);
  };

  // 1. 获取分类名称列表映射
  const { data: categories } = useQuery({
    queryKey: queryKeys.categories(activeLedgerId),
    queryFn: () => transactionsApi.getCategories(),
  });

  const catMap = categories?.reduce((acc, cat) => {
    acc[cat.id] = cat.name;
    return acc;
  }, {} as Record<string, string>) || {};

  const minAmount = minAmountStr ? Math.round(parseFloat(minAmountStr) * 100) : undefined;
  const maxAmount = maxAmountStr ? Math.round(parseFloat(maxAmountStr) * 100) : undefined;
  const transactionFilter = {
    month,
    type: type || undefined,
    category_id: categoryId || undefined,
    keyword: keyword || undefined,
    min_amount: minAmount,
    max_amount: maxAmount,
    payer_user_id: payerUserId || undefined,
    visibility: visibility || undefined,
    tag: tag || undefined,
    page,
    page_size: pageSize,
  };

  // 2. 获取流水列表
  const { 
    data: transactions, 
    isLoading, 
    isError, 
    error,
    refetch 
  } = useQuery({
    queryKey: queryKeys.transactions.list(activeLedgerId, transactionFilter),
    queryFn: () => transactionsApi.list(transactionFilter),
  });

  // 批量打标签 Mutation
  const batchTagMutation = useMutation({
    mutationFn: (payload: { transaction_ids: string[]; tag_names: string[] }) =>
      transactionsApi.batchTag(payload),
    onSuccess: () => {
      setShowBatchTagModal(false);
      setBatchTagsInput('');
      setSelectedTxIds([]);
      setBatchMode(false);
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.root(activeLedgerId) });
    },
  });

  // 3. 删除流水 Mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.deleteTransaction(id),
    onSuccess: () => {
      setShowDeleteModal(false);
      setSelectedTx(null);
      // 失效所有关联缓存，促使全局数据重载
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.root(activeLedgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.root(activeLedgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.reports.root(activeLedgerId) });
    },
  });

  const handleDeleteClick = (tx: TransactionResponse) => {
    setSelectedTx(tx);
    setShowDeleteModal(true);
  };

  const hasNextPage = transactions ? transactions.length === pageSize : false;
  const hasPrevPage = page > 1;

  const getPayerName = (payerId: string) => {
    if (payerId === currentUser?.id) return '我';
    return '伙伴';
  };

  const handleSelectTx = (id: string) => {
    setSelectedTxIds((prev) =>
      prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id]
    );
  };

  const handleSelectAll = (checked: boolean) => {
    if (checked && transactions) {
      const selectable = transactions.filter((tx) => tx.created_by_user_id === currentUser?.id).map((tx) => tx.id);
      setSelectedTxIds(selectable);
    } else {
      setSelectedTxIds([]);
    }
  };

  return (
    <div className="page-content animate-fade-in text-left">
      {/* 头部 Banner */}
      <div className="glass-card header-banner">
        <ReceiptText className="banner-icon" />
        <div>
          <h2>交易明细流水</h2>
          <p className="dimmed">在此您可以查看与维护 {currentMonth} 账期内所有的普通收支及共享支出</p>
        </div>
      </div>

      {/* 快捷控制条：高级筛选与批量管理开关 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '16px', margin: '20px 0', flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button 
            className="btn-secondary mobile-filter-trigger" 
            onClick={() => setMobileFilterOpen(true)}
          >
            <SlidersHorizontal size={15} />
            高级筛选
          </button>
        </div>

        <PermissionGate allow={['owner', 'editor']}>
          <button
            className={`btn-secondary ${batchMode ? 'active' : ''}`}
            style={{
              padding: '8px 16px',
              borderRadius: '8px',
              fontSize: '13px',
              borderColor: batchMode ? 'var(--accent-purple)' : 'rgba(255,255,255,0.08)',
              background: batchMode ? 'rgba(147, 51, 234, 0.15)' : 'none',
              color: batchMode ? '#c084fc' : 'var(--text-primary)',
            }}
            onClick={() => {
              setBatchMode(!batchMode);
              setSelectedTxIds([]);
            }}
          >
            {batchMode ? '退出批量管理' : '批量管理'}
          </button>
        </PermissionGate>
      </div>

      {/* 筛选控制面板 / 移动端 Bottom Sheet */}
      <div className={`glass-card filter-panel ${mobileFilterOpen ? 'mobile-sheet-open' : 'desktop-only'}`}>
        <div className="filter-header mobile-flex" style={{ display: 'none', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px', paddingBottom: '16px', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
          <h3 style={{ margin: 0, fontSize: '18px' }}>高级筛选</h3>
          <button className="btn-close-drawer" onClick={() => setMobileFilterOpen(false)}>
            <X size={20} />
          </button>
        </div>
        <div className="filter-grid" style={{ marginBottom: '16px' }}>
          <div className="filter-item">
            <label>账单类型</label>
            <select 
              className="filter-input" 
              value={type} 
              onChange={(e) => updateFilter({ type: e.target.value })}
            >
              <option value="">全部类型</option>
              <option value="expense">个人支出</option>
              <option value="income">个人收入</option>
              <option value="shared_expense">共同支出</option>
            </select>
          </div>

          <div className="filter-item">
            <label>所属分类</label>
            <select 
              className="filter-input" 
              value={categoryId} 
              onChange={(e) => updateFilter({ category_id: e.target.value })}
            >
              <option value="">全部分类</option>
              {categories?.map((cat) => (
                <option key={cat.id} value={cat.id}>{cat.name}</option>
              ))}
            </select>
          </div>

          <div className="filter-item">
            <label>付款人</label>
            <select 
              className="filter-input" 
              value={payerUserId} 
              onChange={(e) => updateFilter({ payer_user_id: e.target.value })}
            >
              <option value="">全部付款人</option>
              {users.map((u) => (
                <option key={u.user_id} value={u.user_id}>
                  {u.display_name} {u.user_id === currentUser?.id ? '(我)' : ''}
                </option>
              ))}
            </select>
          </div>

          <div className="filter-item">
            <label>可见性</label>
            <select 
              className="filter-input" 
              value={visibility} 
              onChange={(e) => updateFilter({ visibility: e.target.value })}
            >
              <option value="">全部可见性</option>
              <option value="private">仅自己可见</option>
              <option value="partner_readable">对方可见 (只读)</option>
            </select>
          </div>

          <div className="filter-item">
            <label>关联标签</label>
            <input 
              type="text" 
              placeholder="模糊搜索标签..." 
              className="filter-input"
              value={localTag}
              onChange={(e) => setLocalTag(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && updateFilter({ tag: localTag })}
            />
          </div>

          <div className="filter-item">
            <label>最小金额 (元)</label>
            <input 
              type="number" 
              placeholder="最小金额..." 
              className="filter-input"
              value={localMinAmount}
              onChange={(e) => setLocalMinAmount(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && updateFilter({ min_amount: localMinAmount })}
            />
          </div>

          <div className="filter-item">
            <label>最大金额 (元)</label>
            <input 
              type="number" 
              placeholder="最大金额..." 
              className="filter-input"
              value={localMaxAmount}
              onChange={(e) => setLocalMaxAmount(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && updateFilter({ max_amount: localMaxAmount })}
            />
          </div>

          <div className="filter-item">
            <label>关键词</label>
            <input 
              type="text" 
              placeholder="搜索标题或备注..." 
              className="filter-input"
              value={localKeyword}
              onChange={(e) => setLocalKeyword(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && updateFilter({ keyword: localKeyword })}
            />
          </div>
        </div>

        <div className="filter-buttons">
          <button 
            className="btn-secondary" 
            style={{ padding: '6px 14px', fontSize: '13px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '4px' }}
            onClick={handleClearFilters}
          >
            <RotateCcw size={14} />
            清空筛选
          </button>
          <button 
            className="btn-primary" 
            style={{ padding: '6px 16px', fontSize: '13px', borderRadius: '8px' }}
            onClick={() => updateFilter({ 
              min_amount: localMinAmount, 
              max_amount: localMaxAmount, 
              keyword: localKeyword,
              tag: localTag
            })}
          >
            应用筛选
          </button>
        </div>
        {/* 移动端应用后关闭弹窗 */}
        {mobileFilterOpen && (
          <div style={{ marginTop: '16px' }}>
            <button 
              className="btn-primary mobile-full"
              style={{ padding: '12px' }}
              onClick={() => {
                updateFilter({ 
                  min_amount: localMinAmount, 
                  max_amount: localMaxAmount, 
                  keyword: localKeyword,
                  tag: localTag
                });
                setMobileFilterOpen(false);
              }}
            >
              确认并查看结果
            </button>
          </div>
        )}
      </div>

      {/* 批量操作悬浮控制条 */}
      <div className={`glass-card batch-actions-bar ${batchMode && selectedTxIds.length > 0 ? 'show' : ''}`}>
        <span style={{ fontSize: '14px', fontWeight: 500 }}>
          已选择 <strong style={{ color: 'var(--accent-purple)' }}>{selectedTxIds.length}</strong> 笔账单
        </span>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button 
            className="btn-primary" 
            style={{ padding: '8px 16px', fontSize: '13px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '4px' }}
            onClick={() => setShowBatchTagModal(true)}
          >
            <Tags size={15} />
            批量打标签
          </button>
        </div>
      </div>

      {/* 主展示区域 */}
      <PageState 
        isLoading={isLoading} 
        isError={isError} 
        isEmpty={transactions ? transactions.length === 0 : true}
        errorMsg={error instanceof Error ? error.message : '拉取流水明细失败'}
        emptyMessage="暂无任何匹配的流水明细。请尝试调整筛选条件，或点击上方「记一笔」开启首笔记账。"
        skeletonType="table"
        onRetry={refetch}
      >
        {transactions && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            
            {/* 移动端卡片列表 */}
            <div className="mobile-only">
              {transactions.map(tx => (
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', position: 'relative' }} key={tx.id}>
                  {batchMode && tx.created_by_user_id === currentUser?.id && (
                     <div style={{ padding: '0 8px' }}>
                       <input 
                         type="checkbox" 
                         className="checkbox-input"
                         checked={selectedTxIds.includes(tx.id)}
                         onChange={() => handleSelectTx(tx.id)}
                       />
                     </div>
                  )}
                  <div style={{ flex: 1 }}>
                    <TransactionCard 
                      tx={tx} 
                      currentUserId={currentUser?.id || ''} 
                      onClick={() => { setSelectedTx(tx); setDetailOpen(true); }}
                    />
                  </div>
                </div>
              ))}
            </div>

            {/* 桌面端表格 */}
            <div className="glass-card desktop-only" style={{ overflowX: 'auto' }}>
              <table className="transaction-table" style={{ width: '100%', borderCollapse: 'collapse', fontSize: '14px' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid rgba(255,255,255,0.06)', color: 'var(--text-secondary)' }}>
                    {batchMode && (
                      <th className="checkbox-col">
                        <input 
                          type="checkbox" 
                          className="checkbox-input"
                          checked={transactions.length > 0 && transactions.filter(tx => tx.created_by_user_id === currentUser?.id).length > 0 && selectedTxIds.length === transactions.filter(tx => tx.created_by_user_id === currentUser?.id).length}
                          onChange={(e) => handleSelectAll(e.target.checked)}
                        />
                      </th>
                    )}
                    <th style={{ padding: '16px 20px', textAlign: 'left', fontWeight: 500 }}>记账日期</th>
                    <th style={{ padding: '16px 20px', textAlign: 'left', fontWeight: 500 }}>类型</th>
                    <th style={{ padding: '16px 20px', textAlign: 'left', fontWeight: 500 }}>分类</th>
                    <th style={{ padding: '16px 20px', textAlign: 'left', fontWeight: 500 }}>流水标题</th>
                    <th style={{ padding: '16px 20px', textAlign: 'left', fontWeight: 500 }}>付款人</th>
                    <th style={{ padding: '16px 20px', textAlign: 'left', fontWeight: 500 }}>分摊方式</th>
                    <th style={{ padding: '16px 20px', textAlign: 'right', fontWeight: 500 }}>交易金额</th>
                    <th style={{ padding: '16px 20px', textAlign: 'center', fontWeight: 500 }}>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {transactions.map((tx) => {
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

                    // 权限控制：仅允许创建者删除，且不能删除结算
                    const canDelete = tx.created_by_user_id === currentUser?.id && tx.type !== 'settlement';
                    const canEdit = tx.created_by_user_id === currentUser?.id;

                    return (
                      <tr 
                        key={tx.id} 
                        style={{ borderBottom: '1px solid rgba(255,255,255,0.03)', transition: 'background 0.2s', cursor: 'pointer' }}
                        className="table-row-hover"
                        onClick={() => { setSelectedTx(tx); setDetailOpen(true); }}
                      >
                        {batchMode && (
                          <td className="checkbox-col" onClick={(e) => e.stopPropagation()}>
                            <input 
                              type="checkbox" 
                              className="checkbox-input"
                              checked={selectedTxIds.includes(tx.id)}
                              disabled={!canEdit}
                              onChange={() => handleSelectTx(tx.id)}
                            />
                          </td>
                        )}
                        <td style={{ padding: '14px 20px', color: 'var(--text-secondary)' }}>
                          {formatDate(tx.occurred_at).substring(5, 16)}
                        </td>
                        <td style={{ padding: '14px 20px' }}>
                          <span className={`type-badge ${badgeClass}`}>{badgeLabel}</span>
                        </td>
                        <td style={{ padding: '14px 20px', color: 'var(--text-primary)' }}>
                          {tx.category_id ? catMap[tx.category_id] || '已设分类' : '未分类'}
                        </td>
                        <td style={{ padding: '14px 20px', color: 'var(--text-primary)', fontWeight: 500 }}>
                          {tx.title}
                          {tx.note && (
                            <span style={{ display: 'block', fontSize: '11px', color: 'var(--text-muted)', marginTop: '2px', fontWeight: 400 }}>
                              {tx.note}
                            </span>
                          )}
                        </td>
                        <td style={{ padding: '14px 20px', color: 'var(--text-secondary)' }}>
                          {getPayerName(tx.payer_user_id)}
                        </td>
                        <td style={{ padding: '14px 20px', color: 'var(--text-muted)', fontSize: '12px' }}>
                          {tx.type === 'shared_expense' 
                            ? (tx.split_method === 'equal' ? '均摊 (Equal)' : '垫付 (Payer)') 
                            : '—'}
                        </td>
                        <td style={{ padding: '14px 20px', textAlign: 'right', fontWeight: 600 }} className={amountClass}>
                          {amountSign}¥{centsToYuan(tx.amount_cents)}
                        </td>
                        <td style={{ padding: '14px 20px', textAlign: 'center' }}>
                          {canDelete ? (
                            <button 
                              onClick={(e) => { e.stopPropagation(); handleDeleteClick(tx); }}
                              className="btn-logout" 
                              style={{ 
                                padding: '6px 12px', 
                                fontSize: '12px', 
                                display: 'inline-flex', 
                                alignItems: 'center', 
                                gap: '4px',
                                width: 'auto' 
                              }}
                              title="删除此笔账单"
                            >
                              <Trash2 size={13} />
                              删除
                            </button>
                          ) : (
                            <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>—</span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {/* 分页导航控制 */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>
                当前页码：{page}
              </span>
              <div style={{ display: 'flex', gap: '10px' }}>
                <button 
                  onClick={() => updateFilter({ page: Math.max(1, page - 1) })}
                  className="btn-secondary"
                  style={{ padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '4px' }}
                  disabled={!hasPrevPage}
                >
                  <ChevronLeft size={16} /> 上一页
                </button>
                <button 
                  onClick={() => updateFilter({ page: page + 1 })}
                  className="btn-secondary"
                  style={{ padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '4px' }}
                  disabled={!hasNextPage}
                >
                  下一页 <ChevronRight size={16} />
                </button>
              </div>
            </div>
          </div>
        )}
      </PageState>

      {/* ==========================================
         账单详情抽屉 (TransactionDetailDrawer)
         ========================================== */}
      {detailOpen && selectedTx && (
        <div className="drawer-overlay glass-blur show" onClick={() => { setDetailOpen(false); setSelectedTx(null); }}>
          <div className="drawer-container glass-card text-left" onClick={(e) => e.stopPropagation()}>
            {/* 头部 */}
            <div className="drawer-header">
              <div className="header-title">
                <ReceiptText className="title-icon text-glow" />
                <h3>账单详情</h3>
              </div>
              <button className="btn-close-drawer" onClick={() => { setDetailOpen(false); setSelectedTx(null); }}>
                <X size={20} />
              </button>
            </div>

            {/* 详情内容 */}
            <div className="drawer-body" style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
              {/* 金额大字号展示 */}
              <div style={{ textAlign: 'center', padding: '24px 0', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
                <span className="dimmed" style={{ fontSize: '13px' }}>交易金额</span>
                <div style={{ fontSize: '36px', fontWeight: 700, marginTop: '8px', color: selectedTx.type === 'income' ? 'var(--accent-green)' : 'var(--accent-purple)' }}>
                  {selectedTx.type === 'income' ? '+' : '-'}¥{centsToYuan(selectedTx.amount_cents)}
                </div>
              </div>

              {/* 核心信息网格 */}
              <div style={{ display: 'flex', flexDirection: 'column', gap: '14px', fontSize: '14px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span className="dimmed">账单标题</span>
                  <span style={{ fontWeight: 500 }}>{selectedTx.title || '无标题'}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span className="dimmed">账单类型</span>
                  <span>
                    {selectedTx.type === 'expense' && '个人支出'}
                    {selectedTx.type === 'income' && '个人收入'}
                    {selectedTx.type === 'shared_expense' && '共同支出'}
                    {selectedTx.type === 'settlement' && '结算记录'}
                  </span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span className="dimmed">所属分类</span>
                  <span>{selectedTx.category_id ? catMap[selectedTx.category_id] || '已设分类' : '未分类'}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span className="dimmed">发生日期</span>
                  <span>{formatDate(selectedTx.occurred_at).substring(0, 16)}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span className="dimmed">付款人</span>
                  <span>{getPayerName(selectedTx.payer_user_id)}</span>
                </div>

                {selectedTx.type === 'shared_expense' && (
                  <>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span className="dimmed">分摊方式</span>
                      <span>{selectedTx.split_method === 'equal' ? '均等平分 (Equal)' : '付款人全额承担'}</span>
                    </div>
                    {/* 分摊明细展示 */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', background: 'rgba(255,255,255,0.01)', border: '1px solid rgba(255,255,255,0.03)', padding: '12px 16px', borderRadius: '12px', marginTop: '6px' }}>
                      <span style={{ fontSize: '12px', fontWeight: 500 }} className="dimmed">分摊明细：</span>
                      {selectedTx.participants?.map((p) => (
                        <div key={p.user_id} style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px' }}>
                          <span>{p.user_id === currentUser?.id ? '我 (承担)' : '伙伴 (承担)'}</span>
                          <strong>¥{centsToYuan(p.share_amount_cents)}</strong>
                        </div>
                      ))}
                    </div>
                  </>
                )}

                {selectedTx.type !== 'shared_expense' && selectedTx.type !== 'settlement' && (
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span className="dimmed">可见性</span>
                    <span>{selectedTx.visibility === 'private' ? '仅自己可见' : '对方可见 (只读)'}</span>
                  </div>
                )}

                {selectedTx.tags && selectedTx.tags.length > 0 && (
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <span className="dimmed" style={{ minWidth: '80px' }}>账单标签</span>
                    <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', justifyContent: 'flex-end' }}>
                      {selectedTx.tags.map((t) => (
                        <span key={t} className="badge-shared" style={{ padding: '2px 8px', borderRadius: '4px', fontSize: '11px' }}>
                          #{t}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

                {selectedTx.note && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', borderTop: '1px dashed rgba(255,255,255,0.05)', paddingTop: '12px', marginTop: '6px' }}>
                    <span className="dimmed">备注</span>
                    <p style={{ margin: 0, fontSize: '13px', background: 'rgba(255,255,255,0.01)', padding: '10px 14px', borderRadius: '8px', border: '1px solid rgba(255,255,255,0.03)', color: 'var(--text-secondary)' }}>
                      {selectedTx.note}
                    </p>
                  </div>
                )}

                {/* 图片附件回显 */}
                {selectedTx.attachment_paths && selectedTx.attachment_paths.length > 0 && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', borderTop: '1px dashed rgba(255,255,255,0.05)', paddingTop: '12px', marginTop: '6px' }}>
                    <span className="dimmed">图片附件与小票</span>
                    <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', marginTop: '4px' }}>
                      {selectedTx.attachment_paths.map((p, idx) => (
                        <div
                          key={p}
                          style={{
                            width: '64px',
                            height: '64px',
                            borderRadius: '8px',
                            overflow: 'hidden',
                            border: '1px solid rgba(255, 255, 255, 0.12)',
                            background: 'rgba(255, 255, 255, 0.05)',
                            cursor: 'pointer',
                            transition: 'transform 0.2s',
                          }}
                          onClick={() => setActiveLightboxImg(p)}
                          onMouseEnter={(e) => e.currentTarget.style.transform = 'scale(1.05)'}
                          onMouseLeave={(e) => e.currentTarget.style.transform = 'scale(1)'}
                        >
                          <img
                            src={p}
                            alt={`attachment-${idx}`}
                            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                          />
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* 页脚操作按钮 */}
            <div className="drawer-footer" style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
              {/* 删除按钮 (仅创建者允许) */}
              {selectedTx.created_by_user_id === currentUser?.id && selectedTx.type !== 'settlement' && (
                <button 
                  className="btn-secondary" 
                  style={{ color: '#ef4444', borderColor: 'rgba(239, 68, 68, 0.2)', padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} 
                  onClick={() => {
                    setDetailOpen(false);
                    handleDeleteClick(selectedTx);
                  }}
                >
                  <Trash2 size={16} style={{ display: 'inline', marginRight: '6px', verticalAlign: 'middle' }} />
                  删除账单
                </button>
              )}

              {/* 复制一笔 */}
              {selectedTx.type !== 'settlement' && (
                <PermissionGate allow={['owner', 'editor']}>
                  <button
                    className="btn-secondary"
                    style={{
                      padding: '10px 20px',
                      fontSize: '14px',
                      borderRadius: '10px',
                    }}
                    onClick={() => {
                      setCopySourceTransaction(selectedTx);
                      setOpenTemplateSaveOnDrawerOpen(true);
                      setAddDrawerOpen(true);
                      setDetailOpen(false);
                    }}
                  >
                    存为模板
                  </button>
                  <button
                    className="btn-primary"
                    style={{
                      padding: '10px 20px',
                      fontSize: '14px',
                      borderRadius: '10px',
                    }}
                    onClick={() => {
                      setCopySourceTransaction(selectedTx);
                      setOpenTemplateSaveOnDrawerOpen(false);
                      setAddDrawerOpen(true);
                      setDetailOpen(false);
                    }}
                  >
                    复制一笔
                  </button>
                </PermissionGate>
              )}
            </div>
          </div>
        </div>
      )}

      {/* 图片附件大图灯箱 (Lightbox Modal) */}
      {activeLightboxImg && (
        <div
          className="drawer-overlay show"
          style={{
            alignItems: 'center',
            justifyContent: 'center',
            background: 'rgba(0, 0, 0, 0.8)',
            backdropFilter: 'blur(12px)',
            zIndex: 9999,
          }}
          onClick={() => setActiveLightboxImg(null)}
        >
          <div
            style={{
              position: 'relative',
              maxWidth: '90vw',
              maxHeight: '90vh',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <img
              src={activeLightboxImg}
              alt="Lightbox Large"
              style={{
                maxWidth: '100%',
                maxHeight: '80vh',
                borderRadius: '12px',
                border: '1px solid rgba(255, 255, 255, 0.15)',
                boxShadow: '0 20px 40px rgba(0, 0, 0, 0.5)',
              }}
            />
            <button
              onClick={() => setActiveLightboxImg(null)}
              style={{
                position: 'absolute',
                top: '-40px',
                right: '0',
                background: 'rgba(255, 255, 255, 0.1)',
                border: 'none',
                borderRadius: '50%',
                width: '32px',
                height: '32px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: '#fff',
                cursor: 'pointer',
                transition: 'background 0.2s',
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255, 255, 255, 0.2)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(255, 255, 255, 0.1)'}
            >
              <X size={18} />
            </button>
          </div>
        </div>
      )}



      {/* ==========================================
         账单/共同支出删除高风险二次确认模态弹窗
         ========================================== */}
      {showDeleteModal && selectedTx && (
        <div className="drawer-overlay show" style={{ alignItems: 'center', justifyContent: 'center' }}>
          <div className="confirm-modal-box animate-fade-in">
            <div className="drawer-header" style={{ padding: '16px 20px' }}>
              <div className="header-title" style={{ color: '#ef4444' }}>
                <Trash2 size={18} style={{ color: '#ef4444' }} />
                <h3 style={{ fontSize: '16px' }}>
                  {selectedTx.type === 'shared_expense' ? '确认删除这笔共同支出？' : '确认删除这笔账单？'}
                </h3>
              </div>
              <button 
                className="btn-close-drawer" 
                onClick={() => { setShowDeleteModal(false); setSelectedTx(null); }}
              >
                <X size={18} />
              </button>
            </div>

            <div className="modal-body-padding" style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <p className="modal-alert-text">
                {selectedTx.type === 'shared_expense' 
                  ? '删除后，本月双方待结算金额会重新计算。历史结算记录不会被自动删除。此操作无法撤销。' 
                  : '删除后，这笔账单将不再出现在流水和统计中。此操作无法撤销。'}
              </p>

              {/* 关键信息展示 */}
              <div style={{ background: 'rgba(10,12,16,0.4)', border: '1px solid rgba(255,255,255,0.03)', padding: '12px 16px', borderRadius: '12px', display: 'flex', flexDirection: 'column', gap: '8px', fontSize: '13px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span className="dimmed">流水标题</span>
                  <span>{selectedTx.title}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span className="dimmed">付款人</span>
                  <span>{selectedTx.payer_user_id === currentUser?.id ? '我' : '伙伴'}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span className="dimmed">账期时间</span>
                  <span>{formatDate(selectedTx.occurred_at).substring(0, 10)}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', borderTop: '1px dashed rgba(255,255,255,0.05)', paddingTop: '6px', marginTop: '2px' }}>
                  <span className="dimmed">删除金额</span>
                  <strong className="val-expense" style={{ fontSize: '15px' }}>¥{centsToYuan(selectedTx.amount_cents)}</strong>
                </div>
              </div>

              {/* 审计日志警示 */}
              <div style={{ background: 'rgba(239, 68, 68, 0.04)', border: '1px solid rgba(239, 68, 68, 0.15)', borderRadius: '8px', padding: '10px 14px', display: 'flex', alignItems: 'flex-start', gap: '8px', fontSize: '11px', color: '#fca5a5', textAlign: 'left' }}>
                <Info size={14} style={{ marginTop: '2px', flexShrink: 0 }} />
                <span>此操作作为高风险数据变动动作，将被自动记录并同步写入系统的 `audit_logs` 审计表中以备历史追溯。</span>
              </div>

              {/* 模态框页脚操作 */}
              <div className="drawer-footer" style={{ borderTop: 'none', paddingTop: 0, marginTop: '8px', display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
                <button 
                  className="btn-secondary" 
                  style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} 
                  onClick={() => { setShowDeleteModal(false); setSelectedTx(null); }}
                  disabled={deleteMutation.isPending}
                >
                  取消
                </button>
                <button 
                  className="btn-danger" 
                  style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} 
                  onClick={() => deleteMutation.mutate(selectedTx.id)}
                  disabled={deleteMutation.isPending}
                >
                  {deleteMutation.isPending ? '正在删除...' : selectedTx.type === 'shared_expense' ? '删除共同支出' : '删除账单'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ==========================================
         移动端高级筛选底部 Sheet (MobileFilterDrawer)
         ========================================== */}
      {mobileFilterOpen && (
        <div className="drawer-overlay glass-blur show" onClick={() => setMobileFilterOpen(false)}>
          <div className="drawer-container glass-card text-left" onClick={(e) => e.stopPropagation()}>
            <div className="drawer-header">
              <div className="header-title">
                <SlidersHorizontal className="title-icon text-glow" />
                <h3>高级筛选</h3>
              </div>
              <button className="btn-close-drawer" onClick={() => setMobileFilterOpen(false)}>
                <X size={20} />
              </button>
            </div>

            <div className="drawer-body" style={{ display: 'flex', flexDirection: 'column', gap: '16px', padding: '20px', overflowY: 'auto', maxHeight: 'calc(100vh - 200px)' }}>
              <div className="form-group" style={{ marginBottom: '12px' }}>
                <label>账单类型</label>
                <select 
                  className="filter-input" 
                  style={{ padding: '10px' }}
                  value={type} 
                  onChange={(e) => updateFilter({ type: e.target.value })}
                >
                  <option value="">全部类型</option>
                  <option value="expense">个人支出</option>
                  <option value="income">个人收入</option>
                  <option value="shared_expense">共同支出</option>
                </select>
              </div>

              <div className="form-group" style={{ marginBottom: '12px' }}>
                <label>所属分类</label>
                <select 
                  className="filter-input"
                  style={{ padding: '10px' }}
                  value={categoryId} 
                  onChange={(e) => updateFilter({ category_id: e.target.value })}
                >
                  <option value="">全部分类</option>
                  {categories?.map((cat) => (
                    <option key={cat.id} value={cat.id}>{cat.name}</option>
                  ))}
                </select>
              </div>

              <div className="form-group" style={{ marginBottom: '12px' }}>
                <label>付款人</label>
                <select 
                  className="filter-input"
                  style={{ padding: '10px' }}
                  value={payerUserId} 
                  onChange={(e) => updateFilter({ payer_user_id: e.target.value })}
                >
                  <option value="">全部付款人</option>
                  {users.map((u) => (
                    <option key={u.user_id} value={u.user_id}>
                      {u.display_name} {u.user_id === currentUser?.id ? '(我)' : ''}
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-group" style={{ marginBottom: '12px' }}>
                <label>可见性</label>
                <select 
                  className="filter-input"
                  style={{ padding: '10px' }}
                  value={visibility} 
                  onChange={(e) => updateFilter({ visibility: e.target.value })}
                >
                  <option value="">全部可见性</option>
                  <option value="private">仅自己可见</option>
                  <option value="partner_readable">对方可见 (只读)</option>
                </select>
              </div>

              <div className="form-group" style={{ marginBottom: '12px' }}>
                <label>关联标签</label>
                <input 
                  type="text" 
                  placeholder="搜索标签名称..." 
                  className="filter-input"
                  style={{ padding: '10px' }}
                  value={localTag}
                  onChange={(e) => setLocalTag(e.target.value)}
                />
              </div>

              <div className="form-group" style={{ marginBottom: '12px' }}>
                <label>最小金额 (元)</label>
                <input 
                  type="number" 
                  placeholder="最少金额..." 
                  className="filter-input"
                  style={{ padding: '10px' }}
                  value={localMinAmount}
                  onChange={(e) => setLocalMinAmount(e.target.value)}
                />
              </div>

              <div className="form-group" style={{ marginBottom: '12px' }}>
                <label>最大金额 (元)</label>
                <input 
                  type="number" 
                  placeholder="最多金额..." 
                  className="filter-input"
                  style={{ padding: '10px' }}
                  value={localMaxAmount}
                  onChange={(e) => setLocalMaxAmount(e.target.value)}
                />
              </div>

              <div className="form-group" style={{ marginBottom: '12px' }}>
                <label>标题/备注关键词</label>
                <input 
                  type="text" 
                  placeholder="搜索标题或备注关键词..." 
                  className="filter-input"
                  style={{ padding: '10px' }}
                  value={localKeyword}
                  onChange={(e) => setLocalKeyword(e.target.value)}
                />
              </div>
            </div>

            <div className="drawer-footer" style={{ display: 'flex', gap: '10px', padding: '16px 20px' }}>
              <button 
                className="btn-secondary" 
                style={{ flex: 1, padding: '10px' }}
                onClick={() => {
                  handleClearFilters();
                  setMobileFilterOpen(false);
                }}
              >
                重置
              </button>
              <button 
                className="btn-primary" 
                style={{ flex: 2, padding: '10px' }}
                onClick={() => {
                  updateFilter({
                    min_amount: localMinAmount,
                    max_amount: localMaxAmount,
                    keyword: localKeyword,
                    tag: localTag
                  });
                  setMobileFilterOpen(false);
                }}
              >
                应用筛选
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ==========================================
         批量打标签 Modal (BatchTagModal)
         ========================================== */}
      {showBatchTagModal && (
        <div className="modal-overlay" onClick={() => setShowBatchTagModal(false)}>
          <div className="modal-content glass-card animate-fade-in text-left" style={{ maxWidth: '440px' }} onClick={(e) => e.stopPropagation()}>
            <div className="drawer-header" style={{ padding: '0 0 16px 0', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
              <div className="header-title" style={{ color: 'var(--accent-purple)' }}>
                <Tags size={18} />
                <h3 style={{ fontSize: '16px' }}>批量追加标签</h3>
              </div>
              <button className="btn-close-drawer" onClick={() => setShowBatchTagModal(false)}>
                <X size={18} />
              </button>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', marginTop: '20px' }}>
              <p className="modal-alert-text" style={{ fontSize: '13px', margin: 0 }}>
                将为已选中的 <strong style={{ color: 'var(--accent-purple)' }}>{selectedTxIds.length}</strong> 笔账单追加新标签。历史已有标签会被保留并由系统自动去重。
              </p>

              <div className="form-group" style={{ marginBottom: '12px' }}>
                <label>输入追加标签 (空格或逗号分隔)</label>
                <input 
                  type="text" 
                  placeholder="如: 报销 餐饮 六月" 
                  className="filter-input"
                  style={{ padding: '12px' }}
                  value={batchTagsInput}
                  onChange={(e) => setBatchTagsInput(e.target.value)}
                  autoFocus
                />
              </div>

              {/* 审计日志提示 */}
              <div style={{ background: 'rgba(147, 51, 234, 0.04)', border: '1px solid rgba(147, 51, 234, 0.15)', borderRadius: '8px', padding: '10px 14px', display: 'flex', alignItems: 'flex-start', gap: '8px', fontSize: '11px', color: '#c084fc', textAlign: 'left' }}>
                <Info size={14} style={{ marginTop: '2px', flexShrink: 0 }} />
                <span>此操作属于批量操作，修改结果及标签流水变化将由系统后台记录并同步写入 `audit_logs` 审计表中以备历史追溯。</span>
              </div>

              <div className="drawer-footer" style={{ borderTop: 'none', padding: 0, marginTop: '8px', display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
                <button 
                  className="btn-secondary" 
                  style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} 
                  onClick={() => setShowBatchTagModal(false)}
                  disabled={batchTagMutation.isPending}
                >
                  取消
                </button>
                <button 
                  className="btn-primary" 
                  style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} 
                  onClick={() => {
                    const tag_names = batchTagsInput
                      .split(/[\s,，]+/)
                      .map((t) => t.trim())
                      .filter((t) => t.length > 0);
                    if (tag_names.length === 0) return;
                    batchTagMutation.mutate({
                      transaction_ids: selectedTxIds,
                      tag_names
                    });
                  }}
                  disabled={batchTagMutation.isPending || !batchTagsInput.trim()}
                >
                  {batchTagMutation.isPending ? '追加中...' : '确认追加'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
