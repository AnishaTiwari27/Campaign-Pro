import "./Pagination.css";

export function Pagination({
  page,
  pages,
  onPageChange,
  sortLabel,
}: {
  page: number;
  pages: number;
  onPageChange: (p: number) => void;
  sortLabel?: string;
}) {
  if (pages <= 1 && !sortLabel) return null;
  const pageNumbers = Array.from({ length: pages }, (_, i) => i + 1);

  return (
    <div className="pagination">
      {sortLabel && <span className="pagination-sort-label">{sortLabel}</span>}
      {pages > 1 && (
        <div className="pagination-nav">
          <button type="button" className="btn btn-sm btn-ghost" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
            Prev
          </button>
          {pageNumbers.map((n) => (
            <button
              key={n}
              type="button"
              className={`pagination-page${n === page ? " pagination-page-active" : ""}`}
              onClick={() => onPageChange(n)}
            >
              {n}
            </button>
          ))}
          <button type="button" className="btn btn-sm btn-ghost" disabled={page >= pages} onClick={() => onPageChange(page + 1)}>
            Next
          </button>
        </div>
      )}
    </div>
  );
}
