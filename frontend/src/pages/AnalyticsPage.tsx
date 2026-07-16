import { useState } from 'react';
import { useQueries, useQuery } from '@tanstack/react-query';
import { BarChart3, ShieldCheck } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { reportsApi, type MonthlySummary } from '../api/reports.api';
import { queryKeys } from '../api/queryKeys';
import AnalyticsMemberPanel from '../components/analytics/AnalyticsMemberPanel';
import AnalyticsRankingPanel from '../components/analytics/AnalyticsRankingPanel';
import AnalyticsTrendPanel from '../components/analytics/AnalyticsTrendPanel';
import { useLedgerContext } from '../components/ledger/useLedgerContext';
import PageState from '../components/ui/PageState';
import SegmentedControl from '../components/ui/SegmentedControl';
import StatusChip from '../components/ui/StatusChip';
import { useUIStore } from '../stores/ui.store';
import { buildMonthRange, buildTransactionsDrilldown } from './analyticsPageModel';
import './AnalyticsPage.css';

type AnalyticsTab = 'trend' | 'category' | 'member' | 'tag';

const tabOptions = [
  { value: 'trend', label: '趋势' },
  { value: 'category', label: '分类' },
  { value: 'member', label: '成员' },
  { value: 'tag', label: '标签' },
] as const;

function isMonthlySummary(value: MonthlySummary | undefined): value is MonthlySummary {
  return Boolean(value);
}

export default function AnalyticsPage() {
  const navigate = useNavigate();
  const currentMonth = useUIStore((state) => state.currentMonth);
  const { ledgerId, isArchivedView } = useLedgerContext();
  const [activeTab, setActiveTab] = useState<AnalyticsTab>('trend');
  const trendMonths = buildMonthRange(currentMonth, 6);

  const trendQueries = useQueries({
    queries: trendMonths.map((month) => ({
      queryKey: queryKeys.reports.monthly(ledgerId, month),
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        reportsApi.getMonthlySummary(month, signal),
      enabled: Boolean(ledgerId) && activeTab === 'trend',
    })),
  });

  const categoryQuery = useQuery({
    queryKey: queryKeys.reports.category(ledgerId, currentMonth),
    queryFn: ({ signal }) => reportsApi.getCategorySummary(currentMonth, signal),
    enabled: Boolean(ledgerId) && activeTab === 'category',
  });

  const memberQuery = useQuery({
    queryKey: queryKeys.reports.member(ledgerId, currentMonth),
    queryFn: ({ signal }) => reportsApi.getMemberSummary(currentMonth, signal),
    enabled: Boolean(ledgerId) && activeTab === 'member',
  });

  const tagQuery = useQuery({
    queryKey: queryKeys.reports.tag(ledgerId, currentMonth),
    queryFn: ({ signal }) => reportsApi.getTagSummary(currentMonth, signal),
    enabled: Boolean(ledgerId) && activeTab === 'tag',
  });

  const trendPoints = trendQueries.map((query) => query.data).filter(isMonthlySummary);
  const activeState = activeTab === 'trend'
    ? {
        isLoading: trendQueries.some((query) => query.isLoading),
        isError: trendQueries.some((query) => query.isError),
        isEmpty: trendPoints.length !== trendMonths.length,
        retry: () => trendQueries.forEach((query) => query.refetch()),
      }
    : activeTab === 'category'
      ? {
          isLoading: categoryQuery.isLoading,
          isError: categoryQuery.isError,
          isEmpty: !categoryQuery.data?.length,
          retry: categoryQuery.refetch,
        }
      : activeTab === 'member'
        ? {
            isLoading: memberQuery.isLoading,
            isError: memberQuery.isError,
            isEmpty: !memberQuery.data?.length,
            retry: memberQuery.refetch,
          }
        : {
            isLoading: tagQuery.isLoading,
            isError: tagQuery.isError,
            isEmpty: !tagQuery.data?.length,
            retry: tagQuery.refetch,
          };

  const emptyMessage = ledgerId
    ? `${currentMonth} 暂无${activeTab === 'member' ? '成员统计' : activeTab === 'category' ? '分类支出' : activeTab === 'tag' ? '标签支出' : '趋势数据'}。`
    : '当前没有可用账本。';

  return (
    <main className="analytics-page animate-fade-in">
      <header className="analytics-page__header">
        <div className="analytics-page__title">
          <span className="analytics-page__icon"><BarChart3 size={22} aria-hidden="true" /></span>
          <div>
            <span className="analytics-page__eyebrow">{currentMonth} 账期</span>
            <h1>分析</h1>
            <p>基于你可见的账单，从收支变化、消费去向和成员承担理解当前账本。</p>
          </div>
        </div>
        <StatusChip tone="info" icon={<ShieldCheck size={14} />}>
          服务端可见口径
        </StatusChip>
      </header>

      <SegmentedControl
        className="analytics-page__tabs"
        ariaLabel="分析维度"
        value={activeTab}
        options={tabOptions}
        onChange={setActiveTab}
        fullWidth
      />

      <PageState
        isLoading={activeState.isLoading}
        isError={activeState.isError}
        isEmpty={activeState.isEmpty}
        errorMsg="统计数据加载失败，请检查网络后重试。"
        emptyMessage={emptyMessage}
        loadingMessage="正在汇总账本数据..."
        skeletonType="card"
        onRetry={activeState.retry}
      >
        {activeTab === 'trend' && trendPoints.length === trendMonths.length ? (
          <AnalyticsTrendPanel
            points={trendPoints}
            onMonthDrilldown={(month) => navigate(buildTransactionsDrilldown({
              month,
              archivedLedgerId: isArchivedView ? ledgerId ?? undefined : undefined,
            }))}
          />
        ) : null}

        {activeTab === 'category' && categoryQuery.data?.length ? (
          <AnalyticsRankingPanel
            kind="category"
            month={currentMonth}
            items={categoryQuery.data}
            onDrilldown={(item) => {
              if ('id' in item && item.id) {
                navigate(buildTransactionsDrilldown({
                  month: currentMonth,
                  categoryId: item.id,
                  archivedLedgerId: isArchivedView ? ledgerId ?? undefined : undefined,
                }));
              }
            }}
          />
        ) : null}

        {activeTab === 'member' && memberQuery.data?.length ? (
          <AnalyticsMemberPanel
            month={currentMonth}
            members={memberQuery.data}
            onDrilldown={(payerUserId) => navigate(buildTransactionsDrilldown({
              month: currentMonth,
              payerUserId,
              archivedLedgerId: isArchivedView ? ledgerId ?? undefined : undefined,
            }))}
          />
        ) : null}

        {activeTab === 'tag' && tagQuery.data?.length ? (
          <AnalyticsRankingPanel
            kind="tag"
            month={currentMonth}
            items={tagQuery.data}
            onDrilldown={(item) => navigate(buildTransactionsDrilldown({
              month: currentMonth,
              tag: item.name,
              archivedLedgerId: isArchivedView ? ledgerId ?? undefined : undefined,
            }))}
          />
        ) : null}
      </PageState>
    </main>
  );
}
