export interface UserBalance {
  user_id: string;
  paid_cents: number;
  share_cents: number;
  settled_out_cents: number;
  settled_in_cents: number;
  net_cents: number;
}

export interface SuggestedTransfer {
  from_user_id: string;
  to_user_id: string;
  amount_cents: number;
}

export interface BalanceResponse {
  user_balances?: UserBalance[];
  suggested_transfers?: SuggestedTransfer[];
  
  // Backwards compatibility
  user_a_paid_cents?: number;
  user_a_share_cents?: number;
  user_b_paid_cents?: number;
  user_b_share_cents?: number;
  user_a_settled_to_b_cents?: number;
  user_b_settled_to_a_cents?: number;
  user_a_net_cents?: number;
  user_b_net_cents?: number;
  from_user_id?: string;
  to_user_id?: string;
  amount_cents?: number;
}

export interface CreateSettlementPayload {
  from_user_id: string;
  to_user_id: string;
  amount_cents: number;
  occurred_at: string;
  note: string;
}


export interface SettlementResponse {
  id: string;
  ledger_id: string;
  from_user_id: string;
  to_user_id: string;
  amount_cents: number;
  occurred_at: string;
  note?: string;
  created_by_user_id: string;
  created_at: string;
}
