import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const componentDirectory = dirname(fileURLToPath(import.meta.url));

function readComponentFile(relativePath: string) {
  const filePath = resolve(componentDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('UI-FL-04 TransactionFormDrawer contract', () => {
  it('uses the shared modal primitives and keeps orchestration in the drawer', () => {
    const source = readComponentFile('./TransactionFormDrawer.tsx');

    expect(source).toContain("import ConfirmDialog from '../ui/ConfirmDialog'");
    expect(source).toContain("import SegmentedControl from '../ui/SegmentedControl'");
    expect(source).toContain("import useModalSurface from '../ui/useModalSurface'");
    expect(source).toContain("import SharedExpensePreview from './SharedExpensePreview'");
    expect(source).toContain("import TransactionFormFooter from './TransactionFormFooter'");
    expect(source).toContain("import './TransactionFormDrawer.css'");
    expect(source).toContain('formState: { errors, isSubmitting, isDirty }');
    expect(source).toContain('buildSharedExpensePreview(');
    expect(source).toContain('shouldOpenAdvancedFields(');
    expect(source).toContain('transactionsApi.createSharedExpense');
    expect(source).toContain('transactionsApi.createTransaction');
  });

  it('defines an explicit high-frequency, shared and low-frequency hierarchy', () => {
    const source = readComponentFile('./TransactionFormDrawer.tsx');

    expect(source).toContain('className="lt-entry-amount"');
    expect(source).toContain('className="lt-entry-core-grid"');
    expect(source).toContain('className="lt-entry-shared"');
    expect(source).toContain('className="lt-entry-advanced"');
    expect(source).toContain('更多选项');
    expect(source).toContain('放弃本次修改？');
    expect(source).toContain('放弃修改');
  });

  it('keeps stable desktop and mobile geometry without visual gradients', () => {
    const css = readComponentFile('./TransactionFormDrawer.css');

    expect(css).toContain('width: min(440px, 100vw);');
    expect(css).toContain('@media (max-width: 1024px)');
    expect(css).toContain('height: min(92dvh, 900px);');
    expect(css).toContain('@media (max-width: 430px)');
    expect(css).toContain('var(--lt-bg-surface)');
    expect(css).not.toContain('linear-gradient');
    expect(css).not.toContain('letter-spacing: -');
  });
});
