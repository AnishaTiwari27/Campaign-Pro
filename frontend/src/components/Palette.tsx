import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useUIStore } from "../app/useUIStore";
import { useSearch } from "../api/misc";
import type { SearchResult } from "../api/types";
import "./Palette.css";

function targetFor(r: SearchResult): string {
  switch (r.kind) {
    case "section":
      return `/${r.id}`;
    case "campaign":
      return `/campaigns/${r.id}`;
    case "creator":
      return `/creators/${r.id}`;
    case "region":
      return `/regions/${encodeURIComponent(r.id)}`;
    case "action":
      return `/${r.id}`;
    default:
      return "/overview";
  }
}

const GROUP_TITLES: Record<SearchResult["kind"], string> = {
  section: "Go to",
  campaign: "Campaigns",
  creator: "Influencers",
  region: "Regions",
  action: "Actions",
};

export function Palette() {
  const open = useUIStore((s) => s.paletteOpen);
  const close = useUIStore((s) => s.closePalette);
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const id = setTimeout(() => setDebounced(query), 120);
    return () => clearTimeout(id);
  }, [query]);

  const { data } = useSearch(debounced);
  const items = useMemo(() => data?.items ?? [], [data]);

  useEffect(() => {
    if (open) {
      setQuery("");
      setDebounced("");
      setActiveIndex(0);
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [open]);

  useEffect(() => setActiveIndex(0), [items.length]);

  const groups = useMemo(() => {
    const order: SearchResult["kind"][] = ["section", "campaign", "creator", "region", "action"];
    return order.map((kind) => ({ kind, items: items.filter((i) => i.kind === kind) })).filter((g) => g.items.length > 0);
  }, [items]);

  function openResult(r: SearchResult) {
    navigate(targetFor(r));
    close();
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === "Escape") {
      e.preventDefault();
      close();
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIndex((i) => Math.min(items.length - 1, i + 1));
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIndex((i) => Math.max(0, i - 1));
      return;
    }
    if (e.key === "Enter") {
      e.preventDefault();
      const active = items[activeIndex];
      if (active) openResult(active);
    }
  }

  if (!open) return null;

  let flatIndex = -1;

  return (
    <div className="palette-scrim" onClick={close}>
      <div className="palette" onClick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Command palette">
        <div className="palette-input-row">
          <input
            ref={inputRef}
            className="palette-input"
            placeholder="Search campaigns, regions, or go to a section…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
          />
          <kbd className="command-bar-kbd">Esc</kbd>
        </div>

        <div className="palette-results">
          {items.length === 0 ? (
            <div className="palette-empty">No matches.</div>
          ) : (
            groups.map((group) => (
              <div key={group.kind} className="palette-group">
                <div className="palette-group-title">{GROUP_TITLES[group.kind]}</div>
                {group.items.map((r) => {
                  flatIndex += 1;
                  const isActive = flatIndex === activeIndex;
                  return (
                    <button
                      key={`${r.kind}-${r.id}`}
                      type="button"
                      className={`palette-item${isActive ? " palette-item-active" : ""}`}
                      onMouseEnter={() => setActiveIndex(flatIndex)}
                      onClick={() => openResult(r)}
                    >
                      <span className="palette-item-title">{r.title}</span>
                      {r.subtitle && <span className="palette-item-subtitle">{r.subtitle}</span>}
                    </button>
                  );
                })}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
