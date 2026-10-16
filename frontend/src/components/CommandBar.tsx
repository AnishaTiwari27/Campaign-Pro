import { useEffect, useState, type CSSProperties } from "react";
import { Link } from "react-router-dom";
import { useBreadcrumbs } from "../app/BreadcrumbContext";
import { useThemeStore, type ThemeChoice } from "../app/useTheme";
import { useUIStore } from "../app/useUIStore";
import { useSyncStore } from "../app/syncStore";
import { useNow } from "../app/useNow";
import { IconAuto, IconMoon, IconSearch, IconSun } from "./icons";
import "./CommandBar.css";

function syncLabel(lastSyncedAt: number | null, now: number): string {
  if (lastSyncedAt === null) return "Syncing…";
  const mins = Math.floor((now - lastSyncedAt) / 60000);
  if (mins < 1) return "Synced just now";
  if (mins === 1) return "Synced 1 min ago";
  return `Synced ${mins} min ago`;
}

const THEME_OPTIONS: { value: ThemeChoice; label: string; Icon: typeof IconAuto }[] = [
  { value: "auto", label: "Auto", Icon: IconAuto },
  { value: "light", label: "Light", Icon: IconSun },
  { value: "dark", label: "Dark", Icon: IconMoon },
];

/** What the search button offers to find, cycled one at a time.
 *
 *  A button that only says "Search" leaves you guessing what is in
 *  the index. These are the things that actually are.
 */
const SEARCHABLE = ["campaigns", "creators", "brands", "regions", "reports"];

/** Index into SEARCHABLE, advancing every few seconds.
 *
 *  Holds still for anyone who asked for reduced motion — a word
 *  swapping itself in the corner of the eye is exactly the kind of
 *  thing that setting is for.
 */
function useSearchHint(): number {
  const [i, setI] = useState(0);
  useEffect(() => {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    const id = setInterval(() => setI((n) => (n + 1) % SEARCHABLE.length), 3200);
    return () => clearInterval(id);
  }, []);
  return i;
}

export function CommandBar() {
  const { breadcrumbs } = useBreadcrumbs();
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const openPalette = useUIStore((s) => s.openPalette);
  const lastSyncedAt = useSyncStore((s) => s.lastSyncedAt);
  const now = useNow();
  const hint = useSearchHint();

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

      <button
        type="button"
        className="command-bar-search"
        onClick={openPalette}
        aria-label="Search campaigns, creators, brands, regions and reports"
      >
        <IconSearch className="command-bar-search-icon" />
        <span className="command-bar-search-label">
          Search{" "}
          {/* aria-hidden: the cycling word is a hint, and a label that
              rewrites itself every few seconds would be read out again
              each time. The button's own aria-label names all of it. */}
          <span className="command-bar-search-cycle" aria-hidden="true">
            <span key={hint} className="command-bar-search-word">
              {SEARCHABLE[hint]}
            </span>
          </span>
        </span>
        <kbd className="command-bar-kbd">⌘K</kbd>
      </button>

      {/* A track with a thumb that slides to the chosen one, rather
          than three buttons that each light up. Icons, not words:
          three words cost about 130px and squeezed the breadcrumb to
          "Camp" on a phone — three icons fit, so the narrow-screen
          collapse that used to paper over that is gone. */}
      <div
        className="command-bar-theme"
        role="radiogroup"
        aria-label="Theme"
        style={{ "--theme-index": THEME_OPTIONS.findIndex((o) => o.value === theme) } as CSSProperties}
      >
        <span className="command-bar-theme-thumb" aria-hidden="true" />
        {THEME_OPTIONS.map((opt) => (
          <button
            key={opt.value}
            type="button"
            role="radio"
            aria-checked={theme === opt.value}
            aria-label={opt.label}
            title={opt.label}
            className={`command-bar-theme-opt${theme === opt.value ? " command-bar-theme-opt-active" : ""}`}
            onClick={() => setTheme(opt.value)}
          >
            <opt.Icon />
          </button>
        ))}
      </div>
    </header>
  );
}
