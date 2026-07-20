export type MetadataKind = 'categories' | 'tags' | 'accounts';

export interface MetadataItem {
  id: string;
  ledger_id: string;
  system_key?: string;
  name: string;
  type?: string;
  icon?: string;
  color?: string;
  sort_order: number;
  usage_count: number;
  rule_reference_count: number;
  is_archived: boolean;
}

export interface MetadataUpsertPayload {
  name: string;
  type?: string;
  icon?: string;
  color?: string;
}

export interface MetadataArchivePayload {
  replacement_category_id?: string;
}

export interface MetadataArchiveResult {
  archived_id: string;
  fallback_replaced: boolean;
  transferred_system_key?: 'expense_other' | 'income_other';
  replacement_category_id?: string;
}
