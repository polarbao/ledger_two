import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { dashboardApi } from '../api/dashboard.api';
import { queryKeys } from '../api/queryKeys';
import { transactionsApi } from '../api/transactions.api';
import CategorySummary from '../components/dashboard/CategorySummary';
import MemberContributionSummary from '../components/dashboard/MemberContributionSummary';
import MonthlySummary from '../components/dashboard/MonthlySummary';
import RecentTransactionList from '../components/dashboard/RecentTransactionList';
import RecurringReminderList from '../components/dashboard/RecurringReminderList';
import SettlementActionCard from '../components/dashboard/SettlementActionCard';
import {
  getDashboardSummaryMetrics,
  getSettlementAction,
} from '../components/dashboard/dashboardModel';
import PermissionGate from '../components/ledger/PermissionGate';
import { useHasLedgerRole } from '../components/ledger/useLedgerPermission';
import Button from '../components/ui/Button';
import EmptyState from '../components/ui/EmptyState';
import ErrorState from '../components/ui/ErrorState';
import { useAuthStore } from '../stores/auth.store';
import { useLedgerStore } from '../stores/ledger.store';
import { useUIStore } from '../stores/ui.store';
import './DashboardPage.css';

function getMonthLabel(month: string) {
  const [year, monthNumber] = month.split('-');
  return year && monthNumber ? `${year} 年 ${Number(monthNumber)} 月` : month;
}

export default function DashboardPage() {
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((state) => state.user);
  const currentMonth = useUIStore((state) => state.currentMonth);
  const setAddDrawerOpen = useUIStore((state) => state.setAddDrawerOpen);
  const setCopySourceTransaction = useUIStore((state) => state.setCopySourceTransaction);
  const setEditSourceTransaction = useUIStore((state) => state.setEditSourceTransaction);
  const setEditingDraftId = useUIStore((state) => state.setEditingDraftId);
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const canWriteLedger = useHasLedgerRole(['owner', 'editor']);

  const {
    data: dashboardData,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: queryKeys.dashboard.month(activeLedgerId, currentMonth),
    queryFn: ({ signal }) => dashboardApi.getDashboard(currentMonth, signal),
    enabled: Boolean(currentUser && activeLedgerId),
  });

  const { data: reminders = [] } = useQuery({
    queryKey: queryKeys.recurringReminders(activeLedgerId),
    queryFn: ({ signal }) => transactionsApi.listRecurringReminders(signal),
    enabled: Boolean(currentUser && activeLedgerId),
  });

  const confirmReminderMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.confirmReminder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.root(activeLedgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.recurringReminders(activeLedgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.root(activeLedgerId) });
    },
  });

  const skipReminderMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.skipReminder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.recurringReminders(activeLedgerId) });
    },
  });

  const handleQuickAdd = () => {
    setCopySourceTransaction(null);
    setEditSourceTransaction(null);
    setEditingDraftId(null);
    setAddDrawerOpen(true);
  };

  const handleConfirmReminder = (id: string) => {
    confirmReminderMutation.mutate(id);
  };

  const handleSkipReminder = (id: string) => {
    skipReminderMutation.mutate(id);
  };

  const summaryMetrics = dashboardData
    ? getDashboardSummaryMetrics(dashboardData, currentUser?.id)
    : [];
  const settlementAction = dashboardData
    ? getSettlementAction(dashboardData, currentUser?.id)
    : null;
  const isMonthEmpty = dashboardData
    ? dashboardData.total_expense_cents === 0
      && dashboardData.total_income_cents === 0
      && dashboardData.recent_transactions.length === 0
    : false;
  const reminderMutationPending = confirmReminderMutation.isPending || skipReminderMutation.isPending;

  return (
    <div className="lt-dashboard animate-fade-in">
      <header className="lt-dashboard__header">
        <div className="lt-dashboard__title-copy">
          <span className="lt-dashboard__month">{getMonthLabel(currentMonth)}</span>
          <h1>本月概览</h1>
          <p>{currentUser?.display_name ? `${currentUser.display_name}，` : ''}你们的共享账目在这里汇总。</p>
        </div>
        <PermissionGate allow={['owner', 'editor']}>
          <Button
            className="lt-dashboard__record-button"
            variant="primary"
            startIcon={<Plus size={18} />}
            onClick={handleQuickAdd}
          >
            记一笔
          </Button>
        </PermissionGate>
      </header>

      {error ? (
        <ErrorState
          title="本月概览加载失败"
          message={error instanceof Error ? error.message : '请检查网络后重试。'}
          onRetry={refetch}
        />
      ) : (
        <>
          <MonthlySummary metrics={summaryMetrics} isLoading={isLoading} />

          {isLoading || !settlementAction ? (
            <section className="lt-dashboard-settlement lt-dashboard-settlement--loading" aria-busy="true">
              <span className="skeleton-item lt-dashboard-settlement__icon-skeleton" />
              <div className="lt-dashboard-settlement__loading-copy">
                <span className="skeleton-item" />
                <span className="skeleton-item" />
              </div>
            </section>
          ) : (
            <SettlementActionCard action={settlementAction} />
          )}

          {reminders.length > 0 ? (
            <RecurringReminderList
              reminders={reminders}
              isMutating={reminderMutationPending}
              confirmingId={confirmReminderMutation.isPending ? confirmReminderMutation.variables : undefined}
              onConfirm={handleConfirmReminder}
              onSkip={handleSkipReminder}
            />
          ) : null}

          {isMonthEmpty && !isLoading ? (
            <EmptyState
              title="这个月还没有账单"
              description={canWriteLedger
                ? '记录第一笔支出或收入后，这里会出现分类、成员承担和最近流水。'
                : '当前账本还没有本月账单；你可以查看，但不能新增记录。'}
              actionText={canWriteLedger ? '记第一笔' : undefined}
              onAction={canWriteLedger ? handleQuickAdd : undefined}
            />
          ) : (
            <div className="lt-dashboard-content-grid">
              <div className="lt-dashboard-content-stack">
                <CategorySummary
                  items={dashboardData?.category_summary ?? []}
                  isLoading={isLoading}
                />
                {dashboardData ? (
                  <MemberContributionSummary
                    data={dashboardData}
                    currentUserId={currentUser?.id}
                  />
                ) : (
                  <section className="lt-dashboard-section lt-dashboard-section--loading" aria-busy="true">
                    <span className="skeleton-item" />
                    <span className="skeleton-item" />
                    <span className="skeleton-item" />
                  </section>
                )}
              </div>
              <RecentTransactionList
                data={dashboardData}
                currentUserId={currentUser?.id}
                isLoading={isLoading}
              />
            </div>
          )}
        </>
      )}
    </div>
  );
}
