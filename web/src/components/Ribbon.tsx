import { useEffect, useRef, useState, type ReactNode } from "react";
import "./Ribbon.css";

export type RibbonAccent = "accent" | "warn" | "crit" | "good";

export function Ribbon({
  title,
  subtitle,
  count,
  accent = "accent",
  emptyMessage,
  action,
  children,
}: {
  title: string;
  subtitle?: string;
  count?: number;
  accent?: RibbonAccent;
  emptyMessage?: string;
  action?: ReactNode;
  children: ReactNode[];
}) {
  const scrollerRef = useRef<HTMLDivElement>(null);
  const [atStart, setAtStart] = useState(true);
  const [atEnd, setAtEnd] = useState(true);

  function updateEdges() {
    const el = scrollerRef.current;
    if (!el) return;
    setAtStart(el.scrollLeft <= 4);
    setAtEnd(el.scrollLeft + el.clientWidth >= el.scrollWidth - 4);
  }

  useEffect(() => {
    updateEdges();
  }, [children.length]);

  function scrollBy(dir: 1 | -1) {
    scrollerRef.current?.scrollBy({ left: dir * 376, behavior: "smooth" });
  }

  // Both arrows disabled means there's nothing to scroll — hide them rather
  // than show a dead control.
  const scrollable = !(atStart && atEnd);

  return (
    <section className="ribbon">
      <div className="ribbon-header">
        <div className="ribbon-heading">
          <h3>
            {title}
            {count !== undefined && count > 0 && <span className={`ribbon-count ribbon-count-${accent}`}>{count}</span>}
          </h3>
          {subtitle && <p className="ribbon-subtitle">{subtitle}</p>}
        </div>
        <div className="ribbon-controls">
          {action}
          {children.length > 0 && scrollable && (
            <div className="ribbon-nav">
              <button type="button" className="ribbon-nav-btn" aria-label={`Scroll ${title} left`} disabled={atStart} onClick={() => scrollBy(-1)}>
                ‹
              </button>
              <button type="button" className="ribbon-nav-btn" aria-label={`Scroll ${title} right`} disabled={atEnd} onClick={() => scrollBy(1)}>
                ›
              </button>
            </div>
          )}
        </div>
      </div>
      {children.length === 0 ? (
        <div className="ribbon-empty">{emptyMessage ?? "Nothing here right now."}</div>
      ) : (
        <div className="ribbon-track" ref={scrollerRef} onScroll={updateEdges}>
          {children}
        </div>
      )}
    </section>
  );
}
