import { Link } from "react-router-dom";
import type { Campaign, CategoryBenchmark } from "../../api/types";
import { BudgetMeter } from "../../components/Meter";
import { SubjectImage } from "../../components/SubjectImage";
import { formatIndex, formatMoney, formatReach } from "../../lib/format";
import "./SidePanels.css";

export function BudgetPanel({ campaign }: { campaign: Campaign }) {
  return (
    <div className="side-panel card">
      <h4>Budget</h4>
      <div className="budget-panel-figures">
        <span className="mono">{formatMoney(campaign.spend)}</span>
        <span className="side-panel-muted">of {formatMoney(campaign.budget)}</span>
      </div>
      <BudgetMeter spend={campaign.spend} budget={campaign.budget} />
    </div>
  );
}

export function CategoryBenchmarkPanel({ campaign, benchmark }: { campaign: Campaign; benchmark?: CategoryBenchmark }) {
  return (
    <div className="side-panel card">
      <h4>Category benchmark</h4>
      {benchmark ? (
        <div className="category-benchmark-rows">
          <div className="category-benchmark-row">
            <span className="side-panel-muted">{campaign.category} median</span>
            <span className="mono">{formatReach(benchmark.medianReach)}</span>
          </div>
          <div className="category-benchmark-row">
            <span className="side-panel-muted">This campaign</span>
            <span className="mono">{formatReach(campaign.reach)}</span>
          </div>
          <div className="category-benchmark-row">
            <span className="side-panel-muted">Category index</span>
            <span className="mono">{formatIndex(campaign.categoryIndex)}</span>
          </div>
          <div className="category-benchmark-row">
            <span className="side-panel-muted">Top in category</span>
            <span>{benchmark.topCampaign}</span>
          </div>
        </div>
      ) : (
        <p className="side-panel-muted">No other running campaigns in this category yet.</p>
      )}
    </div>
  );
}

export function SimilarPanel({ similar }: { similar: Campaign[] }) {
  return (
    <div className="side-panel card">
      <h4>Similar campaigns</h4>
      {similar.length === 0 ? (
        <p className="side-panel-muted">No similar campaigns found.</p>
      ) : (
        <ul className="similar-list">
          {similar.map((c) => (
            <li key={c.id}>
              <Link to={`/campaigns/${c.id}`} className="similar-item">
                <SubjectImage name={c.name} initials={c.initials} kind={c.subjectType} domain={c.subjectType === "brand" ? c.brandDomain : undefined} seed={c.creatorId ?? c.brandDomain ?? c.name} size={28} />
                <span className="similar-item-body">
                  <span className="similar-item-name truncate">{c.name}</span>
                  <span className="similar-item-meta">
                    {c.category} · {formatReach(c.reach)}
                  </span>
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
