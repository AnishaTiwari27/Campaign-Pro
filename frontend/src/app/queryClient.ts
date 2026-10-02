import { QueryCache, QueryClient } from "@tanstack/react-query";
import { useSyncStore } from "./syncStore";

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onSuccess: () => useSyncStore.getState().markSynced(),
  }),
  defaultOptions: {
    queries: {
      retry: 1,
      // Muted deliberately. Refetching on every tab focus re-ran the
      // session check, and while that was in flight the auth gate fell
      // back to its loading state and unmounted the login form — typing
      // an email, switching tabs to fetch a password, and coming back
      // lost what you had typed. The command bar's "Synced N min ago"
      // already shows how stale the data is.
      refetchOnWindowFocus: false,
    },
  },
});
