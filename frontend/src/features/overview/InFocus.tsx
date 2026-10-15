import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import type { Campaign, CategoryBenchmark } from "../../api/types";
import { Pill } from "../../components/Pill";
import { ApprovalTag } from "../../components/Tag";
import { adTypeStripeStyle } from "../../components/AdTypeTag";
import { SubjectImage } from "../../components/SubjectImage";
import { useCreatorPhotos } from "../../api/creatorPhotos";
import { AreaChart } from "../../components/AreaChart";
import { AdFilm } from "../../components/AdFilm";
import { adFilmFor } from "../../data/adFilms";
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
  // A creator with a photograph should be the same person here as on the
  // tiles below; the roster is already cached, so this costs no request.
  const creatorPhotos = useCreatorPhotos();
  const [active, setActive] = useState(0);
  const [paused, setPaused] = useState(false);
  // A film playing holds the carousel still on its own account. Kept
  // apart from `paused`, which is the pointer being over the card: if
  // they shared one flag, moving the mouse away would rotate the card
  // out from under a video that is still playing.
  const [filmPlaying, setFilmPlaying] = useState(false);
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
    if (paused || filmPlaying || count <= 1) return;
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
  }, [paused, filmPlaying, count]);

  if (count === 0) return null;

  const campaign = candidates[Math.min(active, count - 1)];
  // Only where the brand's own upload has been verified. Everything
  // else keeps the card it has — an approximate stand-in for a real
  // campaign's creative would be worse than none.
  const film = adFilmFor(campaign.id);
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

      {/* Was a single <Link> around the whole card. A play control
          cannot sit inside an anchor — nested interactive content is
          invalid, and the click would navigate instead of playing — so
          the link now covers the half you read and the film owns its
          own half. */}
      <div
        className="in-focus-card card"
        style={adTypeStripeStyle(campaign.adType)}
        key={campaign.id}
      >
        <span className="in-focus-stripe" />
        <Link to={`/campaigns/${campaign.id}`} className="in-focus-main">
          <div className="in-focus-eyebrow-row">
            <SubjectImage
              name={campaign.name}
              initials={campaign.initials}
              kind={campaign.subjectType}
              imageUrl={campaign.creatorId ? creatorPhotos[campaign.creatorId] : undefined}
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

        </Link>

        <div className="in-focus-side">
          {film && (
            <AdFilm film={film} campaignName={campaign.name} onPlayingChange={setFilmPlaying} />
          )}
          <div className="in-focus-chart">
            <AreaChart
              points={points}
              medianValue={categoryMedian}
              formatValue={formatReach}
              interactive={false}
              height={film ? 96 : 150}
            />
          </div>
        </div>
      </div>

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
              imageUrl={c.creatorId ? creatorPhotos[c.creatorId] : undefined}
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
