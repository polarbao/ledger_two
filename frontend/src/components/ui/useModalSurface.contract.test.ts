import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const componentDirectory = dirname(fileURLToPath(import.meta.url));

describe('modal focus restoration contract', () => {
  it('keeps the close callback current without restarting the open modal effect', () => {
    const source = readFileSync(resolve(componentDirectory, './useModalSurface.ts'), 'utf8');

    expect(source).toContain('const onCloseRef = useRef(onClose);');
    expect(source).toContain('onCloseRef.current = onClose;');
    expect(source).toContain('onCloseRef.current();');
    expect(source).toContain('const returnFocusRef = useRef<HTMLElement | null>(null);');
    expect(source).toContain("document.addEventListener('focusin', rememberFocus);");
    expect(source).toContain("target.closest('[role=\"dialog\"], [role=\"alertdialog\"]')");
    expect(source).toContain('returnFocusRef.current?.focus();');
    expect(source).not.toContain('[initialFocusRef, onClose, open, surfaceRef]');
  });
});
