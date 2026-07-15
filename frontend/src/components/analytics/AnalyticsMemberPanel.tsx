import { ArrowRight, CircleDollarSign, Users } from 'lucide-react';
import type { MemberSummaryItem } from '../../api/reports.api';
import { centsToYuan } from '../../utils/money';
import Button from '../ui/Button';
import StatusChip, { type StatusChipTone } from '../ui/StatusChip';

interface AnalyticsMemberPanelProps {
  month: string;
  members: MemberSummaryItem[];
  onDrilldown: (userId: string) => void;
}

function getMemberStatus(finalNet: number): { label: string; tone: StatusChipTone } {
  if (finalNet > 0) return { label: `应收 ¥${centsToYuan(finalNet)}`, tone: 'success' };
  if (finalNet < 0) return { label: `应付 ¥${centsToYuan(-finalNet)}`, tone: 'warning' };
  return { label: '本期已结清', tone: 'neutral' };
}

function signedMoney(value: number) {
  const sign = value > 0 ? '+' : value < 0 ? '-' : '';
  return `${sign}¥${centsToYuan(Math.abs(value))}`;
}

export default function AnalyticsMemberPanel({ month, members, onDrilldown }: AnalyticsMemberPanelProps) {
  return (
    <div className="analytics-member-view">
      <section className="analytics-member-intro">
        <div className="analytics-panel__title">
          <Users size={20} aria-hidden="true" />
          <div>
            <span>{month}</span>
            <h2>成员支付与承担</h2>
          </div>
        </div>
        <p>实际支付减去消费承担得到垫付净额，已登记结算只调整最终未结，不计入消费。</p>
      </section>

      <div className="analytics-member-grid">
        {members.map((member) => {
          const status = getMemberStatus(member.final_net);
          return (
            <article key={member.user_id} className="analytics-member-card">
              <header>
                <span className="analytics-member-avatar" aria-hidden="true">
                  {member.display_name.charAt(0) || 'U'}
                </span>
                <div>
                  <h3>{member.display_name}</h3>
                  <span>账本成员</span>
                </div>
                <StatusChip tone={status.tone}>{status.label}</StatusChip>
              </header>

              <dl className="analytics-member-values">
                <div>
                  <dt>实际支付</dt>
                  <dd>¥{centsToYuan(member.paid_amount)}</dd>
                </div>
                <div>
                  <dt>消费承担</dt>
                  <dd>¥{centsToYuan(member.share_amount)}</dd>
                </div>
                <div>
                  <dt>垫付净额</dt>
                  <dd className={member.raw_net < 0 ? 'analytics-value--negative' : 'analytics-value--positive'}>
                    {signedMoney(member.raw_net)}
                  </dd>
                </div>
                <div>
                  <dt>最终未结</dt>
                  <dd className={member.final_net < 0 ? 'analytics-value--negative' : 'analytics-value--positive'}>
                    {signedMoney(member.final_net)}
                  </dd>
                </div>
              </dl>

              <div className="analytics-member-settlement">
                <CircleDollarSign size={17} aria-hidden="true" />
                <span>已付结算 ¥{centsToYuan(member.settlement_paid)}</span>
                <span>已收结算 ¥{centsToYuan(member.settlement_received)}</span>
              </div>

              <Button
                variant="ghost"
                endIcon={<ArrowRight size={16} />}
                onClick={() => onDrilldown(member.user_id)}
              >
                查看支付流水
              </Button>
            </article>
          );
        })}
      </div>
    </div>
  );
}
