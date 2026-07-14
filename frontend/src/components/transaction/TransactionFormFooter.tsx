import { RotateCw, Save } from 'lucide-react';
import Button from '../ui/Button';

export type TransactionFormMode = 'default' | 'copy' | 'draft' | 'edit' | 'offline';

interface TransactionFormFooterProps {
  mode: TransactionFormMode;
  isPending: boolean;
  activeAction: 'close' | 'continue';
  submitDisabled?: boolean;
  onCancel: () => void;
  onContinue: () => void;
  onPrimary: () => void;
}

const PRIMARY_LABELS: Record<TransactionFormMode, string> = {
  default: '保存账单',
  copy: '保存为新账单',
  draft: '提交正式账单',
  edit: '保存修改',
  offline: '保存为离线草稿',
};

export default function TransactionFormFooter({
  mode,
  isPending,
  activeAction,
  submitDisabled = false,
  onCancel,
  onContinue,
  onPrimary,
}: TransactionFormFooterProps) {
  return (
    <footer className="lt-entry-footer">
      <Button
        className="lt-entry-footer__cancel"
        variant="ghost"
        onClick={onCancel}
        disabled={isPending}
      >
        取消
      </Button>
      <Button
        className="lt-entry-footer__continue"
        variant="secondary"
        startIcon={<RotateCw size={17} />}
        onClick={onContinue}
        isLoading={isPending && activeAction === 'continue'}
        disabled={mode === 'offline' || mode === 'edit' || isPending}
      >
        保存并继续
      </Button>
      <Button
        className="lt-entry-footer__primary"
        type="submit"
        variant="primary"
        startIcon={<Save size={17} />}
        onClick={onPrimary}
        isLoading={isPending && activeAction === 'close'}
        disabled={isPending || submitDisabled}
      >
        {PRIMARY_LABELS[mode]}
      </Button>
    </footer>
  );
}
