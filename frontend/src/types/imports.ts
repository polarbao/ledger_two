export type ImportSourceType = 'wechat' | 'alipay' | 'generic';

export type ImportDirection = 'expense' | 'income' | 'refund' | 'transfer' | 'unknown';

export type ImportTargetTransactionType = 'expense' | 'income' | 'skipped';

export type ImportDuplicateStatus = 'new' | 'duplicate' | 'suspicious' | 'invalid';

export type ImportRowStatus = 'pending' | 'adjusted' | 'skipped' | 'imported' | 'failed';

export type ImportVisibility = 'private' | 'shared' | 'partner_readable';

export type ImportRuleMatchType = 'merchant_contains' | 'description_contains' | 'source_account' | 'amount_range';

export type ImportRuleStatus = 'active' | 'archived';

export interface ImportRowError {
  code: string;
  message: string;
}

export interface ImportPreviewRow {
  id: string;
  batch_id: string;
  row_number: number;
  occurred_at?: string;
  title: string;
  merchant: string;
  description?: string;
  amount_cents: number;
  direction: ImportDirection;
  target_transaction_type: ImportTargetTransactionType;
  duplicate_status: ImportDuplicateStatus;
  row_status: ImportRowStatus;
  source_account?: string;
  external_order_id?: string;
  suspicious_reason?: string;
  suggested_category_id?: string;
  suggested_account_id?: string;
  suggested_tag_ids?: string[];
  suggested_rule_id?: string;
  suggestion_reason?: string;
  selected_category_id?: string;
  selected_account_id?: string;
  selected_tag_ids?: string[];
  visibility?: ImportVisibility;
  error?: ImportRowError;
}

export interface ImportPreviewBatch {
  id: string;
  ledger_id: string;
  source_type: ImportSourceType;
  filename: string;
  file_sha256: string;
  status: 'previewing' | 'ready' | 'committed' | 'failed' | 'expired';
  total_rows: number;
  new_rows: number;
  duplicate_rows: number;
  suspicious_rows: number;
  invalid_rows: number;
  imported_rows: number;
  skipped_rows: number;
  failed_rows: number;
  created_by_user_id: string;
  created_at: string;
  updated_at: string;
  committed_at?: string;
  expires_at?: string;
  rows: ImportPreviewRow[];
}

export interface ImportCommitResult {
  batch_id: string;
  status: 'committed';
  imported_rows: number;
  skipped_rows: number;
  failed_rows: number;
  generated_transaction_ids: string[];
}

export interface ImportRuleResult {
  category_id?: string;
  account_id?: string;
  tag_ids?: string[];
  visibility?: ImportVisibility;
}

export interface ImportRule {
  id: string;
  name: string;
  match_type: ImportRuleMatchType;
  pattern: string;
  amount_min_cents?: number;
  amount_max_cents?: number;
  priority: number;
  status: ImportRuleStatus;
  result: ImportRuleResult;
  created_by_user_id: string;
  created_at: string;
  updated_at: string;
  archived_at?: string;
}

export interface ImportRuleUpsertPayload {
  name?: string;
  match_type: ImportRuleMatchType;
  pattern: string;
  amount_min_cents?: number;
  amount_max_cents?: number;
  priority?: number;
  result: ImportRuleResult;
}

export interface UpdateImportRowPayload {
  target_transaction_type?: ImportTargetTransactionType;
  row_status?: Extract<ImportRowStatus, 'pending' | 'adjusted' | 'skipped'>;
  selected_category_id?: string;
  selected_account_id?: string;
  selected_tag_ids?: string[];
  visibility?: ImportVisibility;
}
