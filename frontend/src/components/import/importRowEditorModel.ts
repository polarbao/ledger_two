import type {
  ImportPreviewRow,
  ImportTargetTransactionType,
  ImportVisibility,
  UpdateImportRowPayload,
} from '../../types/imports';
import type { MetadataItem } from '../../types/metadata';

export interface ImportRowEditorDraft {
  targetTransactionType: ImportTargetTransactionType;
  categoryId: string;
  accountId: string;
  tagIds: string[];
  visibility: Exclude<ImportVisibility, 'shared'>;
}

export const IMPORT_TAG_LIMIT = 8;

export function createImportRowEditorDraft(
  row: ImportPreviewRow,
  categories: MetadataItem[],
  accounts: MetadataItem[],
  tags: MetadataItem[],
): ImportRowEditorDraft {
  return {
    targetTransactionType: row.target_transaction_type === 'skipped'
      ? defaultTargetType(row)
      : row.target_transaction_type,
    categoryId: resolveActiveMetadataId(row.selected_category_id, row.suggested_category_id, categories),
    accountId: resolveActiveMetadataId(row.selected_account_id, row.suggested_account_id, accounts),
    tagIds: resolveActiveTagIds(row.selected_tag_ids, row.suggested_tag_ids, tags),
    visibility: row.visibility === 'partner_readable' ? 'partner_readable' : 'private',
  };
}

export function buildImportRowUpdatePayload(draft: ImportRowEditorDraft): UpdateImportRowPayload {
  const skipped = draft.targetTransactionType === 'skipped';
  return {
    target_transaction_type: draft.targetTransactionType,
    row_status: skipped ? 'skipped' : 'adjusted',
    selected_category_id: skipped ? '' : draft.categoryId,
    selected_account_id: skipped ? '' : draft.accountId,
    selected_tag_ids: skipped ? [] : Array.from(new Set(draft.tagIds.filter(Boolean))),
    visibility: draft.visibility,
  };
}

function defaultTargetType(row: ImportPreviewRow): ImportTargetTransactionType {
  return row.direction === 'income' || row.direction === 'refund' ? 'income' : 'expense';
}

function resolveActiveMetadataId(
  selectedId: string | undefined,
  suggestedId: string | undefined,
  items: MetadataItem[],
) {
  const candidate = selectedId || suggestedId || '';
  return items.some((item) => item.id === candidate && !item.is_archived) ? candidate : '';
}

function resolveActiveTagIds(
  selectedIds: string[] | undefined,
  suggestedIds: string[] | undefined,
  tags: MetadataItem[],
) {
  const activeIds = new Set(tags.filter((tag) => !tag.is_archived).map((tag) => tag.id));
  return Array.from(new Set((selectedIds || suggestedIds || []).filter((id) => activeIds.has(id))));
}

export function toggleImportTag(currentTagIds: string[], tagId: string, limit = IMPORT_TAG_LIMIT) {
  if (currentTagIds.includes(tagId)) {
    return currentTagIds.filter((id) => id !== tagId);
  }
  if (currentTagIds.length >= limit) return currentTagIds;
  return [...currentTagIds, tagId];
}

export function canRememberImportMerchant(row: ImportPreviewRow) {
  return Boolean(row.merchant.trim())
    && row.row_status !== 'imported'
    && row.row_status !== 'skipped'
    && row.duplicate_status !== 'invalid'
    && row.duplicate_status !== 'duplicate';
}
