import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import AuthGate from "./AuthGate";
import "./index.css";
import "./theme.css";

// One shared query cache for the whole app — every tab's useQuery calls
// (tabs/*.jsx) key off it, so switching tabs and back reuses a still-fresh
// result instead of refetching, and identical queries fired from two
// places (e.g. OverviewTab and the shell both wanting /meta) de-dupe into
// one request. See docs/ROADMAP.md's Phase A.
const queryClient = new QueryClient();

ReactDOM.createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <QueryClientProvider client={queryClient}>
        <AuthGate />
      </QueryClientProvider>
    </BrowserRouter>
  </React.StrictMode>
);
