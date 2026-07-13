import { ArrowRight, ReceiptText } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { DashboardResponse } from '../../types/dashboard';
import { formatDate } from '../../utils/date';
import SkeletonTable from '../ui/SkeletonTable';
import StatusChip, { type StatusChipTone } from '../ui/StatusChip';
import {
  formatDashboardAmount,
  getPayerName,
  getTransactionTypePresentation,
} from './dashboardModel';

interface RecentTransactionListProps {
  data: DashboardResponse | undefined;
  currentUserId: string | undefined;
  isLoading: boolean;
}

const TONE_MAP: Record<ReturnType<typeof getTransactionTypePresentation>['tone'], StatusChipTone> = {
  expense: 'danger',
  income: 'success',
  shared: 'accent',
  settlement: 'info',
};

export default function RecentTransactionList({
  data,
  currentUserId,
  isLoading,
}: RecentTransactionListProps) {
  return (
    <section className="lt-dashboard-section lt-dashboard-recent" aria-labelledby="dashboard-recent-title">
      <header className="lt-dashboard-section__header">
        <div className="lt-dashboard-section__heading">
          <ReceiptText size={20} aria-hidden="true" />
          <div>
            <span className="lt-dashboard-section__eyebrow">账本动态</span>
            <h2 id="dashboard-recent-title">最近流水</h2>
          </div>
        </div>
        <Link className="lt-dashboard-text-link" to="/transactions">
          <span>查看全部</span>
          <ArrowRight size={16} aria-hidden="true" />
        </Link>
      </header>

      {isLoading ? (
        <SkeletonTable rows={5} />
      ) : data?.recent_transactions.length ? (
        <div className="lt-dashboard-transaction-list">
          {data.recent_transactions.map((transaction) => {
            const presentation = getTransactionTypePresentation(transaction.type);
            return (
              <article className="lt-dashboard-transaction-row" key={transaction.id}>
                <StatusChip tone={TONE_MAP[presentation.tone]}>{presentation.label}</StatusChip>
                <div className="lt-dashboard-transaction-row__copy">
                  <strong>{transaction.title}</strong>
                  <span>
                    {formatDate(transaction.occurred_at).substring(5)} ·{' '}
                    {getPayerName(transaction.payer_user_id, data, currentUserId)}支付
                  </span>
                </div>
                <strong className={`lt-dashboard-transaction-row__amount lt-dashboard-transaction-row__amount--${presentation.tone}`}>
                  {presentation.amountSign}{formatDashboardAmount(transaction.amount_cents)}
                </strong>
              </article>
            );
          })}
        </div>
      ) : (
        <p className="lt-dashboard-section__empty">本月暂无流水。</p>
      )}
    </section>
  );
}
