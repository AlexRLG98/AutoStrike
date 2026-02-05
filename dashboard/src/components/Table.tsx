import { ReactNode } from 'react';

/**
 * Standard table header cell styling
 */
const TABLE_HEADER_CLASS = 'px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider';

/**
 * Table header cell component with consistent styling
 */
export function TableHeaderCell({ children }: { readonly children: ReactNode }) {
  return <th className={TABLE_HEADER_CLASS}>{children}</th>;
}

/**
 * Table header row component that wraps TableHeaderCell for multiple columns
 */
export function TableHeader({ columns }: { readonly columns: string[] }) {
  return (
    <thead className="bg-gray-50">
      <tr>
        {columns.map((column) => (
          <TableHeaderCell key={column}>{column}</TableHeaderCell>
        ))}
      </tr>
    </thead>
  );
}

/**
 * Standard table data cell styling
 */
export const TABLE_CELL_CLASS = 'px-6 py-4';
export const TABLE_CELL_NOWRAP_CLASS = 'px-6 py-4 whitespace-nowrap';

/**
 * Table row component with hover styling
 */
export function TableRow({ children }: { readonly children: ReactNode }) {
  return <tr className="hover:bg-gray-50">{children}</tr>;
}

/**
 * Table body component with divider styling
 */
export function TableBody({ children }: { readonly children: ReactNode }) {
  return <tbody className="divide-y divide-gray-200">{children}</tbody>;
}
