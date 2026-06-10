export interface TransactionSplitResponse {
  id: string;
  transaction_id: string;
  user_id: string;
  share_amount: number;
  created_at: string;
}

export interface TransactionResponse {
  id: string;
  ledger_id: string;
  type: 'expense' | 'income' | 'shared_expense' | 'settlement';
  title: string;
  amount_cents: number;
  currency: string;
  occurred_at: string;
  owner_user_id: string;
  created_by_user_id: string;
  payer_user_id: string;
  account_id: string;
  category_id: string;
  visibility: 'private' | 'partner_readable' | 'shared';
  split_method?: 'equal' | 'payer_only';
  note?: string;
  status: 'active' | 'deleted';
  tags?: string[];
  splits?: TransactionSplitResponse[];
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

