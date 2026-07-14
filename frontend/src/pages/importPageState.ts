import { ApiError } from '../api/client';
import type { ImportPreviewBatch, ImportPreviewRow, ImportSourceType } from '../types/imports';

export type ImportRowFilter = 'all' | 'new' | 'needs_attention' | 'invalid' | 'suspicious' | 'skipped';

export const IMPORT_ROW_FILTER_LABELS: Record<ImportRowFilter, string> = {
  all: '全部流水',
  new: '新增流水',
  needs_attention: '需要处理',
  invalid: '无效行',
  suspicious: '疑似重复',
  skipped: '已跳过',
};

export function validateImportFile(sourceType: ImportSourceType, filename: string, xlsxEnabled = true) {
  const lowerName = filename.trim().toLowerCase();
  if (lowerName.endsWith('.csv')) {
    return null;
  }
  if (lowerName.endsWith('.xlsx')) {
    if (sourceType === 'alipay') {
      return '支付宝当前导出的账单仅支持 CSV 文件';
    }
    if (sourceType === 'generic') {
      return '通用模板当前仅支持 CSV 文件';
    }
    return xlsxEnabled ? null : '当前环境暂未开启 XLSX 导入，请改用 CSV 文件';
  }
  if (sourceType === 'alipay') return '支付宝当前导出的账单仅支持 CSV 文件';
  if (sourceType === 'generic') return '通用模板当前仅支持 CSV 文件';
  return '微信账单仅支持 CSV 或 XLSX 文件';
}

export function getImportFileAccept(sourceType: ImportSourceType, xlsxEnabled: boolean) {
  return sourceType === 'wechat' && xlsxEnabled ? '.csv,.xlsx' : '.csv';
}

export function getImportSourceDescription(sourceType: ImportSourceType, xlsxEnabled: boolean) {
  if (sourceType === 'wechat') {
    return xlsxEnabled ? '微信支付导出的 CSV 或 XLSX' : '当前环境微信账单仅开放 CSV';
  }
  if (sourceType === 'alipay') return '支付宝当前导出的 CSV';
  return 'LedgerTwo 标准 CSV';
}

export function buildImportCommitSummary(batch: ImportPreviewBatch | null) {
  if (!batch) {
    return {
      importableCount: 0,
      skippedCount: 0,
      unconfirmedSuspiciousCount: 0,
      invalidOpenCount: 0,
      blockingCount: 0,
    };
  }

  const importableCount = batch.rows.filter((row) =>
    row.row_status !== 'skipped' &&
    row.row_status !== 'failed' &&
    row.target_transaction_type !== 'skipped' &&
    row.duplicate_status !== 'duplicate' &&
    row.duplicate_status !== 'invalid' &&
    (row.duplicate_status !== 'suspicious' || row.row_status === 'adjusted')
  ).length;
  const skippedCount = batch.rows.filter(
    (row) => row.row_status === 'skipped' || row.target_transaction_type === 'skipped'
  ).length;
  const unconfirmedSuspiciousCount = batch.rows.filter(
    (row) => row.duplicate_status === 'suspicious' && row.row_status === 'pending'
  ).length;
  const invalidOpenCount = batch.rows.filter(
    (row) => row.duplicate_status === 'invalid' && row.row_status !== 'skipped'
  ).length;

  return {
    importableCount,
    skippedCount,
    unconfirmedSuspiciousCount,
    invalidOpenCount,
    blockingCount: unconfirmedSuspiciousCount + invalidOpenCount,
  };
}

export function importRowMatchesFilter(row: ImportPreviewRow, filter: ImportRowFilter) {
  switch (filter) {
    case 'new':
      return row.duplicate_status === 'new' && row.row_status !== 'skipped';
    case 'needs_attention':
      return (
        row.duplicate_status === 'invalid' && row.row_status !== 'skipped'
      ) || (
        row.duplicate_status === 'suspicious' && row.row_status === 'pending'
      );
    case 'invalid':
      return row.duplicate_status === 'invalid';
    case 'suspicious':
      return row.duplicate_status === 'suspicious';
    case 'skipped':
      return row.row_status === 'skipped' || row.target_transaction_type === 'skipped';
    default:
      return true;
  }
}

export function filterImportRows(rows: ImportPreviewRow[], filter: ImportRowFilter) {
  return rows.filter((row) => importRowMatchesFilter(row, filter));
}

export function defaultImportRowFilter(rows: ImportPreviewRow[]): ImportRowFilter {
  if (rows.some((row) => importRowMatchesFilter(row, 'needs_attention'))) {
    return 'needs_attention';
  }
  return 'all';
}

export function resolveImportErrorMessage(err: unknown, fallback: string) {
  if (!(err instanceof ApiError)) {
    return fallback;
  }
  const details = err.details;
  if (isRecord(details) && typeof details.row_number === 'number') {
    return `第 ${details.row_number} 行：${err.message}`;
  }
  return err.message;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}
