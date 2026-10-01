import { create } from "zustand";

export type Density = "comfortable" | "compact";

interface UIState {
  density: Density;
  setDensity: (d: Density) => void;
  selectedIds: Set<string>;
  toggleSelected: (id: string) => void;
  selectMany: (ids: string[]) => void;
  clearSelection: () => void;
  paletteOpen: boolean;
  openPalette: () => void;
  closePalette: () => void;
}

const DENSITY_KEY = "campaign-tracker-pro:density";

function loadDensity(): Density {
  try {
    const stored = localStorage.getItem(DENSITY_KEY);
    if (stored === "compact") return "compact";
  } catch {
    // ignore
  }
  return "comfortable";
}

export const useUIStore = create<UIState>((set) => ({
  density: loadDensity(),
  setDensity: (d) => {
    try {
      localStorage.setItem(DENSITY_KEY, d);
    } catch {
      // ignore
    }
    set({ density: d });
  },
  selectedIds: new Set(),
  toggleSelected: (id) =>
    set((s) => {
      const next = new Set(s.selectedIds);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return { selectedIds: next };
    }),
  selectMany: (ids) => set({ selectedIds: new Set(ids) }),
  clearSelection: () => set({ selectedIds: new Set() }),
  paletteOpen: false,
  openPalette: () => set({ paletteOpen: true }),
  closePalette: () => set({ paletteOpen: false }),
}));
