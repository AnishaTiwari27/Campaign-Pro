import { Link } from "react-router-dom";
import { useEndorsements } from "../../api/endorsements";
import { useLaunches } from "../../api/radar";
import "./RealWork.css";

/**
 * The part of this dashboard that is not demo data.
 *
 * Everything in the campaign rails below — reach, spend, approvals —
 * is seeded fixture data, because per-campaign performance is not
 * published by anyone: it is precisely what agencies are paid to know.
 * It becomes real when a platform is connected.
 *
 * What is real today is on this row, and it was previously buried two
 * clicks away on pages nobody lands on: real public figures with the
 * brand relationships actually reported about them, and real product
 * launches polled this morning. Putting it first is the honest
 * ordering — the verifiable material should not be the hidden part.
 */
export function RealWork() {
  const { data: people } = useEndorsements();
  const { data: launches } = useLaunches();

  const roster = people?.items ?? [];
  const soon = (launches?.items ?? []).filter((l) => l.readiness === "soon");
  if (roster.length === 0 && soon.length === 0) return null;

  return (
    <section className="real-work">
      <div className="real-work-head">
        <h2>
          Real, sourced data
          <span className="real-work-badge">live</span>
        </h2>
        <p className="real-work-sub">
          Public figures and their documented brand work, and products that launched this week. Every figure below
          links to where it was published. The campaign rails further down are seeded demo data until an ad platform
          is connected.
        </p>
      </div>

      {roster.length > 0 && (
        <div className="real-work-people">
          {roster.slice(0, 6).map((p) => {
            // An ownership stake is not an endorsement, so it is
            // counted apart — but someone whose public brand work is
            // entirely stakes (Dhoni co-owns the brands he fronts)
            // must not read "0 brands reported". Same slip as the
            // radar list, made twice.
            const ads = p.deals.filter((d) => d.kind !== "stake");
            const deals = ads.length > 0 ? ads : p.deals;
            const owned = ads.length === 0 && p.deals.length > 0;
            const lead = deals[0];
            return (
              <Link key={p.id} to={`/endorsements/${p.id}`} className="real-person card">
                <img className="real-person-photo" src={p.photo.url} alt={p.name} loading="lazy" />
                <div className="real-person-body">
                  <div className="real-person-name">{p.name}</div>
                  <div className="real-person-role">{p.role}</div>
                  {lead && (
                    <div className="real-person-deal">
                      <strong>{lead.brand}</strong>
                      {lead.period ? ` · ${lead.period}` : ""}
                    </div>
                  )}
                  <div className="real-person-more">
                    {deals.length} brand{deals.length === 1 ? "" : "s"} {owned ? "owned or invested in" : "reported"}
                  </div>
                </div>
              </Link>
            );
          })}
        </div>
      )}

      {soon.length > 0 && (
        <div className="real-work-launches card">
          <div className="real-launch-head">
            <h4>Launched recently, worth a call</h4>
            <Link to="/radar" className="real-launch-all">
              All launches ›
            </Link>
          </div>
          {soon.slice(0, 4).map((l) => (
            <div key={l.id} className="real-launch">
              <span className="real-launch-name">
                {l.url ? (
                  <a href={l.url} target="_blank" rel="noreferrer noopener">
                    {l.name}
                  </a>
                ) : (
                  l.name
                )}
              </span>
              <span className="real-launch-basis">{l.basis}</span>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
