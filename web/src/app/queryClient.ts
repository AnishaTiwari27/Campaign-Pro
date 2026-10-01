import { QueryCache, QueryClient } from "@tanstack/react-query";
import { useSyncStore } from "./syncStore";

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onSuccess: () => useSyncStore.getState().markSynced(),
  }),
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: true,
    },
  },
});
