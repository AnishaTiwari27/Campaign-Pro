// Metric formulas mirrored 1:1 from api/internal/domain/metrics.go. The
// backend already returns pace/cpm/index/categoryIndex on every campaign,
// so the frontend only needs to reconstruct things the API doesn't
// precompute — chiefly a single campaign's reach curve for its chart.

export type CurveShape = "fast" | "steady" | "slow";

const CURVE_WEIGHTS: Record<CurveShape, number[]> = {
  fast: [0.1, 0.34, 0.55, 0.7, 0.81, 0.89, 0.95, 1],
  steady: [0.08, 0.2, 0.33, 0.46, 0.59, 0.72, 0.86, 1],
  slow: [0.04, 0.09, 0.17, 0.28, 0.42, 0.6, 0.79, 1],
};

export function reachCurve(reach: number, shape: CurveShape): number[] {
  const weights = CURVE_WEIGHTS[shape] ?? CURVE_WEIGHTS.steady;
  return weights.map((w) => reach * w);
}

// valueAtAge interpolates a cumulative value along an 8-point curve at
// `age` days into a daysRunning-length flight.
export function valueAtAge(total: number, shape: CurveShape, daysRunning: number, age: number): number {
  if (daysRunning <= 0 || age <= 0) return 0;
  if (age >= daysRunning) return total;
  const curve = reachCurve(total, shape);
  const frac = age / daysRunning;
  const pos = frac * 7;
  const lo = Math.min(6, Math.floor(pos));
  const hi = lo + 1;
  const t = pos - lo;
  return curve[lo] + (curve[hi] - curve[lo]) * t;
}

export type PaceClass = "over" | "warn" | "good";

export function paceOf(spend: number, budget: number): number {
  if (budget === 0) return 0;
  return Math.round((spend / budget) * 100);
}

export function paceClassOf(pace: number): PaceClass {
  if (pace >= 100) return "over";
  if (pace >= 85) return "warn";
  return "good";
}

// The two budget thresholds that block an approval. Mirrors the Go constants
// in services/campaigns/metrics.go — the server is the enforcement, this is
// only so the UI can explain itself before the click rather than after it.
export const PACE_OVER_BUDGET = 100;
export const PACE_NEARLY_EXHAUSTED = 95;

// approvalBlockOf mirrors campaigns.ApprovalBlock: the reason this campaign
// cannot be approved, or null when nothing blocks it. Kept short — the server
// returns the full sentence; this is button and tooltip text.
export function approvalBlockOf(spend: number, budget: number): string | null {
  const pace = paceOf(spend, budget);
  if (pace >= PACE_OVER_BUDGET) return `Spend has reached ${pace}% of budget`;
  if (pace >= PACE_NEARLY_EXHAUSTED) return `Budget is ${pace}% spent and runs out within days`;
  return null;
}

export function cpmOf(spend: number, reachLakh: number, frequency: number): number {
  const impressions = reachLakh * 100000 * frequency;
  if (impressions <= 0) return 0;
  return spend / (impressions / 1000);
}

export function median(values: number[]): number {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const mid = Math.floor(sorted.length / 2);
  if (sorted.length % 2 === 0) return (sorted[mid - 1] + sorted[mid]) / 2;
  return sorted[mid];
}

export function indexOf(reach: number, med: number): number {
  if (med === 0) return 0;
  return reach / med;
}
