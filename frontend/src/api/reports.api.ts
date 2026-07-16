import { api } from './client';

export interface MonthlySummary {
  month: string;
  total_expense: number;
  total_income: number;
  shared_expense: number;
  personal_expense: number;
  settlement_amount: number;
}

export interface CategorySummaryItem {
  id: string;
  name: string;
  amount_cents: number;
  percent: number;
}

export interface TagSummaryItem {
  name: string;
  amount_cents: number;
  percent: number;
}

export interface MemberSummaryItem {
  user_id: string;
  display_name: string;
  paid_amount: number;
  share_amount: number;
  raw_net: number;
  settlement_paid: number;
  settlement_received: number;
  final_net: number;
}

export const reportsApi = {
  getMonthlySummary: (month: string, signal?: AbortSignal) =>
    api.get<MonthlySummary>(
      `/api/reports/monthly-summary?month=${encodeURIComponent(month)}`,
      { signal },
    ),

  getCategorySummary: (month: string, signal?: AbortSignal) =>
    api.get<CategorySummaryItem[]>(
      `/api/reports/category-summary?month=${encodeURIComponent(month)}`,
      { signal },
    ),

  getTagSummary: (month: string, signal?: AbortSignal) =>
    api.get<TagSummaryItem[]>(
      `/api/reports/tag-summary?month=${encodeURIComponent(month)}`,
      { signal },
    ),

  getMemberSummary: (month: string, signal?: AbortSignal) =>
    api.get<MemberSummaryItem[]>(
      `/api/reports/member-summary?month=${encodeURIComponent(month)}`,
      { signal },
    ),
};
