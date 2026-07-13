import { CalendarClock, Check, SkipForward } from 'lucide-react';
import type { RecurringReminderResponse } from '../../types/transaction';
import Button from '../ui/Button';
import StatusChip from '../ui/StatusChip';
import {
  formatDashboardAmount,
  getRecurringFrequencyLabel,
  getRecurringTypeLabel,
} from './dashboardModel';

interface RecurringReminderListProps {
  reminders: RecurringReminderResponse[];
  isMutating: boolean;
  confirmingId?: string;
  onConfirm: (id: string) => void;
  onSkip: (id: string) => void;
}

export default function RecurringReminderList({
  reminders,
  isMutating,
  confirmingId,
  onConfirm,
  onSkip,
}: RecurringReminderListProps) {
  return (
    <section className="lt-dashboard-section lt-dashboard-reminders" aria-labelledby="dashboard-reminders-title">
      <header className="lt-dashboard-section__header">
        <div className="lt-dashboard-section__heading">
          <CalendarClock size={20} aria-hidden="true" />
          <div>
            <span className="lt-dashboard-section__eyebrow">周期账单</span>
            <h2 id="dashboard-reminders-title">待确认事项</h2>
          </div>
        </div>
        <StatusChip tone="warning">{reminders.length} 笔待确认</StatusChip>
      </header>

      <div className="lt-dashboard-reminders__list">
        {reminders.map((reminder) => (
          <article className="lt-dashboard-reminder-row" key={reminder.id}>
            <div className="lt-dashboard-reminder-row__copy">
              <strong>{reminder.rule_name}</strong>
              <span>
                {reminder.due_date} · {getRecurringFrequencyLabel(reminder.frequency)} ·{' '}
                {getRecurringTypeLabel(reminder.type)}
              </span>
              {reminder.category_name ? <small>分类：{reminder.category_name}</small> : null}
            </div>
            <strong className="lt-dashboard-reminder-row__amount">
              {reminder.amount_cents == null
                ? '金额待确认'
                : formatDashboardAmount(reminder.amount_cents)}
            </strong>
            <div className="lt-dashboard-reminder-row__actions">
              <Button
                className="lt-dashboard-compact-button"
                variant="secondary"
                startIcon={<SkipForward size={16} />}
                disabled={isMutating}
                onClick={() => onSkip(reminder.id)}
              >
                跳过本期
              </Button>
              <Button
                className="lt-dashboard-compact-button"
                variant="primary"
                startIcon={<Check size={16} />}
                isLoading={isMutating && confirmingId === reminder.id}
                disabled={isMutating}
                onClick={() => onConfirm(reminder.id)}
              >
                {confirmingId === reminder.id ? '记账中' : '确认记账'}
              </Button>
            </div>
          </article>
        ))}
      </div>
      <p className="lt-dashboard-reminders__note">确认后生成真实账单；跳过只影响本期提醒。</p>
    </section>
  );
}
