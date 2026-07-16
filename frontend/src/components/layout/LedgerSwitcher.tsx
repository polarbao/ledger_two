import { useEffect, useMemo, useRef, useState } from 'react';
import { BookOpen, Check, ChevronDown, Settings } from 'lucide-react';
import type { LedgerWithRole } from '../../api/ledger.api';
import type { LedgerContextStatus } from './ledgerContextModel';
import { sortLedgersByRecentUse } from './ledgerContextModel';
import { getLedgerRoleLabel } from './appShellModel';
import StatusChip from '../ui/StatusChip';

interface LedgerSwitcherProps {
  ledgers: LedgerWithRole[];
  activeLedgerId: string | null;
  recentLedgerUsedAt: Record<string, number>;
  contextStatus: LedgerContextStatus;
  errorMessage: string | null;
  archivedCount: number;
  isSwitching: boolean;
  onSelect: (ledger: LedgerWithRole) => Promise<void>;
  onRetry: () => void;
  onManage: () => void;
}

export default function LedgerSwitcher({
  ledgers,
  activeLedgerId,
  recentLedgerUsedAt,
  contextStatus,
  errorMessage,
  archivedCount,
  isSwitching,
  onSelect,
  onRetry,
  onManage,
}: LedgerSwitcherProps) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const sortedLedgers = useMemo(
    () => sortLedgersByRecentUse(ledgers, recentLedgerUsedAt),
    [ledgers, recentLedgerUsedAt],
  );
  const activeLedger = sortedLedgers.find((ledger) => ledger.id === activeLedgerId) ?? null;
  const isDisabled = contextStatus !== 'active' || !activeLedger || isSwitching;

  useEffect(() => {
    if (!open) return undefined;

    const handlePointerDown = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      setOpen(false);
      triggerRef.current?.focus();
    };
    document.addEventListener('pointerdown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [open]);

  const triggerLabel = contextStatus === 'validating'
    ? '正在校验账本'
    : contextStatus === 'error'
      ? '账本列表加载失败'
      : activeLedger?.name || '暂无活跃账本';

  return (
    <div ref={rootRef} className="lt-ledger-switcher">
      <span className="lt-shell__field-label">当前账本</span>
      <button
        ref={triggerRef}
        type="button"
        className="lt-ledger-switcher__trigger"
        aria-haspopup="listbox"
        aria-expanded={open}
        disabled={isDisabled}
        title={activeLedger?.name || triggerLabel}
        onClick={() => setOpen((current) => !current)}
      >
        <span className="lt-ledger-switcher__trigger-copy">
          <strong>{triggerLabel}</strong>
          <span>{activeLedger ? getLedgerRoleLabel(activeLedger.role) : errorMessage || '需要创建或恢复账本'}</span>
        </span>
        <ChevronDown size={17} aria-hidden="true" />
      </button>

      {open ? (
        <div className="lt-ledger-switcher__menu">
          <div className="lt-ledger-switcher__list" role="listbox" aria-label="切换活跃账本">
            {sortedLedgers.map((ledger) => {
              const selected = ledger.id === activeLedgerId;
              return (
                <button
                  key={ledger.id}
                  type="button"
                  role="option"
                  aria-selected={selected}
                  className="lt-ledger-switcher__option"
                  disabled={isSwitching}
                  onClick={async () => {
                    await onSelect(ledger);
                    setOpen(false);
                    triggerRef.current?.focus();
                  }}
                >
                  <span className="lt-ledger-switcher__option-icon" aria-hidden="true">
                    {selected ? <Check size={16} /> : <BookOpen size={16} />}
                  </span>
                  <span className="lt-ledger-switcher__option-copy">
                    <strong>{ledger.name}</strong>
                    <span>{getLedgerRoleLabel(ledger.role)}{selected ? ' · 当前' : ''}</span>
                  </span>
                </button>
              );
            })}
          </div>
          <button
            type="button"
            className="lt-ledger-switcher__manage"
            onClick={() => {
              setOpen(false);
              onManage();
            }}
          >
            <Settings size={16} aria-hidden="true" />
            <span>管理账本</span>
            {archivedCount > 0 ? <StatusChip>{`已归档 ${archivedCount}`}</StatusChip> : null}
          </button>
        </div>
      ) : null}

      {contextStatus === 'error' ? (
        <button type="button" className="lt-ledger-switcher__retry" onClick={onRetry}>
          重试读取账本
        </button>
      ) : null}
    </div>
  );
}
