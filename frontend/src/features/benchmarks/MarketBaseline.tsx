import { useMarketFacts } from "../../api/endorsements";
import "./MarketBaseline.css";

/**
 * Published figures for the market this product reports on.
 *
 * The benchmarks above this are computed from the agency's own book —
 * a few dozen campaigns — and a reader has no way to tell whether a
 * category median of 88 lakh is normal without something from outside
 * the system to hold it against. These are that: every number is one
 * somebody published, linked to where they published it, none of it
 * derived or adjusted here.
 */
export function MarketBaseline() {
  const { data, isLoading, isError } = useMarketFacts();
  if (isLoading || isError || !data?.items?.length) return null;

  return (
    <div className="card market-baseline">
      <div className="market-head">
        <h4>Published industry baseline</h4>
        <span className="market-note">
          From outside this system — the benchmarks above come from this deployment's own campaigns
        </span>
      </div>

      <div className="market-grid">
        {data.items.map((f) => (
          <div key={f.label + f.period} className="market-fact">
            <div className="market-label">
              {f.label}
              {f.period && <span className="market-period">{f.period}</span>}
            </div>
            <div className="market-value">{f.value}</div>
            {f.note && <div className="market-qual">{f.note}</div>}
            <a className="market-source" href={f.sourceUrl} target="_blank" rel="noreferrer noopener">
              {f.source}
            </a>
          </div>
        ))}
      </div>
    </div>
  );
}
