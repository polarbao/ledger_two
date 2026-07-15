import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageDirectory = dirname(fileURLToPath(import.meta.url));

function readSource(relativePath: string) {
  return readFileSync(resolve(pageDirectory, relativePath), 'utf8');
}

describe('UI-FL-09 analytics page contract', () => {
  it('uses shared states, tabs and ledger-scoped report keys', () => {
    const source = readSource('./AnalyticsPage.tsx');

    expect(source).toContain('<SegmentedControl');
    expect(source).toContain('<PageState');
    expect(source).toContain('queryKeys.reports.monthly(activeLedgerId, month)');
    expect(source).toContain('queryKeys.reports.category(activeLedgerId, currentMonth)');
    expect(source).toContain('queryKeys.reports.member(activeLedgerId, currentMonth)');
    expect(source).toContain('queryKeys.reports.tag(activeLedgerId, currentMonth)');
  });

  it('keeps drilldowns on the existing transaction URL contract', () => {
    const source = readSource('./AnalyticsPage.tsx');
    const model = readSource('./analyticsPageModel.ts');

    expect(source).toContain('buildTransactionsDrilldown');
    expect(model).toContain("params.set('category_id', categoryId)");
    expect(model).toContain("params.set('tag', tag)");
    expect(model).toContain("params.set('payer_user_id', payerUserId)");
  });

  it('does not add local hardcoded colors or gradients', () => {
    const sources = [
      readSource('./AnalyticsPage.tsx'),
      readSource('../components/analytics/AnalyticsTrendPanel.tsx'),
      readSource('../components/analytics/AnalyticsRankingPanel.tsx'),
      readSource('../components/analytics/AnalyticsMemberPanel.tsx'),
      readSource('./AnalyticsPage.css'),
    ].join('\n');

    expect(sources).not.toMatch(/#[0-9a-f]{3,8}/i);
    expect(sources).not.toMatch(/rgba?\(/i);
    expect(sources).not.toMatch(/gradient\(/i);
  });

  it('states that settlement is excluded from consumption statistics', () => {
    const trendSource = readSource('../components/analytics/AnalyticsTrendPanel.tsx');
    const memberSource = readSource('../components/analytics/AnalyticsMemberPanel.tsx');

    expect(trendSource).toContain('settlement 记录不进入消费统计');
    expect(memberSource).toContain('已登记结算只调整最终未结，不计入消费');
  });
});
