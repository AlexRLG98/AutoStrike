import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Modal, MODAL_OVERLAY_CLASS, MODAL_CONTAINER_CLASS } from './Modal';

// Mock showModal since JSDOM doesn't fully support it
beforeAll(() => {
  HTMLDialogElement.prototype.showModal = vi.fn();
  HTMLDialogElement.prototype.close = vi.fn();
});

describe('Modal', () => {
  const defaultProps = {
    title: 'Test Modal',
    onClose: vi.fn(),
    children: <div>Modal content</div>,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('MODAL_OVERLAY_CLASS', () => {
    it('contains overlay styles for reference', () => {
      expect(MODAL_OVERLAY_CLASS).toContain('fixed');
      expect(MODAL_OVERLAY_CLASS).toContain('inset-0');
      expect(MODAL_OVERLAY_CLASS).toContain('bg-black/50');
      expect(MODAL_OVERLAY_CLASS).toContain('z-50');
    });
  });

  describe('MODAL_CONTAINER_CLASS', () => {
    it('contains container styles', () => {
      expect(MODAL_CONTAINER_CLASS).toContain('bg-white');
      expect(MODAL_CONTAINER_CLASS).toContain('rounded-xl');
      expect(MODAL_CONTAINER_CLASS).toContain('shadow-xl');
    });
  });

  describe('Modal component', () => {
    it('renders with title', () => {
      render(<Modal {...defaultProps} />);
      expect(screen.getByText('Test Modal')).toBeInTheDocument();
    });

    it('renders children content', () => {
      render(<Modal {...defaultProps} />);
      expect(screen.getByText('Modal content')).toBeInTheDocument();
    });

    it('calls showModal on mount', () => {
      render(<Modal {...defaultProps} />);
      expect(HTMLDialogElement.prototype.showModal).toHaveBeenCalled();
    });

    it('calls onClose when close button is clicked', () => {
      render(<Modal {...defaultProps} />);
      const closeButton = screen.getByLabelText('Close modal');
      fireEvent.click(closeButton);
      expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
    });

    it('renders with default maxWidth', () => {
      const { container } = render(<Modal {...defaultProps} />);
      const dialog = container.querySelector('dialog');
      expect(dialog).toHaveClass('max-w-md');
    });

    it('renders with custom maxWidth', () => {
      const { container } = render(<Modal {...defaultProps} maxWidth="max-w-lg" />);
      const dialog = container.querySelector('dialog');
      expect(dialog).toHaveClass('max-w-lg');
    });

    it('does not render footer when not provided', () => {
      const { container } = render(<Modal {...defaultProps} />);
      const footerBorder = container.querySelectorAll('.border-t');
      expect(footerBorder.length).toBe(0);
    });

    it('renders footer when provided', () => {
      render(
        <Modal {...defaultProps} footer={<button>Save</button>} />
      );
      expect(screen.getByText('Save')).toBeInTheDocument();
    });

    it('renders multiple footer elements', () => {
      render(
        <Modal
          {...defaultProps}
          footer={
            <>
              <button>Cancel</button>
              <button>Confirm</button>
            </>
          }
        />
      );
      expect(screen.getByText('Cancel')).toBeInTheDocument();
      expect(screen.getByText('Confirm')).toBeInTheDocument();
    });

    it('has proper header structure with border', () => {
      const { container } = render(<Modal {...defaultProps} />);
      const header = container.querySelector('.border-b');
      expect(header).toBeInTheDocument();
      expect(header).toHaveClass('p-6');
    });

    it('has proper content padding', () => {
      const { container } = render(<Modal {...defaultProps} />);
      const contentArea = container.querySelectorAll('.p-6');
      expect(contentArea.length).toBeGreaterThanOrEqual(2); // header + content
    });

    it('calls onClose when clicking on dialog backdrop', () => {
      const { container } = render(<Modal {...defaultProps} />);
      const dialog = container.querySelector('dialog') as HTMLDialogElement;
      // Simulate click on the dialog itself (backdrop area)
      fireEvent.click(dialog);
      expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
    });

    it('does not call onClose when clicking modal content', () => {
      render(<Modal {...defaultProps} />);
      const content = screen.getByText('Modal content');
      fireEvent.click(content);
      expect(defaultProps.onClose).not.toHaveBeenCalled();
    });

    it('calls onClose when cancel event is triggered (Escape key)', () => {
      const { container } = render(<Modal {...defaultProps} />);
      const dialog = container.querySelector('dialog') as HTMLDialogElement;
      fireEvent(dialog, new Event('cancel', { bubbles: true }));
      expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
    });

    it('uses native dialog element', () => {
      const { container } = render(<Modal {...defaultProps} />);
      const dialog = container.querySelector('dialog');
      expect(dialog).toBeInTheDocument();
    });

    it('has title with correct id for aria-labelledby', () => {
      render(<Modal {...defaultProps} />);
      const title = screen.getByText('Test Modal');
      expect(title).toHaveAttribute('id', 'modal-title');
    });

    it('has aria-labelledby pointing to title', () => {
      const { container } = render(<Modal {...defaultProps} />);
      const dialog = container.querySelector('dialog');
      expect(dialog).toHaveAttribute('aria-labelledby', 'modal-title');
    });
  });
});
