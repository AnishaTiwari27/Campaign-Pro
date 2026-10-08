// The pace/index quadrant: where each campaign sits when delivery is read
// against spend. Both inputs already exist on every campaign — `pace` is
// spend as a percentage of budget, `index` is reach against the
// all-campaign median — so this file only decides what a pair of them
// *means*, and nothing here recomputes either.

import type { Campaign } from "../api/types";

/** Fully spent. At or past this, more budget is a decision, not a drift. */
export const PACE_MIDLINE = 100;

/** 1.0x is exactly typical: the all-campaign median reach (MED_ALL). */
export const INDEX_MIDLINE = 1;

export type QuadrantKey = "scale" | "protect" | "fix" | "cut";

export interface Quadrant {
  key: QuadrantKey;
  /** The verb, because the point of this view is the decision. */
  label: string;
  /** What being here actually means, in one line. */
  blurb: string;
  /** Maps to the app's good/warn/crit tokens. */
  tone: "good" | "accent" | "warn" | "crit";
}

export const QUADRANTS: Record<QuadrantKey, Quadrant> = {
  scale: {
    key: "scale",
    label: "Scale",
    blurb: "Beating the median on less than its budget. The cheapest reach you can buy is more of this.",
    tone: "good",
  },
  protect: {
    key: "protect",
    label: "Protect",
    blurb: "Beating the median and spending to plan. Working as intended — leave it alone.",
    tone: "accent",
  },
  fix: {
    key: "fix",
    label: "Fix",
    blurb: "Below the median, but budget is still unspent. Still time to change the creative or the targeting.",
    tone: "warn",
  },
  cut: {
    key: "cut",
    label: "Cut",
    blurb: "Below the median and the budget is gone. Every further rupee buys less than the fleet average.",
    tone: "crit",
  },
};

/** Fixed order: best to worst, so every list and legend agrees. */
export const QUADRANT_ORDER: QuadrantKey[] = ["scale", "protect", "fix", "cut"];

/**
 * Which quadrant a campaign falls in.
 *
 * Both boundaries are inclusive upward: index exactly 1.0 counts as
 * meeting the median rather than missing it, and pace exactly 100 counts
 * as fully spent. A campaign sitting precisely on the line is the
 * uninteresting case, and reading it the generous way for delivery and
 * the strict way for budget keeps the "Cut" quadrant free of campaigns
 * that have not actually underperformed.
 */
export function quadrantOf(pace: number, index: number): QuadrantKey {
  const delivering = index >= INDEX_MIDLINE;
  const spent = pace >= PACE_MIDLINE;
  if (delivering) return spent ? "protect" : "scale";
  return spent ? "cut" : "fix";
}

export interface SignalPoint {
  campaign: Campaign;
  pace: number;
  index: number;
  quadrant: QuadrantKey;
}

/**
 * Scheduled campaigns are excluded, mirroring the backend's Running():
 * nothing has been delivered or spent yet, so a pace of 0 against an
 * index of 0 would plant every unstarted campaign in "Fix" and drown the
 * quadrant that is supposed to be a to-do list.
 */
export function signalPoints(campaigns: Campaign[]): SignalPoint[] {
  return campaigns
    .filter((c) => c.status !== "scheduled")
    .map((c) => ({
      campaign: c,
      pace: c.pace,
      index: c.index,
      quadrant: quadrantOf(c.pace, c.index),
    }));
}

export interface QuadrantSummary extends Quadrant {
  points: SignalPoint[];
  /** Budget sitting in this quadrant — the number that makes it a decision. */
  budget: number;
  spend: number;
}

export function summarise(points: SignalPoint[]): QuadrantSummary[] {
  return QUADRANT_ORDER.map((key) => {
    const mine = points.filter((p) => p.quadrant === key);
    return {
      ...QUADRANTS[key],
      points: mine,
      budget: mine.reduce((sum, p) => sum + p.campaign.budget, 0),
      spend: mine.reduce((sum, p) => sum + p.campaign.spend, 0),
    };
  });
}
