import type { BalanceResponse } from './settlement';
import type { TransactionResponse } from './transaction';

export interface SummaryItem {
  id: string;
  name: string;
  amount_cents: number;
  percent: number;
}

export interface UserStatItem {
  user_id: string;
  display_name: string;
  paid_cents: number;
  share_cents: number;
}

export interface DashboardResponse {
  month: string;
  total_expense_cents: number;
  total_income_cents: number;
  my_paid_cents: number;
  partner_paid_cents: number;
  shared_balance: BalanceResponse;
  recent_transactions: TransactionResponse[];
  category_summary: SummaryItem[];
  tag_summary: SummaryItem[];
  user_stats: UserStatItem[];
}
