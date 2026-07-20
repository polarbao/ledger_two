import { describe, expect, it } from 'vitest';
import { ApiError } from '../api/client';
import type { ImportPreviewBatch, ImportPreviewRow } from '../types/imports';
import {
  buildImportCommitSummary,
  canUseImportWorkspace,
  defaultImportRowFilter,
  filterImportRowsByClassification,
  filterImportRows,
  getImportFileAccept,
  getImportSourceDescription,
  resolveImportErrorMessage,
  normalizeImportList,
  selectableImportRows,
  validateImportFile,
} from './importPageState';

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
  it('allows owners and editors to import while keeping viewers read-only', () => {
    expect(canUseImportWorkspace('owner')).toBe(true);
    expect(canUseImportWorkspace('editor')).toBe(true);
    expect(canUseImportWorkspace('viewer')).toBe(false);
    expect(canUseImportWorkspace(null)).toBe(false);
  });

  it('normalizes empty API collections before rendering import controls', () => {
    expect(normalizeImportList(null)).toEqual([]);
    expect(normalizeImportList(undefined)).toEqual([]);
    expect(normalizeImportList([{ id: 'category-1' }])).toEqual([{ id: 'category-1' }]);
  });

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

  it('filters and prioritizes rows that need user action', () => {
    const rows = [
      createRow(),
      createRow({ id: 'row-2', row_number: 2, duplicate_status: 'suspicious' }),
      createRow({ id: 'row-3', row_number: 3, duplicate_status: 'invalid', row_status: 'failed' }),
      createRow({ id: 'row-4', row_number: 4, duplicate_status: 'invalid', row_status: 'skipped', target_transaction_type: 'skipped' }),
    ];

    expect(defaultImportRowFilter(rows)).toBe('needs_attention');
    expect(filterImportRows(rows, 'needs_attention').map((row) => row.row_number)).toEqual([2, 3]);
    expect(filterImportRows(rows, 'invalid').map((row) => row.row_number)).toEqual([3, 4]);
    expect(filterImportRows(rows, 'skipped').map((row) => row.row_number)).toEqual([4]);
  });

  it('keeps duplicate rows available as an explicit review filter', () => {
    const rows = [
      createRow(),
      createRow({ id: 'row-2', row_number: 2, duplicate_status: 'duplicate', row_status: 'skipped' }),
    ];

    expect(filterImportRows(rows, 'duplicate').map((row) => row.row_number)).toEqual([2]);
  });

  it('includes the failing row number in import errors', () => {
    const error = new ApiError('IMPORT_ROW_INVALID', '存在未跳过的无效导入行', 400, {
      row_id: 'row-5',
      row_number: 5,
    });

    expect(resolveImportErrorMessage(error, 'fallback')).toBe('第 5 行：存在未跳过的无效导入行');
  });

  it('accepts xlsx only for WeChat and keeps Alipay and generic CSV-only', () => {
    expect(validateImportFile('wechat', 'wechat.xlsx')).toBeNull();
    expect(validateImportFile('alipay', 'alipay.XLSX')).toContain('支付宝');
    expect(validateImportFile('alipay', 'alipay.XLSX')).toContain('CSV');
    expect(validateImportFile('generic', 'template.xlsx')).toContain('通用模板');
    expect(validateImportFile('generic', 'template.csv')).toBeNull();
    expect(validateImportFile('wechat', 'wechat.xls')).toContain('CSV 或 XLSX');
  });

  it('rejects xlsx when the current runtime gate is disabled', () => {
    expect(validateImportFile('wechat', 'wechat.xlsx', false)).toContain('暂未开启 XLSX');
    expect(validateImportFile('alipay', 'alipay.xlsx', false)).toContain('仅支持 CSV');
    expect(validateImportFile('wechat', 'wechat.csv', false)).toBeNull();
  });

  it('keeps the file picker and source copy aligned with the support matrix', () => {
    expect(getImportFileAccept('wechat', true)).toBe('.csv,.xlsx');
    expect(getImportFileAccept('wechat', false)).toBe('.csv');
    expect(getImportFileAccept('alipay', true)).toBe('.csv');
    expect(getImportFileAccept('generic', true)).toBe('.csv');
    expect(getImportSourceDescription('alipay', true)).toContain('支付宝当前导出的 CSV');
    expect(getImportSourceDescription('wechat', false)).toContain('仅开放 CSV');
  });

  it('filters persisted classification states without replacing duplicate review filters', () => {
    const rows = [
      createRow({
        classification: {
          status: 'auto_selected',
          confidence: 'high',
          source: 'user_rule',
          matched_rule_ids: ['rule-1'],
          suggested_tag_ids: [],
        },
      }),
      createRow({
        id: 'row-2',
        classification: {
          status: 'suggested',
          confidence: 'medium',
          source: 'builtin',
          matched_rule_ids: [],
          suggested_category_id: 'category-1',
          suggested_tag_ids: [],
        },
      }),
      createRow({
        id: 'row-3',
        classification: {
          status: 'conflict',
          confidence: 'none',
          matched_rule_ids: ['rule-1', 'rule-2'],
          suggested_tag_ids: [],
        },
      }),
    ];

    expect(filterImportRowsByClassification(rows, 'suggested').map((row) => row.id)).toEqual(['row-2']);
    expect(filterImportRowsByClassification(rows, 'conflict').map((row) => row.id)).toEqual(['row-3']);
    expect(filterImportRowsByClassification(rows, 'all')).toHaveLength(3);
  });

  it('selects only mutable preview rows for explicit bulk actions', () => {
    const rows = [
      createRow({ classification: undefined }),
      createRow({ id: 'row-2', row_status: 'imported' }),
      createRow({ id: 'row-3', duplicate_status: 'duplicate', row_status: 'skipped' }),
      createRow({ id: 'row-4', duplicate_status: 'invalid', row_status: 'failed' }),
    ];

    expect(selectableImportRows(rows).map((row) => row.id)).toEqual(['row-1']);
  });
});
