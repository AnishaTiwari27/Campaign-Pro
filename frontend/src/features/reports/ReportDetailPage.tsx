import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useReport, useReportRuns, useRunReport, useTestReport, useUpdateReport } from "../../api/reports";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonBlock } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { CADENCE_OPTIONS, REPORT_COLUMNS, type Cadence } from "../../api/types";
import { formatDateTime } from "../../lib/format";
import "./ReportDetailPage.css";

function parseRecipients(s: string): string[] {
  return s
    .split(",")
    .map((e) => e.trim())
    .filter(Boolean);
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function ReportDetailPage() {
  const { id = "" } = useParams<{ id: string }>();
  const { data: report, isLoading, isError } = useReport(id);
  const { data: runsData } = useReportRuns(id);
  const update = useUpdateReport(id);
  const run = useRunReport(id);
  const test = useTestReport(id);

  const [recipientsText, setRecipientsText] = useState("");
  const [recipientsError, setRecipientsError] = useState<string | null>(null);

  useEffect(() => {
    if (report) setRecipientsText(report.recipients.join(", "));
  }, [report?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  useSetBreadcrumbs([{ label: "Reports", href: "/reports" }, { label: report?.name ?? "…" }]);

  if (isLoading) return <SkeletonBlock height={400} />;

  if (isError || !report) {
    return (
      <EmptyState
        title="Report not found"
        description="This report doesn't exist, or the link is out of date."
        action={
          <Link to="/reports" className="btn btn-primary">
            Back to Reports
          </Link>
        }
      />
    );
  }

  function saveRecipients() {
    const emails = parseRecipients(recipientsText);
    const invalid = emails.find((e) => !EMAIL_RE.test(e));
    if (invalid) {
      setRecipientsError(`"${invalid}" doesn't look like a valid email.`);
      return;
    }
    setRecipientsError(null);
    update.mutate({ recipients: emails });
  }

  function toggleColumn(col: string) {
    const next = report!.columns.includes(col) ? report!.columns.filter((c) => c !== col) : [...report!.columns, col];
    update.mutate({ columns: next });
  }

  const runs = runsData?.items ?? [];

  return (
    <div className="report-detail-page">
      <div className="page-header">
        <div>
          <h1>{report.name}</h1>
          <p className="report-detail-scope">Scope: {report.scopeLabel}</p>
        </div>
        <div className="page-header-actions">
          <button
            type="button"
            role="switch"
            aria-checked={report.enabled}
            className={`report-toggle${report.enabled ? " report-toggle-on" : ""}`}
            onClick={() => update.mutate({ enabled: !report.enabled })}
          >
            <span className="report-toggle-knob" />
          </button>
          <button type="button" className="btn" onClick={() => test.mutate()} disabled={test.isPending}>
            Send a test
          </button>
          <button type="button" className="btn btn-primary" onClick={() => run.mutate()} disabled={run.isPending}>
            Run now
          </button>
        </div>
      </div>

      <div className="report-detail-body">
        <div className="card report-panel">
          <h4>Recipients</h4>
          <input
            className="input"
            style={{ width: "100%" }}
            value={recipientsText}
            onChange={(e) => setRecipientsText(e.target.value)}
            onBlur={saveRecipients}
            placeholder="name@company.com, name2@company.com"
          />
          {recipientsError && <p className="report-field-error">{recipientsError}</p>}
        </div>

        <div className="card report-panel">
          <h4>Cadence</h4>
          <select
            className="select"
            style={{ width: "100%" }}
            value={report.cadence}
            onChange={(e) => update.mutate({ cadence: e.target.value as Cadence })}
          >
            {CADENCE_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>

        <div className="card report-panel">
          <h4>Columns</h4>
          <div className="report-columns-grid">
            {REPORT_COLUMNS.map((col) => (
              <label key={col} className="report-column-checkbox">
                <input type="checkbox" checked={report.columns.includes(col)} onChange={() => toggleColumn(col)} />
                {col}
              </label>
            ))}
          </div>
        </div>

        <div className="card report-panel">
          <h4>Run history</h4>
          {runs.length === 0 ? (
            <p className="side-panel-muted">No runs yet.</p>
          ) : (
            <table className="report-runs-table">
              <thead>
                <tr>
                  <th>Ran at</th>
                  <th>Result</th>
                  <th>Rows</th>
                </tr>
              </thead>
              <tbody>
                {runs.map((r, i) => (
                  <tr key={i}>
                    <td>{formatDateTime(r.ranAt)}</td>
                    <td>{r.result}</td>
                    <td className="mono">{r.rowCount}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}
