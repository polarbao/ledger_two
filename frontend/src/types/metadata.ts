export type MetadataKind = 'categories' | 'tags' | 'accounts';

export interface MetadataItem {
  id: string;
  ledger_id: string;
  name: string;
  type?: string;
  icon?: string;
  color?: string;
  sort_order: number;
  usage_count: number;
  is_archived: boolean;
}

export interface MetadataUpsertPayload {
  name: string;
  type?: string;
  icon?: string;
  color?: string;
}
