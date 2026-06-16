import { create } from 'zustand';
import type { TransactionResponse } from '../types/transaction';

interface UIStore {
  currentMonth: string;
  addDrawerOpen: boolean;
  detailDrawerTransactionId: string | null;
  filterOpen: boolean;
  copySourceTransaction: TransactionResponse | null;
  editingDraftId: string | null;
  isOffline: boolean;
  setCurrentMonth: (month: string) => void;
  setAddDrawerOpen: (open: boolean) => void;
  setDetailDrawerTransactionId: (id: string | null) => void;
  setFilterOpen: (open: boolean) => void;
  setCopySourceTransaction: (tx: TransactionResponse | null) => void;
  setEditingDraftId: (id: string | null) => void;
  setIsOffline: (offline: boolean) => void;
}

const getInitialMonth = () => {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  return `${year}-${month}`;
};

export const useUIStore = create<UIStore>((set) => ({
  currentMonth: getInitialMonth(),
  addDrawerOpen: false,
  detailDrawerTransactionId: null,
  filterOpen: false,
  copySourceTransaction: null,
  editingDraftId: null,
  isOffline: !navigator.onLine,
  setCurrentMonth: (currentMonth) => set({ currentMonth }),
  setAddDrawerOpen: (addDrawerOpen) => set({ addDrawerOpen }),
  setDetailDrawerTransactionId: (detailDrawerTransactionId) => set({ detailDrawerTransactionId }),
  setFilterOpen: (filterOpen) => set({ filterOpen }),
  setCopySourceTransaction: (copySourceTransaction) => set({ copySourceTransaction }),
  setEditingDraftId: (editingDraftId) => set({ editingDraftId }),
  setIsOffline: (isOffline) => set({ isOffline }),
}));
