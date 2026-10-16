import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SubscriptionGate } from "./SubscriptionGate";
import type { BillingState } from "../api/billing";

const plans = [
  { id: "starter", name: "Starter", amountPaise: 499900, interval: "monthly", blurb: "", includes: [] },
];

function answerWith(state: Partial<BillingState> | "error") {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => {
      if (state === "error") return new Response("nope", { status: 500 });
      const body: BillingState = {
        plan: "starter",
        status: "active",
        allowed: true,
        daysLeft: 10,
        payable: false,
        plans,
        ...state,
      };
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }),
  );
}

function renderGate() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={qc}>
      <SubscriptionGate>
        <div>the dashboard</div>
      </SubscriptionGate>
    </QueryClientProvider>,
  );
}

afterEach(() => vi.unstubAllGlobals());

describe("SubscriptionGate", () => {
  it("shows the app when the organisation is allowed", async () => {
    answerWith({ allowed: true });
    renderGate();
    expect(await screen.findByText("the dashboard")).toBeDefined();
  });

  it("replaces the app with the paywall when it is not", async () => {
    answerWith({ allowed: false, status: "cancelled" });
    renderGate();
    expect(await screen.findByText("Your access has paused")).toBeDefined();
    // Not merely hidden. Anything still mounted would fire the
    // requests the paywall exists to refuse.
    expect(screen.queryByText("the dashboard")).toBeNull();
  });

  // The server gate is what actually protects the data, and it fails
  // closed. This one fails open on purpose: a blip on one request
  // should not show a paywall to somebody who has paid.
  it("lets the app through when the check itself fails", async () => {
    answerWith("error");
    renderGate();
    expect(await screen.findByText("the dashboard")).toBeDefined();
  });

  it("trusts the server's verdict rather than re-deriving it", async () => {
    // Dates that look fine, but the server said no. One rule, in one
    // place — the browser does not get a second opinion.
    answerWith({ allowed: false, status: "active", daysLeft: 300 });
    renderGate();
    expect(await screen.findByText("Your access has paused")).toBeDefined();
  });
});
