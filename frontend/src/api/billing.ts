import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { get, post } from "./client";
import { queryKeys } from "./queryKeys";

export type Plan = {
  id: string;
  name: string;
  amountPaise: number;
  interval: string;
  blurb: string;
  includes: string[];
};

export type BillingState = {
  plan: string;
  status: "none" | "trialing" | "active" | "past_due" | "cancelled";
  /** Whether the app is usable. Sent by the server rather than derived
   *  here, so the rule that decides access lives in one place. */
  allowed: boolean;
  /** -1 where the subscription does not expire. */
  daysLeft: number;
  expiresAt?: string;
  /** False where no payment provider is configured on this deployment. */
  payable: boolean;
  plans: Plan[];
};

/** Reachable even while the organisation is locked out — it is the
 *  endpoint that lets them stop being locked out. */
export function useBilling() {
  return useQuery({
    queryKey: queryKeys.billing(),
    queryFn: () => get<BillingState>("/billing"),
    retry: false,
    staleTime: 60_000,
  });
}

export function useStartTrial() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => post<BillingState>("/billing/trial"),
    onSuccess: () => qc.invalidateQueries(),
  });
}

export function useCheckout() {
  return useMutation({
    mutationFn: (plan: string) => post<{ checkoutUrl: string }>("/billing/checkout", { plan }),
    onSuccess: ({ checkoutUrl }) => {
      // A full navigation, not a fetch: the checkout is Razorpay's own
      // page, which is what keeps card details off this origin.
      window.location.assign(checkoutUrl);
    },
  });
}

/** Rupees, from paise, without the trailing ".00" that every one of
 *  these prices would otherwise carry. */
export function formatPrice(paise: number): string {
  const rupees = paise / 100;
  return `₹${rupees.toLocaleString("en-IN", {
    minimumFractionDigits: rupees % 1 === 0 ? 0 : 2,
    maximumFractionDigits: 2,
  })}`;
}
