import { api } from './client';

export interface Ledger {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface LedgerWithRole extends Ledger {
  role: string; // owner, editor, viewer
}

export interface LedgerMember {
  user_id: string;
  username: string;
  role: string;
}

export const ledgerApi = {
  // 获取当前用户的所有账本
  listUserLedgers: () => api.get<LedgerWithRole[]>('/api/ledgers'),

  // 创建账本
  createLedger: (data: { name: string }) => api.post<Ledger>('/api/ledgers', data),

  // 获取特定账本成员
  getLedgerMembers: (ledgerId: string) => api.get<LedgerMember[]>(`/api/ledgers/${ledgerId}/members`),

  // 添加成员
  addMember: (ledgerId: string, data: { username: string; role: string }) => 
    api.post<null>(`/api/ledgers/${ledgerId}/members`, data),

  // 更新成员角色
  updateMemberRole: (ledgerId: string, userId: string, data: { role: string }) => 
    api.put<null>(`/api/ledgers/${ledgerId}/members/${userId}`, data),

  // 移除成员
  removeMember: (ledgerId: string, userId: string) => 
    api.delete<null>(`/api/ledgers/${ledgerId}/members/${userId}`),
};
