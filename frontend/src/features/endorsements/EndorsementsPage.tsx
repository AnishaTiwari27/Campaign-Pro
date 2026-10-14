import { Link } from "react-router-dom";
import { useEndorsements } from "../../api/endorsements";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonRows } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import "./Endorsements.css";

export function EndorsementsPage() {
  useSetBreadcrumbs([{ label: "Public record" }]);
  const { data, isLoading, isError } = useEndorsements();
  const items = data?.items ?? [];

  return (
    <div className="endorsements-page">
      <div className="page-header">
        <div>
          <h1>Public record</h1>
        </div>
      </div>

      {/* Said once, plainly, at the top. Everything else in this product
          is invented fixture data; these are real people, so the line
          between the two has to be visible rather than implied. */}
      <p className="endorsements-lede">
        Real people and the brand relationships reported about them in public sources. This is reference material, not
        the agency's book — there is no spend, reach or approval here, and none of these campaigns ran through this
        tool.
      </p>

      {isLoading ? (
        <SkeletonRows count={6} height={120} />
      ) : isError ? (
        <EmptyState title="Couldn't load the record" description="Please try refreshing the page." />
      ) : (
        <div className="endorsement-grid">
          {items.map((p) => {
            // An ownership stake is not an ad, so it is counted apart —
            // but someone whose public brand work is *all* stakes (Dhoni
            // co-owns the brands he fronts) must not read "0 reported".
            const ads = p.deals.filter((d) => d.kind !== "stake");
            const shown = ads.length > 0 ? ads : p.deals;
            const label = ads.length > 0 ? "reported" : "owned or invested in";
            return (
              <Link key={p.id} to={`/endorsements/${p.id}`} className="endorsement-card card">
                <span className="endorsement-art">
                  <img className="endorsement-photo" src={p.photo.url} alt={p.name} loading="lazy" />
                  <span className="endorsement-art-text">
                    <span className="endorsement-art-name">{p.name}</span>
                    <span className="endorsement-art-role">{p.role}</span>
                  </span>
                </span>
                <span className="endorsement-card-body">
                  <span className="endorsement-count">
                    {shown.length} brand{shown.length === 1 ? "" : "s"} {label}
                  </span>
                  <span className="endorsement-brands truncate">{shown.map((d) => d.brand).join(" · ")}</span>
                </span>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
