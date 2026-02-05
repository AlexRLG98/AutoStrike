import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  TableHeaderCell,
  TableHeader,
  TableBody,
  TableRow,
  TABLE_CELL_CLASS,
  TABLE_CELL_NOWRAP_CLASS,
} from './Table';

describe('Table Components', () => {
  describe('TableHeaderCell', () => {
    it('renders children with correct styling', () => {
      render(
        <table>
          <thead>
            <tr>
              <TableHeaderCell>Test Header</TableHeaderCell>
            </tr>
          </thead>
        </table>
      );

      const header = screen.getByText('Test Header');
      expect(header).toBeInTheDocument();
      expect(header).toHaveClass('px-6', 'py-3', 'text-left', 'text-xs', 'font-medium', 'text-gray-500', 'uppercase', 'tracking-wider');
    });
  });

  describe('TableHeader', () => {
    it('renders all columns', () => {
      render(
        <table>
          <TableHeader columns={['ID', 'Name', 'Status']} />
        </table>
      );

      expect(screen.getByText('ID')).toBeInTheDocument();
      expect(screen.getByText('Name')).toBeInTheDocument();
      expect(screen.getByText('Status')).toBeInTheDocument();
    });

    it('renders thead with bg-gray-50', () => {
      const { container } = render(
        <table>
          <TableHeader columns={['Test']} />
        </table>
      );

      const thead = container.querySelector('thead');
      expect(thead).toHaveClass('bg-gray-50');
    });

    it('handles empty columns array', () => {
      const { container } = render(
        <table>
          <TableHeader columns={[]} />
        </table>
      );

      const thead = container.querySelector('thead');
      expect(thead).toBeInTheDocument();
      expect(container.querySelectorAll('th')).toHaveLength(0);
    });
  });

  describe('TableBody', () => {
    it('renders children with divider styling', () => {
      const { container } = render(
        <table>
          <TableBody>
            <tr><td>Test</td></tr>
          </TableBody>
        </table>
      );

      const tbody = container.querySelector('tbody');
      expect(tbody).toHaveClass('divide-y', 'divide-gray-200');
    });
  });

  describe('TableRow', () => {
    it('renders children with hover styling', () => {
      const { container } = render(
        <table>
          <tbody>
            <TableRow><td>Test</td></TableRow>
          </tbody>
        </table>
      );

      const row = container.querySelector('tr');
      expect(row).toHaveClass('hover:bg-gray-50');
    });
  });

  describe('CSS Constants', () => {
    it('TABLE_CELL_CLASS has correct value', () => {
      expect(TABLE_CELL_CLASS).toBe('px-6 py-4');
    });

    it('TABLE_CELL_NOWRAP_CLASS has correct value', () => {
      expect(TABLE_CELL_NOWRAP_CLASS).toBe('px-6 py-4 whitespace-nowrap');
    });
  });
});
