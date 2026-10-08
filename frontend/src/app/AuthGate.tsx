import type { ReactNode } from "react";
import { useSession } from "../api/auth";
import { AuthPage } from "../features/auth/AuthPage";
import { SkeletonBlock } from "../components/Skeleton";

// Decides between the app and the sign-in screen. Sits above the router so
// no authenticated screen ever mounts — and therefore never fires a
// request — without a session.
export function AuthGate({ children }: { children: ReactNode }) {
  const { data: me, isError, isFetched } = useSession();

  // Only the very first check shows a placeholder. Keying off a fetch in
  // progress would unmount whatever is on screen every time the session
  // is re-checked, which discards anything typed into the login form.
  if (!isFetched) {
    return (
      <div style={{ padding: 32 }}>
        <SkeletonBlock height={200} />
      </div>
    );
  }

  // Any failure to resolve a session means sign in. A 401 is the normal
  // case, not an error state worth showing.
  if (isError || !me) return <AuthPage />;

  return <>{children}</>;
}
