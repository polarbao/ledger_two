import type { DashboardSummaryMetric } from './dashboardModel';
import { formatDashboardAmount } from './dashboardModel';

interface MonthlySummaryProps {
  metrics: DashboardSummaryMetric[];
  isLoading?: boolean;
}

export default function MonthlySummary({ metrics, isLoading = false }: MonthlySummaryProps) {
  return (
    <section className="lt-dashboard-summary" aria-label="月度摘要" aria-busy={isLoading || undefined}>
      {isLoading
        ? Array.from({ length: 5 }, (_, index) => (
            <div className="lt-dashboard-metric lt-dashboard-metric--loading" key={index} aria-hidden="true">
              <span className="skeleton-item lt-dashboard-metric__label-skeleton" />
              <span className="skeleton-item lt-dashboard-metric__amount-skeleton" />
              <span className="skeleton-item lt-dashboard-metric__detail-skeleton" />
            </div>
          ))
        : metrics.map((metric) => (
            <article
              className={`lt-dashboard-metric lt-dashboard-metric--${metric.tone}`}
              key={metric.id}
            >
              <span className="lt-dashboard-metric__label">{metric.label}</span>
              <strong className="lt-dashboard-metric__amount">
                {formatDashboardAmount(metric.amountCents)}
              </strong>
              <span className="lt-dashboard-metric__detail">{metric.detail}</span>
            </article>
          ))}
    </section>
  );
}
