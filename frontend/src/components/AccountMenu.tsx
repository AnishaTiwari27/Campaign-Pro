import { useCallback, useEffect, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { useNavigate } from "react-router-dom";
import { useLogout, useSession } from "../api/auth";
import { useThemeStore, type ThemeChoice } from "../app/useTheme";
import {
  IconAuto,
  IconBell,
  IconBilling,
  IconLifebuoy,
  IconLogout,
  IconMoon,
  IconSettings,
  IconSun,
  IconUsers,
  IconSparkle,
  IconHistory,
  IconHandshake,
  IconChevronDown,
} from "./icons";
import "./AccountMenu.css";

const ROLE_LABEL: Record<string, string> = {
  admin: "Admin",
  approver: "Approver",
  analyst: "Analyst",
  client: "Client",
  viewer: "Analyst",
};

const THEMES: { value: ThemeChoice; label: string; Icon: typeof IconAuto }[] = [
  { value: "auto", label: "Auto", Icon: IconAuto },
  { value: "light", label: "Light", Icon: IconSun },
  { value: "dark", label: "Dark", Icon: IconMoon },
];

/** The account menu: everything about you, rather than about the data.
 *
 *  A popover off the rail's user chip, with one nested level for the
 *  account itself. The theme switch lives in here rather than in the
 *  command bar — two controls for one setting, both on screen at once,
 *  is how a settings menu starts to rot.
 *
 *  Items that lead somewhere real and items that do not are both here,
 *  but they do not look alike: an unbuilt one is marked "Soon" and
 *  says so when opened, rather than quietly going nowhere.
 */
export function AccountMenu({ canManageUsers = false }: { canManageUsers?: boolean }) {
  const { data: me } = useSession();
  const logout = useLogout();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [submenu, setSubmenu] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const menu = useRef<HTMLDivElement>(null);
  // Where to put the popover, measured from the chip.
  const [at, setAt] = useState<{ left: number; bottom: number }>({ left: 0, bottom: 0 });

  // The rail scrolls, so it clips anything that leaves it — which the
  // nested flyout does by definition, since it opens sideways. The
  // popover is rendered into the body instead and positioned from the
  // chip's own rectangle. Measured rather than guessed, because the
  // rail's width changes with the viewport.
  const place = useCallback(() => {
    const r = root.current?.getBoundingClientRect();
    if (!r) return;
    setAt({ left: r.left, bottom: window.innerHeight - r.top + 8 });
  }, []);

  // Close on a click anywhere else, and on Escape. Both, because a
  // popover that only closes one way strands people who reached for
  // the other.
  useEffect(() => {
    if (!open) return;
    place();
    function onPointer(e: MouseEvent) {
      // Both, now that the menu is not inside the chip's subtree.
      if (root.current?.contains(e.target as Node)) return;
      if (menu.current?.contains(e.target as Node)) return;
      setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    // Anything that moves the chip invalidates the position. Closing
    // is honest and cheap; re-measuring mid-scroll would have the
    // popover chase the chip around.
    const close = () => setOpen(false);
    document.addEventListener("mousedown", onPointer);
    document.addEventListener("keydown", onKey);
    window.addEventListener("resize", close);
    window.addEventListener("scroll", close, true);
    return () => {
      document.removeEventListener("mousedown", onPointer);
      document.removeEventListener("keydown", onKey);
      window.removeEventListener("resize", close);
      window.removeEventListener("scroll", close, true);
    };
  }, [open, place]);

  useEffect(() => {
    if (!open) setSubmenu(false);
  }, [open]);

  if (!me) return null;

  const initials = me.name
    .split(" ")
    .map((p) => p[0])
    .join("")
    .slice(0, 2);

  function go(to: string) {
    setOpen(false);
    navigate(to);
  }

  return (
    <div className="account" ref={root}>
      <button
        type="button"
        className={`account-chip${open ? " account-chip-open" : ""}`}
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <span className="account-initials">{initials}</span>
        <span className="account-who">
          <span className="account-name truncate">{me.name}</span>
          <span className="account-role">
            {ROLE_LABEL[me.role] ?? me.role}
            {me.canApprove ? " · approver" : ""}
          </span>
        </span>
        <IconChevronDown className="account-caret" />
      </button>

      {open &&
        createPortal(
          <div
            className="account-menu"
            role="menu"
            ref={menu}
            style={{ left: at.left, bottom: at.bottom }}
          >
          <Row icon={<IconSettings />} label="Profile" onClick={() => go("/settings")} />
          <Row icon={<IconBell />} label="Notifications" soon onClick={() => go("/soon/notifications")} />

          <div
            className={`account-sub${submenu ? " account-sub-open" : ""}`}
            onMouseEnter={() => setSubmenu(true)}
            onMouseLeave={() => setSubmenu(false)}
          >
            <button
              type="button"
              role="menuitem"
              className="account-row account-row-parent"
              aria-haspopup="menu"
              aria-expanded={submenu}
              onClick={() => setSubmenu((s) => !s)}
            >
              <span className="account-row-icon">
                <IconBilling />
              </span>
              <span className="account-row-label">Account</span>
              <span className="account-row-arrow" aria-hidden="true">
                ›
              </span>
            </button>

            {submenu && (
              <div className="account-flyout" role="menu">
                <Row icon={<IconBilling />} label="Plan & billing" onClick={() => go("/billing")} />
                <Row icon={<IconSparkle />} label="AI credits" soon onClick={() => go("/soon/ai-credits")} />
                <Row icon={<IconSettings />} label="Preferences" onClick={() => go("/settings")} />
                <Row icon={<IconHistory />} label="Login history" onClick={() => go("/sign-ins")} />
                {/* Only for an admin: the roster is theirs to manage,
                    and an item that 403s is worse than one that is not
                    there. */}
                {canManageUsers && (
                  <Row icon={<IconUsers />} label="Team" onClick={() => go("/users")} />
                )}
                <Row icon={<IconHandshake />} label="Affiliate" soon onClick={() => go("/soon/affiliate")} />
              </div>
            )}
          </div>

          <Row icon={<IconLifebuoy />} label="Troubleshooting" soon onClick={() => go("/soon/troubleshooting")} />

          <div className="account-divider" />

          <div className="account-theme-row">
            <span className="account-row-icon">
              <IconAuto />
            </span>
            <span className="account-row-label">Theme</span>
            <ThemeSwitch />
          </div>

          <div className="account-divider" />

          <button
            type="button"
            role="menuitem"
            className="account-row account-row-danger"
            disabled={logout.isPending}
            onClick={() => logout.mutate()}
          >
            <span className="account-row-icon">
              <IconLogout />
            </span>
            <span className="account-row-label">{logout.isPending ? "Signing out…" : "Log out"}</span>
          </button>
          </div>,
          document.body,
        )}
    </div>
  );
}

function Row({
  icon,
  label,
  onClick,
  soon = false,
}: {
  icon: ReactNode;
  label: string;
  onClick: () => void;
  /** Marks an item with nothing behind it yet. Shown rather than
   *  hidden, but never dressed up as working. */
  soon?: boolean;
}) {
  return (
    <button type="button" role="menuitem" className="account-row" onClick={onClick}>
      <span className="account-row-icon">{icon}</span>
      <span className="account-row-label">{label}</span>
      {soon && <span className="account-soon">Soon</span>}
    </button>
  );
}

/** The same sliding-thumb switch the command bar had, moved in here. */
function ThemeSwitch() {
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const index = THEMES.findIndex((t) => t.value === theme);

  return (
    <div
      className="account-theme"
      role="radiogroup"
      aria-label="Theme"
      style={{ "--theme-index": index } as CSSProperties}
    >
      <span className="account-theme-thumb" aria-hidden="true" />
      {THEMES.map((t) => (
        <button
          key={t.value}
          type="button"
          role="radio"
          aria-checked={theme === t.value}
          aria-label={t.label}
          title={t.label}
          className={`account-theme-opt${theme === t.value ? " account-theme-opt-active" : ""}`}
          onClick={() => setTheme(t.value)}
        >
          <t.Icon />
        </button>
      ))}
    </div>
  );
}
