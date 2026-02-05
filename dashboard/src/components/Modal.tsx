import { ReactNode, useRef, useEffect } from 'react';
import { XMarkIcon } from '@heroicons/react/24/outline';

/**
 * Standard modal overlay classes (for backdrop styling)
 */
export const MODAL_OVERLAY_CLASS = 'fixed inset-0 bg-black/50 flex items-center justify-center z-50';

/**
 * Standard modal container classes
 */
export const MODAL_CONTAINER_CLASS = 'bg-white rounded-xl shadow-xl';

interface ModalProps {
  /** Modal title displayed in the header */
  readonly title: string;
  /** Callback when modal is closed (X button, overlay click, or Escape key) */
  readonly onClose: () => void;
  /** Modal content */
  readonly children: ReactNode;
  /** Optional max width class (default: 'max-w-md') */
  readonly maxWidth?: string;
  /** Optional footer content */
  readonly footer?: ReactNode;
}

/**
 * Reusable modal component with consistent styling.
 * Uses native HTML dialog element for accessibility.
 * Includes header with title and close button, content area, and optional footer.
 * Supports closing via Escape key, backdrop click, or close button.
 */
export function Modal({ title, onClose, children, maxWidth = 'max-w-md', footer }: ModalProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (dialog && !dialog.open) {
      dialog.showModal();
    }

    // Handle backdrop click by checking if click is outside content
    const handleBackdropClick = (e: MouseEvent) => {
      if (contentRef.current && !contentRef.current.contains(e.target as Node)) {
        onClose();
      }
    };

    dialog?.addEventListener('click', handleBackdropClick);
    return () => dialog?.removeEventListener('click', handleBackdropClick);
  }, [onClose]);

  // Handle native dialog cancel event (Escape key)
  const handleCancel = (e: React.SyntheticEvent) => {
    e.preventDefault();
    onClose();
  };

  return (
    <dialog
      ref={dialogRef}
      className={`${MODAL_CONTAINER_CLASS} ${maxWidth} w-full mx-4 p-0 backdrop:bg-black/50`}
      onCancel={handleCancel}
      aria-labelledby="modal-title"
    >
      <div ref={contentRef}>
        <div className="flex items-center justify-between p-6 border-b">
          <h2 id="modal-title" className="text-xl font-semibold">{title}</h2>
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 rounded-lg"
            aria-label="Close modal"
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>
        <div className="p-6">
          {children}
        </div>
        {footer && (
          <div className="flex justify-end gap-3 p-6 border-t">
            {footer}
          </div>
        )}
      </div>
    </dialog>
  );
}
