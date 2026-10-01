import { useHealth, useMe } from "../../api/misc";
import { useOverview } from "../../api/overview";
import { useRegions } from "../../api/regions";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { useThemeStore, type ThemeChoice } from "../../app/useTheme";
import { useUIStore } from "../../app/useUIStore";
import { SkeletonBlock } from "../../components/Skeleton";
import { CATEGORIES } from "../../lib/constants";
import "./SettingsPage.css";

const THEME_OPTIONS: { value: ThemeChoice; label: string }[] = [
  { value: "auto", label: "Auto" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

export function SettingsPage() {
  useSetBreadcrumbs([{ label: "Settings" }]);

  const { data: me } = useMe();
  const { data: health } = useHealth();
  const { data: overview } = useOverview();
  const { data: regions } = useRegions();
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const density = useUIStore((s) => s.density);
  const setDensity = useUIStore((s) => s.setDensity);

  if (!me) return <SkeletonBlock height={320} />;

  const totalCampaigns = regions?.items.reduce((sum, r) => sum + r.count, 0) ?? 0;

  return (
    <div className="settings-page">
      <div className="page-header">
        <h1>Settings</h1>
      </div>

      <div className="settings-grid">
        <section className="card settings-panel">
          <h4>Your access</h4>
          <dl className="settings-rows">
            <div className="settings-row">
              <dt>Email</dt>
              <dd>{me.email}</dd>
            </div>
            <div className="settings-row">
              <dt>Role</dt>
              <dd className="settings-capitalize">{me.role}</dd>
            </div>
            <div className="settings-row">
              <dt>Approve rights</dt>
              <dd>{me.canApprove ? "Can approve campaigns" : "View only"}</dd>
            </div>
            <div className="settings-row">
              <dt>Waiting on you</dt>
              <dd className="mono">{overview?.pendingCount ?? 0}</dd>
            </div>
          </dl>
        </section>

        <section className="card settings-panel">
          <h4>Appearance</h4>
          <div className="settings-rows">
            <div className="settings-row">
              <dt>Theme</dt>
              <dd>
                <div className="settings-segmented" role="radiogroup" aria-label="Theme">
                  {THEME_OPTIONS.map((opt) => (
                    <button
                      key={opt.value}
                      type="button"
                      role="radio"
                      aria-checked={theme === opt.value}
                      className={`settings-segmented-opt${theme === opt.value ? " settings-segmented-opt-active" : ""}`}
                      onClick={() => setTheme(opt.value)}
                    >
                      {opt.label}
                    </button>
                  ))}
                </div>
              </dd>
            </div>
            <div className="settings-row">
              <dt>Compact tables</dt>
              <dd>
                <button
                  type="button"
                  role="switch"
                  aria-checked={density === "compact"}
                  aria-label="Compact tables"
                  className={`report-toggle${density === "compact" ? " report-toggle-on" : ""}`}
                  onClick={() => setDensity(density === "compact" ? "comfortable" : "compact")}
                >
                  <span className="report-toggle-knob" />
                </button>
              </dd>
            </div>
          </div>
        </section>

        <section className="card settings-panel">
          <h4>Workspace</h4>
          <dl className="settings-rows">
            <div className="settings-row">
              <dt>Market</dt>
              <dd>India · all regions</dd>
            </div>
            <div className="settings-row">
              <dt>Data window</dt>
              <dd>Last 21 days of delivery</dd>
            </div>
            <div className="settings-row">
              <dt>Currency</dt>
              <dd>INR (₹), lakh &amp; crore units</dd>
            </div>
            <div className="settings-row">
              <dt>Tracked</dt>
              <dd className="mono">
                {totalCampaigns} campaigns · {regions?.items.length ?? 0} regions · {CATEGORIES.length} categories
              </dd>
            </div>
          </dl>
        </section>

        <section className="card settings-panel">
          <h4>Data sources</h4>
          <dl className="settings-rows">
            {(health?.sources ?? []).map((s) => (
              <div key={s.name} className="settings-row">
                <dt>{s.name}</dt>
                <dd>
                  <span className={`pill ${s.status === "healthy" ? "pill-live" : "pill-paused"}`}>
                    {s.status === "healthy" ? "Healthy" : "Degraded"}
                  </span>
                </dd>
              </div>
            ))}
          </dl>
        </section>
      </div>
    </div>
  );
}
