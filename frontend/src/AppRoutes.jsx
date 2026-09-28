import React, { Suspense, lazy } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import DashboardLayout from "./DashboardLayout";

// Each tab is its own chunk, loaded only when its route is actually
// visited — Vite code-splits automatically on a dynamic import(), no
// bundler config needed. Cuts the single ~570kB bundle every build warned
// about (docs/ROADMAP.md's Phase A) into one chunk per tab.
//
// Campaigns/Region/Reports were folded into Overview (see
// tabs/OverviewTab.jsx's own header comment) once the ribbon/spotlight
// redesign made their separate tabs feel like a different design
// language — this is down to two routes, not five, on purpose.
const OverviewTab = lazy(() => import("./tabs/OverviewTab"));
const SettingsTab = lazy(() => import("./tabs/SettingsTab"));

// null fallback, not a spinner: each tab's own useQuery already renders
// its panel shell immediately and dims it via ctp-fetching while loading
// (the same "fetching" treatment every tab already had) — a Suspense
// spinner on top of that would just be a second, redundant loading state
// for the ~100ms a chunk takes to load on a warm cache.
function TabRoute({ children }) {
  return <Suspense fallback={null}>{children}</Suspense>;
}

export default function AppRoutes({ onSignOut }) {
  return (
    <Routes>
      <Route path="/" element={<DashboardLayout onSignOut={onSignOut} />}>
        <Route index element={<Navigate to="/overview" replace />} />
        <Route path="overview" element={<TabRoute><OverviewTab /></TabRoute>} />
        <Route path="settings" element={<TabRoute><SettingsTab /></TabRoute>} />
        <Route path="*" element={<Navigate to="/overview" replace />} />
      </Route>
    </Routes>
  );
}
