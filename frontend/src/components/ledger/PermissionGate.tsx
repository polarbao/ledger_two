import type { ReactNode } from 'react';
import { useLedgerStore } from '../../stores/ledger.store';

export type LedgerRole = 'owner' | 'editor' | 'viewer';

interface PermissionGateProps {
  allow: LedgerRole[];
  children: ReactNode;
  fallback?: ReactNode;
}

export function useHasLedgerRole(allow: LedgerRole[]) {
  const activeRole = useLedgerStore((state) => state.activeRole);
  return !!activeRole && allow.includes(activeRole as LedgerRole);
}

export default function PermissionGate({ allow, children, fallback = null }: PermissionGateProps) {
  const allowed = useHasLedgerRole(allow);
  return allowed ? <>{children}</> : <>{fallback}</>;
}
