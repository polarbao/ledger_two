import { UsersRound } from 'lucide-react';
import type { DashboardResponse } from '../../types/dashboard';
import { formatDashboardAmount } from './dashboardModel';

interface MemberContributionSummaryProps {
  data: DashboardResponse;
  currentUserId: string | undefined;
}

export default function MemberContributionSummary({
  data,
  currentUserId,
}: MemberContributionSummaryProps) {
  const orderedMembers = [...data.user_stats].sort((left, right) => {
    if (left.user_id === currentUserId) return -1;
    if (right.user_id === currentUserId) return 1;
    return left.display_name.localeCompare(right.display_name, 'zh-CN');
  });

  return (
    <section className="lt-dashboard-section" aria-labelledby="dashboard-member-title">
      <header className="lt-dashboard-section__header">
        <div className="lt-dashboard-section__heading">
          <UsersRound size={20} aria-hidden="true" />
          <div>
            <span className="lt-dashboard-section__eyebrow">共同支出</span>
            <h2 id="dashboard-member-title">支付与承担</h2>
          </div>
        </div>
      </header>

      <div className="lt-dashboard-member-list">
        {orderedMembers.map((member) => (
          <article className="lt-dashboard-member-row" key={member.user_id}>
            <div>
              <strong>{member.user_id === currentUserId ? '我' : member.display_name}</strong>
              <span>{member.user_id === currentUserId ? member.display_name : '账本伙伴'}</span>
            </div>
            <dl>
              <div>
                <dt>已支付</dt>
                <dd>{formatDashboardAmount(member.paid_cents)}</dd>
              </div>
              <div>
                <dt>应承担</dt>
                <dd>{formatDashboardAmount(member.share_cents)}</dd>
              </div>
            </dl>
          </article>
        ))}
      </div>
    </section>
  );
}
