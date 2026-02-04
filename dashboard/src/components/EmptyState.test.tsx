import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { EmptyState } from './EmptyState';

// Mock icon component
const MockIcon = ({ className }: { className?: string }) => (
  <svg data-testid="mock-icon" className={className} />
);

describe('EmptyState', () => {
  it('renders with required props', () => {
    render(
      <EmptyState
        icon={MockIcon}
        title="No items found"
        description="Start by adding some items"
      />
    );

    expect(screen.getByText('No items found')).toBeInTheDocument();
    expect(screen.getByText('Start by adding some items')).toBeInTheDocument();
    expect(screen.getByTestId('mock-icon')).toBeInTheDocument();
  });

  it('renders icon with correct styling', () => {
    render(
      <EmptyState
        icon={MockIcon}
        title="Test"
        description="Test description"
      />
    );

    const icon = screen.getByTestId('mock-icon');
    expect(icon).toHaveClass('h-12', 'w-12', 'text-gray-400', 'mx-auto', 'mb-4');
  });

  it('renders action button when provided', () => {
    const handleClick = vi.fn();

    render(
      <EmptyState
        icon={MockIcon}
        title="No agents"
        description="Deploy an agent to get started"
        action={{ label: 'Add Agent', onClick: handleClick }}
      />
    );

    const button = screen.getByRole('button', { name: 'Add Agent' });
    expect(button).toBeInTheDocument();
    expect(button).toHaveClass('btn-primary');
  });

  it('calls action onClick when button is clicked', () => {
    const handleClick = vi.fn();

    render(
      <EmptyState
        icon={MockIcon}
        title="No agents"
        description="Deploy an agent"
        action={{ label: 'Add Agent', onClick: handleClick }}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Add Agent' }));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('does not render action button when not provided', () => {
    render(
      <EmptyState
        icon={MockIcon}
        title="No items"
        description="Nothing here"
      />
    );

    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('applies custom className to container', () => {
    const { container } = render(
      <EmptyState
        icon={MockIcon}
        title="Test"
        description="Test description"
        className="custom-class"
      />
    );

    expect(container.firstChild).toHaveClass('custom-class');
  });

  it('renders with default className when not provided', () => {
    const { container } = render(
      <EmptyState
        icon={MockIcon}
        title="Test"
        description="Test description"
      />
    );

    expect(container.firstChild).toHaveClass('text-center', 'py-12');
  });

  it('renders title with correct styling', () => {
    render(
      <EmptyState
        icon={MockIcon}
        title="Empty State Title"
        description="Description"
      />
    );

    const title = screen.getByText('Empty State Title');
    expect(title.tagName).toBe('H3');
    expect(title).toHaveClass('text-lg', 'font-medium', 'text-gray-900');
  });

  it('renders description with correct styling', () => {
    render(
      <EmptyState
        icon={MockIcon}
        title="Title"
        description="This is a description"
      />
    );

    const description = screen.getByText('This is a description');
    expect(description.tagName).toBe('P');
    expect(description).toHaveClass('text-gray-500', 'mt-1');
  });
});
