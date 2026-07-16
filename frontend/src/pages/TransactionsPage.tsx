import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, ChevronLeft, ChevronRight, Download, ReceiptText, Tags } from 'lucide-react';
import { useSearchParams } from 'react-router-dom';
import { dashboardApi } from '../api/dashboard.api';
import { queryKeys } from '../api/queryKeys';
import { transactionsApi } from '../api/transactions.api';
import TransactionCard from '../components/transaction/TransactionCard';
import TransactionDetailDrawer from '../components/transaction/TransactionDetailDrawer';
import TransactionFilterFields, { type TransactionFilterDraft } from '../components/transaction/TransactionFilterFields';
import TransactionTable from '../components/transaction/TransactionTable';
import TransactionToolbar from '../components/transaction/TransactionToolbar';
import {
  buildTransactionFilterChips,
  type TransactionFilterState,
  type TransactionQuickType,
  yuanFilterToCents,
} from '../components/transaction/transactionsPageModel';
import { getTransactionEditBlockReason } from '../components/transaction/transactionFormState';
import ActiveFilterChips from '../components/ui/ActiveFilterChips';
import BottomSheet from '../components/ui/BottomSheet';
import Button from '../components/ui/Button';
import ConfirmDialog from '../components/ui/ConfirmDialog';
import PageState from '../components/ui/PageState';
import ResponsiveDataList from '../components/ui/ResponsiveDataList';
import { useLedgerContext } from '../components/ledger/useLedgerContext';
import { useAuthStore } from '../stores/auth.store';
import { useUIStore } from '../stores/ui.store';
import type { TransactionResponse } from '../types/transaction';
import './TransactionsPage.css';

const pageSize = 15;
const quickTypes: TransactionQuickType[] = ['', 'expense', 'income', 'shared_expense', 'settlement'];

