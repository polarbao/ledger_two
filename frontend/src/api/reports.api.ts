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
  getMonthlySummary: (month: string) =>
    api.get<MonthlySummary>(`/api/reports/monthly-summary?month=${encodeURIComponent(month)}`),

  getCategorySummary: (month: string) =>
    api.get<CategorySummaryItem[]>(`/api/reports/category-summary?month=${encodeURIComponent(month)}`),

  getTagSummary: (month: string) =>
    api.get<TagSummaryItem[]>(`/api/reports/tag-summary?month=${encodeURIComponent(month)}`),

  getMemberSummary: (month: string) =>
    api.get<MemberSummaryItem[]>(`/api/reports/member-summary?month=${encodeURIComponent(month)}`),
};
