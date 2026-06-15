import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface LedgerState {
  activeLedgerId: string | null;
  activeRole: string | null;
  setActiveLedger: (id: string, role: string) => void;
  clearActiveLedger: () => void;
}

export const useLedgerStore = create<LedgerState>()(
  persist(
    (set) => ({
      activeLedgerId: null,
      activeRole: null,
      setActiveLedger: (id, role) => set({ activeLedgerId: id, activeRole: role }),
      clearActiveLedger: () => set({ activeLedgerId: null, activeRole: null }),
    }),
    {
      name: 'ledger-storage',
    }
  )
);
