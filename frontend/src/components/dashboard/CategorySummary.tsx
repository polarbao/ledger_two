import { ArrowRight, ChartNoAxesColumn } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { SummaryItem } from '../../types/dashboard';
import SkeletonTable from '../ui/SkeletonTable';
import { formatDashboardAmount } from './dashboardModel';

interface CategorySummaryProps {
  items: SummaryItem[];
  isLoading: boolean;
}

export default function CategorySummary({ items, isLoading }: CategorySummaryProps) {
  return (
    <section className="lt-dashboard-section" aria-labelledby="dashboard-category-title">
      <header className="lt-dashboard-section__header">
        <div className="lt-dashboard-section__heading">
          <ChartNoAxesColumn size={20} aria-hidden="true" />
          <div>
            <span className="lt-dashboard-section__eyebrow">本月结构</span>
            <h2 id="dashboard-category-title">分类支出</h2>
          </div>
        </div>
        <Link className="lt-dashboard-text-link" to="/analytics">
          <span>查看分析</span>
          <ArrowRight size={16} aria-hidden="true" />
        </Link>
      </header>

      {isLoading ? (
        <SkeletonTable rows={3} />
      ) : items.length > 0 ? (
        <div className="lt-dashboard-category-list">
          {items.map((item) => {
            const percent = Math.min(100, Math.max(0, item.percent));
            return (
              <div className="lt-dashboard-category-row" key={item.id}>
                <div className="lt-dashboard-category-row__copy">
                  <strong>{item.name}</strong>
                  <span>
                    {formatDashboardAmount(item.amount_cents)} · {item.percent.toFixed(1)}%
                  </span>
                </div>
                <div
                  className="lt-dashboard-progress"
                  role="progressbar"
                  aria-label={`${item.name}占比`}
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-valuenow={percent}
                >
                  <span style={{ width: `${percent}%` }} />
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <p className="lt-dashboard-section__empty">本月暂无分类支出。</p>
      )}
    </section>
  );
}
