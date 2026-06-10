import { create } from 'zustand';

interface UIStore {
  currentMonth: string;
  addDrawerOpen: boolean;
  detailDrawerTransactionId: string | null;
  filterOpen: boolean;
  setCurrentMonth: (month: string) => void;
  setAddDrawerOpen: (open: boolean) => void;
  setDetailDrawerTransactionId: (id: string | null) => void;
  setFilterOpen: (open: boolean) => void;
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
  setCurrentMonth: (currentMonth) => set({ currentMonth }),
  setAddDrawerOpen: (addDrawerOpen) => set({ addDrawerOpen }),
  setDetailDrawerTransactionId: (detailDrawerTransactionId) => set({ detailDrawerTransactionId }),
  setFilterOpen: (filterOpen) => set({ filterOpen }),
}));
