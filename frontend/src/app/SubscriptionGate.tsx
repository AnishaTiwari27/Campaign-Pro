import type { ReactNode } from "react";
import { useBilling } from "../api/billing";
import { BillingScreen } from "../features/billing/BillingScreen";
import { SkeletonBlock } from "../components/Skeleton";

/** Stands between a signed-in person and the app.
 *
 *  Sits inside AuthGate, not beside it: the subscription belongs to an
 *  organisation, and there is no organisation until there is a
 *  session. Above the router, so no screen behind the paywall ever
 *  mounts and fires a request the server would only answer with a 402.
 *
 *  The server decides. This reads `allowed` rather than re-deriving it
 *  from status and dates, so there is one rule and the browser cannot
 *  disagree with the gate that is actually enforcing it.
 */
export function SubscriptionGate({ children }: { children: ReactNode }) {
  const { data, isFetched, isError } = useBilling();

  if (!isFetched) {
    return (
      <div style={{ padding: 32 }}>
        <SkeletonBlock height={200} />
      </div>
    );
  }

  // If the check itself fails, let the app through. This is the one
  // place that fails open, and deliberately: the server gate is what
  // actually protects the data, and it fails closed. Blocking here as
  // well would turn a blip in one request into a paywall shown to a
  // paying customer.
  if (isError || !data) return <>{children}</>;

  if (!data.allowed) return <BillingScreen locked />;

  return <>{children}</>;
}
