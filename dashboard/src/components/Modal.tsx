import { ReactNode, useCallback, useEffect } from 'react';
import { XMarkIcon } from '@heroicons/react/24/outline';

/**
 * Standard modal overlay classes
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
 * Includes header with title and close button, content area, and optional footer.
 * Supports closing via Escape key, overlay click, or close button.
 */
export function Modal({ title, onClose, children, maxWidth = 'max-w-md', footer }: ModalProps) {
  // Handle Escape key to close modal
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose();
    }
  }, [onClose]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  return (
    <div
      className={MODAL_OVERLAY_CLASS}
      onClick={onClose}
      role="presentation"
    >
      <div
        className={`${MODAL_CONTAINER_CLASS} ${maxWidth} w-full mx-4`}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
      >
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
    </div>
  );
}
