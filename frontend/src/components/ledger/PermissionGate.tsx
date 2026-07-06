import type { ReactNode } from 'react';
import { useHasLedgerRole, type LedgerRole } from './useLedgerPermission';

interface PermissionGateProps {
  allow: LedgerRole[];
  children: ReactNode;
  fallback?: ReactNode;
}

export default function PermissionGate({ allow, children, fallback = null }: PermissionGateProps) {
  const allowed = useHasLedgerRole(allow);
  return allowed ? <>{children}</> : <>{fallback}</>;
}
