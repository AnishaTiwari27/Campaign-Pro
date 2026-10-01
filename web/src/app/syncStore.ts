import { create } from "zustand";

interface SyncState {
  lastSyncedAt: number | null;
  markSynced: () => void;
}

// Bumped by the QueryClient's global onSuccess callback (see queryClient.ts)
// so the command bar's "Synced N min ago" reflects real fetch activity.
export const useSyncStore = create<SyncState>((set) => ({
  lastSyncedAt: null,
  markSynced: () => set({ lastSyncedAt: Date.now() }),
}));
