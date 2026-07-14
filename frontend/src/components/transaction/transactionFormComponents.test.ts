import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

describe('UI-FL-04 transaction form components', () => {
  it('renders an explainable shared-expense preview without exposing ids', async () => {
    const component = await import('./SharedExpensePreview').catch(() => null);

    expect(component).not.toBeNull();
    if (!component) return;

    const markup = renderToStaticMarkup(createElement(component.default, {
      items: [
        {
          userId: 'private-user-a',
          displayName: '林然',
          shareAmountCents: 5001,
          isPayer: true,
          isParticipating: true,
        },
        {
          userId: 'private-user-b',
          displayName: '北北',
          shareAmountCents: 5000,
          isPayer: false,
          isParticipating: true,
        },
      ],
      currentUserId: 'private-user-a',
    }));

    expect(markup).toContain('承担预览');
    expect(markup).toContain('林然（我）');
    expect(markup).toContain('付款人');
    expect(markup).toContain('¥50.01');
    expect(markup).toContain('¥50.00');
    expect(markup).not.toContain('private-user-a');
    expect(markup).not.toContain('private-user-b');
  });

  it('keeps the footer command hierarchy and offline wording explicit', async () => {
    const component = await import('./TransactionFormFooter').catch(() => null);

    expect(component).not.toBeNull();
    if (!component) return;

    const onlineMarkup = renderToStaticMarkup(createElement(component.default, {
      mode: 'copy',
      isPending: false,
      activeAction: 'close',
      onCancel: () => undefined,
      onContinue: () => undefined,
      onPrimary: () => undefined,
    }));
    const offlineMarkup = renderToStaticMarkup(createElement(component.default, {
      mode: 'offline',
      isPending: false,
      activeAction: 'close',
      onCancel: () => undefined,
      onContinue: () => undefined,
      onPrimary: () => undefined,
    }));
    const editMarkup = renderToStaticMarkup(createElement(component.default, {
      mode: 'edit',
      isPending: false,
      activeAction: 'close',
      onCancel: () => undefined,
      onContinue: () => undefined,
      onPrimary: () => undefined,
    }));

    expect(onlineMarkup).toContain('取消');
    expect(onlineMarkup).toContain('保存并继续');
    expect(onlineMarkup).toContain('保存为新账单');
    expect(offlineMarkup).toContain('保存为离线草稿');
    expect(offlineMarkup).toContain('disabled=""');
    expect(editMarkup).toContain('保存修改');
    expect(editMarkup).toContain('保存并继续');
    expect(editMarkup).toContain('disabled=""');
  });
});
