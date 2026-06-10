// SplitResponse 对应后端 transaction.SplitResponse
export interface TransactionSplitResponse {
  user_id: string;
  share_amount_cents: number; // 后端 json:"share_amount_cents"
}

export interface TransactionResponse {
  id: string;
  ledger_id?: string;
  type: 'expense' | 'income' | 'shared_expense' | 'settlement';
  title: string;
  amount_cents: number;
  currency: string;
  occurred_at: string;         // ISO8601 string
  owner_user_id: string;
  created_by_user_id: string;
  payer_user_id: string;
  account_id?: string | null;
  category_id?: string | null;
  visibility: 'private' | 'partner_readable' | 'shared';
  split_method?: 'equal' | 'payer_only';
  note?: string;
  status: 'active' | 'deleted';
  tags?: string[];
  participants?: TransactionSplitResponse[]; // 后端 json:"participants,omitempty"
  created_at: string;
  updated_at: string;
}

export interface Category {
  id: string;
  name: string;
}

export interface CreateTransactionPayload {
  type: 'expense' | 'income';
  title?: string;
  amount_cents: number;
  currency: string;
  occurred_at: string;
  payer_user_id: string;
  category_id?: string;
  visibility?: 'private' | 'partner_readable';
  tag_names?: string[];
  note?: string;
}

export interface CreateSharedExpensePayload {
  title?: string;
  amount_cents: number;
  currency: string;
  occurred_at: string;
  payer_user_id: string;
  category_id?: string | null;
  split_method: 'equal' | 'payer_only';
  tag_names?: string[];
  note?: string;
}

