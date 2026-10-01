import { create } from "zustand";

export type ThemeChoice = "auto" | "light" | "dark";

interface ThemeState {
  theme: ThemeChoice;
  setTheme: (t: ThemeChoice) => void;
}

const STORAGE_KEY = "campaign-tracker-pro:theme";

function applyTheme(theme: ThemeChoice) {
  const root = document.documentElement;
  if (theme === "auto") root.removeAttribute("data-theme");
  else root.setAttribute("data-theme", theme);
}

function loadInitial(): ThemeChoice {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "light" || stored === "dark" || stored === "auto") return stored;
  } catch {
    // private mode / storage blocked — fall back to auto
  }
  return "auto";
}

const initial = loadInitial();
if (typeof document !== "undefined") applyTheme(initial);

export const useThemeStore = create<ThemeState>((set) => ({
  theme: initial,
  setTheme: (t) => {
    try {
      localStorage.setItem(STORAGE_KEY, t);
    } catch {
      // ignore write failures
    }
    applyTheme(t);
    set({ theme: t });
  },
}));
