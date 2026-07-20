import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageDirectory = dirname(fileURLToPath(import.meta.url));

function read(relativePath: string) {
  return readFileSync(resolve(pageDirectory, relativePath), 'utf8');
}

describe('Task53U category and tag intelligence UI contract', () => {
  it('exposes persisted classification states and explicit preview-only actions', () => {
    const page = read('./ImportPage.tsx');
    const rows = read('../components/import/ImportPreviewRows.tsx');

    expect(page).toContain('IMPORT_CLASSIFICATION_FILTER_LABELS');
    expect(page).toContain("action: 'accept_suggestions'");
    expect(page).toContain("action: 'apply_values'");
    expect(page).toContain('importsApi.reclassify');
    expect(page).toContain('尚未提交导入批次');
    expect(rows).toContain('classification.reason_text');
    expect(rows).toContain('接受该行已保存的分类和标签建议，不会提交批次');
  });

  it('keeps row saving, merchant learning and rule health independently visible', () => {
    const page = read('./ImportPage.tsx');
    const editor = read('../components/import/ImportRowEditor.tsx');
    const rules = read('../components/import/ImportRuleManager.tsx');

    expect(page).toContain('本行已保存，长期规则未创建');
    expect(editor).toContain('记住此商户');
    expect(editor).toContain('IMPORT_TAG_LIMIT');
    expect(rules).toContain('系统为你记住的规则');
    expect(rules).toContain('committed_hit_count');
    expect(rules).toContain('stale_reference_ids');
    expect(rules).toContain('merchant_equals');
  });

  it('adds default profiles and fallback replacement without parallel UI foundations', () => {
    const metadata = read('./MetadataManagePage.tsx');
    const ledgers = read('./LedgerManagementPage.tsx');
    const css = read('./ImportPage.css');

    expect(metadata).toContain('补充基础分类与标签');
    expect(metadata).toContain('replacement_category_id');
    expect(ledgers).toContain('metadata_profile: metadataProfile');
    expect(css).toContain('.import-classification-summary');
    expect(css).toContain('.import-bulk-bar');
    expect(css).not.toContain('linear-gradient');
  });
});
