import { useMemo, useState } from "react";
import { useLaunches } from "../../api/radar";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonRows } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import type { LaunchReadiness } from "../../api/types";
import "./RadarPage.css";

const BANDS: { key: LaunchReadiness; label: string; blurb: string }[] = [
  { key: "soon", label: "Worth a call", blurb: "Launched recently, people noticed, and there is somewhere to send traffic." },
  { key: "building", label: "Building", blurb: "Launched, but missing one of the signals that usually comes before a push." },
  { key: "quiet", label: "Quiet", blurb: "Past the window for a launch campaign, or nobody noticed." },
  { key: "unclear", label: "Unclear", blurb: "Not enough signal from the source to say anything." },
];

export function RadarPage() {
  useSetBreadcrumbs([{ label: "Launch radar" }]);
  const { data, isLoading, isError } = useLaunches();
  const [band, setBand] = useState<LaunchReadiness | "all">("soon");

  // Memoised because `?? []` builds a new array on every render, which
  // would make the memo below recompute every time and defeat itself.
  const items = useMemo(() => data?.items ?? [], [data]);
  const counts = useMemo(() => {
    const c: Record<string, number> = {};
    for (const l of items) c[l.readiness] = (c[l.readiness] ?? 0) + 1;
    return c;
  }, [items]);

  const shown = band === "all" ? items : items.filter((l) => l.readiness === band);
  const settled = items.filter((l) => l.daysToMarket !== undefined);

  return (
    <div className="radar-page">
      <div className="page-header">
        <div>
          <h1>Launch radar</h1>
        </div>
      </div>

      {/* Said plainly and up front, because the honest description is
          not the flattering one and a reader deciding whether to act
          on this is entitled to it. */}
      <p className="radar-lede">
        Products and brands that have just launched, ranked by signals anyone can check — how recent, how much
        attention the source reports, whether there is a site yet. This is <strong>not a forecast</strong>: nothing
        here is fitted to what past launches actually did, because the system has never watched one through to its
        first campaign. It is a prioritised list with its reasoning shown.{" "}
        {settled.length > 0
          ? `${settled.length} launch${settled.length === 1 ? "" : "es"} tracked through to a campaign so far.`
          : "Once launches start reaching campaigns, the gap is recorded here and the ranking can be checked against it."}
      </p>

      <div className="radar-bands" role="radiogroup" aria-label="Readiness">
        {BANDS.map((b) => (
          <button
            key={b.key}
            type="button"
            role="radio"
            aria-checked={band === b.key}
            className={`radar-band radar-band-${b.key}${band === b.key ? " radar-band-active" : ""}`}
            onClick={() => setBand(b.key)}
            title={b.blurb}
          >
            {b.label}
            <span className="radar-band-count">{counts[b.key] ?? 0}</span>
          </button>
        ))}
        <button
          type="button"
          role="radio"
          aria-checked={band === "all"}
          className={`radar-band${band === "all" ? " radar-band-active" : ""}`}
          onClick={() => setBand("all")}
        >
          All<span className="radar-band-count">{items.length}</span>
        </button>
      </div>

      {isLoading ? (
        <SkeletonRows count={6} height={56} />
      ) : isError ? (
        <EmptyState title="Couldn't load the radar" description="Please try refreshing the page." />
      ) : shown.length === 0 ? (
        <EmptyState title="Nothing in this band" description="Try another band, or wait for the next poll." />
      ) : (
        <div className="card radar-list">
          {shown.map((l) => (
            <div key={l.id} className="radar-row">
              <div className="radar-row-main">
                <div className="radar-row-name">
                  {l.url ? (
                    <a href={l.url} target="_blank" rel="noreferrer noopener">
                      {l.name}
                    </a>
                  ) : (
                    l.name
                  )}
                  <span className={`radar-pill radar-pill-${l.readiness}`}>{l.readiness}</span>
                </div>
                {l.tagline && <div className="radar-row-tagline truncate">{l.tagline}</div>}
                {/* The working, not decoration: this is what the band
                    was computed from. */}
                <div className="radar-row-basis">{l.basis}</div>
              </div>
              <div className="radar-row-outcome">
                {l.daysToMarket !== undefined ? (
                  <>
                    <span className="radar-outcome-label">Reached market</span>
                    <span className="radar-outcome-value mono">{l.daysToMarket}d</span>
                  </>
                ) : (
                  <span className="radar-outcome-pending">not yet advertising</span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
