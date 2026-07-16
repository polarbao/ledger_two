import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { LedgerRole, LedgerWithRole } from '../api/ledger.api';
import {
  migratePersistedLedgerState,
  selectActiveLedger,
  selectPersistedLedgerState,
  type LedgerContextStatus,
} from '../components/layout/ledgerContextModel';

export interface LedgerContextNotice {
  kind: 'fallback' | 'no-active';
  previousLedgerId: string | null;
  nextLedgerName?: string;
}

export interface LedgerState {
  activeLedgerId: string | null;
  activeRole: LedgerRole | null;
  archivedViewingLedger: LedgerWithRole | null;
  recentLedgerUsedAt: Record<string, number>;
  contextStatus: LedgerContextStatus;
  contextNotice: LedgerContextNotice | null;
  validationError: string | null;
  setActiveLedger: (id: string, role: LedgerRole, usedAt?: number) => void;
  enterArchivedLedgerView: (ledger: LedgerWithRole) => void;
  exitArchivedLedgerView: () => void;
  reconcileActiveLedgers: (ledgers: LedgerWithRole[], usedAt?: number) => void;
  beginLedgerValidation: () => void;
  failLedgerValidation: (message: string) => void;
  clearContextNotice: () => void;
  clearActiveLedger: () => void;
}

export const useLedgerStore = create<LedgerState>()(
  persist<LedgerState, [], [], ReturnType<typeof selectPersistedLedgerState>>(
    (set) => ({
      activeLedgerId: null,
      activeRole: null,
      archivedViewingLedger: null,
      recentLedgerUsedAt: {},
      contextStatus: 'validating',
      contextNotice: null,
      validationError: null,
      setActiveLedger: (id, role, usedAt = Date.now()) => set((state) => ({
        activeLedgerId: id,
        activeRole: role,
        recentLedgerUsedAt: {
          ...state.recentLedgerUsedAt,
          [id]: usedAt,
        },
        contextStatus: 'active',
        contextNotice: null,
        validationError: null,
        archivedViewingLedger: null,
      })),
      enterArchivedLedgerView: (ledger) => set({
        archivedViewingLedger: ledger.status === 'archived' ? ledger : null,
      }),
      exitArchivedLedgerView: () => set({ archivedViewingLedger: null }),
      reconcileActiveLedgers: (ledgers, usedAt = Date.now()) => set((state) => {
        const selection = selectActiveLedger(
          ledgers,
          state.activeLedgerId,
          state.recentLedgerUsedAt,
        );
        if (!selection.ledger) {
          return {
            activeLedgerId: null,
            activeRole: null,
            contextStatus: 'no-active' as const,
            contextNotice: selection.previousLedgerId
              ? {
                  kind: 'no-active' as const,
                  previousLedgerId: selection.previousLedgerId,
                }
              : state.contextNotice?.kind === 'no-active'
                ? state.contextNotice
                : null,
            validationError: null,
          };
        }

        return {
          activeLedgerId: selection.ledger.id,
          activeRole: selection.ledger.role,
          recentLedgerUsedAt: {
            ...state.recentLedgerUsedAt,
            [selection.ledger.id]: usedAt,
          },
          contextStatus: 'active' as const,
          contextNotice: selection.reason === 'preferred-unavailable'
            ? {
                kind: 'fallback' as const,
                previousLedgerId: selection.previousLedgerId,
                nextLedgerName: selection.ledger.name,
              }
            : state.contextNotice,
          validationError: null,
        };
      }),
      beginLedgerValidation: () => set({
        activeRole: null,
        contextStatus: 'validating',
        contextNotice: null,
        validationError: null,
      }),
      failLedgerValidation: (message) => set({
        activeRole: null,
        contextStatus: 'error',
        validationError: message,
      }),
      clearContextNotice: () => set({ contextNotice: null }),
      clearActiveLedger: () => set({
        activeLedgerId: null,
        activeRole: null,
        archivedViewingLedger: null,
        contextStatus: 'no-active',
        contextNotice: null,
        validationError: null,
      }),
    }),
    {
      name: 'ledger-storage',
      version: 2,
      migrate: migratePersistedLedgerState,
      partialize: (state) => selectPersistedLedgerState(state),
    }
  )
);
