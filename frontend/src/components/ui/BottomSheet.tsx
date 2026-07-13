import { useId, useRef, type MouseEvent, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { X } from 'lucide-react';
import Button from './Button';
import useModalSurface from './useModalSurface';

export interface BottomSheetProps {
  open: boolean;
  title: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  closeOnBackdrop?: boolean;
  onClose: () => void;
}

export default function BottomSheet({
  open,
  title,
  description,
  children,
  footer,
  closeOnBackdrop = true,
  onClose,
}: BottomSheetProps) {
  const titleId = useId();
  const descriptionId = useId();
  const surfaceRef = useRef<HTMLElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);

  useModalSurface({ open, onClose, surfaceRef, initialFocusRef: closeRef });

  if (!open) return null;

  const handleBackdropClick = (event: MouseEvent<HTMLDivElement>) => {
    if (closeOnBackdrop && event.target === event.currentTarget) onClose();
  };

  const sheet = (
    <div className="ui-overlay ui-overlay--bottom-sheet" onMouseDown={handleBackdropClick}>
      <section
        ref={surfaceRef}
        className="ui-overlay__surface ui-bottom-sheet"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        tabIndex={-1}
      >
        <div className="ui-bottom-sheet__handle" aria-hidden="true" />
        <header className="ui-bottom-sheet__header">
          <div className="ui-bottom-sheet__copy">
            <h2 id={titleId} className="ui-bottom-sheet__title">{title}</h2>
            {description ? (
              <p id={descriptionId} className="ui-bottom-sheet__description">{description}</p>
            ) : null}
          </div>
          <Button
            ref={closeRef}
            variant="ghost"
            iconOnly
            aria-label={`关闭${title}`}
            title={`关闭${title}`}
            onClick={onClose}
          >
            <X size={20} />
          </Button>
        </header>
        <div className="ui-bottom-sheet__body">{children}</div>
        {footer ? <footer className="ui-bottom-sheet__footer">{footer}</footer> : null}
      </section>
    </div>
  );

  return typeof document === 'undefined' ? sheet : createPortal(sheet, document.body);
}
