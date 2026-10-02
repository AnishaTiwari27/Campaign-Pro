import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useCampaignDetail, useDecision, usePause } from "../../api/campaigns";
import { useBenchmark } from "../../api/overview";
import { useSession } from "../../api/auth";
import { useListParams } from "../../lib/useListParams";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { useKeyboardShortcuts } from "../../lib/keyboard";
import { Pill } from "../../components/Pill";
import { ApprovalTag, FlaggedTag } from "../../components/Tag";
import { AdTypeTag } from "../../components/AdTypeTag";
import { SubjectImage } from "../../components/SubjectImage";
import { EmptyState } from "../../components/EmptyState";
import { SkeletonBlock } from "../../components/Skeleton";
import { ApiError } from "../../api/client";
import { StatRail } from "./StatRail";
import { PerformanceTab } from "./PerformanceTab";
import { CreativesTab } from "./CreativesTab";
import { ActivityTab } from "./ActivityTab";
import { BudgetPanel, CategoryBenchmarkPanel, SimilarPanel } from "./SidePanels";
import "./CampaignDetailPage.css";

const TABS = [
  { key: "performance", label: "Performance" },
  { key: "creatives", label: "Creatives" },
  { key: "activity", label: "Activity" },
];

export function CampaignDetailPage() {
  const { id = "" } = useParams<{ id: string }>();
  const [listParams] = useListParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();

  const tab = searchParams.get("tab") ?? "performance";
  function setTab(t: string) {
    const next = new URLSearchParams(searchParams);
    next.set("tab", t);
    setSearchParams(next, { replace: true });
  }

  const { data: campaign, isLoading, error } = useCampaignDetail(id, listParams);
  const { data: benchmark } = useBenchmark();
  const decision = useDecision();
  const pause = usePause();
  const { data: me } = useSession();
  const canApprove = me?.canApprove ?? false;

  useSetBreadcrumbs([
    { label: "Campaigns", href: `/campaigns${window.location.search}` },
    { label: campaign?.name ?? "…" },
  ]);

  useKeyboardShortcuts({
    onPrev: () => campaign?.prevId && navigate({ pathname: `/campaigns/${campaign.prevId}`, search: window.location.search }),
    onNext: () => campaign?.nextId && navigate({ pathname: `/campaigns/${campaign.nextId}`, search: window.location.search }),
  });

  if (isLoading) {
    return <SkeletonBlock height={400} />;
  }

  const notFound = error instanceof ApiError && error.status === 404;
  if (notFound || !campaign) {
    return (
      <EmptyState
        title="Campaign not found"
        description="This campaign doesn't exist, or the link is out of date."
        action={
          <Link to="/campaigns" className="btn btn-primary">
            Back to Campaigns
          </Link>
        }
      />
    );
  }

  const categoryBenchmark = benchmark?.categories.find((c) => c.category === campaign.category);
  const canPause = campaign.status === "live" || campaign.status === "paused";

  return (
    <div className="campaign-detail-page">
      <div className="campaign-detail-header">
        <div className="campaign-detail-header-main">
          <SubjectImage
            name={campaign.name}
            initials={campaign.initials}
            kind={campaign.subjectType}
            domain={campaign.subjectType === "brand" ? campaign.brandDomain : undefined}
            seed={campaign.creatorId ?? campaign.brandDomain ?? campaign.name}
            size={52}
          />
          <div>
            <div className="campaign-detail-title-row">
              <h1>{campaign.name}</h1>
              <Pill status={campaign.status} />
              <ApprovalTag approval={campaign.approval} />
              <AdTypeTag adType={campaign.adType} />
              {campaign.flagReason && <FlaggedTag />}
            </div>
            <div className="campaign-detail-subtitle">
              {campaign.role || campaign.category} ·{" "}
              <Link to={`/regions/${encodeURIComponent(campaign.region)}`}>{campaign.region}</Link> · {campaign.platform}
              {campaign.flagReason && <span className="campaign-detail-flag-reason"> · {campaign.flagReason}</span>}
            </div>
          </div>
        </div>

        <div className="campaign-detail-actions">
          <div className="campaign-detail-stepper">
            <button
              type="button"
              className="btn btn-icon btn-ghost"
              disabled={!campaign.prevId}
              aria-label="Previous campaign"
              onClick={() => campaign.prevId && navigate({ pathname: `/campaigns/${campaign.prevId}`, search: window.location.search })}
            >
              ‹
            </button>
            <span className="mono campaign-detail-stepper-count">
              {campaign.position} / {campaign.total}
            </span>
            <button
              type="button"
              className="btn btn-icon btn-ghost"
              disabled={!campaign.nextId}
              aria-label="Next campaign"
              onClick={() => campaign.nextId && navigate({ pathname: `/campaigns/${campaign.nextId}`, search: window.location.search })}
            >
              ›
            </button>
          </div>

          {canApprove && canPause && (
            <button type="button" className="btn" onClick={() => pause.mutate(campaign.id)}>
              {campaign.status === "live" ? "Pause" : "Resume"}
            </button>
          )}

          {!canApprove ? null : campaign.approval === "pending" ? (
            <>
              <button type="button" className="btn btn-primary" onClick={() => decision.mutate({ id: campaign.id, action: "approve" })}>
                Approve
              </button>
              <button type="button" className="btn btn-danger" onClick={() => decision.mutate({ id: campaign.id, action: "reject" })}>
                Reject
              </button>
            </>
          ) : (
            <button type="button" className="btn" onClick={() => decision.mutate({ id: campaign.id, action: "reopen" })}>
              Reopen review
            </button>
          )}
        </div>
      </div>

      <StatRail campaign={campaign} />

      <div className="campaign-detail-body">
        <div className="campaign-detail-main">
          <div className="campaign-detail-tabs" role="tablist">
            {TABS.map((t) => (
              <button
                key={t.key}
                type="button"
                role="tab"
                aria-selected={tab === t.key}
                className={`campaign-detail-tab${tab === t.key ? " campaign-detail-tab-active" : ""}`}
                onClick={() => setTab(t.key)}
              >
                {t.label}
                {t.key === "creatives" && ` (${campaign.creatives.length})`}
              </button>
            ))}
          </div>

          <div className="campaign-detail-tab-panel">
            {tab === "performance" && <PerformanceTab campaign={campaign} categoryMedian={categoryBenchmark?.medianReach} />}
            {tab === "creatives" && <CreativesTab creatives={campaign.creatives} />}
            {tab === "activity" && <ActivityTab campaignId={campaign.id} audit={campaign.audit} />}
          </div>
        </div>

        <div className="campaign-detail-side">
          <BudgetPanel campaign={campaign} />
          <CategoryBenchmarkPanel campaign={campaign} benchmark={categoryBenchmark} />
          <SimilarPanel similar={campaign.similar} />
        </div>
      </div>
    </div>
  );
}
