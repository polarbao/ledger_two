import { ArrowLeft, Archive, RotateCcw } from 'lucide-react';
import type { LedgerWithRole } from '../../api/ledger.api';
import Button from '../ui/Button';
import StatusChip from '../ui/StatusChip';

interface ArchivedLedgerBannerProps {
  ledger: LedgerWithRole;
  isRestoring: boolean;
  errorMessage?: string | null;
  onRestore: () => void;
  onReturn: () => void;
}

export default function ArchivedLedgerBanner({
  ledger,
  isRestoring,
  errorMessage,
  onRestore,
  onReturn,
}: ArchivedLedgerBannerProps) {
  return (
    <section className="archived-ledger-banner" aria-label="归档账本只读状态">
      <Archive size={19} aria-hidden="true" />
      <div className="archived-ledger-banner__copy">
        <div>
          <strong>正在查看已归档账本</strong>
          <StatusChip tone="warning">历史数据不会被修改</StatusChip>
        </div>
        <span>{ledger.name} · {ledger.role === 'owner' ? 'Owner' : ledger.role === 'editor' ? 'Editor' : 'Viewer'} · 全员只读</span>
        {errorMessage ? <small role="alert">{errorMessage}</small> : null}
      </div>
      <div className="archived-ledger-banner__actions">
        <Button variant="secondary" startIcon={<ArrowLeft size={16} />} onClick={onReturn}>
          返回活跃账本
        </Button>
        {ledger.role === 'owner' ? (
          <Button
            variant="primary"
            startIcon={<RotateCcw size={16} />}
            isLoading={isRestoring}
            onClick={onRestore}
          >
            恢复账本
          </Button>
        ) : null}
      </div>
    </section>
  );
}
