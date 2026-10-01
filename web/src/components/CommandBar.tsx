import { Link } from "react-router-dom";
import { useBreadcrumbs } from "../app/BreadcrumbContext";
import { useThemeStore, type ThemeChoice } from "../app/useTheme";
import { useUIStore } from "../app/useUIStore";
import { useSyncStore } from "../app/syncStore";
import { useNow } from "../app/useNow";
import { IconSearch } from "./icons";
import "./CommandBar.css";

function syncLabel(lastSyncedAt: number | null, now: number): string {
  if (lastSyncedAt === null) return "Syncing…";
  const mins = Math.floor((now - lastSyncedAt) / 60000);
  if (mins < 1) return "Synced just now";
  if (mins === 1) return "Synced 1 min ago";
  return `Synced ${mins} min ago`;
}

const THEME_OPTIONS: { value: ThemeChoice; label: string }[] = [
  { value: "auto", label: "Auto" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

export function CommandBar() {
  const { breadcrumbs } = useBreadcrumbs();
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const openPalette = useUIStore((s) => s.openPalette);
  const lastSyncedAt = useSyncStore((s) => s.lastSyncedAt);
  const now = useNow();

  return (
    <header className="command-bar">
      <nav className="command-bar-breadcrumbs" aria-label="Breadcrumb">
        {breadcrumbs.map((crumb, i) => {
          const isLast = i === breadcrumbs.length - 1;
          return (
            <span key={i} className="command-bar-crumb">
              {crumb.href && !isLast ? (
                <Link to={crumb.href}>{crumb.label}</Link>
              ) : (
                <span aria-current={isLast ? "page" : undefined}>{crumb.label}</span>
              )}
              {!isLast && <span className="command-bar-crumb-sep">/</span>}
            </span>
          );
        })}
      </nav>

      <div className="command-bar-sync">
        <span className="command-bar-sync-dot" aria-hidden="true" />
        {syncLabel(lastSyncedAt, now)}
      </div>

      <button type="button" className="command-bar-search" onClick={openPalette}>
        <IconSearch />
        <span className="command-bar-search-label">Search</span>
        <kbd className="command-bar-kbd">⌘K</kbd>
      </button>

      <div className="command-bar-theme" role="radiogroup" aria-label="Theme">
        {THEME_OPTIONS.map((opt) => (
          <button
            key={opt.value}
            type="button"
            role="radio"
            aria-checked={theme === opt.value}
            className={`command-bar-theme-opt${theme === opt.value ? " command-bar-theme-opt-active" : ""}`}
            onClick={() => setTheme(opt.value)}
          >
            {opt.label}
          </button>
        ))}
      </div>
    </header>
  );
}
