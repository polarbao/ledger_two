import type { QueryClient, QueryKey } from '@tanstack/react-query';
import type { LedgerRole, LedgerWithRole } from '../../api/ledger.api';

export type LedgerContextStatus = 'validating' | 'active' | 'no-active' | 'error';
export type ActiveLedgerSelectionReason =
  | 'preferred-valid'
  | 'preferred-unavailable'
  | 'initial-selection'
  | 'no-active';

export interface ActiveLedgerSelection {
  ledger: LedgerWithRole | null;
  reason: ActiveLedgerSelectionReason;
  previousLedgerId: string | null;
}

export interface LedgerPersistenceSource {
  activeLedgerId: string | null;
  activeRole: LedgerRole | null;
  contextStatus: LedgerContextStatus;
  recentLedgerUsedAt: Record<string, number>;
}

export interface PersistedLedgerState {
  activeLedgerId: string | null;
  recentLedgerUsedAt: Record<string, number>;
}

export function sortLedgersByRecentUse(
  ledgers: LedgerWithRole[],
  recentLedgerUsedAt: Record<string, number>,
) {
  return ledgers
    .map((ledger, index) => ({
      ledger,
      index,
      usedAt: recentLedgerUsedAt[ledger.id] ?? Number.NEGATIVE_INFINITY,
    }))
    .sort((left, right) => right.usedAt - left.usedAt || left.index - right.index)
    .map(({ ledger }) => ledger);
}

export function selectActiveLedger(
  ledgers: LedgerWithRole[],
  preferredLedgerId: string | null,
  recentLedgerUsedAt: Record<string, number>,
): ActiveLedgerSelection {
  const activeLedgers = ledgers.filter((ledger) => ledger.status === 'active');
  const preferredLedger = activeLedgers.find((ledger) => ledger.id === preferredLedgerId);

  if (preferredLedger) {
    return {
      ledger: preferredLedger,
      reason: 'preferred-valid',
      previousLedgerId: preferredLedgerId,
    };
  }

  const fallbackLedger = sortLedgersByRecentUse(activeLedgers, recentLedgerUsedAt)[0] ?? null;
  if (!fallbackLedger) {
    return {
      ledger: null,
      reason: 'no-active',
      previousLedgerId: preferredLedgerId,
    };
  }

  return {
    ledger: fallbackLedger,
    reason: preferredLedgerId ? 'preferred-unavailable' : 'initial-selection',
    previousLedgerId: preferredLedgerId,
  };
}

export function selectPersistedLedgerState(
  state: LedgerPersistenceSource,
): PersistedLedgerState {
  return {
    activeLedgerId: state.activeLedgerId,
    recentLedgerUsedAt: state.recentLedgerUsedAt,
  };
}

export function migratePersistedLedgerState(persistedState: unknown): PersistedLedgerState {
  if (!persistedState || typeof persistedState !== 'object') {
    return { activeLedgerId: null, recentLedgerUsedAt: {} };
  }

  const candidate = persistedState as {
    activeLedgerId?: unknown;
    recentLedgerUsedAt?: unknown;
  };
  const recentLedgerUsedAt = Object.fromEntries(
    candidate.recentLedgerUsedAt && typeof candidate.recentLedgerUsedAt === 'object'
      ? Object.entries(candidate.recentLedgerUsedAt).filter(
          (entry): entry is [string, number] =>
            typeof entry[1] === 'number' && Number.isFinite(entry[1]),
        )
      : [],
  );

  return {
    activeLedgerId: typeof candidate.activeLedgerId === 'string'
      ? candidate.activeLedgerId
      : null,
    recentLedgerUsedAt,
  };
}

export function isLedgerScopedQueryKey(queryKey: QueryKey, ledgerId: string) {
  return queryKey.some((part) => part === ledgerId);
}

interface SwitchActiveLedgerOptions {
  queryClient: QueryClient;
  currentLedgerId: string | null;
  nextLedger: LedgerWithRole;
  commit: (nextLedger: LedgerWithRole) => void;
}

export async function switchActiveLedgerContext({
  queryClient,
  currentLedgerId,
  nextLedger,
  commit,
}: SwitchActiveLedgerOptions) {
  if (currentLedgerId === nextLedger.id) return;

  if (currentLedgerId) {
    const predicate = (query: { queryKey: QueryKey }) =>
      isLedgerScopedQueryKey(query.queryKey, currentLedgerId);
    await queryClient.cancelQueries({ predicate });
    await queryClient.invalidateQueries({ predicate, refetchType: 'none' });
  }

  commit(nextLedger);
}
