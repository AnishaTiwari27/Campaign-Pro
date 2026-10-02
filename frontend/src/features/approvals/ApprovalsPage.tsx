import { Link } from "react-router-dom";
import { useCampaignsList, useDecision } from "../../api/campaigns";
import { useSession } from "../../api/auth";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { Pill } from "../../components/Pill";
import { FlaggedTag } from "../../components/Tag";
import { AdTypeTag, adTypeStripeStyle } from "../../components/AdTypeTag";
import { SubjectImage } from "../../components/SubjectImage";
import { SkeletonRows } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { formatIndex, formatMoney, formatReach, formatPct, timeAgo } from "../../lib/format";
import { approvalBlockOf } from "../../lib/metrics";
import "./ApprovalsPage.css";

export function ApprovalsPage() {
  useSetBreadcrumbs([{ label: "Approvals" }]);

  const { data, isLoading } = useCampaignsList({ approval: "pending", sort: "spend", dir: "desc", page: 1, per: 500 });
  const decision = useDecision();
  const { data: me } = useSession();
  const canApprove = me?.canApprove ?? false;

  const items = data?.items ?? [];
  const totalSpend = items.reduce((sum, c) => sum + c.spend, 0);

  return (
    <div className="approvals-page">
      <div className="page-header">
        <div>
          <h1>Approvals</h1>
          {items.length > 0 && (
            <p className="approvals-subhead">
              {items.length} campaign{items.length === 1 ? "" : "s"} waiting · {formatMoney(totalSpend)} unreviewed spend
            </p>
          )}
        </div>
      </div>

      {isLoading ? (
        <SkeletonRows count={4} height={140} />
      ) : items.length === 0 ? (
        <EmptyState title="Inbox zero" description="Nothing is waiting on a decision right now." />
      ) : (
        <div className="approval-cards">
          {items.map((c) => {
            // Over-budget campaigns can be rejected but not approved: the
            // budget has to move first. Mirrors the server's own refusal.
            const approvalBlock = approvalBlockOf(c.spend, c.budget);
            return (
              <div key={c.id} className="approval-card card" style={adTypeStripeStyle(c.adType)}>
                <div className="approval-card-stripe" />
                <div className="approval-card-main">
                  <SubjectImage name={c.name} initials={c.initials} kind={c.subjectType} domain={c.subjectType === "brand" ? c.brandDomain : undefined} seed={c.creatorId ?? c.brandDomain ?? c.name} size={40} />
                  <div className="approval-card-body">
                    <div className="approval-card-title-row">
                      <span className="approval-card-name">{c.name}</span>
                      <Pill status={c.status} />
                      <AdTypeTag adType={c.adType} />
                      {c.flagReason && <FlaggedTag />}
                    </div>
                    <div className="approval-card-meta">
                      {c.role || c.category} · {c.region} · {c.platform} · submitted {timeAgo(c.createdAt)}
                    </div>
                    {c.flagReason && <div className="approval-card-flag-reason">{c.flagReason}</div>}

                    <div className="approval-card-facts">
                      <div>
                        <div className="approval-fact-label">Reach</div>
                        <div className="approval-fact-value mono">{formatReach(c.reach)}</div>
                      </div>
                      <div>
                        <div className="approval-fact-label">Spend</div>
                        <div className="approval-fact-value mono">{formatMoney(c.spend)}</div>
                      </div>
                      <div>
                        <div className="approval-fact-label">Budget pace</div>
                        <div className={`approval-fact-value mono${c.pace >= 100 ? " approval-fact-value-crit" : ""}`}>{formatPct(c.pace)}</div>
                      </div>
                      <div>
                        <div className="approval-fact-label">Index vs median</div>
                        <div className="approval-fact-value mono">{formatIndex(c.index)}</div>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="approval-card-actions">
                  <Link to={`/campaigns/${c.id}`} className="btn">
                    Open
                  </Link>
                  {canApprove && (
                    <>
                      <button type="button" className="btn btn-danger" onClick={() => decision.mutate({ id: c.id, action: "reject" })}>
                        Reject
                      </button>
                      <button
                        type="button"
                        className="btn btn-primary"
                        disabled={approvalBlock !== null}
                        title={approvalBlock ? `${approvalBlock} — raise the budget or pause the campaign first` : undefined}
                        onClick={() => decision.mutate({ id: c.id, action: "approve" })}
                      >
                        Approve
                      </button>
                    </>
                  )}
                </div>
                {canApprove && approvalBlock && (
                  <p className="approval-card-blocked">Can't approve: {approvalBlock.toLowerCase()}. Raise the budget or pause the campaign first.</p>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
