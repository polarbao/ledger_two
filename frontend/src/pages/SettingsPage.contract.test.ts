import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageDirectory = dirname(fileURLToPath(import.meta.url));

function readPageFile(relativePath: string) {
  const filePath = resolve(pageDirectory, relativePath);
  return existsSync(filePath) ? readFileSync(filePath, 'utf8') : '';
}

describe('UI-FL-07 settings page contract', () => {
  it('keeps safety actions explicit and aligned with the existing service permissions', () => {
    const source = readPageFile('./SettingsPage.tsx');

    expect(source).toContain("import ConfirmDialog from '../components/ui/ConfirmDialog'");
    expect(source).toContain("const canImportData = useHasLedgerRole(['owner'])");
    expect(source).toContain("const canExportData = useHasLedgerRole(['owner', 'editor'])");
		expect(source).toContain('const canManageSafety = Boolean(currentUser?.instance_admin)');
    expect(source).toContain('preview 不会写入正式账单');
    expect(source).toContain('只读数据包');
    expect(source).toContain('不能替代 SQLite 物理备份或直接恢复');
    expect(source).toContain('不会在线替换运行中的数据库');
    expect(source).toContain('不展示密码、Cookie、密钥、DSN 或服务器绝对路径');
    expect(source).not.toContain('style={{');
  });

  it('defines stable Fresh Light settings geometry without decorative gradients', () => {
    const css = readPageFile('./SettingsPage.css');

    expect(css).toContain('max-width: 1240px;');
    expect(css).toContain('grid-template-columns: repeat(6, minmax(0, 1fr));');
    expect(css).toContain('@media (max-width: 1024px)');
    expect(css).toContain('@media (max-width: 768px)');
    expect(css).toContain('@media (max-width: 430px)');
    expect(css).toContain('var(--lt-bg-surface)');
    expect(css).not.toContain('linear-gradient');
    expect(css).not.toContain('letter-spacing: -');
  });
});
