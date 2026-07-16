import { z } from 'zod';
import { ApiError } from '../../api/client';
import type { LedgerWithRole } from '../../api/ledger.api';

export const ledgerNameSchema = z.object({
  name: z.string().trim().min(1, '请输入账本名称').max(60, '账本名称最多 60 个字符'),
});

export type LedgerNameValues = z.infer<typeof ledgerNameSchema>;

export interface LedgerCapabilities {
  canSwitch: boolean;
  canViewHistory: boolean;
  canRename: boolean;
  canArchive: boolean;
  canRestore: boolean;
  canManageMembers: boolean;
  canLeave: boolean;
  canExport: boolean;
}

export type LedgerErrorRecovery =
  | 'none'
  | 'refresh'
  | 'imports'
  | 'transfer-owner'
  | 'ledger-list';

export interface LedgerErrorPresentation {
  message: string;
  recovery: LedgerErrorRecovery;
}

export function getLedgerCapabilities(ledger: LedgerWithRole): LedgerCapabilities {
  const isActive = ledger.status === 'active';
  const isOwner = ledger.role === 'owner';
  const canExport = ledger.role === 'owner' || ledger.role === 'editor';

  return {
    canSwitch: isActive,
    canViewHistory: !isActive,
    canRename: isActive && isOwner,
    canArchive: isActive && isOwner,
    canRestore: !isActive && isOwner,
    canManageMembers: isActive && isOwner,
    canLeave: isActive && !isOwner,
    canExport,
  };
}

export function getLedgerErrorPresentation(error: unknown): LedgerErrorPresentation {
  if (!(error instanceof ApiError)) {
    return {
      message: error instanceof Error ? error.message : '操作失败，请稍后重试。',
      recovery: 'none',
    };
  }

  const frozenErrors: Record<string, LedgerErrorPresentation> = {
    LEDGER_VERSION_CONFLICT: {
      message: '账本已在另一处更新，请刷新账本信息后重新确认。',
      recovery: 'refresh',
    },
    LEDGER_READY_IMPORT_EXISTS: {
      message: '有待确认导入阻止归档，请先完成或放弃该批次。',
      recovery: 'imports',
    },
    LEDGER_OWNER_TRANSFER_REQUIRED: {
      message: 'Owner 不能直接离开，请先移交所有权。',
      recovery: 'transfer-owner',
    },
    LEDGER_MEMBER_LIMIT_REACHED: {
      message: '当前账本已达到两人上限。',
      recovery: 'none',
    },
    LEDGER_ARCHIVED: {
      message: '该账本已归档，只能查看历史或由 Owner 恢复。',
      recovery: 'ledger-list',
    },
    LEDGER_ACCESS_DENIED: {
      message: '你已失去该账本访问权限，或当前角色不能执行此操作。',
      recovery: 'ledger-list',
    },
  };

  return frozenErrors[error.code] ?? {
    message: error.message || '操作失败，请稍后重试。',
    recovery: 'none',
  };
}

export function buildArchivedLedgerPath(path: string, ledgerId: string) {
  const url = new URL(path, 'https://ledger-two.local');
  url.searchParams.set('archived_ledger_id', ledgerId);
  return `${url.pathname}${url.search}`;
}
