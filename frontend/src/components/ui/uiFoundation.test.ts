import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

describe('UI-FL-01 base components', () => {
  it('renders an accessible loading button without enabling interaction', async () => {
    const buttonModule = await import('./Button').catch(() => null);

    expect(buttonModule).not.toBeNull();
    if (!buttonModule) return;

    const markup = renderToStaticMarkup(createElement(
      buttonModule.default,
      { variant: 'primary', isLoading: true },
      '保存账单',
    ));

    expect(markup).toContain('class="ui-button ui-button--primary');
    expect(markup).toContain('aria-busy="true"');
    expect(markup).toContain('disabled=""');
    expect(markup).toContain('保存账单');
  });

  it('renders status text in addition to its semantic color class', async () => {
    const chipModule = await import('./StatusChip').catch(() => null);

    expect(chipModule).not.toBeNull();
    if (!chipModule) return;

    const markup = renderToStaticMarkup(createElement(
      chipModule.default,
      { tone: 'warning' },
      '疑似，需确认',
    ));

    expect(markup).toContain('ui-status-chip--warning');
    expect(markup).toContain('疑似，需确认');
  });

  it('exposes selected and disabled segmented options to assistive technology', async () => {
    const segmentModule = await import('./SegmentedControl').catch(() => null);

    expect(segmentModule).not.toBeNull();
    if (!segmentModule) return;

    const markup = renderToStaticMarkup(createElement(segmentModule.default, {
      ariaLabel: '流水类型',
      value: 'shared',
      options: [
        { value: 'all', label: '全部' },
        { value: 'shared', label: '共同', count: 2 },
        { value: 'disabled', label: '不可用', disabled: true },
      ],
      onChange: () => undefined,
    }));

    expect(markup).toContain('role="group"');
    expect(markup).toContain('aria-label="流水类型"');
    expect(markup).toContain('aria-pressed="true"');
    expect(markup).toContain('disabled=""');
    expect(markup).toContain('共同');
  });

  it('renders reusable empty and error state content with an explicit action', async () => {
    const stateModule = await import('./StatePanel').catch(() => null);

    expect(stateModule).not.toBeNull();
    if (!stateModule) return;

    const markup = renderToStaticMarkup(createElement(stateModule.default, {
      tone: 'danger',
      title: '加载失败',
      description: '请检查网络后重试。',
      action: { label: '立即重试', onClick: () => undefined },
    }));

    expect(markup).toContain('ui-state-panel--danger');
    expect(markup).toContain('加载失败');
    expect(markup).toContain('请检查网络后重试。');
    expect(markup).toContain('立即重试');
  });

  it('renders a labelled confirmation dialog with explicit cancel and confirm actions', async () => {
    const dialogModule = await import('./ConfirmDialog').catch(() => null);

    expect(dialogModule).not.toBeNull();
    if (!dialogModule) return;

    const markup = renderToStaticMarkup(createElement(dialogModule.default, {
      open: true,
      title: '删除这笔账单？',
      description: '账单会进入回收状态，不会立即物理删除。',
      confirmLabel: '删除账单',
      cancelLabel: '保留账单',
      tone: 'danger',
      onConfirm: () => undefined,
      onClose: () => undefined,
    }));

    expect(markup).toContain('role="alertdialog"');
    expect(markup).toContain('aria-modal="true"');
    expect(markup).toContain('删除这笔账单？');
    expect(markup).toContain('保留账单');
    expect(markup).toContain('删除账单');
  });

  it('renders a labelled bottom sheet with a familiar close control', async () => {
    const sheetModule = await import('./BottomSheet').catch(() => null);

    expect(sheetModule).not.toBeNull();
    if (!sheetModule) return;

    const markup = renderToStaticMarkup(createElement(
      sheetModule.default,
      { open: true, title: '筛选流水', onClose: () => undefined },
      createElement('p', null, '筛选条件'),
    ));

    expect(markup).toContain('role="dialog"');
    expect(markup).toContain('aria-modal="true"');
    expect(markup).toContain('aria-label="关闭筛选流水"');
    expect(markup).toContain('筛选条件');
  });
});
