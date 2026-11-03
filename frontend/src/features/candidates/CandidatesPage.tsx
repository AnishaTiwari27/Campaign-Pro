import { useState } from "react";
import { useCandidates, usePromoteCandidate, useRejectCandidate } from "../../api/candidates";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { SkeletonRows } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { formatFollowers } from "../creators/CreatorBits";
import type { Candidate } from "../../api/types";
import "./CandidatesPage.css";

/** Promoting a candidate needs three things no search result carries.
 *
 *  Category, region and language are the axes every benchmark slices
 *  on, and none of them can be read off a channel title. Asking is the
 *  only honest option: a guess here would quietly skew the medians a
 *  buyer reads, and it would be invisible once it had.
 */
function PromoteForm({ candidate, onDone }: { candidate: Candidate; onDone: () => void }) {
  const promote = usePromoteCandidate();
  const [category, setCategory] = useState("");
  const [region, setRegion] = useState("");
  const [languages, setLanguages] = useState("");
  const [followers, setFollowers] = useState("");

  const needsFollowers = !candidate.hasFollowers;
  const ready = category.trim() && region.trim() && (!needsFollowers || Number(followers) > 0);

  return (
    <form
      className="candidate-promote"
      onSubmit={(e) => {
        e.preventDefault();
        if (!ready) return;
        promote.mutate(
          {
            id: candidate.id,
            body: {
              category: category.trim(),
              region: region.trim(),
              languages: languages.split(",").map((l) => l.trim()).filter(Boolean),
              ...(needsFollowers ? { followers: Number(followers) } : {}),
            },
          },
          { onSuccess: onDone },
        );
      }}
    >
      <label className="field">
        <span>Category</span>
        <input value={category} onChange={(e) => setCategory(e.target.value)} placeholder="Comedy" required />
      </label>
      <label className="field">
        <span>Region</span>
        <input value={region} onChange={(e) => setRegion(e.target.value)} placeholder="Maharashtra" required />
      </label>
      <label className="field">
        <span>Languages</span>
        <input value={languages} onChange={(e) => setLanguages(e.target.value)} placeholder="Hindi, English" />
      </label>
      {needsFollowers && (
        <label className="field">
          <span>Subscribers</span>
          <input
            type="number"
            min={1}
            value={followers}
            onChange={(e) => setFollowers(e.target.value)}
            placeholder="Hidden by the channel"
            required
          />
        </label>
      )}
      <div className="candidate-promote-actions">
        <button type="submit" className="btn btn-sm btn-primary" disabled={!ready || promote.isPending}>
          {promote.isPending ? "Adding…" : "Add to roster"}
        </button>
        <button type="button" className="btn btn-sm" onClick={onDone}>
          Cancel
        </button>
      </div>
      {promote.isError && (
        <p className="candidate-error">{(promote.error as Error).message}</p>
      )}
    </form>
  );
}

export function CandidatesPage() {
  useSetBreadcrumbs([{ label: "Discovered" }]);
  const { data, isLoading } = useCandidates();
  const reject = useRejectCandidate();
  const [promoting, setPromoting] = useState<string | null>(null);
  const items = data?.items ?? [];

  return (
    <div className="candidates-page">
      <div className="page-header">
        <div>
          <h1>Discovered</h1>
          <p className="candidates-lede">
            Channels YouTube search turned up, waiting on a decision. Nothing here affects any figure in the product
            until it is added to the roster — every benchmark is computed from the roster, so a channel nobody has
            looked at would move medians a buyer reads.
          </p>
        </div>
      </div>

      {isLoading ? (
        <SkeletonRows count={5} height={76} />
      ) : items.length === 0 ? (
        <EmptyState
          title="Nothing waiting"
          description="Run a discovery search to find channels. Each brief costs 100 of the day's 10,000 YouTube quota units."
        />
      ) : (
        <div className="candidate-list">
          {items.map((c) => (
            <div key={c.id} className="candidate">
              <div className="candidate-main">
                <div className="candidate-identity">
                  <a className="candidate-name" href={c.url} target="_blank" rel="noreferrer">
                    {c.name}
                  </a>
                  <span className="candidate-meta">
                    {/* Hidden is not zero: YouTube lets a channel withhold
                        its count, and showing 0 would read as a dead
                        channel. */}
                    {c.hasFollowers ? `${formatFollowers(c.followers)} subscribers` : "Subscribers hidden"}
                    {" · "}
                    {c.videos.toLocaleString()} videos
                  </span>
                  {c.foundFor && (
                    <span className="candidate-found" title="The search that found this channel">
                      found for “{c.foundFor}”
                    </span>
                  )}
                </div>
                {promoting !== c.id && (
                  <div className="candidate-actions">
                    <button type="button" className="btn btn-sm btn-primary" onClick={() => setPromoting(c.id)}>
                      Add to roster
                    </button>
                    <button
                      type="button"
                      className="btn btn-sm"
                      onClick={() => reject.mutate(c.id)}
                      disabled={reject.isPending}
                      title="Keeps them out of future searches too"
                    >
                      Not relevant
                    </button>
                  </div>
                )}
              </div>
              {c.description && <p className="candidate-desc">{c.description}</p>}
              {promoting === c.id && <PromoteForm candidate={c} onDone={() => setPromoting(null)} />}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
