import { describe, expect, it } from 'vitest';
import { safetyApi } from './safety.api';

describe('Task50.3C instance operation API contract', () => {
  it('builds a backup download URL from a managed relative backup key', () => {
    expect(safetyApi.backupDownloadUrl('manual/backup_20260716.db'))
      .toBe('/api/admin/backups/manual/backup_20260716.db');
  });

  it('keeps nested backup key segments instead of encoding their separators', () => {
    expect(safetyApi.backupDownloadUrl('manual/daily/pre_restore.db'))
      .toBe('/api/admin/backups/manual/daily/pre_restore.db');
  });
});
