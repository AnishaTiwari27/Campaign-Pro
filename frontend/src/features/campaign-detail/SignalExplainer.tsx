import { useInsight } from "../../api/insights";
import "./SignalExplainer.css";

/**
 * Why this campaign is flagged, in a sentence.
 *
 * The figures it cites are shown beneath it. That is not decoration:
 * every number in the text is checked server-side against this exact
 * list before the text is released, and showing the list is what lets
 * a reader confirm that for themselves rather than taking it on faith.
 */
export function SignalExplainer({ campaignId, flagReason }: { campaignId: string; flagReason?: string }) {
  // Only asked for when there is a signal to explain.
  const { data, isLoading, isError } = useInsight(campaignId, Boolean(flagReason));

  if (!flagReason) return null;
  if (isError) return null;

  return (
    <div className="card signal-explainer">
      <div className="signal-explainer-head">
        <h4>Why this is flagged</h4>
        {data?.source === "fallback" && (
          // Said plainly rather than hidden. A reader is entitled to
          // know whether a sentence was written or assembled, and a
          // silent fallback is a bug nobody ever finds.
          <span className="signal-explainer-badge" title="No model is configured, or its answer cited a figure that is not in the data">
            assembled from the figures
          </span>
        )}
      </div>

      {isLoading ? (
        <p className="signal-explainer-text signal-explainer-loading">Reading the figures…</p>
      ) : (
        <p className="signal-explainer-text">{data?.text}</p>
      )}

      {data?.facts && data.facts.length > 0 && (
        <dl className="signal-explainer-facts">
          {data.facts.map((f) => (
            <div key={f.label} className="signal-explainer-fact">
              <dt>{f.label}</dt>
              <dd className="mono">{f.display}</dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  );
}
