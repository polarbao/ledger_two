import type {
  ImportRuleApplyMode,
  ImportRuleMatchType,
  ImportRuleUpsertPayload,
  ImportSourceType,
} from '../../types/imports';

export type ImportRuleStatusFilter = 'all' | 'active' | 'archived';

export interface ImportRuleForm {
  name: string;
  match_type: ImportRuleMatchType;
  pattern: string;
  category_id: string;
  account_id: string;
  tag_ids: string[];
  priority: string;
  source_type: ImportSourceType | 'all';
  apply_mode: ImportRuleApplyMode;
}

export const createDefaultImportRuleForm = (): ImportRuleForm => ({
  name: '',
  match_type: 'merchant_contains',
  pattern: '',
  category_id: '',
  account_id: '',
  tag_ids: [],
  priority: '100',
  source_type: 'all',
  apply_mode: 'suggest',
});

export function buildImportRulePayload(form: ImportRuleForm): ImportRuleUpsertPayload | null {
  const pattern = form.pattern.trim();
  const tagIds = Array.from(new Set(form.tag_ids.filter(Boolean)));
  if (!pattern || (!form.category_id && !form.account_id && tagIds.length === 0)) {
    return null;
  }
  const priority = Number.parseInt(form.priority || '100', 10);
  return {
    name: form.name.trim() || pattern,
    match_type: form.match_type,
    pattern,
    priority: Number.isFinite(priority) ? priority : 100,
    source_type: form.source_type === 'all' ? null : form.source_type,
    apply_mode: form.apply_mode,
    result: {
      category_id: form.category_id || undefined,
      account_id: form.account_id || undefined,
      tag_ids: tagIds,
      visibility: 'private',
    },
  };
}
