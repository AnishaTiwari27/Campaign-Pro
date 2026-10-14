import { Link, useParams } from "react-router-dom";
import { useEndorsementPerson } from "../../api/endorsements";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonBlock } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import "./Endorsements.css";

const KIND_LABEL: Record<string, string> = {
  endorsement: "Endorsement",
  ambassador: "Ambassador",
  stake: "Stake",
};

export function EndorsementDetailPage() {
  const { id = "" } = useParams<{ id: string }>();
  const { data, isLoading, isError } = useEndorsementPerson(id);

  useSetBreadcrumbs([{ label: "Public record", href: "/endorsements" }, { label: data?.name ?? "…" }]);

  if (isLoading) return <SkeletonBlock height={400} />;
  if (isError || !data) {
    return (
      <EmptyState
        title="Not in the public record"
        description="No one by that id is listed here."
        action={
          <Link to="/endorsements" className="btn btn-primary">
            Back to the record
          </Link>
        }
      />
    );
  }

  // An ownership stake is not an advertisement, and listing the two
  // together would misreport both.
  const ads = data.deals.filter((d) => d.kind !== "stake");
  const stakes = data.deals.filter((d) => d.kind === "stake");

  return (
    <div className="endorsement-detail-page">
      <div className="endorsement-hero card">
        <img className="endorsement-hero-photo" src={data.photo.url} alt={data.name} />
        <div className="endorsement-hero-text">
          <h1>{data.name}</h1>
          <p className="endorsement-hero-sub">
            {data.role} · {data.field}
          </p>
          <p className="endorsement-hero-note">
            Brand relationships reported in public sources. No performance data: this person is not in the agency's
            book, and every spend and reach figure elsewhere in this product is invented.
          </p>
          <p className="endorsement-hero-links">
            <a href={data.source} target="_blank" rel="noreferrer noopener">
              Source
            </a>
          </p>
        </div>
      </div>

      <div className="card endorsement-panel">
        <div className="endorsement-panel-head">
          <h4>Brand work</h4>
          <span className="endorsement-panel-count">{ads.length} reported</span>
        </div>
        <div className="endorsement-list">
          {ads.map((d) => (
            <div key={d.brand + d.period} className="endorsement-row">
              <span className="endorsement-brand">{d.brand}</span>
              <span className={`endorsement-kind endorsement-kind-${d.kind}`}>{KIND_LABEL[d.kind]}</span>
              <span className="endorsement-period mono">{d.period || "—"}</span>
              <span className="endorsement-note truncate">{d.note || ""}</span>
            </div>
          ))}
        </div>
      </div>

      {stakes.length > 0 && (
        <div className="card endorsement-panel">
          <div className="endorsement-panel-head">
            <h4>Ownership and investment</h4>
            <span className="endorsement-panel-count">Not advertising</span>
          </div>
          <div className="endorsement-list">
            {stakes.map((d) => (
              <div key={d.brand} className="endorsement-row">
                <span className="endorsement-brand">{d.brand}</span>
                <span className="endorsement-kind endorsement-kind-stake">Stake</span>
                <span className="endorsement-period mono">{d.period || "—"}</span>
                <span className="endorsement-note truncate">{d.note || ""}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* A condition of every licence these photographs carry, so it is
          part of the page rather than a credits file nobody opens. */}
      <p className="endorsement-credit">
        Photograph by {data.photo.artist} ·{" "}
        <a href={data.photo.licenseUrl} target="_blank" rel="noreferrer noopener">
          {data.photo.license}
        </a>{" "}
        ·{" "}
        <a href={data.photo.source} target="_blank" rel="noreferrer noopener">
          Wikimedia Commons
        </a>
      </p>
    </div>
  );
}
