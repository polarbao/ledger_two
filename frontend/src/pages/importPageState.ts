import { ApiError } from '../api/client';
import type { ImportPreviewBatch } from '../types/imports';

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
