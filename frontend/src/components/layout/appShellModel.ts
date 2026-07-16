export type AppPrimaryNavigationId = 'dashboard' | 'transactions' | 'analytics' | 'settlement' | 'settings';
export type AppToolNavigationId = 'import' | 'recurring';

export interface AppNavigationItem<TId extends string> {
  id: TId;
  label: string;
  path: string;
}

export const APP_PRIMARY_NAV_ITEMS: readonly AppNavigationItem<AppPrimaryNavigationId>[] = [
  { id: 'dashboard', label: '首页', path: '/' },
  { id: 'transactions', label: '流水', path: '/transactions' },
  { id: 'analytics', label: '分析', path: '/analytics' },
  { id: 'settlement', label: '结算', path: '/settlement' },
  { id: 'settings', label: '设置', path: '/settings' },
];

export const APP_TOOL_NAV_ITEMS: readonly AppNavigationItem<AppToolNavigationId>[] = [
  { id: 'import', label: '账单导入', path: '/import' },
  { id: 'recurring', label: '周期规则', path: '/recurring-rules' },
];

const LEDGER_ROLE_LABELS: Record<string, string> = {
  owner: '所有者',
  editor: '可编辑',
  viewer: '只读',
};

export function isAppRouteActive(pathname: string, itemPath: string) {
  if (itemPath === '/') return pathname === '/';
  return pathname === itemPath || pathname.startsWith(`${itemPath}/`);
}

export function getLedgerRoleLabel(role: string | null | undefined) {
  if (!role) return '角色未知';
  return LEDGER_ROLE_LABELS[role] ?? role;
}

export function canCreateTransaction(role: string | null | undefined) {
  return role === 'owner' || role === 'editor';
}

export function shouldShowQuickRecordAction(pathname: string) {
  return !isAppRouteActive(pathname, '/import');
}

export function isLedgerManagementRoute(pathname: string) {
  return pathname === '/settings/ledgers' || pathname.startsWith('/settings/ledgers/');
}

export function isArchivedHistoryRoute(pathname: string) {
  return pathname === '/'
    || pathname === '/transactions'
    || pathname === '/analytics'
    || pathname === '/settlement';
}

export function isArchivedContextCleanupPending(
  archivedViewingLedgerId: string | null | undefined,
  archivedRequestedId: string | null | undefined,
) {
  return Boolean(archivedViewingLedgerId && !archivedRequestedId);
}

export function buildShellNavigationPath(
  itemPath: string,
  archivedLedgerId: string | null,
) {
  if (!archivedLedgerId) return itemPath;
  if (itemPath === '/settings') return `/settings/ledgers/${encodeURIComponent(archivedLedgerId)}`;
  if (!isArchivedHistoryRoute(itemPath)) return itemPath;
  const separator = itemPath.includes('?') ? '&' : '?';
  return `${itemPath}${separator}archived_ledger_id=${encodeURIComponent(archivedLedgerId)}`;
}