export default function TransactionsPage() {
  const currentMonth = useUIStore((state) => state.currentMonth);
  const currentUser = useAuthStore((state) => state.user);
  const { ledgerId, role, isArchivedView } = useLedgerContext();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  const month = searchParams.get('month') || currentMonth;
  const rawType = searchParams.get('type') || '';
  const type = quickTypes.includes(rawType as TransactionQuickType) ? rawType as TransactionQuickType : '';
  const categoryId = searchParams.get('category_id') || '';
  const keyword = searchParams.get('keyword') || '';
  const minAmount = searchParams.get('min_amount') || '';
  const maxAmount = searchParams.get('max_amount') || '';
  const payerUserId = searchParams.get('payer_user_id') || '';
  const visibility = searchParams.get('visibility') || '';
  const tag = searchParams.get('tag') || '';
  const parsedPage = Number.parseInt(searchParams.get('page') || '1', 10);
  const page = Number.isFinite(parsedPage) && parsedPage > 0 ? parsedPage : 1;
  const canWrite = !isArchivedView && (role === 'owner' || role === 'editor');
  const canExport = role === 'owner' || role === 'editor';

  const [selectedTransaction, setSelectedTransaction] = useState<TransactionResponse | null>(null);
  const [transactionToDelete, setTransactionToDelete] = useState<TransactionResponse | null>(null);
  const [mobileFilterOpen, setMobileFilterOpen] = useState(false);
  const [desktopFilterOpen, setDesktopFilterOpen] = useState(false);
  const [batchMode, setBatchMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchTagOpen, setBatchTagOpen] = useState(false);
  const [batchTags, setBatchTags] = useState('');
  const [exportOpen, setExportOpen] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [pageMessage, setPageMessage] = useState<string | null>(null);

  const setCopySourceTransaction = useUIStore((state) => state.setCopySourceTransaction);
  const setEditSourceTransaction = useUIStore((state) => state.setEditSourceTransaction);
  const setOpenTemplateSaveOnDrawerOpen = useUIStore((state) => state.setOpenTemplateSaveOnDrawerOpen);
  const setAddDrawerOpen = useUIStore((state) => state.setAddDrawerOpen);
  const setEditingDraftId = useUIStore((state) => state.setEditingDraftId);
  const isOffline = useUIStore((state) => state.isOffline);

  const updateFilter = (changes: Record<string, string | number | undefined>) => {
    const next = new URLSearchParams(searchParams);
    if (!Object.prototype.hasOwnProperty.call(changes, 'page')) next.set('page', '1');
    Object.entries(changes).forEach(([key, value]) => {
      if (value === undefined || value === '') next.delete(key);
      else next.set(key, String(value));
    });
    setSelectedIds([]);
    setSearchParams(next);
  };

  const clearFilters = () => {
    const next = new URLSearchParams();
    next.set('month', month);
    next.set('page', '1');
    if (isArchivedView && ledgerId) next.set('archived_ledger_id', ledgerId);
    setSelectedIds([]);
    setSearchParams(next);
  };

  const { data: dashboardData } = useQuery({
    queryKey: queryKeys.dashboard.month(ledgerId, month),
    queryFn: ({ signal }) => dashboardApi.getDashboard(month, signal),
    enabled: Boolean(ledgerId),
  });
  const users = dashboardData?.user_stats || [];
  const ledgerUserIds = users.map((user) => user.user_id);
  const payerNames = users.reduce<Record<string, string>>((names, user) => {
    names[user.user_id] = user.user_id === currentUser?.id ? `${user.display_name}（我）` : user.display_name;
    return names;
  }, {});
  const getUserName = (userId: string) => payerNames[userId] || (userId === currentUser?.id ? '我' : '账本成员');

  const { data: categories = [] } = useQuery({
    queryKey: queryKeys.categories(ledgerId, true),
    queryFn: ({ signal }) => transactionsApi.getCategories({ includeArchived: true }, signal),
    enabled: Boolean(ledgerId),
  });
  const normalizedCategories = categories ?? [];
  const categoryNames = normalizedCategories.reduce<Record<string, string>>((names, category) => {
    names[category.id] = category.is_archived ? `${category.name}（已归档）` : category.name;
    return names;
  }, {});
  const getCategoryLabel = (tx: TransactionResponse) => {
    if (tx.category_name) return tx.category_is_archived ? `${tx.category_name}（已归档）` : tx.category_name;
    if (!tx.category_id) return '未分类';
    return categoryNames[tx.category_id] || '已设分类';
  };

  const filterState: TransactionFilterState = {
    type,
    categoryId,
    keyword,
    minAmount,
    maxAmount,
    payerUserId,
    visibility,
    tag,
  };
  const activeFilters = buildTransactionFilterChips(filterState, categoryNames, payerNames);
  const advancedFilterValues: TransactionFilterDraft = {
    categoryId,
    payerUserId,
    visibility,
    tag,
    minAmount,
    maxAmount,
  };
  const filterFieldsKey = [categoryId, payerUserId, visibility, tag, minAmount, maxAmount].join('|');

  const transactionFilter = {
    month,
    type: type || undefined,
    category_id: categoryId || undefined,
    keyword: keyword || undefined,
    min_amount: yuanFilterToCents(minAmount),
    max_amount: yuanFilterToCents(maxAmount),
    payer_user_id: payerUserId || undefined,
    visibility: visibility || undefined,
    tag: tag || undefined,
    page,
    page_size: pageSize,
  };
  const {
    data: transactions,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: queryKeys.transactions.list(ledgerId, transactionFilter),
    queryFn: ({ signal }) => transactionsApi.list(transactionFilter, signal),
    enabled: Boolean(ledgerId),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.deleteTransaction(id),
    onSuccess: () => {
      setTransactionToDelete(null);
      setSelectedTransaction(null);
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.root(ledgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.root(ledgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.reports.root(ledgerId) });
    },
  });
  const batchTagMutation = useMutation({
    mutationFn: (payload: { transaction_ids: string[]; tag_names: string[] }) => transactionsApi.batchTag(payload),
    onSuccess: () => {
      setBatchTagOpen(false);
      setBatchTags('');
      setSelectedIds([]);
      setBatchMode(false);
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.root(ledgerId) });
    },
  });

  const handleOpenFilters = () => {
    if (window.matchMedia('(max-width: 768px)').matches) setMobileFilterOpen(true);
    else setDesktopFilterOpen((open) => !open);
  };
  const applyAdvancedFilters = (filters: TransactionFilterDraft) => {
    updateFilter({
      category_id: filters.categoryId,
      payer_user_id: filters.payerUserId,
      visibility: filters.visibility,
      tag: filters.tag,
      min_amount: filters.minAmount,
      max_amount: filters.maxAmount,
    });
    setMobileFilterOpen(false);
  };
  const removeFilter = (key: string) => {
    const paramsByKey: Record<string, string> = {
      type: 'type',
      categoryId: 'category_id',
      keyword: 'keyword',
      minAmount: 'min_amount',
      maxAmount: 'max_amount',
      payerUserId: 'payer_user_id',
      visibility: 'visibility',
      tag: 'tag',
    };
    updateFilter({ [paramsByKey[key]]: undefined });
  };
  const handleCopy = (tx: TransactionResponse, saveAsTemplate: boolean) => {
    setEditSourceTransaction(null);
    setEditingDraftId(null);
    setCopySourceTransaction(tx);
    setOpenTemplateSaveOnDrawerOpen(saveAsTemplate);
    setAddDrawerOpen(true);
    setSelectedTransaction(null);
  };
  const editBlockReason = (tx: TransactionResponse) => getTransactionEditBlockReason(
    tx,
    currentUser?.id || '',
    canWrite,
    ledgerUserIds,
    isOffline,
  );
  const handleEdit = (tx: TransactionResponse) => {
    if (editBlockReason(tx)) return;
    setCopySourceTransaction(null);
    setEditingDraftId(null);
    setOpenTemplateSaveOnDrawerOpen(false);
    setEditSourceTransaction(tx);
    setAddDrawerOpen(true);
    setSelectedTransaction(null);
  };
  const handleSelectAll = (checked: boolean) => {
    if (!checked || !transactions) return setSelectedIds([]);
    setSelectedIds(transactions.filter((tx) => tx.created_by_user_id === currentUser?.id).map((tx) => tx.id));
  };
  const handleSelect = (id: string) => {
    setSelectedIds((ids) => ids.includes(id) ? ids.filter((item) => item !== id) : [...ids, id]);
  };
  const parsedBatchTags = batchTags.split(/[\s,，]+/).map((item) => item.trim()).filter(Boolean);

  const downloadCSV = async () => {
    setExporting(true);
    setPageMessage(null);
    try {
		if (!ledgerId) {
			throw new Error('请先选择账本');
		}
      const response = await fetch(`/api/export/transactions.csv?month=${encodeURIComponent(month)}`, {
        credentials: 'include',
			headers: { 'X-Ledger-Id': ledgerId },
      });
      if (!response.ok) throw new Error('导出失败，请稍后重试');
      const blobUrl = window.URL.createObjectURL(await response.blob());
      const anchor = document.createElement('a');
      anchor.href = blobUrl;
      anchor.download = `transactions_${month}.csv`;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      window.URL.revokeObjectURL(blobUrl);
      setExportOpen(false);
      setPageMessage(`${month} 流水已导出`);
    } catch (downloadError) {
      setPageMessage(downloadError instanceof Error ? downloadError.message : '导出失败，请稍后重试');
    } finally {
      setExporting(false);
    }
  };

  const hasPreviousPage = page > 1;
  const hasNextPage = transactions?.length === pageSize;
  const emptyMessage = activeFilters.length
    ? '没有符合当前筛选条件的流水，请调整或清除筛选。'
    : '本账期暂无流水，记一笔后会显示在这里。';

  return (
    <main className="transactions-page animate-fade-in">
      <header className="transactions-page__header">
        <div>
          <span className="transactions-page__eyebrow"><ReceiptText size={17} /> {month} 账期</span>
          <h1>流水</h1>
        </div>
        {pageMessage ? <p role="status">{pageMessage}</p> : null}
      </header>

      <TransactionToolbar
        month={month}
        keyword={keyword}
        type={type}
        activeFilterCount={activeFilters.length}
        batchMode={batchMode}
        canWrite={canWrite}
        canExport={canExport}
        onMonthChange={(value) => updateFilter({ month: value })}
        onKeywordChange={(value) => updateFilter({ keyword: value })}
        onTypeChange={(value) => updateFilter({ type: value })}
        onOpenFilters={handleOpenFilters}
        onToggleBatch={() => {
          setBatchMode((enabled) => !enabled);
          setSelectedIds([]);
        }}
        onExport={() => setExportOpen(true)}
      />

      {desktopFilterOpen ? (
        <section className="transactions-filter-panel" aria-label="更多筛选">
          <TransactionFilterFields
            key={filterFieldsKey}
            categories={normalizedCategories}
            users={users.map((user) => ({ userId: user.user_id, label: getUserName(user.user_id) }))}
            values={advancedFilterValues}
            onApply={applyAdvancedFilters}
            onReset={clearFilters}
          />
        </section>
      ) : null}

      <ActiveFilterChips filters={activeFilters} onRemove={removeFilter} onClear={clearFilters} />

      {batchMode ? (
        <div className="transactions-batch-bar">
          <span>已选择 <strong>{selectedIds.length}</strong> 笔可操作账单</span>
          <Button
            variant="primary"
            startIcon={<Tags size={17} />}
            disabled={selectedIds.length === 0}
            onClick={() => setBatchTagOpen(true)}
          >
            批量打标签
          </Button>
        </div>
      ) : null}

      <PageState
        isLoading={isLoading}
        isError={isError}
        isEmpty={!transactions?.length}
        errorMsg={error instanceof Error ? error.message : '拉取流水失败'}
        emptyMessage={emptyMessage}
        skeletonType="table"
        onRetry={refetch}
      >
        {transactions ? (
          <ResponsiveDataList
            desktopLabel="桌面流水表格"
            mobileLabel="移动端流水卡片"
            desktop={(
              <TransactionTable
                transactions={transactions}
                currentUserId={currentUser?.id || ''}
                canWrite={canWrite}
                batchMode={batchMode}
                selectedIds={selectedIds}
                categoryLabel={getCategoryLabel}
                payerName={getUserName}
                onSelectAll={handleSelectAll}
                onSelect={handleSelect}
                onView={setSelectedTransaction}
                onCopy={handleCopy}
                onEdit={handleEdit}
                onDelete={(tx) => setTransactionToDelete(tx)}
                editBlockReason={editBlockReason}
              />
            )}
            mobile={(
              <div className="transactions-mobile-list">
                {transactions.map((tx) => (
                  <TransactionCard
                    key={tx.id}
                    tx={tx}
                    categoryName={getCategoryLabel(tx)}
                    payerName={getUserName(tx.payer_user_id)}
                    selectable={batchMode && tx.created_by_user_id === currentUser?.id}
                    selected={selectedIds.includes(tx.id)}
                    onSelectedChange={() => handleSelect(tx.id)}
                    onClick={() => setSelectedTransaction(tx)}
                  />
                ))}
              </div>
            )}
          />
        ) : null}
      </PageState>

      <nav className="transactions-pagination" aria-label="流水分页">
        <span>第 {page} 页</span>
        <div>
          <Button
            variant="secondary"
            startIcon={<ChevronLeft size={17} />}
            disabled={!hasPreviousPage}
            onClick={() => updateFilter({ page: page - 1 })}
          >
            上一页
          </Button>
          <Button
            variant="secondary"
            endIcon={<ChevronRight size={17} />}
            disabled={!hasNextPage}
            onClick={() => updateFilter({ page: page + 1 })}
          >
            下一页
          </Button>
        </div>
      </nav>

      <TransactionDetailDrawer
        open={Boolean(selectedTransaction)}
        transaction={selectedTransaction}
        currentUserId={currentUser?.id || ''}
        canWrite={canWrite}
        categoryLabel={selectedTransaction ? getCategoryLabel(selectedTransaction) : ''}
        payerName={selectedTransaction ? getUserName(selectedTransaction.payer_user_id) : ''}
        userName={getUserName}
        onClose={() => setSelectedTransaction(null)}
        onCopy={handleCopy}
        onEdit={handleEdit}
        onDelete={(tx) => {
          setSelectedTransaction(null);
          setTransactionToDelete(tx);
        }}
        editBlockReason={selectedTransaction ? editBlockReason(selectedTransaction) : null}
      />

      <BottomSheet
        open={mobileFilterOpen}
        title="筛选流水"
        description="按分类、付款人、可见范围、标签或金额缩小结果"
        onClose={() => setMobileFilterOpen(false)}
      >
        <TransactionFilterFields
          key={`mobile-${filterFieldsKey}`}
          categories={normalizedCategories}
          users={users.map((user) => ({ userId: user.user_id, label: getUserName(user.user_id) }))}
          values={advancedFilterValues}
          onApply={applyAdvancedFilters}
          onReset={() => {
            clearFilters();
            setMobileFilterOpen(false);
          }}
        />
      </BottomSheet>

      <ConfirmDialog
        open={Boolean(transactionToDelete)}
        title={transactionToDelete?.type === 'shared_expense' ? '删除这笔共同支出？' : '删除这笔账单？'}
        description={transactionToDelete?.type === 'shared_expense'
          ? '删除后双方待结算金额会重新计算，历史结算记录不会被改写。'
          : '删除后该账单不再出现在流水和统计中。'}
        confirmLabel={transactionToDelete?.type === 'shared_expense' ? '删除共同支出' : '删除账单'}
        tone="danger"
        icon={<AlertTriangle size={22} />}
        isConfirming={deleteMutation.isPending}
        onClose={() => setTransactionToDelete(null)}
        onConfirm={() => transactionToDelete && deleteMutation.mutate(transactionToDelete.id)}
      >
        {transactionToDelete ? (
          <div className="transactions-confirm-summary">
            <span>{transactionToDelete.title || '无标题'}</span>
            <strong>¥{(transactionToDelete.amount_cents / 100).toFixed(2)}</strong>
          </div>
        ) : null}
      </ConfirmDialog>

      <ConfirmDialog
        open={batchTagOpen}
        title={`为 ${selectedIds.length} 笔账单追加标签`}
        description="已有标签会保留，新标签会自动去重。"
        confirmLabel="追加标签"
        confirmDisabled={parsedBatchTags.length === 0}
        isConfirming={batchTagMutation.isPending}
        onClose={() => setBatchTagOpen(false)}
        onConfirm={() => batchTagMutation.mutate({ transaction_ids: selectedIds, tag_names: parsedBatchTags })}
      >
        <label className="transactions-dialog-field">
          <span>标签名称</span>
          <input value={batchTags} onChange={(event) => setBatchTags(event.target.value)} placeholder="使用空格或逗号分隔" />
        </label>
      </ConfirmDialog>

      <ConfirmDialog
        open={exportOpen}
        title={`导出 ${month} 流水？`}
        description="CSV 会包含当前账本中你有权查看的明文账单，请妥善保管。"
        confirmLabel="下载 CSV"
        icon={<Download size={22} />}
        isConfirming={exporting}
        onClose={() => setExportOpen(false)}
        onConfirm={downloadCSV}
      />
    </main>
  );
}
