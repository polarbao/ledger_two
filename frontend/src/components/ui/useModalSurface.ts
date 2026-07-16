import { useEffect, useRef, type RefObject } from 'react';

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',');

interface ModalSurfaceOptions {
  open: boolean;
  onClose: () => void;
  surfaceRef: RefObject<HTMLElement | null>;
  initialFocusRef?: RefObject<HTMLElement | null>;
}

export default function useModalSurface({
  open,
  onClose,
  surfaceRef,
  initialFocusRef,
}: ModalSurfaceOptions) {
  const onCloseRef = useRef(onClose);
  const returnFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  useEffect(() => {
    if (open || typeof document === 'undefined') return;

    const rememberFocus = (event?: FocusEvent) => {
      const target = event?.target ?? document.activeElement;
      const isInsideModal = target instanceof HTMLElement
        && target.closest('[role="dialog"], [role="alertdialog"]');
      if (target instanceof HTMLElement && target !== document.body && !isInsideModal) {
        returnFocusRef.current = target;
      }
    };

    rememberFocus();
    document.addEventListener('focusin', rememberFocus);
    return () => document.removeEventListener('focusin', rememberFocus);
  }, [open]);

  useEffect(() => {
    if (!open || typeof document === 'undefined') return;

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    const focusFrame = window.requestAnimationFrame(() => {
      const surface = surfaceRef.current;
      const firstFocusable = surface?.querySelector<HTMLElement>(FOCUSABLE_SELECTOR);
      (initialFocusRef?.current ?? firstFocusable ?? surface)?.focus();
    });

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onCloseRef.current();
        return;
      }

      if (event.key !== 'Tab') return;

      const surface = surfaceRef.current;
      if (!surface) return;
      const focusable = Array.from(surface.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR));

      if (focusable.length === 0) {
        event.preventDefault();
        surface.focus();
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (!surface.contains(document.activeElement)) {
        event.preventDefault();
        (event.shiftKey ? last : first).focus();
      } else if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      window.cancelAnimationFrame(focusFrame);
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = previousOverflow;
      returnFocusRef.current?.focus();
    };
  }, [initialFocusRef, open, surfaceRef]);
}
