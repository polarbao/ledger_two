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

export type MetadataProfileKey = 'basic_cn_v1' | 'empty';
export type MetadataProfileAction = 'create' | 'reuse' | 'skip' | 'conflict' | 'existing';

export interface MetadataProfileItem {
  system_key: string;
  kind: 'expense_category' | 'income_category' | 'tag';
  name: string;
  icon?: string;
  color?: string;
  action: MetadataProfileAction;
  existing_id?: string;
}

export interface MetadataDefaultProfile {
  key: MetadataProfileKey;
  version: number;
  items: MetadataProfileItem[];
}

export interface MetadataProfilePreviewResult {
  profile: MetadataDefaultProfile;
  create_count: number;
  reuse_count: number;
  conflict_count: number;
}

export interface MetadataProfileConflictResolution {
  system_key: string;
  action: 'reuse' | 'skip';
  existing_id?: string;
}

export interface MetadataProfileApplyResult {
  profile: MetadataProfileKey;
  created_count: number;
  reused_count: number;
  skipped_count: number;
  metadata_profile_version: number;
}
