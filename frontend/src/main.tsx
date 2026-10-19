import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter } from "react-router-dom";
import { queryClient } from "./app/queryClient";
import { AppRoutes } from "./app/AppRoutes";
import { AuthGate } from "./app/AuthGate";
import { SubscriptionGate } from "./app/SubscriptionGate";
import { cleanLocationInPlace } from "./app/cleanLocation";
import "./styles/base.css";

// Before the router reads the location. A path carrying a newline —
// which is what a URL pasted with its line break still attached looks
// like — matches no route at all, not even the catch-all, and the app
// renders a blank screen.
cleanLocationInPlace();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <AuthGate>
          {/* Inside AuthGate: a subscription belongs to an
              organisation, and there is no organisation until someone
              has signed in. */}
          <SubscriptionGate>
            <AppRoutes />
          </SubscriptionGate>
        </AuthGate>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
