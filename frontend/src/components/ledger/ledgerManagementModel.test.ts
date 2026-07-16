import { describe, expect, it } from 'vitest';
import type { LedgerWithRole } from '../../api/ledger.api';
import { ApiError } from '../../api/client';
import {
  buildArchivedLedgerPath,
  getLedgerCapabilities,
  getLedgerErrorPresentation,
  ledgerNameSchema,
} from './ledgerManagementModel';

function ledger(
  role: LedgerWithRole['role'],
  status: LedgerWithRole['status'],
): LedgerWithRole {
  return {
    id: `ledger-${role}-${status}`,
    name: '共同生活',
    role,
    status,
    version: 3,
    member_count: 2,
    archived_at: status === 'archived' ? '2026-07-15T00:00:00Z' : null,
    archived_by_user_id: status === 'archived' ? 'user-owner' : null,
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-15T00:00:00Z',
  };
}

describe('Task50.5 ledger capability matrix', () => {
  it('keeps lifecycle and member management owner-only on active ledgers', () => {
    expect(getLedgerCapabilities(ledger('owner', 'active'))).toEqual({
      canSwitch: true,
      canViewHistory: false,
      canRename: true,
      canArchive: true,
      canRestore: false,
      canManageMembers: true,
      canLeave: false,
      canExport: true,
    });
  });

  it('lets active editors leave and export without exposing owner actions', () => {
    expect(getLedgerCapabilities(ledger('editor', 'active'))).toEqual({
      canSwitch: true,
      canViewHistory: false,
      canRename: false,
      canArchive: false,
      canRestore: false,
      canManageMembers: false,
      canLeave: true,
      canExport: true,
    });
  });

  it('makes archived ledgers read-only while preserving owner restore and permitted export', () => {
    expect(getLedgerCapabilities(ledger('owner', 'archived'))).toEqual({
      canSwitch: false,
      canViewHistory: true,
      canRename: false,
      canArchive: false,
      canRestore: true,
      canManageMembers: false,
      canLeave: false,
      canExport: true,
    });
    expect(getLedgerCapabilities(ledger('viewer', 'archived')).canExport).toBe(false);
  });
});

describe('Task50.5 ledger form and error contract', () => {
  it('accepts trimmed 1-60 character names and does not reject duplicate wording', () => {
    expect(ledgerNameSchema.parse({ name: '  共同生活  ' })).toEqual({ name: '共同生活' });
    expect(() => ledgerNameSchema.parse({ name: '同名账本' })).not.toThrow();
    expect(() => ledgerNameSchema.parse({ name: '' })).toThrow();
    expect(() => ledgerNameSchema.parse({ name: '账'.repeat(61) })).toThrow();
  });

  it('turns frozen conflict codes into actionable recovery copy', () => {
    expect(getLedgerErrorPresentation(
      new ApiError('LEDGER_VERSION_CONFLICT', 'conflict', 409),
    )).toEqual({
      message: '账本已在另一处更新，请刷新账本信息后重新确认。',
      recovery: 'refresh',
    });
    expect(getLedgerErrorPresentation(
      new ApiError('LEDGER_READY_IMPORT_EXISTS', 'ready', 409),
    )).toEqual({
      message: '有待确认导入阻止归档，请先完成或放弃该批次。',
      recovery: 'imports',
    });
    expect(getLedgerErrorPresentation(
      new ApiError('LEDGER_OWNER_TRANSFER_REQUIRED', 'owner', 409),
    )).toEqual({
      message: 'Owner 不能直接离开，请先移交所有权。',
      recovery: 'transfer-owner',
    });
  });
});

describe('Task50.5 archived viewing routes', () => {
  it('adds an explicit temporary archived context without replacing existing filters', () => {
    expect(buildArchivedLedgerPath('/transactions?month=2026-07&page=2', 'ledger archived'))
      .toBe('/transactions?month=2026-07&page=2&archived_ledger_id=ledger+archived');
  });

  it('supports dashboard history without persisting the archived ledger as active', () => {
    expect(buildArchivedLedgerPath('/', 'ledger-a'))
      .toBe('/?archived_ledger_id=ledger-a');
  });
});
