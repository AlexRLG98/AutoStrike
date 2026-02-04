import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { LoadingState } from './LoadingState';

describe('LoadingState', () => {
  it('renders with default message', () => {
    render(<LoadingState />);

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('renders with custom message', () => {
    render(<LoadingState message="Loading agents..." />);

    expect(screen.getByText('Loading agents...')).toBeInTheDocument();
    expect(screen.queryByText('Loading...')).not.toBeInTheDocument();
  });

  it('has aria-live="polite" for accessibility', () => {
    render(<LoadingState />);

    const output = screen.getByRole('status');
    expect(output).toHaveAttribute('aria-live', 'polite');
  });

  it('renders as output element', () => {
    const { container } = render(<LoadingState />);

    const output = container.querySelector('output');
    expect(output).toBeInTheDocument();
  });

  it('applies animate-pulse class', () => {
    const { container } = render(<LoadingState />);

    expect(container.firstChild).toHaveClass('animate-pulse');
  });

  it('applies text-gray-500 class', () => {
    const { container } = render(<LoadingState />);

    expect(container.firstChild).toHaveClass('text-gray-500');
  });

  it('applies custom className when provided', () => {
    const { container } = render(<LoadingState className="custom-loading" />);

    expect(container.firstChild).toHaveClass('custom-loading');
    expect(container.firstChild).toHaveClass('animate-pulse');
  });

  it('renders bouncing dot indicator', () => {
    const { container } = render(<LoadingState />);

    const dot = container.querySelector('.animate-bounce');
    expect(dot).toBeInTheDocument();
    expect(dot).toHaveClass('h-4', 'w-4', 'rounded-full', 'bg-gray-300');
  });

  it('renders with flex container for alignment', () => {
    const { container } = render(<LoadingState />);

    const flexContainer = container.querySelector('.flex');
    expect(flexContainer).toBeInTheDocument();
    expect(flexContainer).toHaveClass('items-center', 'gap-2');
  });

  it('works with empty custom className', () => {
    const { container } = render(<LoadingState className="" />);

    expect(container.firstChild).toHaveClass('animate-pulse', 'text-gray-500');
  });

  it('renders message inside span', () => {
    render(<LoadingState message="Please wait..." />);

    const span = screen.getByText('Please wait...');
    expect(span.tagName).toBe('SPAN');
  });
});
