import { api, type ApiRequestOptions } from './client';

export type LedgerRole = 'owner' | 'editor' | 'viewer';
export type LedgerStatus = 'active' | 'archived';
export type LedgerListStatus = LedgerStatus | 'all';

export interface Ledger {
  id: string;
  name: string;
  role: LedgerRole;
  status: LedgerStatus;
  version: number;
  member_count: number;
  archived_at: string | null;
  archived_by_user_id: string | null;
  created_at: string;
  updated_at: string;
}

export type LedgerWithRole = Ledger;

export interface LedgerMember {
  user_id: string;
  username: string;
  role: LedgerRole;
  joined_at: string;
}

export interface LedgerMemberList {
  ledger: Ledger;
  members: LedgerMember[];
}

export interface AddLedgerMemberRequest {
  username: string;
  role: Exclude<LedgerRole, 'owner'>;
  acknowledge_history_visibility: true;
}

export interface UpdateLedgerMemberRoleRequest {
  role: Exclude<LedgerRole, 'owner'>;
}

export interface TransferLedgerOwnerRequest {
  acknowledge_permission_change: true;
}

export interface UnsettledBalanceSnapshot {
  from_user_id: string | null;
  to_user_id: string | null;
  amount_cents: number;
}

export interface LedgerArchivePreflight {
  ledger: Ledger;
  unsettled_balance: UnsettledBalanceSnapshot;
  ready_import_batch_count: number;
  can_archive: boolean;
  requires_unsettled_acknowledgement: boolean;
}

export interface ArchiveLedgerRequest {
  acknowledge_unsettled_balance: boolean;
}

export function formatLedgerETag(ledgerId: string, version: number): string {
  return `"ledger:${ledgerId}:v${version}"`;
}

function lifecycleMutationOptions(ledgerId: string, version: number): ApiRequestOptions {
  return {
    ledgerId,
    headers: { 'If-Match': formatLedgerETag(ledgerId, version) },
  };
}

export const ledgerApi = {
  listUserLedgers: (status: LedgerListStatus = 'active') =>
    api.get<Ledger[]>(`/api/ledgers?status=${status}`, { ledgerScope: 'none' }),

  createLedger: (data: { name: string }) =>
    api.post<Ledger>('/api/ledgers', data, { ledgerScope: 'none' }),

  getLedger: (ledgerId: string) =>
    api.get<Ledger>(`/api/ledgers/${ledgerId}`, { ledgerId }),

  renameLedger: (ledgerId: string, version: number, data: { name: string }) =>
    api.patch<Ledger>(`/api/ledgers/${ledgerId}`, data, lifecycleMutationOptions(ledgerId, version)),

  getArchivePreflight: (ledgerId: string) =>
    api.get<LedgerArchivePreflight>(`/api/ledgers/${ledgerId}/archive-preflight`, { ledgerId }),

  archiveLedger: (ledgerId: string, version: number, data: ArchiveLedgerRequest) =>
    api.post<Ledger>(`/api/ledgers/${ledgerId}/archive`, data, lifecycleMutationOptions(ledgerId, version)),

  restoreLedger: (ledgerId: string, version: number) =>
    api.post<Ledger>(`/api/ledgers/${ledgerId}/restore`, undefined, lifecycleMutationOptions(ledgerId, version)),

  getLedgerMembers: (ledgerId: string) =>
    api.get<LedgerMemberList>(`/api/ledgers/${ledgerId}/members`, { ledgerId }),

  addMember: (ledgerId: string, version: number, data: AddLedgerMemberRequest) =>
    api.post<LedgerMemberList>(
      `/api/ledgers/${ledgerId}/members`,
      data,
      lifecycleMutationOptions(ledgerId, version),
    ),

  updateMemberRole: (
    ledgerId: string,
    version: number,
    userId: string,
    data: UpdateLedgerMemberRoleRequest,
  ) =>
    api.patch<LedgerMemberList>(
      `/api/ledgers/${ledgerId}/members/${userId}`,
      data,
      lifecycleMutationOptions(ledgerId, version),
    ),

  removeMember: (ledgerId: string, version: number, userId: string) =>
    api.delete<LedgerMemberList>(
      `/api/ledgers/${ledgerId}/members/${userId}`,
      lifecycleMutationOptions(ledgerId, version),
    ),

  transferOwner: (
    ledgerId: string,
    version: number,
    userId: string,
    data: TransferLedgerOwnerRequest,
  ) =>
    api.post<LedgerMemberList>(
      `/api/ledgers/${ledgerId}/members/${userId}/transfer-owner`,
      data,
      lifecycleMutationOptions(ledgerId, version),
    ),

  leaveLedger: (ledgerId: string, version: number) =>
    api.post<null>(
      `/api/ledgers/${ledgerId}/leave`,
      undefined,
      lifecycleMutationOptions(ledgerId, version),
    ),
};
