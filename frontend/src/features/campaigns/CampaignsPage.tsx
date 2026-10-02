import { useLocation, useNavigate } from "react-router-dom";
import { useEffect, useMemo } from "react";
import { useCampaignsList, useBulkDecision, useDecision } from "../../api/campaigns";
import { useListParams } from "../../lib/useListParams";
import { useUIStore } from "../../app/useUIStore";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { useLastFilterStore } from "../../app/lastFilterStore";
import { useSession } from "../../api/auth";
import { FilterBar } from "../../components/FilterBar";
import { DataTable, type Column } from "../../components/DataTable";
import { Pagination } from "../../components/Pagination";
import { Pill } from "../../components/Pill";
import { ApprovalTag } from "../../components/Tag";
import { AdTypeTag, adTypeStripeStyle } from "../../components/AdTypeTag";
import { SubjectImage } from "../../components/SubjectImage";
import { PaceMeter } from "../../components/Meter";
import { SkeletonRows } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { formatMoney, formatReach } from "../../lib/format";
import { approvalBlockOf } from "../../lib/metrics";
import { exportUrl } from "../../api/client";
import type { Campaign } from "../../api/types";
import "./CampaignsPage.css";

export function CampaignsPage() {
  useSetBreadcrumbs([{ label: "Campaigns" }]);

  const [params, setParams] = useListParams();
  const { data, isLoading, isError } = useCampaignsList(params);
  const density = useUIStore((s) => s.density);
  const setDensity = useUIStore((s) => s.setDensity);
  const selectedIds = useUIStore((s) => s.selectedIds);
  const toggleSelected = useUIStore((s) => s.toggleSelected);
  const selectMany = useUIStore((s) => s.selectMany);
  const clearSelection = useUIStore((s) => s.clearSelection);
  const decision = useDecision();
  const { data: me } = useSession();
  const canApprove = me?.canApprove ?? false;
  const bulkDecision = useBulkDecision();
  const navigate = useNavigate();
  const location = useLocation();
  const setLastCampaignsParams = useLastFilterStore((s) => s.setLastCampaignsParams);

  const items = data?.items ?? [];

  // Reports' "New from current filters" reads this after navigating away.
  useEffect(() => {
    setLastCampaignsParams(params);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params.q, params.type, params.category, params.region, params.status, params.approval, params.range, params.sort, params.dir]);

  // Selection is page-scoped: clear it whenever the visible rows change.
  useEffect(() => {
    clearSelection();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params.page, params.sort, params.dir, params.category, params.region, params.status, params.approval, params.q, params.type, params.range]);

  function handleSort(key: string) {
    if (params.sort === key) {
      setParams({ dir: params.dir === "asc" ? "desc" : "asc" });
    } else {
      setParams({ sort: key, dir: "desc" });
    }
  }

  const columns: Column<Campaign>[] = useMemo(
    () => [
      {
        key: "name",
        label: "Subject",
        sortable: true,
        render: (c) => (
          <div className="campaign-subject" style={adTypeStripeStyle(c.adType)}>
            <span className="campaign-subject-stripe" />
            <SubjectImage name={c.name} initials={c.initials} kind={c.subjectType} domain={c.subjectType === "brand" ? c.brandDomain : undefined} seed={c.creatorId ?? c.brandDomain ?? c.name} size={28} />
            <span className="campaign-subject-text">
              <span className="campaign-subject-name truncate">{c.name}</span>
              <span className="campaign-subject-hint">Open →</span>
            </span>
          </div>
        ),
      },
      { key: "category", label: "Category", sortable: true, render: (c) => c.category },
      { key: "region", label: "Region", sortable: true, render: (c) => c.region },
      { key: "adType", label: "Ad type", render: (c) => <AdTypeTag adType={c.adType} /> },
      { key: "status", label: "Status", sortable: true, render: (c) => <Pill status={c.status} /> },
      { key: "reach", label: "Reach", sortable: true, className: "mono", render: (c) => formatReach(c.reach) },
      { key: "spend", label: "Spend", sortable: true, className: "mono", render: (c) => formatMoney(c.spend) },
      {
        key: "pace",
        label: "Budget pace",
        sortable: true,
        className: "campaigns-pace-col",
        render: (c) => <PaceMeter pace={c.pace} />,
      },
      { key: "approval", label: "Approval", sortable: true, render: (c) => <ApprovalTag approval={c.approval} /> },
      {
        key: "actions",
        label: "",
        render: (c) => {
          if (!canApprove || c.approval !== "pending") return null;
          // Over budget: the server refuses the approval, so don't offer it.
          const approvalBlock = approvalBlockOf(c.spend, c.budget);
          return (
            <div className="campaign-row-actions">
              <button
                type="button"
                className="btn btn-sm btn-primary"
                disabled={approvalBlock !== null}
                title={approvalBlock ? `${approvalBlock} — raise the budget or pause the campaign first` : undefined}
                onClick={(e) => {
                  e.stopPropagation();
                  decision.mutate({ id: c.id, action: "approve" });
                }}
              >
                Approve
              </button>
              <button
                type="button"
                className="btn btn-sm btn-danger"
                onClick={(e) => {
                  e.stopPropagation();
                  decision.mutate({ id: c.id, action: "reject" });
                }}
              >
                Reject
              </button>
            </div>
          );
        },
      },
    ],
    [decision, canApprove],
  );

  return (
    <div className="campaigns-page">
      <div className="page-header">
        <div>
          <h1>Campaigns</h1>
        </div>
        <div className="page-header-actions">
          <div className="density-toggle" role="radiogroup" aria-label="Table density">
            <button
              type="button"
              role="radio"
              aria-checked={density === "comfortable"}
              className={`density-opt${density === "comfortable" ? " density-opt-active" : ""}`}
              onClick={() => setDensity("comfortable")}
            >
              Comfortable
            </button>
            <button
              type="button"
              role="radio"
              aria-checked={density === "compact"}
              className={`density-opt${density === "compact" ? " density-opt-active" : ""}`}
              onClick={() => setDensity("compact")}
            >
              Compact
            </button>
          </div>
          <a className="btn" href={exportUrl(params as Record<string, string | number | undefined>)}>
            Export CSV
          </a>
        </div>
      </div>

      <FilterBar params={params} onChange={setParams} shown={items.length} total={data?.total ?? 0} />

      {canApprove && selectedIds.size > 0 && (
        <div className="bulk-bar">
          <span>{selectedIds.size} selected</span>
          <button
            type="button"
            className="btn btn-sm btn-primary"
            onClick={() => bulkDecision.mutate({ ids: [...selectedIds], action: "approve" }, { onSuccess: clearSelection })}
          >
            Approve
          </button>
          <button
            type="button"
            className="btn btn-sm btn-danger"
            onClick={() => bulkDecision.mutate({ ids: [...selectedIds], action: "reject" }, { onSuccess: clearSelection })}
          >
            Reject
          </button>
          <button type="button" className="btn btn-sm btn-ghost" onClick={clearSelection}>
            Clear
          </button>
        </div>
      )}

      {isLoading ? (
        <SkeletonRows count={8} height={48} />
      ) : isError ? (
        <EmptyState title="Couldn't load campaigns" description="Please try refreshing the page." />
      ) : items.length === 0 ? (
        <EmptyState title="No campaigns match these filters" description="Try clearing a filter or widening the date range." />
      ) : (
        <>
          <DataTable
            columns={columns}
            rows={items}
            sort={params.sort}
            dir={params.dir}
            onSort={handleSort}
            density={density}
            selectable
            selectedIds={selectedIds}
            onToggleSelect={toggleSelected}
            onToggleSelectAll={() => (selectedIds.size === items.length ? clearSelection() : selectMany(items.map((c) => c.id)))}
            onRowClick={(c) => navigate({ pathname: `/campaigns/${c.id}`, search: location.search })}
          />
          <Pagination
            page={data?.page ?? 1}
            pages={data?.pages ?? 1}
            onPageChange={(p) => setParams({ page: p })}
            sortLabel={`Sorted by ${params.sort}, ${params.dir === "asc" ? "ascending" : "descending"}`}
          />
        </>
      )}
    </div>
  );
}
