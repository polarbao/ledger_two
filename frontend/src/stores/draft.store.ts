import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { migrateLegacyDrafts } from './draftLedgerModel';

export interface TransactionDraft {
  id: string; // 本地生成的唯一 ID
  ledgerId: string | null;
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
  clearLedgerDrafts: (ledgerId: string) => void;
}

interface PersistedDraftState {
  drafts: TransactionDraft[];
}

function readPersistedLedgerId() {
  if (typeof localStorage === 'undefined') return null;
  try {
    const value = JSON.parse(localStorage.getItem('ledger-storage') || 'null') as {
      state?: { activeLedgerId?: unknown };
    } | null;
    return typeof value?.state?.activeLedgerId === 'string'
      ? value.state.activeLedgerId
      : null;
  } catch {
    return null;
  }
}

export const useDraftStore = create<DraftState>()(
  persist<DraftState, [], [], PersistedDraftState>(
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
      clearLedgerDrafts: (ledgerId) => set((state) => ({
        drafts: state.drafts.filter((draft) => draft.ledgerId !== ledgerId),
      })),
    }),
    {
      name: 'ledger-two-drafts',
      storage: createJSONStorage(() => localStorage),
      version: 2,
      migrate: (persistedState) => {
        const candidate = persistedState as Partial<PersistedDraftState> | undefined;
        return {
          drafts: migrateLegacyDrafts(
            Array.isArray(candidate?.drafts) ? candidate.drafts : [],
            readPersistedLedgerId(),
          ),
        };
      },
      partialize: (state) => ({ drafts: state.drafts }),
    }
  )
);
