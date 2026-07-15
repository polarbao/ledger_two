import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const pageDirectory = dirname(fileURLToPath(import.meta.url));

describe('import page copy', () => {
  it('describes the empty preview for every supported bill file format', () => {
    const source = readFileSync(resolve(pageDirectory, './ImportPage.tsx'), 'utf8');

    expect(source).toContain('上传账单文件后会在这里看到行级状态和错误原因。');
    expect(source).not.toContain('上传 CSV 后会在这里看到行级状态和错误原因。');
  });
});
