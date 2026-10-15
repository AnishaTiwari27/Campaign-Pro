import type { Campaign } from "../../api/types";
import { formatMoney, formatReach } from "../../lib/format";
import "./Projection.css";

/**
 * Where this campaign lands if it keeps following its own curve.
 *
 * Arithmetic, not a model: the delivery curve is linear in its total,
 * so dividing what has actually been delivered by the share the curve
 * says should be in by now inverts it. Nothing is fitted or learned,
 * which is why it can be shown at all — there is no history of
 * finished campaigns to have learned from.
 *
 * The panel is absent rather than empty when the projection was
 * refused. Dividing by the delivered share amplifies early noise
 * enormously, so a campaign less than a fifth through its flight gets
 * no number, and one that has finished gets its actuals instead of a
 * guess beside a fact.
 */
export function ProjectionPanel({ campaign }: { campaign: Campaign }) {
  const p = campaign.projection;

  if (!p) {
    // Say why rather than showing nothing, so a missing panel does not
    // read as a bug.
    const tooEarly = campaign.flightDays > 0 && campaign.daysRunning < campaign.flightDays * 0.2;
    const finished = campaign.flightDays > 0 && campaign.daysRunning >= campaign.flightDays;
    if (!tooEarly && !finished) return null;
    return (
      <div className="card projection">
        <h4>Projected finish</h4>
        <p className="projection-none">
          {finished
            ? "This flight is over — the figures above are the result, not an estimate."
            : `Too early to project. ${campaign.daysRunning} of ${campaign.flightDays} days have run; a projection this early multiplies one day's noise across the whole flight.`}
        </p>
      </div>
    );
  }

  const elapsed = Math.round(p.elapsedShare * 100);
  const overspending = Boolean(p.exhaustsOnDay);

  return (
    <div className={`card projection${overspending ? " projection-over" : ""}`}>
      <div className="projection-head">
        <h4>Projected finish</h4>
        <span className="projection-confidence">from {elapsed}% of the flight</span>
      </div>

      <div className="projection-figures">
        <div className="projection-figure">
          <span className="projection-label">Reach by day {campaign.flightDays}</span>
          <span className="projection-value mono">{formatReach(p.projectedReach)}</span>
          <span className="projection-from">now {formatReach(campaign.reach)}</span>
        </div>
        <div className="projection-figure">
          <span className="projection-label">Spend by day {campaign.flightDays}</span>
          <span className="projection-value mono">{formatMoney(p.projectedSpend)}</span>
          <span className="projection-from">now {formatMoney(campaign.spend)}</span>
        </div>
        <div className="projection-figure">
          <span className="projection-label">Budget</span>
          <span className="projection-value mono">{formatMoney(campaign.budget)}</span>
          {overspending && p.overspendBy ? (
            <span className="projection-from projection-over-text">over by {formatMoney(p.overspendBy)}</span>
          ) : (
            <span className="projection-from">within plan</span>
          )}
        </div>
      </div>

      {overspending && (
        // The thing the page is actually for: a flag says "over budget
        // pace", this says which day it runs out.
        <p className="projection-warning">
          On this curve the budget is gone on <strong>day {p.exhaustsOnDay}</strong> of {campaign.flightDays} — with{" "}
          {campaign.flightDays - (p.exhaustsOnDay ?? 0)} days of the flight still to run.
        </p>
      )}

      <p className="projection-basis">
        Assumes it keeps to its {campaign.curveShape} delivery curve. Not fitted to past campaigns — this system has
        no record of finished flights to fit to yet.
      </p>
    </div>
  );
}
