import React, { useEffect, useState } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  LayoutDashboard, Settings, Search, Download, Mail,
  ArrowUpRight, ArrowDownRight, Radio, X, RotateCcw, AlertTriangle, LogOut,
} from "lucide-react";
import { getMeta, getKPIs, exportCampaignsCSV, emailReport } from "./api/client";
import { fmtReach, fmtMoney } from "./format";
import { useFilters } from "./filters";
import CampaignDrawer from "./components/CampaignDrawer";

// ---------------------------------------------------------------------------
// This is the shell: left-rail nav, top bar, the shared filter row, and the
// KPI ticker. Filters live in the URL (useFilters, not component state —
// see docs/ROADMAP.md's Phase A for why), so every tab reads them the same
// way regardless of how it got navigated to. Each route in AppRoutes.jsx
// owns its own data fetch (useQuery) and its own view, backed by whichever
// service that data actually comes from (docs/ARCHITECTURE.md):
//   Overview   → everything: trend, the spotlight/ribbon discovery feed,
//                region breakdown, and competitor benchmarking, all
//                against campaigns-service + analytics-service. Folded
//                in from four separate tabs (Overview/Campaigns/Region/
//                Reports) once the ribbon/spotlight redesign made the
//                old exhaustive table, region chart, and benchmark table
//                feel like a different design language — see
//                tabs/OverviewTab.jsx's own comment for the detail.
//   Settings   → admin-only user/role management + the audit log
//                (auth-service + audit-service)
// CSV export and "email this report" (formerly Reports' own tab) live
// here in the global top bar, not inside Overview's content — both act
// on whatever's currently filtered, not on any one section of the page.
// A tab reaches the drawer/error-banner this layout owns via
// useOutletContext() (see tabs/*.jsx), not props — routed children don't
// get props from their parent route the way plain children would.
// ---------------------------------------------------------------------------

const NAV = [
  { path: "/overview", label: "Overview", icon: LayoutDashboard },
  { path: "/settings", label: "Settings", icon: Settings },
];
const RANGE_OPTIONS = [{ label: "7d", days: 7 }, { label: "30d", days: 30 }, { label: "90d", days: 90 }, { label: "All", days: 9999 }];
const TYPE_OPTIONS = [{ label: "All", value: "All" }, { label: "Brands", value: "brand" }, { label: "People", value: "person" }];

