export type ImportSourceType = 'wechat' | 'alipay' | 'generic';

export type ImportDirection = 'expense' | 'income' | 'refund' | 'transfer' | 'unknown';

export type ImportTargetTransactionType = 'expense' | 'income' | 'skipped';

export type ImportDuplicateStatus = 'new' | 'duplicate' | 'suspicious' | 'invalid';

export type ImportRowStatus = 'pending' | 'adjusted' | 'skipped' | 'imported' | 'failed';

export type ImportVisibility = 'private' | 'shared' | 'partner_readable';

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

export interface UpdateImportRowPayload {
  target_transaction_type?: ImportTargetTransactionType;
  row_status?: Extract<ImportRowStatus, 'pending' | 'adjusted' | 'skipped'>;
  selected_category_id?: string;
  selected_account_id?: string;
  selected_tag_ids?: string[];
  visibility?: ImportVisibility;
}
