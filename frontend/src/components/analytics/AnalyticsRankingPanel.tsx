import { ArrowRight, Info, PieChart, Tag } from 'lucide-react';
import type { CategorySummaryItem, TagSummaryItem } from '../../api/reports.api';
import { centsToYuan } from '../../utils/money';

type RankingItem = CategorySummaryItem | TagSummaryItem;

interface AnalyticsRankingPanelProps {
  kind: 'category' | 'tag';
  month: string;
  items: RankingItem[];
  onDrilldown: (item: RankingItem) => void;
}

function isCategoryItem(item: RankingItem): item is CategorySummaryItem {
  return 'id' in item;
}

export default function AnalyticsRankingPanel({ kind, month, items, onDrilldown }: AnalyticsRankingPanelProps) {
  const isCategory = kind === 'category';
  const Icon = isCategory ? PieChart : Tag;

  return (
    <section className="analytics-panel analytics-ranking-panel">
      <header className="analytics-panel__header">
        <div className="analytics-panel__title">
          <Icon size={19} aria-hidden="true" />
          <div>
            <span>{month}</span>
            <h2>{isCategory ? '分类支出' : '标签场景'}</h2>
          </div>
        </div>
      </header>

      {!isCategory ? (
        <div className="analytics-caliber-notice">
          <Info size={17} aria-hidden="true" />
          <span>多标签账单会按全额计入每个标签，标签金额合计不等于账期总支出。</span>
        </div>
      ) : null}

      <div className="analytics-ranking-list">
        {items.map((item, index) => {
          const canDrilldown = !isCategoryItem(item) || Boolean(item.id);
          const content = (
            <>
              <span className="analytics-ranking-position">{String(index + 1).padStart(2, '0')}</span>
              <span className="analytics-ranking-copy">
                <strong>{isCategory ? item.name : `#${item.name}`}</strong>
                <progress max="100" value={Math.max(0, Math.min(100, item.percent))}>
                  {item.percent.toFixed(1)}%
                </progress>
              </span>
              <span className="analytics-ranking-value">
                <strong>¥{centsToYuan(item.amount_cents)}</strong>
                <small>{item.percent.toFixed(1)}%</small>
              </span>
              {canDrilldown ? <ArrowRight size={17} aria-hidden="true" /> : null}
            </>
          );

          return canDrilldown ? (
            <button
              key={isCategoryItem(item) ? item.id || item.name : item.name}
              type="button"
              className="analytics-ranking-row"
              aria-label={`查看${item.name}流水`}
              onClick={() => onDrilldown(item)}
            >
              {content}
            </button>
          ) : (
            <div key={`uncategorized-${item.name}`} className="analytics-ranking-row analytics-ranking-row--static">
              {content}
            </div>
          );
        })}
      </div>
    </section>
  );
}