export default function DashboardLayout({ onSignOut }) {
  const {
    category, region, adType, subjectType, days, subject, q,
    filters, setCategory, setRegion, setAdType, setSubjectType, setQuery, setDays, setSubject, resetAll,
  } = useFilters();

  // The search box needs to feel instant while typing but shouldn't push a
  // new URL (and refetch) on every keystroke — a local draft, debounced
  // into the URL-backed query param. Guarded by `rawQuery !== q`: without
  // it, this fires an unconditional setQuery("") 200ms after every mount
  // (even when nothing was typed) — harmless on its own, but a real bug
  // once another filter's setSearchParams call can land in that same
  // window: two updates racing means the second's "previous params" can
  // miss the first's not-yet-committed change, silently dropping it (e.g.
  // a region just clicked). Only firing on an actual change closes that
  // window in the case that matters — nothing ever calls setQuery on a
  // mount-only tick with real user input to race against.
  const [rawQuery, setRawQuery] = useState(q);
  useEffect(() => {
    if (rawQuery === q) return;
    const t = setTimeout(() => setQuery(rawQuery), 200);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rawQuery]);

  const [selected, setSelected] = useState(null);
  const [error, setError] = useState(null);
  const queryClient = useQueryClient();

  // CSV export and "email this report" — moved here from the old Reports
  // tab (see this file's own header comment): both act on whatever's
  // currently filtered, so the global top bar is theirs regardless of
  // which section of the page someone's looking at.
  const [exporting, setExporting] = useState(false);
  const [emailing, setEmailing] = useState(false);
  const [emailStatus, setEmailStatus] = useState(null); // { ok: bool, message: string } | null

  async function handleExport() {
    setExporting(true);
    try {
      const blob = await exportCampaignsCSV(filters);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url; a.download = "campaign-report.csv"; a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      setError(e.message);
    } finally {
      setExporting(false);
    }
  }

  // Always resolves to a {sent, reason} shape rather than throwing just
  // because delivery isn't configured in this environment — see
  // api/client.js's emailReport doc comment — so this shows that
  // honestly instead of claiming success it didn't achieve.
  async function handleEmailReport() {
    setEmailing(true);
    setEmailStatus(null);
    try {
      const { sent, reason } = await emailReport(filters);
      setEmailStatus(sent ? { ok: true, message: "Sent to your account email." } : { ok: false, message: reason });
    } catch (e) {
      setEmailStatus({ ok: false, message: e.message });
    } finally {
      setEmailing(false);
    }
  }

  // A budget change re-derives `pacing` server-side but doesn't change
  // anything else about a campaign, so refreshing just the drawer's own
  // copy (rather than the whole cache) would leave the Campaigns table's
  // row stale until some other refetch happened to occur. Invalidating
  // every "campaigns"-keyed query (a prefix match — every tab's queryKey
  // starts with it, whatever filters follow) covers Overview's feed,
  // the paginated table, and this drawer's own campaign in one call.
  function handleBudgetUpdated(updatedCampaign) {
    setSelected(updatedCampaign);
    queryClient.invalidateQueries({ queryKey: ["campaigns"] });
  }

  // Same shape as handleBudgetUpdated — an approval change doesn't touch
  // reach/spend, so it can't change this campaign's own anomaly status,
  // but it does need the Campaigns table's Approval column and Overview's
  // pending-approval count to reflect it without a manual refresh.
  function handleApprovalUpdated(updatedCampaign) {
    setSelected(updatedCampaign);
    queryClient.invalidateQueries({ queryKey: ["campaigns"] });
  }

  useEffect(() => {
    function onKey(e) { if (e.key === "Escape") setSelected(null); }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const metaQuery = useQuery({ queryKey: ["meta"], queryFn: getMeta });
  const meta = metaQuery.data ?? { categories: { brand: [], person: [] }, regions: [], adTypes: [] };

  const kpisQuery = useQuery({ queryKey: ["kpis", filters], queryFn: () => getKPIs(filters) });
  useEffect(() => {
    setError(metaQuery.error?.message ?? kpisQuery.error?.message ?? null);
  }, [metaQuery.error, kpisQuery.error]);

  // Category options depend on the Type filter — "Luxury" is the only
  // category that applies to both brands and people.
  const categoryOptions =
    subjectType === "brand" ? meta.categories.brand :
    subjectType === "person" ? meta.categories.person :
    [...new Set([...meta.categories.brand, ...meta.categories.person])];

  // Switching Type can strand the category filter on a value that no longer
  // applies (e.g. "Fintech" selected, then Type flipped to "People").
  useEffect(() => {
    if (category !== "All" && !categoryOptions.includes(category)) setCategory("All");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subjectType, meta]);

  const activeFilterChips = [
    subjectType !== "All" && { key: "subjectType", label: subjectType === "brand" ? "Brands" : "People", clear: () => setSubjectType("All") },
    category !== "All" && { key: "category", label: category, clear: () => setCategory("All") },
    region !== "All" && { key: "region", label: region, clear: () => setRegion("All") },
    adType !== "All" && { key: "adType", label: adType, clear: () => setAdType("All") },
    subject && { key: "subject", label: subject, clear: () => setSubject(null) },
  ].filter(Boolean);

  function handleResetAll() {
    setRawQuery("");
    resetAll();
  }

  function focusSubjectAndClose(name) {
    setSubject(name);
    setSelected(null);
  }

  const kpis = kpisQuery.data;
  const kpiTiles = kpis ? [
    { label: "Active campaigns", value: kpis.activeCampaigns.value, delta: kpis.activeCampaigns.delta, up: kpis.activeCampaigns.up, spark: kpis.activeCampaigns.spark },
    { label: "Subjects tracked", value: kpis.subjectsTracked.value, delta: kpis.subjectsTracked.delta, up: kpis.subjectsTracked.up, spark: kpis.subjectsTracked.spark },
    { label: "Estimated reach", value: fmtReach(kpis.estimatedReach.value), delta: kpis.estimatedReach.delta, up: kpis.estimatedReach.up, spark: kpis.estimatedReach.spark },
    { label: "Est. spend", value: fmtMoney(kpis.estimatedSpend.value), delta: kpis.estimatedSpend.delta, up: kpis.estimatedSpend.up, spark: kpis.estimatedSpend.spark },
  ] : [];

  return (
    <div className="ctp-root flex min-h-screen w-full text-sm">
      <style>{`
        .ctp-root { background:var(--base); color:var(--ink); font-family:Inter,ui-sans-serif,system-ui; }
        .ctp-root .ctp-display { font-family:Archivo,Inter,ui-sans-serif,system-ui; }
        .ctp-panel { background:var(--surface); border:1px solid var(--line); }
        .ctp-nav-item { color:var(--muted); border-left:2px solid transparent; cursor:pointer; transition:background .12s ease; text-decoration:none; }
        .ctp-nav-item.active { color:var(--ink); border-left:2px solid var(--accent); background:var(--accent-soft); }
        .ctp-nav-item:hover:not(.active) { background:#F1EFE7; }
        .ctp-tag { background:var(--secondary-soft); color:var(--secondary); }
        .ctp-select { border:1px solid var(--line); background:var(--surface); color:var(--ink); }
        .ctp-range-btn { cursor:pointer; transition:background .12s ease, color .12s ease; }
        .ctp-range-btn.active { background:var(--ink); color:var(--base); border-color:var(--ink); }
        .ctp-chip { background:var(--accent-soft); color:var(--accent); display:inline-flex; align-items:center; gap:6px; }
        .ctp-row { cursor:pointer; transition:background .1s ease; }
        .ctp-row:hover { background:#F6F4EC; }
        .ctp-th { cursor:pointer; user-select:none; }
        .ctp-th:hover { color:var(--ink); }
        .ctp-drawer-backdrop { position:fixed; inset:0; background:rgba(23,24,26,.35); z-index:40; animation:ctpFade .15s ease; }
        .ctp-drawer { position:fixed; top:0; right:0; height:100%; width:360px; max-width:92vw; background:var(--surface);
          border-left:1px solid var(--line); z-index:50; box-shadow:-8px 0 24px rgba(0,0,0,.08); animation:ctpSlide .18s ease; }
        @keyframes ctpFade { from{opacity:0} to{opacity:1} }
        @keyframes ctpSlide { from{transform:translateX(24px); opacity:.6} to{transform:translateX(0); opacity:1} }
        .ctp-toggle { display:flex; border:1px solid var(--line); overflow:hidden; }
        .ctp-toggle button { padding:5px 10px; font-size:11px; cursor:pointer; background:var(--surface); color:var(--muted); }
        .ctp-toggle button.active { background:var(--ink); color:var(--base); }
        .ctp-spark rect { transition: height .15s ease; }
        .ctp-brand-link { cursor:pointer; }
        .ctp-brand-link:hover { text-decoration:underline; text-decoration-color:var(--accent); text-underline-offset:3px; }
        .ctp-status-live { color:var(--status-live); }
        .ctp-status-completed { color:var(--muted); }
        .ctp-fetching { opacity:.55; pointer-events:none; transition:opacity .15s ease; }
        .ctp-error-banner { background:var(--alert); color:#fff; }

        /* ---- Campaigns tab: spotlight + ribbons (tabs/CampaignsTab.jsx) ---- */
        .ctp-section-label { font-size:11px; letter-spacing:.1em; text-transform:uppercase; color:var(--secondary); font-weight:700; margin-bottom:10px; }
        .ctp-spotlight {
          position:relative; overflow:hidden; border-radius:14px; border:1px solid var(--line); cursor:pointer;
          padding:26px 28px; display:flex; align-items:flex-end; min-height:190px; margin-bottom:26px;
          background:linear-gradient(135deg, color-mix(in srgb, var(--stripe) 22%, var(--surface)) 0%, var(--surface) 75%);
          transition:border-color .12s ease;
        }
        .ctp-spotlight:hover { border-color:var(--stripe); }
        .ctp-spotlight-avatar {
          position:absolute; top:-30px; right:-24px; width:200px; height:200px; border-radius:50%;
          background:color-mix(in srgb, var(--stripe) 34%, var(--surface));
          display:flex; align-items:center; justify-content:center;
        }
        .ctp-spotlight-avatar span { font-weight:800; font-size:56px; color:color-mix(in srgb, var(--stripe) 75%, black 15%); opacity:.55; }
        .ctp-spotlight-body { position:relative; z-index:1; max-width:600px; }
        .ctp-spotlight-name { font-size:24px; font-weight:800; margin:0 0 4px; display:flex; align-items:center; gap:8px; }
        .ctp-spotlight-sub { font-size:12.5px; color:var(--muted); margin-bottom:12px; }
        .ctp-spotlight-reason { font-size:13px; line-height:1.55; margin:0 0 14px; max-width:480px; background:var(--surface); border:1px solid var(--line); border-radius:10px; padding:10px 13px; }
        .ctp-spotlight-reason strong { color:var(--stripe); }
        .ctp-spotlight-stats { display:flex; gap:24px; }
        .ctp-spotlight-stats .stat { display:flex; flex-direction:column; gap:2px; }
        .ctp-spotlight-stats .stat .k { font-size:10px; letter-spacing:.06em; text-transform:uppercase; color:var(--muted); }
        .ctp-spotlight-stats .stat .v { font-size:17px; font-weight:700; font-variant-numeric:tabular-nums; }

        .ctp-ribbon-head { display:flex; align-items:baseline; justify-content:space-between; margin-bottom:11px; }
        .ctp-ribbon-head h3 { font-size:15px; font-weight:700; margin:0; }
        .ctp-ribbon-head .count { font-size:11px; color:var(--muted); font-weight:600; }
        /* The track gets left/right padding sized to the nav buttons, and
           the wrap cancels it back out with a matching negative margin —
           tiles never sit under a button this way, whatever the scroll
           position, and the buttons never overlap the tile flow. The
           gutter (and the button) is sized to exactly 24px — this tab's
           own outer edge (px-6, Tailwind's 1.5rem) — on purpose: the
           bleed this negative margin creates reaches exactly the real
           page edge and no further. A wider gutter (the standalone
           mockup this was built from used 38px) is invisible on a
           centered page with margin to spare either side, but here
           main runs flush to the actual viewport edge, so anything
           past 24px pushed the "next" button partly off-screen — a real
           bug, caught live, not a hypothetical. */
        .ctp-ribbon-track-wrap { position:relative; padding:0 24px; margin:0 -24px; }
        .ctp-ribbon-track { display:flex; gap:13px; overflow-x:auto; scroll-behavior:smooth; scroll-snap-type:x proximity; padding:2px 2px 6px; scrollbar-width:none; }
        .ctp-ribbon-track::-webkit-scrollbar { display:none; }
        .ctp-ribbon-nav {
          position:absolute; top:50%; transform:translateY(-50%); width:24px; height:24px; border-radius:50%;
          display:flex; align-items:center; justify-content:center; border:1px solid var(--line);
          background:var(--surface); cursor:pointer; z-index:2; color:var(--ink);
        }
        .ctp-ribbon-nav:hover { border-color:var(--accent); color:var(--accent); }
        .ctp-ribbon-nav.prev { left:0; }
        .ctp-ribbon-nav.next { right:0; }

        .ctp-tile { scroll-snap-align:start; flex:0 0 166px; width:166px; border-radius:12px; overflow:hidden; border:1px solid var(--line); background:var(--surface); cursor:pointer; transition:transform .12s ease, border-color .12s ease; }
        .ctp-tile:hover { transform:translateY(-3px); border-color:var(--stripe); }
        .ctp-tile-art {
          position:relative; height:82px; display:flex; align-items:center; justify-content:center;
          background:linear-gradient(150deg, color-mix(in srgb, var(--stripe) 55%, var(--surface)) 0%, color-mix(in srgb, var(--stripe) 18%, var(--surface)) 100%);
        }
        .ctp-tile-initials { font-weight:800; font-size:24px; color:color-mix(in srgb, var(--stripe) 82%, black 18%); opacity:.7; }
        .ctp-tile-status { position:absolute; top:6px; left:6px; display:flex; align-items:center; gap:4px; font-size:9px; font-weight:700; padding:2px 6px; border-radius:999px; background:rgba(10,12,24,.42); color:#fff; }
        .ctp-tile-status .dot { width:5px; height:5px; border-radius:50%; background:#fff; }
        .ctp-tile-flag { position:absolute; top:6px; right:6px; color:#fff; background:var(--alert); border-radius:50%; width:17px; height:17px; display:flex; align-items:center; justify-content:center; }
        .ctp-tile-body { padding:9px 10px 10px; display:flex; flex-direction:column; gap:3px; }
        .ctp-tile-name { display:flex; align-items:center; gap:4px; font-size:12.5px; font-weight:700; line-height:1.2; }
        .ctp-tile-name svg { flex-shrink:0; }
        .ctp-tile-name span { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; min-width:0; }
        .ctp-tile-meta { font-size:10px; color:var(--muted); white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
        .ctp-tile-stat { font-size:11px; font-weight:600; margin-top:2px; font-variant-numeric:tabular-nums; }
        .ctp-tile-badge { font-size:9px; font-weight:700; margin-top:1px; }
        .ctp-tile-badge.pending { color:var(--alert); }
        .ctp-tile-spark { display:block; width:100%; height:14px; margin-top:2px; }

        /* ---- Overview tab: "Recently tracked" feed rows (tabs/OverviewTab.jsx) ----
           The same ad-type-stripe-gradient + initials treatment as a
           ribbon tile's art block (DashboardLayout above), just circular
           and sized for a compact vertical list rather than a 166px card —
           one visual language, two layouts. */
        .ctp-feed-row { display:flex; align-items:center; gap:10px; padding:9px 0; cursor:pointer; border-bottom:1px solid var(--line); }
        .ctp-feed-avatar {
          width:34px; height:34px; border-radius:50%; flex-shrink:0; display:flex; align-items:center; justify-content:center;
          background:linear-gradient(150deg, color-mix(in srgb, var(--stripe) 55%, var(--surface)) 0%, color-mix(in srgb, var(--stripe) 18%, var(--surface)) 100%);
        }
        .ctp-feed-avatar span { font-weight:800; font-size:11.5px; color:color-mix(in srgb, var(--stripe) 82%, black 18%); opacity:.75; }
      `}</style>

      {/* Left rail */}
      <aside className="ctp-panel hidden md:flex w-56 shrink-0 flex-col justify-between border-r py-6">
        <div>
          <div className="px-6 pb-8">
            <div className="ctp-display text-lg font-bold leading-none">Campaign</div>
            <div className="ctp-display text-lg font-bold leading-none" style={{ color: "var(--accent)" }}>Tracker Pro</div>
          </div>
          <nav className="flex flex-col gap-1">
            {NAV.map((n) => (
              <NavLink key={n.path} to={n.path} className={({ isActive }) => `ctp-nav-item ${isActive ? "active" : ""} flex items-center gap-3 px-6 py-2.5`}>
                <n.icon size={16} strokeWidth={2} /><span>{n.label}</span>
              </NavLink>
            ))}
          </nav>
        </div>
        <div className="px-6 text-xs" style={{ color: "var(--muted)" }}>
          Data window: last 150 days<br />Indian market · all regions
        </div>
      </aside>

      {/* Main. min-w-0 matters once a tab (Campaigns' ribbons) contains a
          fixed-width, horizontally-scrolling row: without it, a flex
          child's default min-width is its content's intrinsic width, so
          the ribbon's full un-scrolled width would push this whole
          column (and the page) wider than the viewport instead of
          scrolling inside .ctp-ribbon-track as intended. */}
      <main className="flex-1 flex flex-col min-w-0">
        {error && (
          <div className="ctp-error-banner flex items-center gap-2 px-6 py-2 text-xs">
            <AlertTriangle size={14} />
            Couldn't reach the API: {error}
            <button className="ml-auto underline" onClick={() => setError(null)}>Dismiss</button>
          </div>
        )}

        {/* Top bar */}
        <div className="ctp-panel flex items-center justify-between gap-4 border-b px-6 py-4">
          <div className="flex items-center gap-2 flex-1 max-w-sm">
            <Search size={16} style={{ color: "var(--muted)" }} />
            <input
              value={rawQuery} onChange={(e) => setRawQuery(e.target.value)}
              placeholder="Search a brand or person…" className="w-full bg-transparent outline-none text-sm"
            />
            {rawQuery && (
              <button onClick={() => setRawQuery("")} style={{ color: "var(--muted)" }}><X size={13} /></button>
            )}
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {emailStatus && (
              <span className="text-xs" style={{ color: emailStatus.ok ? "var(--status-live)" : "var(--muted)" }}>
                {emailStatus.message}
              </span>
            )}
            <button
              onClick={handleEmailReport} disabled={emailing}
              className="flex items-center gap-2 px-3 py-1.5 text-xs font-medium ctp-panel hover:opacity-80 disabled:opacity-50"
              title="Email the filtered CSV to your account"
            >
              <Mail size={14} />{emailing ? "Sending…" : "Email report"}
            </button>
            <button
              onClick={handleExport} disabled={exporting}
              className="flex items-center gap-2 px-3 py-1.5 text-xs font-medium ctp-panel hover:opacity-80 disabled:opacity-50"
              title="Download the filtered campaigns as CSV"
            >
              <Download size={14} />{exporting ? "Exporting…" : "Export CSV"}
            </button>
            {onSignOut && (
              <button
                onClick={onSignOut}
                className="flex items-center gap-2 px-3 py-1.5 text-xs font-medium ctp-panel hover:opacity-80"
                title="Sign out"
              >
                <LogOut size={14} />Sign out
              </button>
            )}
          </div>
        </div>

        {/* KPI ticker with sparklines — shown above every tab */}
        <div className={`ctp-panel flex flex-wrap items-stretch border-b ${kpisQuery.isFetching ? "ctp-fetching" : ""}`}>
          {kpiTiles.map((k, i) => (
            <div key={k.label} className="flex items-center gap-4 px-6 py-4" style={{ borderRight: i < kpiTiles.length - 1 ? "1px solid var(--line)" : "none" }}>
              <div>
                <div className="text-xs" style={{ color: "var(--muted)" }}>{k.label}</div>
                <div className="ctp-display text-2xl font-bold">{k.value}</div>
                <div className="flex items-center gap-0.5 text-xs font-medium" style={{ color: k.up ? "var(--secondary)" : "var(--alert)" }}>
                  {k.up ? <ArrowUpRight size={13} /> : <ArrowDownRight size={13} />}{k.delta}
                </div>
              </div>
              <svg className="ctp-spark" width="36" height="28" viewBox="0 0 36 28">
                {k.spark.map((v, idx) => {
                  const max = Math.max(...k.spark, 1);
                  const h = Math.max(2, (v / max) * 24);
                  return <rect key={idx} x={idx * 4.6} y={26 - h} width="3" height={h} fill={idx === k.spark.length - 1 ? "var(--accent)" : "var(--line)"} />;
                })}
              </svg>
            </div>
          ))}
          <div className="flex items-center gap-2 px-6 py-4 ml-auto text-xs" style={{ color: "var(--muted)" }}>
            <Radio size={13} style={{ color: "var(--accent)" }} />Live tracking active
          </div>
        </div>

        {/* Filters — shared across every tab */}
        <div className="flex flex-wrap items-center gap-3 px-6 py-4">
          <div className="ctp-toggle">
            {TYPE_OPTIONS.map((t) => (
              <button key={t.value} className={subjectType === t.value ? "active" : ""} onClick={() => setSubjectType(t.value)}>
                {t.label}
              </button>
            ))}
          </div>
          <select className="ctp-select px-3 py-1.5 text-xs" value={category} onChange={(e) => setCategory(e.target.value)}>
            <option>All</option>{categoryOptions.map((c) => <option key={c}>{c}</option>)}
          </select>
          <select className="ctp-select px-3 py-1.5 text-xs" value={region} onChange={(e) => setRegion(e.target.value)}>
            <option>All</option>{meta.regions.map((t) => <option key={t}>{t}</option>)}
          </select>
          <select className="ctp-select px-3 py-1.5 text-xs" value={adType} onChange={(e) => setAdType(e.target.value)}>
            <option>All</option>{meta.adTypes.map((a) => <option key={a}>{a}</option>)}
          </select>
          <div className="flex" style={{ border: "1px solid var(--line)" }}>
            {RANGE_OPTIONS.map((r) => (
              <button key={r.label} onClick={() => setDays(r.days)}
                className={`ctp-range-btn px-3 py-1.5 text-xs ${days === r.days ? "active" : ""}`}
                style={{ borderRight: "1px solid var(--line)" }}>{r.label}</button>
            ))}
          </div>
          {activeFilterChips.length > 0 && (
            <button onClick={handleResetAll} className="flex items-center gap-1 text-xs ml-1" style={{ color: "var(--muted)" }}>
              <RotateCcw size={12} />Reset
            </button>
          )}
          <div className="flex flex-wrap gap-2 ml-auto">
            {activeFilterChips.map((chip) => (
              <span key={chip.key} className="ctp-chip px-2 py-1 text-xs">
                {chip.label}
                <X size={11} className="cursor-pointer" onClick={chip.clear} />
              </span>
            ))}
          </div>
        </div>

        <Outlet context={{ onSelect: setSelected, onError: setError }} />
      </main>

      <CampaignDrawer
        campaign={selected} onClose={() => setSelected(null)} onFocusSubject={focusSubjectAndClose}
        onBudgetUpdated={handleBudgetUpdated} onApprovalUpdated={handleApprovalUpdated}
      />
    </div>
  );
}
