import { describe, expect, it } from 'vitest';
import { ApiError } from '../api/client';
import type { ImportPreviewBatch, ImportPreviewRow } from '../types/imports';
import { buildImportCommitSummary, resolveImportErrorMessage, validateImportFile } from './importPageState';

const createRow = (overrides: Partial<ImportPreviewRow> = {}): ImportPreviewRow => ({
  id: 'row-1',
  batch_id: 'batch-1',
  row_number: 1,
  title: '午餐',
  merchant: '快餐店',
  amount_cents: 3200,
  direction: 'expense',
  target_transaction_type: 'expense',
  duplicate_status: 'new',
  row_status: 'pending',
  ...overrides,
});

const createBatch = (rows: ImportPreviewRow[]): ImportPreviewBatch => ({
  id: 'batch-1',
  ledger_id: 'ledger-1',
  source_type: 'generic',
  file_format: 'csv',
  parser_metadata: {
    parser_version: 'tabular-v1',
    header_row_number: 1,
    parsed_rows: rows.length,
    max_columns: 4,
  },
  filename: 'test.csv',
  file_sha256: 'hash',
  status: 'ready',
  total_rows: rows.length,
  new_rows: 0,
  duplicate_rows: 0,
  suspicious_rows: 0,
  invalid_rows: 0,
  imported_rows: 0,
  skipped_rows: 0,
  failed_rows: 0,
  created_by_user_id: 'owner-1',
  created_at: '2026-07-09T00:00:00Z',
  updated_at: '2026-07-09T00:00:00Z',
  rows,
});

describe('import page state', () => {
  it('blocks commit until suspicious and invalid rows are handled', () => {
    const summary = buildImportCommitSummary(createBatch([
      createRow(),
      createRow({ id: 'row-2', row_number: 2, duplicate_status: 'suspicious' }),
      createRow({ id: 'row-3', row_number: 3, duplicate_status: 'invalid', row_status: 'failed' }),
      createRow({ id: 'row-4', row_number: 4, duplicate_status: 'duplicate', row_status: 'skipped', target_transaction_type: 'skipped' }),
    ]));

    expect(summary).toEqual({
      importableCount: 1,
      skippedCount: 1,
      unconfirmedSuspiciousCount: 1,
      invalidOpenCount: 1,
      blockingCount: 2,
    });
  });

  it('allows adjusted suspicious rows and skipped invalid rows', () => {
    const summary = buildImportCommitSummary(createBatch([
      createRow({ duplicate_status: 'suspicious', row_status: 'adjusted' }),
      createRow({ id: 'row-2', row_number: 2, duplicate_status: 'invalid', row_status: 'skipped', target_transaction_type: 'skipped' }),
    ]));

    expect(summary.importableCount).toBe(1);
    expect(summary.skippedCount).toBe(1);
    expect(summary.blockingCount).toBe(0);
  });

  it('includes the failing row number in import errors', () => {
    const error = new ApiError('IMPORT_ROW_INVALID', '存在未跳过的无效导入行', 400, {
      row_id: 'row-5',
      row_number: 5,
    });

    expect(resolveImportErrorMessage(error, 'fallback')).toBe('第 5 行：存在未跳过的无效导入行');
  });

  it('accepts xlsx for WeChat and Alipay but keeps generic CSV-only', () => {
    expect(validateImportFile('wechat', 'wechat.xlsx')).toBeNull();
    expect(validateImportFile('alipay', 'alipay.XLSX')).toBeNull();
    expect(validateImportFile('generic', 'template.xlsx')).toContain('通用模板');
    expect(validateImportFile('generic', 'template.csv')).toBeNull();
    expect(validateImportFile('wechat', 'wechat.xls')).toContain('CSV 或 XLSX');
  });
});
