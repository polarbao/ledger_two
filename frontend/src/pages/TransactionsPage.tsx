import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useUIStore } from '../stores/ui.store';
import { useAuthStore } from '../stores/auth.store';
import { transactionsApi } from '../api/transactions.api';
import { centsToYuan } from '../utils/money';
import { formatDate } from '../utils/date';
import PageState from '../components/ui/PageState';
import { 
  ReceiptText, 
  Trash2, 
  ChevronLeft, 
  ChevronRight, 
  X, 
  Info
} from 'lucide-react';
import type { TransactionResponse } from '../types/transaction';

export default function TransactionsPage() {
  const currentMonth = useUIStore((state) => state.currentMonth);
  const currentUser = useAuthStore((state) => state.user);
  const queryClient = useQueryClient();

  const [page, setPage] = useState(1);
  const pageSize = 15;

  const [selectedTx, setSelectedTx] = useState<TransactionResponse | null>(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  // 1. 获取分类名称列表映射
  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: () => transactionsApi.getCategories(),
  });

  const catMap = categories?.reduce((acc, cat) => {
    acc[cat.id] = cat.name;
    return acc;
  }, {} as Record<string, string>) || {};

  // 2. 获取流水列表
  const { 
    data: transactions, 
    isLoading, 
    isError, 
    error,
    refetch 
  } = useQuery({
    queryKey: ['transactions', currentMonth, page],
    queryFn: () => transactionsApi.list({
      month: currentMonth,
      page,
      page_size: pageSize,
    }),
  });

  // 3. 删除流水 Mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.deleteTransaction(id),
    onSuccess: () => {
      setShowDeleteModal(false);
      setSelectedTx(null);
      // 失效所有关联缓存，促使全局数据重载
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      queryClient.invalidateQueries({ queryKey: ['reports-monthly'] });
      queryClient.invalidateQueries({ queryKey: ['reports-category'] });
      queryClient.invalidateQueries({ queryKey: ['reports-tag'] });
      queryClient.invalidateQueries({ queryKey: ['reports-member'] });
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

      {/* 主展示区域 */}
      <PageState 
        isLoading={isLoading} 
        isError={isError} 
        isEmpty={transactions ? transactions.length === 0 : true}
        errorMsg={error instanceof Error ? error.message : '拉取流水明细失败'}
        emptyMessage="本月账期暂无任何流水明细。点击上方「记一笔」开启首笔记账。"
        skeletonType="table"
        onRetry={refetch}
      >
        {transactions && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div className="glass-card" style={{ overflowX: 'auto' }}>
              <table className="transaction-table" style={{ width: '100%', borderCollapse: 'collapse', fontSize: '14px' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid rgba(255,255,255,0.06)', color: 'var(--text-secondary)' }}>
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

                    // 权限控制：仅允许创建者删除
                    const canDelete = tx.created_by_user_id === currentUser?.id && tx.type !== 'settlement';

                    return (
                      <tr 
                        key={tx.id} 
                        style={{ borderBottom: '1px solid rgba(255,255,255,0.03)', transition: 'background 0.2s' }}
                        className="table-row-hover"
                      >
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
                              onClick={() => handleDeleteClick(tx)}
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
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  className="btn-secondary"
                  style={{ padding: '8px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '4px' }}
                  disabled={!hasPrevPage}
                >
                  <ChevronLeft size={16} /> 上一页
                </button>
                <button 
                  onClick={() => setPage(p => p + 1)}
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
    </div>
  );
}
