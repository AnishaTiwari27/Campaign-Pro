import { Navigate, Outlet, Route, Routes } from "react-router-dom";
import { Suspense, lazy, type ReactNode } from "react";
import { useSession } from "../api/auth";
import { AppLayout } from "./AppLayout";
import { NotFoundPage } from "./NotFoundPage";
import { SkeletonBlock } from "../components/Skeleton";
import { OverviewPage } from "../features/overview/OverviewPage";

// Overview is imported eagerly: it is where "/" redirects, so every
// session lands on it and splitting it out would only add a round trip
// before the first paint. Everything else is reached by a deliberate
// click, which is the moment to fetch it.
//
// Detail pages are split from their list pages on purpose even though a
// click usually follows — the charts and panels they pull in are the
// heaviest part of the bundle, and someone who only ever looks at the
// Overview should not pay for the campaign detail's chart code.
const CampaignsPage = lazy(() => import("../features/campaigns/CampaignsPage").then((m) => ({ default: m.CampaignsPage })));
const CampaignDetailPage = lazy(() => import("../features/campaign-detail/CampaignDetailPage").then((m) => ({ default: m.CampaignDetailPage })));
const ApprovalsPage = lazy(() => import("../features/approvals/ApprovalsPage").then((m) => ({ default: m.ApprovalsPage })));
const CreatorsPage = lazy(() => import("../features/creators/CreatorsPage").then((m) => ({ default: m.CreatorsPage })));
const CreatorDetailPage = lazy(() => import("../features/creators/CreatorDetailPage").then((m) => ({ default: m.CreatorDetailPage })));
const EndorsementsPage = lazy(() => import("../features/endorsements/EndorsementsPage").then((m) => ({ default: m.EndorsementsPage })));
const EndorsementDetailPage = lazy(() => import("../features/endorsements/EndorsementDetailPage").then((m) => ({ default: m.EndorsementDetailPage })));
const RadarPage = lazy(() => import("../features/radar/RadarPage").then((m) => ({ default: m.RadarPage })));
const RegionsPage = lazy(() => import("../features/regions/RegionsPage").then((m) => ({ default: m.RegionsPage })));
const RegionDetailPage = lazy(() => import("../features/regions/RegionDetailPage").then((m) => ({ default: m.RegionDetailPage })));
const BenchmarksPage = lazy(() => import("../features/benchmarks/BenchmarksPage").then((m) => ({ default: m.BenchmarksPage })));
const SignalsPage = lazy(() => import("../features/signals/SignalsPage").then((m) => ({ default: m.SignalsPage })));
const ReportsPage = lazy(() => import("../features/reports/ReportsPage").then((m) => ({ default: m.ReportsPage })));
const ReportDetailPage = lazy(() => import("../features/reports/ReportDetailPage").then((m) => ({ default: m.ReportDetailPage })));
const SignIns = lazy(() => import("../features/account/SignIns").then((m) => ({ default: m.SignIns })));
const ComingSoon = lazy(() => import("../features/account/ComingSoon").then((m) => ({ default: m.ComingSoon })));
const BillingScreen = lazy(() => import("../features/billing/BillingScreen").then((m) => ({ default: m.BillingScreen })));
const SettingsPage = lazy(() => import("../features/settings/SettingsPage").then((m) => ({ default: m.SettingsPage })));
const UsersPage = lazy(() => import("../features/users/UsersPage").then((m) => ({ default: m.UsersPage })));

// AgencyOnly keeps a client account off the pages the rail already hides
// from them, so a typed URL or an old bookmark lands on the Overview
// instead. The server refuses the data either way — this is about not
// showing an empty, broken-looking page.
function AgencyOnly({ children }: { children: ReactNode }) {
  const { data: me } = useSession();
  if (me?.isClient) return <Navigate to="/overview" replace />;
  return <>{children}</>;
}

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<Navigate to="/overview" replace />} />
        <Route path="overview" element={<OverviewPage />} />
        {/* One boundary inside the layout rather than one per route: the
            rail and header stay painted while a chunk arrives, so a
            navigation reads as the page filling in rather than the app
            blanking. */}
        <Route
          element={
            <Suspense fallback={<SkeletonBlock height={420} />}>
              <Outlet />
            </Suspense>
          }
        >
          <Route path="campaigns" element={<CampaignsPage />} />
          <Route path="campaigns/:id" element={<CampaignDetailPage />} />
          <Route path="approvals" element={<AgencyOnly><ApprovalsPage /></AgencyOnly>} />
          <Route path="creators" element={<CreatorsPage />} />
          <Route path="creators/:id" element={<CreatorDetailPage />} />
          <Route path="endorsements" element={<EndorsementsPage />} />
          <Route path="endorsements/:id" element={<EndorsementDetailPage />} />
          <Route path="radar" element={<AgencyOnly><RadarPage /></AgencyOnly>} />
          <Route path="regions" element={<RegionsPage />} />
          <Route path="regions/:id" element={<RegionDetailPage />} />
          <Route path="signals" element={<SignalsPage />} />
          <Route path="benchmarks" element={<AgencyOnly><BenchmarksPage /></AgencyOnly>} />
          <Route path="reports" element={<ReportsPage />} />
          <Route path="reports/:id" element={<ReportDetailPage />} />
          <Route path="users" element={<UsersPage />} />
          <Route path="settings" element={<SettingsPage />} />
          <Route path="billing" element={<BillingScreen />} />
          <Route path="sign-ins" element={<SignIns />} />
          {/* Listed in the account menu, so they need somewhere to
              land. Each says plainly that it is not built. */}
          <Route
            path="soon/notifications"
            element={
              <ComingSoon
                title="Notifications"
                description="Alerts for flagged campaigns and approvals waiting on you will arrive here. Today those reach you by scheduled report instead."
              />
            }
          />
          <Route
            path="soon/ai-credits"
            element={
              <ComingSoon
                title="AI credits"
                description="Signal explanations are written by a model when one is configured, and assembled from the figures when not. There is no metering behind this yet."
              />
            }
          />
          <Route
            path="soon/affiliate"
            element={
              <ComingSoon
                title="Affiliate"
                description="Nothing here yet. There is no referral programme to join."
              />
            }
          />
          <Route
            path="soon/troubleshooting"
            element={
              <ComingSoon
                title="Troubleshooting"
                description="Health, ingestion runs and recent errors would live here. For now the health of each source is on the Overview, and ingestion runs are under Settings."
              />
            }
          />
        </Route>
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
