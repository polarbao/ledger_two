import { useLedgerStore } from '../../stores/ledger.store';

export type LedgerRole = 'owner' | 'editor' | 'viewer';

export function useHasLedgerRole(allow: LedgerRole[]) {
  const activeRole = useLedgerStore((state) => state.activeRole);
  return !!activeRole && allow.includes(activeRole as LedgerRole);
}
