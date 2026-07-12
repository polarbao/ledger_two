import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';
import { useLedgerStore } from '../stores/ledger.store';
import ImportPage from './ImportPage';

describe('import page copy', () => {
  it('describes the empty preview for every supported bill file format', () => {
    useLedgerStore.setState({ activeLedgerId: 'ledger-1', activeRole: 'owner' });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    const html = renderToStaticMarkup(
      createElement(QueryClientProvider, { client }, createElement(ImportPage)),
    );

    expect(html).toContain('上传账单文件后会在这里看到行级状态和错误原因。');
    expect(html).not.toContain('上传 CSV 后会在这里看到行级状态和错误原因。');
  });
});
