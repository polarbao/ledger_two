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

export interface TransactionTemplateResponse {
  id: string;
  name: string;
  type: 'expense' | 'income' | 'shared_expense';
  title: string;
  amount_cents?: number | null;
  category_id: string;
  account_id: string;
  payer_user_id: string;
  split_method: string;
  tag_names: string[];
  note: string;
  created_by_user_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTemplatePayload {
  name: string;
  type: 'expense' | 'income' | 'shared_expense';
  title?: string;
  amount_cents?: number;
  category_id?: string;
  account_id?: string;
  payer_user_id?: string;
  split_method?: string;
  tag_names?: string[];
  note?: string;
}

export interface RecurringRuleResponse {
  id: string;
  name: string;
  type: 'expense' | 'income' | 'shared_expense';
  title: string;
  amount_cents?: number | null;
  category_id: string;
  payer_user_id: string;
  split_method: string;
  tag_names: string[];
  note: string;
  frequency: 'weekly' | 'monthly' | 'yearly';
  next_due_date: string;
  created_by_user_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateRecurringRulePayload {
  name: string;
  type: 'expense' | 'income' | 'shared_expense';
  title?: string;
  amount_cents?: number;
  category_id?: string;
  payer_user_id?: string;
  split_method?: string;
  tag_names?: string[];
  note?: string;
  frequency: 'weekly' | 'monthly' | 'yearly';
  next_due_date: string;
}

export interface RecurringReminderResponse {
  id: string;
  rule_id: string;
  rule_name: string;
  type: 'expense' | 'income' | 'shared_expense';
  title: string;
  amount_cents?: number | null;
  category_id: string;
  category_name: string;
  payer_user_id: string;
  split_method: string;
  tag_names: string[];
  note: string;
  frequency: 'weekly' | 'monthly' | 'yearly';
  due_date: string;
  status: 'pending' | 'confirmed' | 'ignored';
  transaction_id?: string;
  created_at: string;
  updated_at: string;
}

export interface CSVParseResponse {
  headers: string[];
  rows: string[][];
}

export interface ImportItemPayload {
  occurred_at: string;
  amount_cents: number;
  title: string;
  merchant: string;
  category_id: string;
  account_id: string;
  payer_user_id: string;
  type: 'expense' | 'shared_expense';
  tag_names: string[];
  note: string;
}

export interface AnalyzeImportPayload {
  items: ImportItemPayload[];
}

export interface AnalyzeImportResponse {
  total_count: number;
  import_count: number;
  skip_count: number;
}

export interface CommitImportPayload {
  filename: string;
  items: ImportItemPayload[];
}

export interface Account {
  id: string;
  ledger_id: string;
  owner_user_id: string;
  name: string;
  type: string;
  currency: string;
  initial_balance: number;
  is_archived: boolean;
}

export interface ImportRuleResponse {
  id: string;
  keyword: string;
  category_id: string;
  tag_names: string[];
  account_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateImportRulePayload {
  keyword: string;
  category_id?: string;
  tag_names?: string[];
  account_id?: string;
}




