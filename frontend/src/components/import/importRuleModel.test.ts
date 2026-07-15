import { describe, expect, it } from 'vitest';
import { buildImportRulePayload, createDefaultImportRuleForm } from './importRuleModel';

describe('import rule model', () => {
  it('requires a matcher and at least one recommendation target', () => {
    expect(buildImportRulePayload(createDefaultImportRuleForm())).toBeNull();
    expect(buildImportRulePayload({
      ...createDefaultImportRuleForm(),
      pattern: '星巴克',
    })).toBeNull();
  });

  it('deduplicates tags and keeps recommendations private by default', () => {
    expect(buildImportRulePayload({
      ...createDefaultImportRuleForm(),
      name: '咖啡',
      pattern: '星巴克',
      category_id: 'category-1',
      tag_ids: ['tag-1', 'tag-1'],
      priority: '20',
    })).toEqual({
      name: '咖啡',
      match_type: 'merchant_contains',
      pattern: '星巴克',
      priority: 20,
      result: {
        category_id: 'category-1',
        account_id: undefined,
        tag_ids: ['tag-1'],
        visibility: 'private',
      },
    });
  });
});
