import { useLedgerStore } from '../../stores/ledger.store';

export function useLedgerContext() {
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const activeRole = useLedgerStore((state) => state.activeRole);
  const archivedViewingLedger = useLedgerStore((state) => state.archivedViewingLedger);

  return {
    ledgerId: archivedViewingLedger?.id ?? activeLedgerId,
    role: archivedViewingLedger?.role ?? activeRole,
    status: archivedViewingLedger?.status ?? (activeLedgerId ? 'active' : null),
    isArchivedView: archivedViewingLedger?.status === 'archived',
    archivedViewingLedger,
    activeLedgerId,
    activeRole,
  } as const;
}
