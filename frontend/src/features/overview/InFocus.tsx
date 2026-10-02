import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import type { Campaign, CategoryBenchmark } from "../../api/types";
import { Pill } from "../../components/Pill";
import { ApprovalTag } from "../../components/Tag";
import { adTypeStripeStyle } from "../../components/AdTypeTag";
import { SubjectImage } from "../../components/SubjectImage";
import { AreaChart } from "../../components/AreaChart";
import { reachCurve } from "../../lib/metrics";
import { formatIndex, formatMoney, formatReach } from "../../lib/format";
import "./InFocus.css";

const ROTATE_MS = 30_000;

export function InFocus({
  candidates,
  medAll,
  benchmarks,
}: {
  candidates: Campaign[];
  medAll: number;
  benchmarks?: CategoryBenchmark[];
}) {
  const [active, setActive] = useState(0);
  const [paused, setPaused] = useState(false);
  // Drives the progress ring. Kept separate from `active` so restarting the
  // cycle doesn't require re-rendering the whole card.
  const [elapsed, setElapsed] = useState(0);
  const startedAt = useRef(Date.now());

  const count = candidates.length;

  const goTo = useCallback((i: number) => {
    setActive(((i % count) + count) % count);
    startedAt.current = Date.now();
    setElapsed(0);
  }, [count]);

  useEffect(() => {
    if (paused || count <= 1) return;
    const id = setInterval(() => {
      const since = Date.now() - startedAt.current;
      if (since >= ROTATE_MS) {
        startedAt.current = Date.now();
        setElapsed(0);
        setActive((i) => (i + 1) % count);
      } else {
        setElapsed(since);
      }
    }, 250);
    return () => clearInterval(id);
  }, [paused, count]);

  if (count === 0) return null;

  const campaign = candidates[Math.min(active, count - 1)];
  const curve = reachCurve(campaign.reach, campaign.curveShape);
  const points = curve.map((v, i) => ({
    label: `Day ${Math.round((i / 7) * campaign.daysRunning)}`,
    value: v,
  }));
  const categoryMedian = benchmarks?.find((b) => b.category === campaign.category)?.medianReach;
  const eyebrow = campaign.flagReason
    ? `Flagged ${campaign.adType} · ${campaign.region}`
    : `${campaign.adType} · ${campaign.region}`;
  const progress = Math.min(1, elapsed / ROTATE_MS);

  return (
    <section
      className="in-focus"
      aria-label="In focus"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocusCapture={() => setPaused(true)}
      onBlurCapture={() => setPaused(false)}
    >
      <div className="in-focus-header">
        <div className="in-focus-heading">
          <h3>
            In focus
            <span className="in-focus-count">
              {active + 1}/{count}
            </span>
          </h3>
          <p className="in-focus-subtitle">
            What most deserves a look right now, worst signal first. Rotates every 30s — hover to hold.
          </p>
        </div>

        <div className="in-focus-controls">
          <button type="button" className="in-focus-nav" aria-label="Previous" onClick={() => goTo(active - 1)}>
            ‹
          </button>
          <button
            type="button"
            className="in-focus-nav"
            aria-label={paused ? "Resume rotation" : "Pause rotation"}
            aria-pressed={paused}
            onClick={() => setPaused((p) => !p)}
          >
            {paused ? "▶" : "❚❚"}
          </button>
          <button type="button" className="in-focus-nav" aria-label="Next" onClick={() => goTo(active + 1)}>
            ›
          </button>
        </div>
      </div>

      <Link
        to={`/campaigns/${campaign.id}`}
        className="in-focus-card card"
        style={adTypeStripeStyle(campaign.adType)}
        key={campaign.id}
      >
        <span className="in-focus-stripe" />
        <div className="in-focus-main">
          <div className="in-focus-eyebrow-row">
            <SubjectImage
              name={campaign.name}
              initials={campaign.initials}
              kind={campaign.subjectType}
              domain={campaign.subjectType === "brand" ? campaign.brandDomain : undefined}
              seed={campaign.creatorId ?? campaign.brandDomain ?? campaign.name}
              size={40}
            />
            <span className="in-focus-eyebrow">{eyebrow}</span>
          </div>

          <div className="in-focus-title-row">
            <h2>{campaign.name}</h2>
            <Pill status={campaign.status} />
            <ApprovalTag approval={campaign.approval} />
          </div>
          <div className="in-focus-sub">
            {campaign.role || campaign.category} · {campaign.platform} · Day {campaign.daysRunning}
          </div>
          {campaign.flagReason && <div className="in-focus-flag">{campaign.flagReason}</div>}

          <div className="in-focus-stats">
            <div>
              <div className="in-focus-stat-label">Reach</div>
              <div className="in-focus-stat-value mono">{formatReach(campaign.reach)}</div>
            </div>
            <div>
              <div className="in-focus-stat-label">Spend</div>
              <div className="in-focus-stat-value mono">{formatMoney(campaign.spend)}</div>
            </div>
            <div>
              <div className="in-focus-stat-label">All-campaign median</div>
              <div className="in-focus-stat-value mono">{formatReach(medAll)}</div>
            </div>
            <div>
              <div className="in-focus-stat-label">Index</div>
              <div className="in-focus-stat-value in-focus-index mono">{formatIndex(campaign.index)}</div>
            </div>
          </div>
        </div>

        <div className="in-focus-chart">
          <AreaChart points={points} medianValue={categoryMedian} formatValue={formatReach} interactive={false} height={150} />
        </div>
      </Link>

      <div className="in-focus-strip" role="tablist" aria-label="In focus campaigns">
        {candidates.map((c, i) => (
          <button
            key={c.id}
            type="button"
            role="tab"
            aria-selected={i === active}
            aria-label={c.name}
            className={`in-focus-thumb${i === active ? " in-focus-thumb-active" : ""}`}
            style={adTypeStripeStyle(c.adType)}
            onClick={() => goTo(i)}
          >
            <SubjectImage
              name={c.name}
              initials={c.initials}
              kind={c.subjectType}
              domain={c.subjectType === "brand" ? c.brandDomain : undefined}
              seed={c.creatorId ?? c.brandDomain ?? c.name}
              size={26}
            />
            <span className="in-focus-thumb-text">
              <span className="in-focus-thumb-name truncate">{c.name}</span>
              <span className="in-focus-thumb-meta truncate">{c.flagReason ?? `${formatIndex(c.index)} vs median`}</span>
            </span>
            {i === active && count > 1 && (
              <span
                className={`in-focus-thumb-progress${paused ? " in-focus-thumb-progress-paused" : ""}`}
                style={{ transform: `scaleX(${progress})` }}
                aria-hidden="true"
              />
            )}
          </button>
        ))}
      </div>
    </section>
  );
}
