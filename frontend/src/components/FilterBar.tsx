import type { ListParams } from "../api/types";
import { APPROVALS, CATEGORIES, REGIONS, STATUSES } from "../lib/constants";
import "./FilterBar.css";

const RANGE_OPTIONS: { value: string; label: string }[] = [
  { value: "", label: "All time" },
  { value: "7", label: "7d" },
  { value: "30", label: "30d" },
];

const CHIP_LABELS: Partial<Record<keyof ListParams, (v: string) => string>> = {
  q: (v) => `"${v}"`,
  type: (v) => (v === "brand" ? "Brands" : "People"),
  category: (v) => v,
  region: (v) => v,
  status: (v) => v[0].toUpperCase() + v.slice(1),
  approval: (v) => v[0].toUpperCase() + v.slice(1),
  range: (v) => (v === "7" ? "Last 7 days" : v === "30" ? "Last 30 days" : v),
};

const CHIP_KEYS: (keyof ListParams)[] = ["q", "type", "category", "region", "status", "approval", "range"];

export function FilterBar({
  params,
  onChange,
  shown,
  total,
}: {
  params: ListParams;
  onChange: (patch: Partial<ListParams>) => void;
  shown: number;
  total: number;
}) {
  const activeChips = CHIP_KEYS.filter((k) => !!params[k]).map((k) => ({
    key: k,
    label: CHIP_LABELS[k]!(String(params[k])),
  }));

  return (
    <div className="filter-bar">
      <div className="filter-bar-row">
        <input
          className="input filter-bar-search"
          type="search"
          placeholder="Search name, category, region, ad type, role…"
          value={params.q ?? ""}
          onChange={(e) => onChange({ q: e.target.value })}
          aria-label="Search campaigns"
        />

        <select className="select" value={params.type ?? ""} onChange={(e) => onChange({ type: e.target.value })} aria-label="Subject type">
          <option value="">All subjects</option>
          <option value="brand">Brands</option>
          <option value="person">People</option>
        </select>

        <select className="select" value={params.category ?? ""} onChange={(e) => onChange({ category: e.target.value })} aria-label="Category">
          <option value="">All categories</option>
          {CATEGORIES.map((c) => (
            <option key={c} value={c}>
              {c}
            </option>
          ))}
        </select>

        <select className="select" value={params.region ?? ""} onChange={(e) => onChange({ region: e.target.value })} aria-label="Region">
          <option value="">All regions</option>
          {REGIONS.map((r) => (
            <option key={r} value={r}>
              {r}
            </option>
          ))}
        </select>

        <select className="select" value={params.status ?? ""} onChange={(e) => onChange({ status: e.target.value })} aria-label="Status">
          <option value="">All statuses</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {s[0].toUpperCase() + s.slice(1)}
            </option>
          ))}
        </select>

        <select className="select" value={params.approval ?? ""} onChange={(e) => onChange({ approval: e.target.value })} aria-label="Approval">
          <option value="">All approvals</option>
          {APPROVALS.map((a) => (
            <option key={a} value={a}>
              {a[0].toUpperCase() + a.slice(1)}
            </option>
          ))}
        </select>

        <div className="filter-bar-range" role="radiogroup" aria-label="Date range">
          {RANGE_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              type="button"
              role="radio"
              aria-checked={(params.range ?? "") === opt.value}
              className={`filter-bar-range-opt${(params.range ?? "") === opt.value ? " filter-bar-range-opt-active" : ""}`}
              onClick={() => onChange({ range: opt.value })}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      <div className="filter-bar-chips">
        <span className="filter-bar-count">
          Showing {shown} of {total}
        </span>
        {activeChips.map((chip) => (
          <button key={chip.key} type="button" className="filter-chip" onClick={() => onChange({ [chip.key]: undefined })}>
            {chip.label}
            <span aria-hidden="true">×</span>
          </button>
        ))}
        {activeChips.length > 0 && (
          <button
            type="button"
            className="filter-bar-clear"
            onClick={() => onChange(Object.fromEntries(CHIP_KEYS.map((k) => [k, undefined])))}
          >
            Clear all
          </button>
        )}
      </div>
    </div>
  );
}
