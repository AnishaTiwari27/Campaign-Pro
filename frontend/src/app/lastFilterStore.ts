import { create } from "zustand";
import type { ListParams } from "../api/types";

interface LastFilterState {
  lastCampaignsParams: ListParams;
  setLastCampaignsParams: (p: ListParams) => void;
}

// Tracked so Reports' "New from current filters" can scope a draft report
// to whatever the Campaigns page was last filtered to, even after
// navigating away.
export const useLastFilterStore = create<LastFilterState>((set) => ({
  lastCampaignsParams: {},
  setLastCampaignsParams: (p) => set({ lastCampaignsParams: p }),
}));
