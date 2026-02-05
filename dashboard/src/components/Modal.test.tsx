import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Modal, MODAL_OVERLAY_CLASS, MODAL_CONTAINER_CLASS } from './Modal';

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
    it('contains fixed positioning and overlay styles', () => {
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

    it('calls onClose when close button is clicked', () => {
      render(<Modal {...defaultProps} />);
      const closeButton = screen.getByLabelText('Close modal');
      fireEvent.click(closeButton);
      expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
    });

    it('renders with default maxWidth', () => {
      const { container } = render(<Modal {...defaultProps} />);
      const modalContainer = container.querySelector('.max-w-md');
      expect(modalContainer).toBeInTheDocument();
    });

    it('renders with custom maxWidth', () => {
      const { container } = render(<Modal {...defaultProps} maxWidth="max-w-lg" />);
      const modalContainer = container.querySelector('.max-w-lg');
      expect(modalContainer).toBeInTheDocument();
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

    it('calls onClose when clicking overlay', () => {
      const { container } = render(<Modal {...defaultProps} />);
      const overlay = container.firstChild as HTMLElement;
      fireEvent.click(overlay);
      expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
    });

    it('does not call onClose when clicking modal content', () => {
      render(<Modal {...defaultProps} />);
      const content = screen.getByText('Modal content');
      fireEvent.click(content);
      expect(defaultProps.onClose).not.toHaveBeenCalled();
    });
  });
});
