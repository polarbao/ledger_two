import { ArrowRight, CheckCircle2, Scale } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { SettlementAction } from './dashboardModel';
import { formatDashboardAmount } from './dashboardModel';

interface SettlementActionCardProps {
  action: SettlementAction;
  archivedLedgerId?: string | null;
}

export default function SettlementActionCard({
  action,
  archivedLedgerId,
}: SettlementActionCardProps) {
  const settled = action.state === 'settled';

  return (
    <section className={`lt-dashboard-settlement lt-dashboard-settlement--${action.state}`}>
      <div className="lt-dashboard-settlement__icon" aria-hidden="true">
        {settled ? <CheckCircle2 size={22} /> : <Scale size={22} />}
      </div>
      <div className="lt-dashboard-settlement__copy">
        <span className="lt-dashboard-section__eyebrow">{action.eyebrow}</span>
        <h2>{action.title}</h2>
        <p>{action.description}</p>
      </div>
      <div className="lt-dashboard-settlement__action">
        <strong>{formatDashboardAmount(action.amountCents)}</strong>
        <Link
          className="lt-dashboard-text-link"
          to={archivedLedgerId
            ? `/settlement?archived_ledger_id=${encodeURIComponent(archivedLedgerId)}`
            : '/settlement'}
        >
          <span>查看结算</span>
          <ArrowRight size={17} aria-hidden="true" />
        </Link>
      </div>
    </section>
  );
}
