import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageDirectory = dirname(fileURLToPath(import.meta.url));

function readFile(relativePath: string) {
  const filePath = resolve(pageDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('UI-FL-08 import workbench contract', () => {
  it('reuses shared primitives and preserves the frozen import state machine', () => {
    const source = readFile('./ImportPage.tsx');

    expect(source).toContain("import ConfirmDialog from '../components/ui/ConfirmDialog'");
    expect(source).toContain("import SegmentedControl from '../components/ui/SegmentedControl'");
    expect(source).toContain("import ImportPreviewRows from '../components/import/ImportPreviewRows'");
    expect(source).toContain("import ImportRowEditor from '../components/import/ImportRowEditor'");
    expect(source).toContain('importsApi.preview({ file, sourceType })');
    expect(source).toContain('importsApi.updateRow');
    expect(source).toContain('importsApi.commit(batch.id)');
    expect(source).toContain('Preview 不写正式账单');
    expect(source).toContain('不会写入 transactions');
    expect(source).not.toContain('createPortal');
    expect(source).not.toContain('style={{');
  });

  it('uses responsive table/cards and an explicit row editor without changing parser inputs', () => {
    const rows = readFile('../components/import/ImportPreviewRows.tsx');
    const editor = readFile('../components/import/ImportRowEditor.tsx');
    const css = readFile('./ImportPage.css');

    expect(rows).toContain("import ResponsiveDataList from '../ui/ResponsiveDataList'");
    expect(rows).toContain('className="import-preview-table"');
    expect(rows).toContain('className="import-row-card');
    expect(editor).toContain("import BottomSheet from '../ui/BottomSheet'");
    expect(editor).toContain('分类');
    expect(editor).toContain('支付账户');
    expect(editor).toContain('可见范围');
    expect(editor).not.toContain("['shared', '共同账单']");
    expect(css).toContain('min-width: 1040px;');
    expect(css).toContain('@media (max-width: 1024px)');
    expect(css).toContain('@media (max-width: 768px)');
    expect(css).toContain('@media (max-width: 430px)');
    expect(css).toContain('font-variant-numeric: tabular-nums;');
    expect(css).not.toContain('linear-gradient');
    expect(css).not.toContain('letter-spacing: -');
  });
});
