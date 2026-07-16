import { useEffect, useState, type ReactNode } from 'react';
import Button from '../ui/Button';
import BottomSheet from '../ui/BottomSheet';
import ConfirmDialog from '../ui/ConfirmDialog';

interface LedgerActionSurfaceProps {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  children?: ReactNode;
  tone?: 'primary' | 'danger';
  icon?: ReactNode;
  isConfirming?: boolean;
  confirmDisabled?: boolean;
  onConfirm: () => void;
  onClose: () => void;
}

function useMobileSurface() {
  const [isMobile, setIsMobile] = useState(
    () => typeof window !== 'undefined' && window.matchMedia('(max-width: 768px)').matches,
  );

  useEffect(() => {
    const media = window.matchMedia('(max-width: 768px)');
    const handleChange = () => setIsMobile(media.matches);
    handleChange();
    media.addEventListener('change', handleChange);
    return () => media.removeEventListener('change', handleChange);
  }, []);

  return isMobile;
}

export default function LedgerActionSurface({
  open,
  title,
  description,
  confirmLabel,
  children,
  tone = 'primary',
  icon,
  isConfirming = false,
  confirmDisabled = false,
  onConfirm,
  onClose,
}: LedgerActionSurfaceProps) {
  const isMobile = useMobileSurface();

  if (isMobile) {
    return (
      <BottomSheet
        open={open}
        title={title}
        description={description}
        closeOnBackdrop={false}
        onClose={onClose}
        footer={(
          <div className="ledger-action-surface__footer">
            <Button variant="secondary" disabled={isConfirming} onClick={onClose}>
              取消
            </Button>
            <Button
              variant={tone === 'danger' ? 'danger' : 'primary'}
              isLoading={isConfirming}
              disabled={confirmDisabled}
              onClick={onConfirm}
            >
              {confirmLabel}
            </Button>
          </div>
        )}
      >
        {children}
      </BottomSheet>
    );
  }

  return (
    <ConfirmDialog
      open={open}
      title={title}
      description={description}
      confirmLabel={confirmLabel}
      tone={tone}
      icon={icon}
      isConfirming={isConfirming}
      confirmDisabled={confirmDisabled}
      closeOnBackdrop={false}
      onConfirm={onConfirm}
      onClose={onClose}
    >
      {children}
    </ConfirmDialog>
  );
}
