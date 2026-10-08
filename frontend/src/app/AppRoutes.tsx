import { Navigate, Route, Routes } from "react-router-dom";
import type { ReactNode } from "react";
import { useSession } from "../api/auth";
import { AppLayout } from "./AppLayout";
import { NotFoundPage } from "./NotFoundPage";
import { CampaignsPage } from "../features/campaigns/CampaignsPage";
import { CampaignDetailPage } from "../features/campaign-detail/CampaignDetailPage";
import { ApprovalsPage } from "../features/approvals/ApprovalsPage";
import { OverviewPage } from "../features/overview/OverviewPage";
import { CreatorsPage } from "../features/creators/CreatorsPage";
import { CreatorDetailPage } from "../features/creators/CreatorDetailPage";
import { RegionsPage } from "../features/regions/RegionsPage";
import { RegionDetailPage } from "../features/regions/RegionDetailPage";
import { BenchmarksPage } from "../features/benchmarks/BenchmarksPage";
import { SignalsPage } from "../features/signals/SignalsPage";
import { ReportsPage } from "../features/reports/ReportsPage";
import { ReportDetailPage } from "../features/reports/ReportDetailPage";
import { SettingsPage } from "../features/settings/SettingsPage";

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
        <Route path="campaigns" element={<CampaignsPage />} />
        <Route path="campaigns/:id" element={<CampaignDetailPage />} />
        <Route path="approvals" element={<AgencyOnly><ApprovalsPage /></AgencyOnly>} />
        <Route path="creators" element={<CreatorsPage />} />
        <Route path="creators/:id" element={<CreatorDetailPage />} />
        <Route path="regions" element={<RegionsPage />} />
        <Route path="regions/:id" element={<RegionDetailPage />} />
        <Route path="signals" element={<SignalsPage />} />
        <Route path="benchmarks" element={<AgencyOnly><BenchmarksPage /></AgencyOnly>} />
        <Route path="reports" element={<ReportsPage />} />
        <Route path="reports/:id" element={<ReportDetailPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
