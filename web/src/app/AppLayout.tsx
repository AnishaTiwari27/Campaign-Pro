import { Outlet, useNavigate } from "react-router-dom";
import { Rail } from "../components/Rail";
import { CommandBar } from "../components/CommandBar";
import { Toasts } from "../components/Toasts";
import { Palette } from "../components/Palette";
import { useUIStore } from "./useUIStore";
import { useKeyboardShortcuts } from "../lib/keyboard";
import { BreadcrumbProvider } from "./BreadcrumbContext";
import "./AppLayout.css";

function Shell() {
  const openPalette = useUIStore((s) => s.openPalette);
  const navigate = useNavigate();

  useKeyboardShortcuts({
    onPalette: openPalette,
    onBack: () => navigate(-1),
  });

  return (
    <div className="app-shell">
      <Rail />
      <div className="app-main">
        <CommandBar />
        <main className="app-content">
          <Outlet />
        </main>
      </div>
      <Toasts />
      <Palette />
    </div>
  );
}

export function AppLayout() {
  return (
    <BreadcrumbProvider>
      <Shell />
    </BreadcrumbProvider>
  );
}
