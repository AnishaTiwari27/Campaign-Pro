import type { Creative } from "../../api/types";
import { formatReach } from "../../lib/format";
import { EmptyState } from "../../components/EmptyState";
import "./CreativesTab.css";

const KIND_ICON: Record<Creative["kind"], string> = {
  video: "▶",
  banner: "▭",
  text: "✎",
};

export function CreativesTab({ creatives }: { creatives: Creative[] }) {
  if (creatives.length === 0) {
    return <EmptyState title="No creatives yet" description="Creative assets for this campaign haven't been uploaded." />;
  }

  return (
    <div className="creatives-grid">
      {creatives.map((c) => (
        <div key={c.id} className="creative-card card">
          <div className="creative-card-kind" aria-hidden="true">
            {KIND_ICON[c.kind]}
          </div>
          <div className="creative-card-headline">{c.headline}</div>
          <div className="creative-card-meta">
            {c.kind} · {c.durationLabel}
          </div>
          <div className="creative-card-stats">
            <span>{formatReach(c.reach)} reach</span>
            <span>{c.ctr.toFixed(1)}% CTR</span>
          </div>
        </div>
      ))}
    </div>
  );
}
