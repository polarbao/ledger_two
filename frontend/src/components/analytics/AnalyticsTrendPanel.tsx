import type { CSSProperties } from 'react';
import { ArrowRight, Scale, TrendingDown, TrendingUp, WalletCards } from 'lucide-react';
import type { MonthlySummary } from '../../api/reports.api';
import { centsToYuan } from '../../utils/money';
import StatusChip from '../ui/StatusChip';
import {
  getChartPercent,
  getChartScale,
  getExpenseChange,
  getExpenseChangeLabel,
  getShortMonthLabel,
} from '../../pages/analyticsPageModel';

interface AnalyticsTrendPanelProps {
  points: MonthlySummary[];
  onMonthDrilldown: (month: string) => void;
}

type BarStyle = CSSProperties & { '--analytics-bar-height': string };

function money(value: number) {
  return `¥${centsToYuan(value)}`;
}

export default function AnalyticsTrendPanel({ points, onMonthDrilldown }: AnalyticsTrendPanelProps) {
  const current = points.at(-1)!;
  const previous = points.at(-2);
  const expenseChange = getExpenseChange(current.total_expense, previous?.total_expense ?? 0);
  const scale = getChartScale(points.flatMap((point) => [point.total_expense, point.total_income]));
  const changeTone = expenseChange.direction === 'up'
    ? 'warning'
    : expenseChange.direction === 'down'
      ? 'success'
      : 'neutral';

  return (
    <div className="analytics-trend">
      <section className="analytics-summary-band" aria-label={`${current.month} 月度摘要`}>
        <article className="analytics-summary-metric">
          <span>本月支出</span>
          <strong>{money(current.total_expense)}</strong>
          <StatusChip tone={changeTone} icon={expenseChange.direction === 'up' ? <TrendingUp size={13} /> : <TrendingDown size={13} />}>
            {getExpenseChangeLabel(expenseChange)}
          </StatusChip>
        </article>
        <article className="analytics-summary-metric">
          <span>本月收入</span>
          <strong className="analytics-amount--income">{money(current.total_income)}</strong>
          <small>不与支出相抵</small>
        </article>
        <article className="analytics-summary-metric">
          <span>个人支出</span>
          <strong>{money(current.personal_expense)}</strong>
          <small>普通个人消费</small>
        </article>
        <article className="analytics-summary-metric">
          <span>共同支出</span>
          <strong className="analytics-amount--shared">{money(current.shared_expense)}</strong>
          <small>按账单总额统计</small>
        </article>
        <article className="analytics-summary-metric">
          <span>当前未结</span>
          <strong className="analytics-amount--warning">{money(current.settlement_amount)}</strong>
          <small>结算中心当前轧差</small>
        </article>
      </section>

      <div className="analytics-trend-grid">
        <section className="analytics-panel analytics-chart-panel">
          <header className="analytics-panel__header">
            <div className="analytics-panel__title">
              <TrendingUp size={19} aria-hidden="true" />
              <div>
                <span>近六个月</span>
                <h2>收支变化</h2>
              </div>
            </div>
            <div className="analytics-chart-legend" aria-label="图例">
              <span><i className="analytics-legend-dot analytics-legend-dot--expense" />支出</span>
              <span><i className="analytics-legend-dot analytics-legend-dot--income" />收入</span>
            </div>
          </header>

          <div className="analytics-chart" role="group" aria-label="近六个月支出与收入柱状图">
            {points.map((point) => {
              const expenseStyle: BarStyle = {
                '--analytics-bar-height': `${getChartPercent(point.total_expense, scale)}%`,
              };
              const incomeStyle: BarStyle = {
                '--analytics-bar-height': `${getChartPercent(point.total_income, scale)}%`,
              };
              return (
                <button
                  key={point.month}
                  type="button"
                  className="analytics-chart-column"
                  aria-label={`${point.month}，支出 ${money(point.total_expense)}，收入 ${money(point.total_income)}，查看流水`}
                  onClick={() => onMonthDrilldown(point.month)}
                >
                  <span className="analytics-chart-bars" aria-hidden="true">
                    <i className="analytics-chart-bar analytics-chart-bar--expense" style={expenseStyle} />
                    <i className="analytics-chart-bar analytics-chart-bar--income" style={incomeStyle} />
                  </span>
                  <span>{getShortMonthLabel(point.month)}</span>
                </button>
              );
            })}
          </div>
          <p className="analytics-panel__footnote">柱状图仅包含消费与收入，settlement 记录不进入消费统计。</p>
        </section>

        <section className="analytics-panel analytics-caliber-panel">
          <header className="analytics-panel__header">
            <div className="analytics-panel__title">
              <Scale size={19} aria-hidden="true" />
              <div>
                <span>统计口径</span>
                <h2>本月支出构成</h2>
              </div>
            </div>
          </header>
          <dl className="analytics-caliber-list">
            <div>
              <dt><WalletCards size={16} />个人消费</dt>
              <dd>{money(current.personal_expense)}</dd>
            </div>
            <div>
              <dt><WalletCards size={16} />共同消费</dt>
              <dd>{money(current.shared_expense)}</dd>
            </div>
            <div className="analytics-caliber-list__total">
              <dt>消费合计</dt>
              <dd>{money(current.total_expense)}</dd>
            </div>
          </dl>
          <button type="button" className="analytics-text-action" onClick={() => onMonthDrilldown(current.month)}>
            查看本月流水 <ArrowRight size={16} aria-hidden="true" />
          </button>
        </section>
      </div>
    </div>
  );
}
