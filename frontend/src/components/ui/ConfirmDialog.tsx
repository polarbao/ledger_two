import { useId, useRef, type MouseEvent, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import Button from './Button';
import useModalSurface from './useModalSurface';

export interface ConfirmDialogProps {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  cancelLabel?: string;
  tone?: 'primary' | 'danger';
  icon?: ReactNode;
  children?: ReactNode;
  isConfirming?: boolean;
  confirmDisabled?: boolean;
  closeOnBackdrop?: boolean;
  onConfirm: () => void;
  onClose: () => void;
}

export default function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  cancelLabel = '取消',
  tone = 'primary',
  icon,
  children,
  isConfirming = false,
  confirmDisabled = false,
  closeOnBackdrop = false,
  onConfirm,
  onClose,
}: ConfirmDialogProps) {
  const titleId = useId();
  const descriptionId = useId();
  const surfaceRef = useRef<HTMLElement>(null);
  const cancelRef = useRef<HTMLButtonElement>(null);

  useModalSurface({ open, onClose, surfaceRef, initialFocusRef: cancelRef });

  if (!open) return null;

  const handleBackdropClick = (event: MouseEvent<HTMLDivElement>) => {
    if (closeOnBackdrop && event.target === event.currentTarget) onClose();
  };

  const dialog = (
    <div className="ui-overlay" onMouseDown={handleBackdropClick}>
      <section
        ref={surfaceRef}
        className={`ui-overlay__surface ui-confirm-dialog ui-confirm-dialog--${tone}`}
        role={tone === 'danger' ? 'alertdialog' : 'dialog'}
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descriptionId}
        tabIndex={-1}
      >
        <header className="ui-confirm-dialog__header">
          {icon ? <div className="ui-confirm-dialog__icon" aria-hidden="true">{icon}</div> : null}
          <div className="ui-confirm-dialog__copy">
            <h2 id={titleId} className="ui-confirm-dialog__title">{title}</h2>
            <p id={descriptionId} className="ui-confirm-dialog__description">{description}</p>
          </div>
        </header>
        {children ? <div className="ui-confirm-dialog__body">{children}</div> : null}
        <footer className="ui-confirm-dialog__actions">
          <Button ref={cancelRef} variant="secondary" onClick={onClose} disabled={isConfirming}>
            {cancelLabel}
          </Button>
          <Button
            variant={tone === 'danger' ? 'danger' : 'primary'}
            onClick={onConfirm}
            isLoading={isConfirming}
            disabled={confirmDisabled}
          >
            {confirmLabel}
          </Button>
        </footer>
      </section>
    </div>
  );

  return typeof document === 'undefined' ? dialog : createPortal(dialog, document.body);
}
