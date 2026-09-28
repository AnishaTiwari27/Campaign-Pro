import React, { useEffect, useMemo, useRef, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  AreaChart, Area, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from "recharts";
import {
  AlertTriangle, ChevronLeft, ChevronRight, ClipboardCheck, Inbox, TrendingUp, Wallet,
} from "lucide-react";
import { getTrend, listCampaigns, getMeta, getAnomalies, getBenchmark, getRegions } from "../api/client";
import { fmtDate, fmtReach, fmtMoney, initialsOf } from "../format";
import { useFilters } from "../filters";
import { PACING_COLOR } from "../pacing";
import { APPROVAL_COLOR } from "../approval";
import SubjectTypeIcon from "../components/SubjectTypeIcon";
import { AD_TYPE_COLOR } from "../adTypeColors";

// This is now the whole product's one screen (Settings aside) — folded in
// from what used to be four separate tabs (Overview/Campaigns/Region/
// Reports) at the user's own direction, once the ribbon/spotlight
// redesign made the old exhaustive table, the region bar chart, and the
// benchmark table all feel like leftovers from a different design
// language. Concretely:
//   - Campaigns' Spotlight + ribbons lead the page (discovery-first).
//   - Region's bar chart becomes a "Regions" ribbon (still click-to-filter).
//   - Reports' benchmark table becomes "Top brands" (still sortable, via
//     the ribbon header's Count/Reach toggle; still shows a trend
//     sparkline; platforms/last-seen move to a hover tooltip rather than
//     dedicated columns) — CSV export and "email this report" move to
//     the global top bar (DashboardLayout.jsx), since they act on
//     whatever's currently filtered, not on any one section of this page.
//   - The trend-by-ad-type chart and the alert-chips summary (this tab's
//     own original content) stay, chart moved to the end as supporting
//     detail behind the discovery content above it.

// How many of the current filtered set a page of campaigns actually
// covers — one fetch backs the alert-chip counts, every ribbon below,
// and the header's own count. 100 is the API's own max `limit` and
// comfortably covers this app's real campaign count today (see
// docs/ROADMAP.md's Phase A) — a documented bounded sample, not a true
// upper bound; revisit with a real sorted/paginated endpoint if that
// count ever exceeds it.
const BATCH_SAMPLE_SIZE = 100;
const RIBBON_ITEM_CAP = 10; // how many tiles a ribbon shows at most
const RIBBON_SCROLL_PX = 358; // ~2 tiles' worth (166px tile + 13px gap) per click

// Campaign dates are "YYYY-MM-DD" (see format.js's fmtDate) — parsed the
// same defensive way here.
function startedLabel(iso) {
  const start = new Date(iso.length === 10 ? `${iso}T00:00:00` : iso);
  const days = Math.max(0, Math.round((Date.now() - start.getTime()) / 86400000));
  if (days === 0) return "today";
  if (days === 1) return "1 day ago";
  return `${days} days ago`;
}

// Compact bars, same rendering ReportsTab's own Sparkline used before
// this merge — renders nothing until a subject has at least one
// benchmark_snapshots point (see docs/ROADMAP.md's Phase D), rather than
// a flat, meaningless line.
function Sparkline({ points }) {
  if (!points || points.length === 0) return null;
  const max = Math.max(...points, 1);
  return (
    <svg className="ctp-tile-spark" viewBox="0 0 100 16" preserveAspectRatio="none">
      {points.map((v, idx) => {
        const w = 100 / points.length;
        const h = Math.max(2, (v / max) * 14);
        return <rect key={idx} x={idx * w + 0.6} y={16 - h} width={w - 1.2} height={h} fill={idx === points.length - 1 ? "var(--secondary)" : "var(--line)"} />;
      })}
    </svg>
  );
}

// A horizontally-scrollable row of tiles — nav buttons live in a padding
// "gutter" outside the tile flow (see DashboardLayout.jsx's
// .ctp-ribbon-track-wrap comment) so they never clip a tile's text.
// Renders nothing once its item list is empty, rather than an empty
// shell. `right` overrides the plain count label with an interactive
// control (Top brands' Count/Reach sort toggle).
function Ribbon({ label, count, right, items, renderItem }) {
  const trackRef = useRef(null);
  if (items.length === 0) return null;
  return (
    <div>
      <div className="ctp-ribbon-head">
        <h3 className="ctp-display">{label}</h3>
        {right ?? <span className="count">{count}</span>}
      </div>
      <div className="ctp-ribbon-track-wrap">
        <button
          type="button" className="ctp-ribbon-nav prev" aria-label={`Scroll ${label} left`}
          onClick={() => trackRef.current?.scrollBy({ left: -RIBBON_SCROLL_PX, behavior: "smooth" })}
        >
          <ChevronLeft size={14} />
        </button>
        <div className="ctp-ribbon-track" ref={trackRef}>
          {items.map(renderItem)}
        </div>
        <button
          type="button" className="ctp-ribbon-nav next" aria-label={`Scroll ${label} right`}
          onClick={() => trackRef.current?.scrollBy({ left: RIBBON_SCROLL_PX, behavior: "smooth" })}
        >
          <ChevronRight size={14} />
        </button>
      </div>
    </div>
  );
}

// One campaign, real row shape straight from /campaigns — the same object
// CampaignDrawer already expects via onSelect. `stat` is a plain trailing
// line (e.g. reach, recency); `badge` replaces it with a colored label
// (e.g. pending review) when the two aren't both relevant.
function CampaignTile({ c, flagged, stat, badge, onSelect }) {
  return (
    <div className="ctp-tile" style={{ "--stripe": AD_TYPE_COLOR[c.adType] }} onClick={() => onSelect(c)}>
      <div className="ctp-tile-art">
        <span className="ctp-tile-status"><span className="dot" />{c.status}</span>
        {flagged && (
          <span className="ctp-tile-flag" title="Flagged by anomaly detection">
            <AlertTriangle size={10} />
          </span>
        )}
        <span className="ctp-tile-initials ctp-display">{initialsOf(c.subject)}</span>
      </div>
      <div className="ctp-tile-body">
        <div className="ctp-tile-name"><SubjectTypeIcon type={c.subjectType} /><span>{c.subject}</span></div>
        <div className="ctp-tile-meta">{c.adType} · {c.region}</div>
        {badge ? (
          <div className={`ctp-tile-badge ${badge.variant}`}>{badge.label}</div>
        ) : (
          <div className="ctp-tile-stat">{stat}</div>
        )}
      </div>
    </div>
  );
}

// One subject aggregate, from /benchmark — a different shape from
// CampaignTile on purpose: there's no single status/ad type/anomaly flag
// that honestly applies to a brand as a whole, so this tile doesn't
// invent one. Platforms and last-seen (real columns on the old Reports
// table) live in the hover tooltip rather than more on-card text; the
// sparkline is the same benchmark_snapshots trend that table's own
// Sparkline column showed. Clicking drills the whole page into that
// subject (setSubject), the same "focus subject" filter the drawer's own
// link already sets.
function SubjectTile({ row, onSelect }) {
  return (
    <div
      className="ctp-tile" style={{ "--stripe": "var(--secondary)" }} onClick={() => onSelect(row.subject)}
      title={`${row.platforms.join(", ")} · last seen ${fmtDate(row.last)}`}
    >
      <div className="ctp-tile-art">
        <span className="ctp-tile-initials ctp-display">{initialsOf(row.subject)}</span>
      </div>
      <div className="ctp-tile-body">
        <div className="ctp-tile-name"><span>{row.subject}</span></div>
        <div className="ctp-tile-meta">{row.category}</div>
        <div className="ctp-tile-stat">{row.count} campaign{row.count === 1 ? "" : "s"} · {fmtReach(row.reach)}</div>
        <Sparkline points={row.trend} />
      </div>
    </div>
  );
}

// One region, from /regions (always all 6, zero-filled — see
// api/client.js). Replaces the old Region tab's bar chart: same
// click-to-toggle-filter behavior, same --data-accent identity that
// chart's own bars used (a single-series magnitude color, kept separate
// from the ad-type palette and the brand accent — see theme.css).
function RegionTile({ d, total, active, onToggle }) {
  const pct = total > 0 ? Math.round((d.campaigns / total) * 100) : 0;
  return (
    <div
      className="ctp-tile" onClick={() => onToggle(d.region)}
      style={{ "--stripe": "var(--data-accent)", borderColor: active ? "var(--ink)" : undefined }}
    >
      <div className="ctp-tile-art"><span className="ctp-tile-initials ctp-display">{initialsOf(d.region)}</span></div>
      <div className="ctp-tile-body">
        <div className="ctp-tile-name"><span style={{ fontWeight: active ? 800 : 700 }}>{d.region}</span></div>
        <div className="ctp-tile-meta">{pct}% of all campaigns</div>
        <div className="ctp-tile-stat">{d.campaigns} campaign{d.campaigns === 1 ? "" : "s"}</div>
      </div>
    </div>
  );
}

export default function OverviewTab() {
  const { onSelect, onError } = useOutletContext();
  const { filters, region, setRegion, setSubject } = useFilters();
  const [chartMode, setChartMode] = useState("area");
  const [brandSort, setBrandSort] = useState("count"); // "count" | "reach" — Top brands' sort toggle

  const metaQuery = useQuery({ queryKey: ["meta"], queryFn: getMeta });
  const adTypes = metaQuery.data?.adTypes ?? [];

  const trendQuery = useQuery({ queryKey: ["trend", filters], queryFn: () => getTrend(filters) });
  // One fetch backs the header count, the alert chips, and every
  // campaign-level ribbon below — see BATCH_SAMPLE_SIZE's comment.
  const batchQuery = useQuery({
    queryKey: ["campaigns", { ...filters, limit: BATCH_SAMPLE_SIZE }],
    queryFn: () => listCampaigns({ ...filters, limit: BATCH_SAMPLE_SIZE }),
  });
  // A global "what needs attention" signal, not scoped to the current
  // filters — see api/client.js's getAnomalies doc comment. Same
  // ["anomalies"] key CampaignDrawer uses, so React Query shares one
  // request rather than issuing two.
  const anomaliesQuery = useQuery({ queryKey: ["anomalies"], queryFn: getAnomalies });
  const regionsQuery = useQuery({ queryKey: ["regions", filters], queryFn: () => getRegions(filters) });
  // "Top brands" is sourced from analytics-service's own SQL-side
  // aggregation (GET /benchmark) rather than re-counted client-side over
  // the batch above — that's the service already doing this the right
  // way (docs/ROADMAP.md's Phase A). Skipped entirely (never
  // force-overridden) when the shared Type filter is already pinned to
  // People — showing brands under a "People"-only filter would
  // contradict the filter the user just set, rather than just being an
  // empty ribbon.
  const showBrandsRibbon = filters.subjectType !== "person";
  const benchmarkQuery = useQuery({
    queryKey: ["benchmark", { ...filters, subjectType: "brand", sort: brandSort }],
    queryFn: () => getBenchmark({ ...filters, subjectType: "brand", sort: brandSort }),
    enabled: showBrandsRibbon,
  });

  useEffect(() => {
    onError(
      trendQuery.error?.message ?? batchQuery.error?.message ?? anomaliesQuery.error?.message ??
      regionsQuery.error?.message ?? benchmarkQuery.error?.message ?? null
    );
  }, [trendQuery.error, batchQuery.error, anomaliesQuery.error, regionsQuery.error, benchmarkQuery.error, onError]);

  const trendData = trendQuery.data ?? [];
  const batch = batchQuery.data?.data ?? [];
  const total = batchQuery.data?.total ?? 0;
  const loading = trendQuery.isFetching || batchQuery.isFetching || anomaliesQuery.isFetching || regionsQuery.isFetching || benchmarkQuery.isFetching;

  const anomalyByID = useMemo(() => {
    const map = new Map();
    for (const a of anomaliesQuery.data?.campaigns ?? []) map.set(a.id, a.reason);
    return map;
  }, [anomaliesQuery.data]);

  // Both pacing and approval alerts come from the same batch above — one
  // fetch, two derived counts, rather than a second network call.
  const sampleAlerts = useMemo(() => ({
    over: batch.filter((c) => c.pacing === "over").length,
    under: batch.filter((c) => c.pacing === "under").length,
    pending: batch.filter((c) => c.approvalStatus === "pending").length,
  }), [batch]);

  const anomalyCount = anomaliesQuery.data?.campaigns?.length ?? 0;
  const staleCategoryCount = anomaliesQuery.data?.staleCategories?.length ?? 0;

  // The leading ad format this period — summed straight out of trendData,
  // which is already fetched for the chart below; no separate call for it.
  const leadingAdType = useMemo(() => {
    const totals = {};
    for (const row of trendData) {
      for (const a of adTypes) totals[a] = (totals[a] || 0) + (row[a] || 0);
    }
    const [name, count] = Object.entries(totals).sort((a, b) => b[1] - a[1])[0] || [];
    return count > 0 ? { name, count } : null;
  }, [trendData, adTypes]);

  // Trending — every campaign in the current filtered batch that anomaly
  // detection has flagged (a global, unfiltered signal — see
  // api/client.js's getAnomalies doc comment — intersected here against
  // whatever the shared filter bar currently shows).
  const trending = useMemo(
    () => batch.filter((c) => anomalyByID.has(c.id)).slice(0, RIBBON_ITEM_CAP),
    [batch, anomalyByID]
  );
  const pendingReview = useMemo(
    () => batch.filter((c) => c.approvalStatus === "pending").slice(0, RIBBON_ITEM_CAP),
    [batch]
  );
  const biggestByReach = useMemo(() => [...batch].sort((a, b) => b.reach - a.reach).slice(0, RIBBON_ITEM_CAP), [batch]);
  // /campaigns already returns newest start_date first — no re-sort needed.
  const latestTracked = useMemo(() => batch.slice(0, RIBBON_ITEM_CAP), [batch]);
  const peopleCampaigns = useMemo(
    () => batch.filter((c) => c.subjectType === "person").slice(0, RIBBON_ITEM_CAP),
    [batch]
  );
  const topBrands = useMemo(() => (benchmarkQuery.data ?? []).slice(0, RIBBON_ITEM_CAP), [benchmarkQuery.data]);

  const regionsTotal = useMemo(() => (regionsQuery.data ?? []).reduce((sum, d) => sum + d.campaigns, 0), [regionsQuery.data]);
  // Biggest region first — same "largest first" convention every other
  // ribbon here uses, rather than the fixed catalog-insertion order the
  // old Region tab's chart happened to render in.
  const regionTiles = useMemo(() => [...(regionsQuery.data ?? [])].sort((a, b) => b.campaigns - a.campaigns), [regionsQuery.data]);
  function toggleRegion(name) {
    setRegion(region === name ? "All" : name);
  }

  // Spotlight — the single most notable campaign right now: the
  // highest-reach anomaly in view, or, once none are flagged, simply the
  // biggest campaign in view, honestly relabeled rather than implying an
  // anomaly that isn't there.
  const spotlight = useMemo(() => {
    const flagged = trending.length > 0 ? [...trending].sort((a, b) => b.reach - a.reach)[0] : null;
    if (flagged) return { campaign: flagged, isAnomaly: true, reason: anomalyByID.get(flagged.id) };
    if (biggestByReach.length > 0) return { campaign: biggestByReach[0], isAnomaly: false };
    return null;
  }, [trending, biggestByReach, anomalyByID]);

  return (
    <div className={`px-6 pb-8 ${loading ? "ctp-fetching" : ""}`}>
      <div className="flex items-baseline justify-between gap-4 mb-4 pb-4 flex-wrap" style={{ borderBottom: "1px solid var(--line)" }}>
        <div>
          <h1 className="ctp-display text-xl font-bold mb-1">Overview</h1>
          <p className="text-xs" style={{ color: "var(--muted)" }}>
            {total > 0 ? `${total} campaigns match your current filters.` : "No campaigns match your current filters."}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {leadingAdType && (
            // Colored by the ad type's own series color, not the brand
            // accent — this callout names a specific series in the chart
            // below, so it should read as that series.
            <div className="flex items-center gap-2 px-3 py-2 text-xs shrink-0" style={{ background: `color-mix(in srgb, ${AD_TYPE_COLOR[leadingAdType.name]} 14%, var(--surface))`, borderRadius: 8 }}>
              <TrendingUp size={14} style={{ color: AD_TYPE_COLOR[leadingAdType.name] }} />
              <span>
                <strong style={{ color: AD_TYPE_COLOR[leadingAdType.name] }}>{leadingAdType.name}</strong> is leading —{" "}
                {leadingAdType.count} campaign{leadingAdType.count === 1 ? "" : "s"} this period
              </span>
            </div>
          )}
          {(sampleAlerts.over > 0 || sampleAlerts.under > 0) && (
            <div className="flex items-center gap-2 px-3 py-2 text-xs shrink-0" style={{ background: "var(--base)", border: "1px solid var(--line)", borderRadius: 8 }}>
              <Wallet size={14} style={{ color: PACING_COLOR.over }} />
              <span>
                {sampleAlerts.over > 0 && (
                  <><strong style={{ color: PACING_COLOR.over }}>{sampleAlerts.over}</strong> over pace</>
                )}
                {sampleAlerts.over > 0 && sampleAlerts.under > 0 && <span style={{ color: "var(--muted)" }}> · </span>}
                {sampleAlerts.under > 0 && (
                  <><strong style={{ color: PACING_COLOR.under }}>{sampleAlerts.under}</strong> under pace</>
                )}
              </span>
            </div>
          )}
          {sampleAlerts.pending > 0 && (
            <div className="flex items-center gap-2 px-3 py-2 text-xs shrink-0" style={{ background: "var(--base)", border: "1px solid var(--line)", borderRadius: 8 }}>
              <ClipboardCheck size={14} style={{ color: APPROVAL_COLOR.pending }} />
              <span><strong style={{ color: APPROVAL_COLOR.pending }}>{sampleAlerts.pending}</strong> pending approval</span>
            </div>
          )}
          {(anomalyCount > 0 || staleCategoryCount > 0) && (
            <div className="flex items-center gap-2 px-3 py-2 text-xs shrink-0" style={{ background: "var(--base)", border: "1px solid var(--line)", borderRadius: 8 }}>
              <AlertTriangle size={14} style={{ color: "var(--alert)" }} />
              <span>
                {anomalyCount > 0 && (
                  <><strong style={{ color: "var(--alert)" }}>{anomalyCount}</strong> flagged</>
                )}
                {anomalyCount > 0 && staleCategoryCount > 0 && <span style={{ color: "var(--muted)" }}> · </span>}
                {staleCategoryCount > 0 && (
                  <><strong style={{ color: "var(--alert)" }}>{staleCategoryCount}</strong> categor{staleCategoryCount === 1 ? "y" : "ies"} quiet</>
                )}
              </span>
            </div>
          )}
        </div>
      </div>

      {total === 0 ? (
        <div className="flex flex-col items-center gap-2 py-16 text-center" style={{ color: "var(--muted)" }}>
          <Inbox size={22} strokeWidth={1.5} />
          <span className="text-xs">No campaigns match these filters yet.</span>
        </div>
      ) : (
        <>
          {spotlight && (
            <>
              <div className="ctp-section-label">
                {spotlight.isAnomaly ? "Spotlight · the most statistically notable signal right now" : "Spotlight · biggest campaign in your current filters"}
              </div>
              <div
                className="ctp-spotlight" style={{ "--stripe": AD_TYPE_COLOR[spotlight.campaign.adType] }}
                onClick={() => onSelect(spotlight.campaign)}
              >
                <div className="ctp-spotlight-avatar"><span className="ctp-display">{initialsOf(spotlight.campaign.subject)}</span></div>
                <div className="ctp-spotlight-body">
                  <div className="ctp-spotlight-name ctp-display">
                    <SubjectTypeIcon type={spotlight.campaign.subjectType} />
                    {spotlight.campaign.subject}
                    {spotlight.isAnomaly && (
                      <span style={{ color: "var(--alert)", display: "flex" }}>
                        <AlertTriangle size={18} />
                      </span>
                    )}
                  </div>
                  <div className="ctp-spotlight-sub">
                    {spotlight.campaign.subjectType === "person" ? "Person" : "Brand"} · {spotlight.campaign.category} · {spotlight.campaign.adType} · {spotlight.campaign.region} ·{" "}
                    {spotlight.campaign.status === "Live" ? `Live · started ${startedLabel(spotlight.campaign.start)}` : "Completed"}
                  </div>
                  <p className="ctp-spotlight-reason">
                    {spotlight.isAnomaly ? (
                      <>Flagged by anomaly detection: <strong>{spotlight.reason}</strong> — the biggest signal across your current filters.</>
                    ) : (
                      "No anomalies flagged right now — this is simply the largest campaign (by reach) matching your current filters."
                    )}
                  </p>
                  <div className="ctp-spotlight-stats">
                    <div className="stat"><span className="k">Reach</span><span className="v">{fmtReach(spotlight.campaign.reach)}</span></div>
                    <div className="stat"><span className="k">Spend</span><span className="v">{fmtMoney(spotlight.campaign.spend)}</span></div>
                    <div className="stat"><span className="k">Platform</span><span className="v">{spotlight.campaign.platform}</span></div>
                  </div>
                </div>
              </div>
            </>
          )}

          <div className="flex flex-col" style={{ gap: 30 }}>
            <Ribbon
              label="Trending — flagged by anomaly detection"
              count={`${trending.length} campaign${trending.length === 1 ? "" : "s"}`}
              items={trending}
              renderItem={(c) => <CampaignTile key={c.id} c={c} flagged stat={`${fmtReach(c.reach)} reach`} onSelect={onSelect} />}
            />
            <Ribbon
              label="Pending your review"
              count={`${pendingReview.length} campaign${pendingReview.length === 1 ? "" : "s"}`}
              items={pendingReview}
              renderItem={(c) => (
                <CampaignTile key={c.id} c={c} flagged={anomalyByID.has(c.id)} badge={{ label: "Pending review", variant: "pending" }} onSelect={onSelect} />
              )}
            />
            <Ribbon
              label="Regions"
              count="click a region to filter"
              items={regionTiles}
              renderItem={(d) => <RegionTile key={d.region} d={d} total={regionsTotal} active={region === d.region} onToggle={toggleRegion} />}
            />
            {showBrandsRibbon && (
              <Ribbon
                label="Top brands"
                right={
                  <div className="ctp-toggle">
                    <button type="button" className={brandSort === "count" ? "active" : ""} onClick={() => setBrandSort("count")}>Count</button>
                    <button type="button" className={brandSort === "reach" ? "active" : ""} onClick={() => setBrandSort("reach")}>Reach</button>
                  </div>
                }
                items={topBrands}
                renderItem={(row) => <SubjectTile key={row.subject} row={row} onSelect={setSubject} />}
              />
            )}
            <Ribbon
              label="Biggest campaigns — by reach"
              count="this window"
              items={biggestByReach}
              renderItem={(c) => <CampaignTile key={c.id} c={c} flagged={anomalyByID.has(c.id)} stat={`${fmtReach(c.reach)} reach`} onSelect={onSelect} />}
            />
            <Ribbon
              label="Latest tracked"
              count="newest first"
              items={latestTracked}
              renderItem={(c) => <CampaignTile key={c.id} c={c} flagged={anomalyByID.has(c.id)} stat={startedLabel(c.start)} onSelect={onSelect} />}
            />
            <Ribbon
              label="People campaigns"
              count={`${peopleCampaigns.length} campaign${peopleCampaigns.length === 1 ? "" : "s"}`}
              items={peopleCampaigns}
              renderItem={(c) => <CampaignTile key={c.id} c={c} flagged={anomalyByID.has(c.id)} stat={`${fmtReach(c.reach)} reach`} onSelect={onSelect} />}
            />
          </div>

          {/* Trend chart — detail behind the discovery content above it,
              same "lead with what needs attention, end with the summary
              chart" ordering the ribbons/KPI-strip split already uses. */}
          <div className="ctp-panel p-5 mt-8">
            <div className="flex items-baseline justify-between mb-4">
              <h2 className="ctp-display font-semibold">Campaign activity by format</h2>
              <div className="ctp-toggle">
                <button type="button" className={chartMode === "area" ? "active" : ""} onClick={() => setChartMode("area")}>Area</button>
                <button type="button" className={chartMode === "bar" ? "active" : ""} onClick={() => setChartMode("bar")}>Bar</button>
              </div>
            </div>
            <ResponsiveContainer width="100%" height={300}>
              {chartMode === "area" ? (
                <AreaChart data={trendData}>
                  <CartesianGrid stroke="var(--line)" vertical={false} />
                  <XAxis dataKey="month" tick={{ fontSize: 11, fill: "var(--muted)" }} axisLine={{ stroke: "var(--line)" }} tickLine={false} />
                  <YAxis tick={{ fontSize: 11, fill: "var(--muted)" }} axisLine={false} tickLine={false} width={24} />
                  <Tooltip contentStyle={{ fontSize: 12, border: "1px solid var(--line)" }} />
                  {adTypes.map((a) => (
                    // fillOpacity, not a hex-alpha suffix on AD_TYPE_COLOR[a]: the
                    // palette is now a CSS var() reference (theme.css), which a
                    // string-concatenated alpha suffix can't apply to.
                    <Area key={a} type="monotone" dataKey={a} stackId="1" stroke={AD_TYPE_COLOR[a]} fill={AD_TYPE_COLOR[a]} fillOpacity={0.2} />
                  ))}
                </AreaChart>
              ) : (
                <BarChart data={trendData}>
                  <CartesianGrid stroke="var(--line)" vertical={false} />
                  <XAxis dataKey="month" tick={{ fontSize: 11, fill: "var(--muted)" }} axisLine={{ stroke: "var(--line)" }} tickLine={false} />
                  <YAxis tick={{ fontSize: 11, fill: "var(--muted)" }} axisLine={false} tickLine={false} width={24} />
                  <Tooltip contentStyle={{ fontSize: 12, border: "1px solid var(--line)" }} />
                  {adTypes.map((a) => (
                    <Bar key={a} dataKey={a} stackId="1" fill={AD_TYPE_COLOR[a]} />
                  ))}
                </BarChart>
              )}
            </ResponsiveContainer>
            {/* A 6-series stacked chart needs a legend — identity was color-only
                before this (only revealed on hover, via Tooltip); this is the
                "secondary encoding" the palette's one floor-band-adjacent pair
                requires, and just a real gap either way. */}
            <div className="flex flex-wrap gap-x-4 gap-y-1.5 mt-3 pt-3" style={{ borderTop: "1px solid var(--line)" }}>
              {adTypes.map((a) => (
                <div key={a} className="flex items-center gap-1.5 text-xs" style={{ color: "var(--muted)" }}>
                  <span style={{ width: 8, height: 8, borderRadius: 2, background: AD_TYPE_COLOR[a], flexShrink: 0 }} />
                  {a}
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
