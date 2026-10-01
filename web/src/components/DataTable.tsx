import type { ReactNode } from "react";
import type { Density } from "../app/useUIStore";
import "./DataTable.css";

export interface Column<T> {
  key: string;
  label: string;
  sortable?: boolean;
  className?: string;
  render: (row: T) => ReactNode;
}

interface DataTableProps<T extends { id: string }> {
  columns: Column<T>[];
  rows: T[];
  sort?: string;
  dir?: string;
  onSort?: (key: string) => void;
  onRowClick?: (row: T) => void;
  selectable?: boolean;
  selectedIds?: Set<string>;
  onToggleSelect?: (id: string) => void;
  onToggleSelectAll?: () => void;
  density?: Density;
}

export function DataTable<T extends { id: string }>({
  columns,
  rows,
  sort,
  dir,
  onSort,
  onRowClick,
  selectable,
  selectedIds,
  onToggleSelect,
  onToggleSelectAll,
  density = "comfortable",
}: DataTableProps<T>) {
  const allSelected = !!selectable && rows.length > 0 && rows.every((r) => selectedIds?.has(r.id));

  return (
    <div className={`data-table-wrap data-table-${density}`}>
      <table className="data-table">
        <thead>
          <tr>
            {selectable && (
              <th className="data-table-checkbox-col">
                <input type="checkbox" checked={allSelected} onChange={onToggleSelectAll} aria-label="Select all rows" />
              </th>
            )}
            {columns.map((col) => (
              <th key={col.key} className={col.className}>
                {col.sortable ? (
                  <button type="button" className="data-table-sort-btn" onClick={() => onSort?.(col.key)}>
                    {col.label}
                    {sort === col.key && <span className="data-table-sort-arrow">{dir === "asc" ? "↑" : "↓"}</span>}
                  </button>
                ) : (
                  col.label
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.id} className="data-table-row" onClick={() => onRowClick?.(row)}>
              {selectable && (
                <td onClick={(e) => e.stopPropagation()}>
                  <input
                    type="checkbox"
                    checked={!!selectedIds?.has(row.id)}
                    onChange={() => onToggleSelect?.(row.id)}
                    aria-label="Select row"
                  />
                </td>
              )}
              {columns.map((col) => (
                <td key={col.key} className={col.className}>
                  {col.render(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
