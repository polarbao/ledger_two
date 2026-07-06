import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

export interface TransactionDraft {
  id: string; // 本地生成的唯一 ID
  formValues: {
    type: 'expense' | 'income' | 'shared_expense';
    amount: string;
    title?: string;
    category_id?: string | null;
    account_id?: string | null;
    tag_names?: string;
    payer_user_id: string;
    split_method: 'equal' | 'payer_only';
    occurred_at: string;
    note?: string;
    visibility: 'private' | 'partner_readable';
    attachment_paths?: string[];
  };
  createdAt: string;
}

interface DraftState {
  drafts: TransactionDraft[];
  addDraft: (draft: TransactionDraft) => void;
  removeDraft: (id: string) => void;
  updateDraft: (id: string, updatedDraft: TransactionDraft) => void;
  clearDrafts: () => void;
}

export const useDraftStore = create<DraftState>()(
  persist(
    (set) => ({
      drafts: [],
      addDraft: (draft) => set((state) => ({ drafts: [draft, ...state.drafts] })),
      removeDraft: (id) =>
        set((state) => ({
          drafts: state.drafts.filter((d) => d.id !== id),
        })),
      updateDraft: (id, updatedDraft) =>
        set((state) => ({
          drafts: state.drafts.map((d) => (d.id === id ? updatedDraft : d)),
        })),
      clearDrafts: () => set({ drafts: [] }),
    }),
    {
      name: 'ledger-two-drafts',
      storage: createJSONStorage(() => localStorage),
    }
  )
);
