export interface BalanceResponse {
  paid_map: Record<string, number>;
  share_map: Record<string, number>;
  settled_map: Record<string, number>;
  net_map: Record<string, number>;
  has_debt: boolean;
  from_user_id: string;
  from_user_name: string;
  to_user_id: string;
  to_user_name: string;
  amount_cents: number;
}

export interface SettlementResponse {
  id: string;
  ledger_id: string;
  from_user_id: string;
  to_user_id: string;
  amount_cents: number;
  occurred_at: string;
  created_by_user_id: string;
  created_at: string;
}
