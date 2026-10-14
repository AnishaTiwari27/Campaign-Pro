import { useMemo, useState } from "react";
import { useBenchmarkTrend } from "../../api/overview";
import { AreaChart } from "../../components/AreaChart";
import { formatMoney, formatReach } from "../../lib/format";
import "./BenchmarkTrend.css";

type Measure = "medianReach" | "medianCostPerLakh";

const MEASURES: { key: Measure; label: string; format: (v: number) => string; note: string }[] = [
  {
    key: "medianReach",
    label: "Median reach",
    format: formatReach,
    note: "Half the category's running campaigns reached more than this, half less.",
  },
  {
    key: "medianCostPerLakh",
    label: "Cost per lakh",
    format: formatMoney,
    note: "Median rupees per lakh of reach. Rising means the category is getting dearer to buy.",
  },
];

/**
 * The benchmark page answers "where does this category stand" from the
 * campaigns running right now. This answers the question people ask
 * next: which way is it moving.
 *
 * One category at a time rather than every series at once. Eight
 * overlapping lines is the chart nobody can read, and the comparison
 * across categories is already on the page above — this is here for the
 * direction of travel within one.
 */
export function BenchmarkTrend({ categories }: { categories: string[] }) {
  const { data, isLoading, isError } = useBenchmarkTrend();
  const [category, setCategory] = useState(categories[0] ?? "");
  const [measure, setMeasure] = useState<Measure>("medianReach");

  const points = useMemo(() => {
    const rows = (data?.items ?? []).filter((p) => p.category === category);
    return rows.map((p) => ({ label: p.day.slice(5), value: p[measure] }));
  }, [data, category, measure]);

  // The history starts the day this shipped, so for a while there is
  // genuinely nothing to draw. Say that plainly rather than showing an
  // empty chart that reads as a broken one.
  if (isError) return null;
  if (isLoading) return null;
  if (points.length < 2) {
    return (
      <div className="card trend-panel">
        <div className="trend-head">
          <h4>Over time</h4>
        </div>
        <p className="trend-empty">
          {points.length === 0
            ? "No history recorded yet."
            : "One day recorded so far — a trend needs at least two."}{" "}
          Benchmarks are captured once a day; the line appears from tomorrow.
        </p>
      </div>
    );
  }

  const active = MEASURES.find((m) => m.key === measure) ?? MEASURES[0];

  return (
    <div className="card trend-panel">
      <div className="trend-head">
        <h4>Over time</h4>
        <div className="trend-controls">
          <select
            className="trend-select"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            aria-label="Category"
          >
            {categories.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
          <div className="trend-measures" role="radiogroup" aria-label="Measure">
            {MEASURES.map((m) => (
              <button
                key={m.key}
                type="button"
                role="radio"
                aria-checked={measure === m.key}
                className={`trend-measure${measure === m.key ? " trend-measure-active" : ""}`}
                onClick={() => setMeasure(m.key)}
              >
                {m.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      <AreaChart points={points} formatValue={active.format} height={180} interactive />
      <p className="trend-note">{active.note}</p>
    </div>
  );
}
