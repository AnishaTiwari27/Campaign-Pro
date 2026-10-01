import { Link, useNavigate } from "react-router-dom";
import { useCreateReport, useReports, useToggleReportEnabled } from "../../api/reports";
import { useLastFilterStore } from "../../app/lastFilterStore";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonRows } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { describeScope, paramsToScope } from "../../lib/reportScope";
import { formatDateTime } from "../../lib/format";
import "./ReportsPage.css";

export function ReportsPage() {
  useSetBreadcrumbs([{ label: "Reports" }]);
  const { data, isLoading } = useReports();
  const toggle = useToggleReportEnabled();
  const createReport = useCreateReport();
  const lastCampaignsParams = useLastFilterStore((s) => s.lastCampaignsParams);
  const navigate = useNavigate();

  const items = data?.items ?? [];

  function createFromCurrentFilters() {
    createReport.mutate(
      {
        name: "New report",
        cadence: "weekly_mon_9",
        recipients: [],
        scopeFilters: paramsToScope(lastCampaignsParams),
        scopeLabel: describeScope(lastCampaignsParams),
      },
      { onSuccess: (rep) => navigate(`/reports/${rep.id}`) },
    );
  }

  return (
    <div className="reports-page">
      <div className="page-header">
        <h1>Reports</h1>
        <button type="button" className="btn btn-primary" onClick={createFromCurrentFilters} disabled={createReport.isPending}>
          New from current filters
        </button>
      </div>

      {isLoading ? (
        <SkeletonRows count={3} height={80} />
      ) : items.length === 0 ? (
        <EmptyState title="No reports yet" description="Create one from your current Campaigns filters." />
      ) : (
        <div className="reports-list">
          {items.map((r) => (
            <div key={r.id} className="report-row card">
              <div className="report-row-main">
                <div className="report-row-title-line">
                  <span className="report-row-name">{r.name}</span>
                  <span className={`pill ${r.enabled ? "pill-live" : "pill-ended"}`}>{r.enabled ? "On" : "Paused"}</span>
                </div>
                <div className="report-row-meta">
                  {r.cadenceLabel} · {r.scopeLabel} · {r.recipients.length} recipient{r.recipients.length === 1 ? "" : "s"}
                </div>
                <div className="report-row-last-run">
                  {r.lastRun ? `Last run ${formatDateTime(r.lastRun.ranAt)} · ${r.lastRun.result}` : "Never run"}
                </div>
              </div>
              <div className="report-row-actions">
                <button
                  type="button"
                  role="switch"
                  aria-checked={r.enabled}
                  aria-label={r.enabled ? "Disable report" : "Enable report"}
                  className={`report-toggle${r.enabled ? " report-toggle-on" : ""}`}
                  onClick={() => toggle.mutate({ id: r.id, enabled: !r.enabled })}
                >
                  <span className="report-toggle-knob" />
                </button>
                <Link to={`/reports/${r.id}`} className="btn">
                  Open
                </Link>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
