import { Navigate, Route, Routes } from "react-router-dom";
import { AppLayout } from "./AppLayout";
import { NotFoundPage } from "./NotFoundPage";
import { CampaignsPage } from "../features/campaigns/CampaignsPage";
import { CampaignDetailPage } from "../features/campaign-detail/CampaignDetailPage";
import { ApprovalsPage } from "../features/approvals/ApprovalsPage";
import { OverviewPage } from "../features/overview/OverviewPage";
import { RegionsPage } from "../features/regions/RegionsPage";
import { RegionDetailPage } from "../features/regions/RegionDetailPage";
import { BenchmarksPage } from "../features/benchmarks/BenchmarksPage";
import { ReportsPage } from "../features/reports/ReportsPage";
import { ReportDetailPage } from "../features/reports/ReportDetailPage";
import { SettingsPage } from "../features/settings/SettingsPage";

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<Navigate to="/overview" replace />} />
        <Route path="overview" element={<OverviewPage />} />
        <Route path="campaigns" element={<CampaignsPage />} />
        <Route path="campaigns/:id" element={<CampaignDetailPage />} />
        <Route path="approvals" element={<ApprovalsPage />} />
        <Route path="regions" element={<RegionsPage />} />
        <Route path="regions/:id" element={<RegionDetailPage />} />
        <Route path="benchmarks" element={<BenchmarksPage />} />
        <Route path="reports" element={<ReportsPage />} />
        <Route path="reports/:id" element={<ReportDetailPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
